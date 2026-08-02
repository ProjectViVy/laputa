// Example: Hybrid search combining BM25 + vector similarity.
// This shows how to replace the standard searcher with a hybrid searcher.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dashimaki/mentle/internal/embedder"
	"github.com/dashimaki/mentle/internal/hybrid"
	govector "github.com/dashimaki/mentle/storage/govector"
)

func main() {
	ctx := context.Background()
	palacePath := os.ExpandEnv("$HOME/.mempalace")

	// 1. Create embedder (same as before).
	emb, err := embedder.New("", palacePath+"/models")
	if err != nil {
		panic(err)
	}
	defer emb.Close()

	// 2. Create govector store (same as before).
	store, err := govector.NewStore(palacePath+"/vectors.db", 384)
	if err != nil {
		panic(err)
	}
	defer store.Close()

	// 3. Create HYBRID searcher (alpha=0.7 means 70% vector, 30% BM25).
	hybridSearcher := hybrid.NewSearcher(store, emb, 0.7)

	// 4. If you have existing data, rebuild the BM25 index from the vector store.
	if err := hybridSearcher.RebuildBM25Index(ctx); err != nil {
		panic(err)
	}

	// 5. Search — now combines vector similarity + lexical matching.
	results, err := hybridSearcher.Search(ctx, "authentication system design", "", "", 5)
	if err != nil {
		panic(err)
	}

	for _, r := range results {
		fmt.Printf("[%s/%s] %s\n", r.Wing, r.Room, r.Content[:min(100, len(r.Content))])
	}
}

// --- Migration Guide ---
//
// BEFORE (pure vector search):
//
//	store, _ := govector.NewStore(path, 384)
//	searcher := search.NewSearcher(store, embedder)
//
// AFTER (hybrid search):
//
//	store, _ := govector.NewStore(path, 384)
//	hybridSearcher := hybrid.NewSearcher(store, embedder, 0.7)
//	hybridSearcher.RebuildBM25Index(ctx)  // One-time rebuild on startup.
//
// The hybrid searcher has the same interface as the standard searcher,
// so the rest of your code remains unchanged.
//
// --- Alpha Tuning ---
//
// alpha=1.0: Pure vector search (semantic only)
// alpha=0.7: 70% vector, 30% BM25 (recommended default)
// alpha=0.5: Equal weight (balanced)
// alpha=0.3: Mostly BM25 (lexical focused)
// alpha=0.0: Pure BM25 (keyword only)
//
// --- Why Reciprocal Rank Fusion? ---
//
// RRF combines rankings from different systems without requiring score normalization.
// This is important because:
// - Vector scores are cosine similarities (0.0-1.0)
// - BM25 scores are unbounded TF-IDF-like values
// RRF avoids the need to normalize these incompatible scales.
