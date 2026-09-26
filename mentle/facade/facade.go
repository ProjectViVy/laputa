// Package facade exposes a unified entry point for mentle memory services.
package facade

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/dashimaki/mentle/internal/config"
	"github.com/dashimaki/mentle/internal/diary"
	"github.com/dashimaki/mentle/internal/embedder"
	"github.com/dashimaki/mentle/internal/hybrid"
	"github.com/dashimaki/mentle/internal/kg"
	"github.com/dashimaki/mentle/internal/layers"
	"github.com/dashimaki/mentle/internal/palace"
	"github.com/dashimaki/mentle/internal/search"
	govector "github.com/dashimaki/mentle/storage/govector"
)

// Options configures facade initialization.
type Options struct {
	ConfigDir  string
	PalacePath string
	ModelsDir  string
}

// Service aggregates mentle internal components for garden and cmd/server.
type Service struct {
	Cfg      *config.Config
	Embedder *embedder.Embedder
	// EmbeddingIdentity is the authoritative runtime identity provider. It is
	// injectable so canonical mutation and health tests can exercise identity
	// drift without loading an ONNX model.
	EmbeddingIdentity func() embedder.Identity
	Searcher          *search.Searcher
	Hybrid            *hybrid.Searcher
	Stack             *layers.MemoryStack
	KG                *kg.KnowledgeGraph
	PalaceGraph       *palace.Graph
	Diary             *diary.Diary
	PalacePath        string
	Catalog           *Catalog
	mutationMu        sync.Mutex
}

// Init loads config and wires the same components as cmd/server.
func (s *Service) Init(ctx context.Context, opts Options) error {
	var cfg *config.Config
	if opts.ConfigDir == "" && opts.PalacePath != "" && opts.ModelsDir != "" {
		// Fully specified embedded paths must not consult ambient config or env.
		defaults := config.DefaultConfig
		defaults.TopicWings = append([]string(nil), defaults.TopicWings...)
		cfg = &defaults
	} else {
		var err error
		cfg, err = config.Load(opts.ConfigDir)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}
	}

	if opts.PalacePath != "" {
		cfg.PalacePath = opts.PalacePath
	}
	if opts.ModelsDir != "" {
		cfg.ModelsDir = opts.ModelsDir
	}

	modelsDir := cfg.GetModelsDir()
	emb, err := embedder.New("", modelsDir)
	if err != nil {
		return fmt.Errorf("embedder: %w", err)
	}

	palacePath := expandPalacePath(cfg.PalacePath)
	if err := os.MkdirAll(palacePath, 0700); err != nil {
		emb.Close()
		return fmt.Errorf("create palace directory: %w", err)
	}

	vectorDB, err := govector.NewStore(palacePath+"/vectors.db", 384)
	if err != nil {
		emb.Close()
		return fmt.Errorf("vector store: %w", err)
	}

	kgDB, err := kg.New(palacePath + "/knowledge_graph.sqlite3")
	if err != nil {
		emb.Close()
		return fmt.Errorf("knowledge graph: %w", err)
	}

	searcher := search.NewSearcher(vectorDB, emb)
	hybridSearcher := hybrid.NewSearcher(vectorDB, emb, 0.7)
	stack := layers.NewMemoryStack(cfg, searcher)

	taxonomy, err := searcher.GetTaxonomy(ctx)
	if err != nil {
		kgDB.Close()
		emb.Close()
		return fmt.Errorf("taxonomy: %w", err)
	}

	palaceGraph := palace.NewGraph()
	for wingName, wingNode := range taxonomy {
		for roomName, roomNode := range wingNode.Rooms {
			for i := 0; i < roomNode.Count; i++ {
				palaceGraph.AddDrawer(wingName, roomName, "")
			}
		}
	}
	palaceGraph.BuildEdges()

	agentDiary, err := diary.New(palacePath)
	if err != nil {
		kgDB.Close()
		emb.Close()
		return fmt.Errorf("diary: %w", err)
	}

	s.Cfg = cfg
	s.Embedder = emb
	s.EmbeddingIdentity = emb.Identity
	s.Searcher = searcher
	s.Hybrid = hybridSearcher
	s.Stack = stack
	s.KG = kgDB
	s.PalaceGraph = palaceGraph
	s.Diary = agentDiary
	s.PalacePath = palacePath
	catalog, err := OpenCatalog(palacePath + "/canonical.sqlite3")
	if err != nil {
		s.Close()
		return fmt.Errorf("canonical catalog: %w", err)
	}
	s.Catalog = catalog
	// BM25 is disposable and must be rebuilt from canonical active/current
	// memories, never from an arbitrary tombstone-bearing vector listing.
	snapshot, err := s.readCanonicalSnapshot(ctx)
	if err != nil {
		s.Close()
		return fmt.Errorf("canonical snapshot for lexical index: %w", err)
	}
	hybridSearcher.RebuildBM25FromDrawers(snapshot.Drawers)
	if err := s.replayIndexJobs(ctx); err != nil {
		s.Close()
		return fmt.Errorf("canonical index recovery: %w", err)
	}
	return nil
}

// Close releases resources held by the service.
func (s *Service) Close() error {
	var closeErr error
	if s.Catalog != nil {
		closeErr = s.Catalog.Close()
		s.Catalog = nil
	}
	if s.Searcher != nil {
		closeErr = s.Searcher.Close()
		s.Searcher = nil
		s.Hybrid = nil
	}
	if s.Embedder != nil {
		s.Embedder.Close()
		s.Embedder = nil
	}
	if s.KG != nil {
		if err := s.KG.Close(); err != nil {
			if closeErr == nil {
				closeErr = err
			}
		}
		s.KG = nil
	}
	return closeErr
}

func expandPalacePath(path string) string {
	palacePath := os.ExpandEnv(path)
	if palacePath == path && strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		palacePath = strings.Replace(path, "~", home, 1)
	}
	return palacePath
}
