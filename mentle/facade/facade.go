// Package facade exposes a unified entry point for mentle memory services.
package facade

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	"github.com/dashimaki/mentle/storage/sqlite"
)

// Options configures facade initialization.
type Options struct {
	ConfigDir  string
	PalacePath string
	ModelsDir  string
	// RequireLocalModel disables cwd fallback and downloads; ModelsDir must be explicit.
	RequireLocalModel bool
	// LexicalOnly opens an existing canonical catalog read-only and builds only
	// an in-memory BM25 projection; it never loads or downloads a model.
	LexicalOnly bool
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
	lexicalOnly       bool
	mutationMu        sync.Mutex
}

// Init loads config and wires the same components as cmd/server.
func (s *Service) Init(ctx context.Context, opts Options) error {
	if opts.LexicalOnly {
		if opts.RequireLocalModel {
			return fmt.Errorf("LexicalOnly and RequireLocalModel cannot both be enabled")
		}
		return s.initLexicalOnly(ctx, opts)
	}
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
	var emb *embedder.Embedder
	var err error
	if opts.RequireLocalModel {
		// Strict mode uses the caller's explicit directory, never config defaults.
		emb, err = embedder.NewLocal("", opts.ModelsDir)
	} else {
		emb, err = embedder.New("", modelsDir)
	}
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
		vectorDB.Close()
		emb.Close()
		return fmt.Errorf("knowledge graph: %w", err)
	}

	searcher := search.NewSearcher(vectorDB, emb)
	hybridSearcher := hybrid.NewSearcher(vectorDB, emb, 0.7)
	stack := layers.NewMemoryStack(cfg, searcher)

	taxonomy, err := searcher.GetTaxonomy(ctx)
	if err != nil {
		kgDB.Close()
		vectorDB.Close()
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
		vectorDB.Close()
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

// IsReadOnly reports whether this service is a lexical-only projection
// (read paths open, canonical writes rejected).
func (s *Service) IsReadOnly() bool { return s != nil && s.lexicalOnly }

// OpenCatalogService opens a canonical-authority service without a local
// embedding model: writes are accepted into canonical SQLite, retrieval is
// lexical (BM25) rebuilt from canonical state, vector search is
// unavailable. It is the supported fixture/development seam for
// model-free deployments — not a silent degradation of a configured model.
func OpenCatalogService(ctx context.Context, palacePath string) (*Service, error) {
	if err := os.MkdirAll(palacePath, 0700); err != nil {
		return nil, err
	}
	s := &Service{PalacePath: palacePath}
	catalog, err := OpenCatalog(palacePath + "/canonical.sqlite3")
	if err != nil {
		return nil, err
	}
	s.Catalog = catalog
	s.Hybrid = hybrid.NewSearcher(unavailableVectorStore{}, nil, 0)
	snapshot, err := s.readCanonicalSnapshot(ctx)
	if err != nil {
		s.Close()
		return nil, err
	}
	s.Hybrid.RebuildBM25FromDrawers(snapshot.Drawers)
	return s, nil
}

// initLexicalOnly never opens models or persistent derived indexes and does not
// run the index outbox. Canonical SQLite must already exist and is opened with
// SQLite's read-only mode so even unguarded maintenance cannot mutate it.
func (s *Service) initLexicalOnly(ctx context.Context, opts Options) error {
	if strings.TrimSpace(opts.PalacePath) == "" {
		return fmt.Errorf("lexical-only requires an explicit palace path")
	}
	palacePath := expandPalacePath(opts.PalacePath)
	canonicalPath := filepath.Join(palacePath, "canonical.sqlite3")
	if info, err := os.Stat(canonicalPath); err != nil {
		return fmt.Errorf("canonical catalog: %w", err)
	} else if info.IsDir() {
		return fmt.Errorf("canonical catalog is a directory: %s", canonicalPath)
	}
	db, err := sqlite.OpenReadOnly(canonicalPath)
	if err != nil {
		return fmt.Errorf("canonical catalog: %w", err)
	}
	s.Catalog = &Catalog{db: db}
	s.Hybrid = hybrid.NewSearcher(unavailableVectorStore{}, nil, 0)
	s.PalacePath = palacePath
	s.lexicalOnly = true
	snapshot, err := s.readCanonicalSnapshot(ctx)
	if err != nil {
		_ = s.Close()
		return fmt.Errorf("canonical snapshot for lexical index: %w", err)
	}
	s.Hybrid.RebuildBM25FromDrawers(snapshot.Drawers)
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
	}
	s.Hybrid = nil
	s.lexicalOnly = false
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
