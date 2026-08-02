package longmemeval

import (
	"context"
	"fmt"
	"strings"

	"github.com/dashimaki/mentle/internal/dialect"
	"github.com/dashimaki/mentle/internal/embedder"
)

// collectUniqueSessionTexts scans all entries and returns the deduplicated set
// of session texts that will need embedding. The returned slice is deterministic
// (insertion-order of first occurrence).
func collectUniqueSessionTexts(entries []Entry, mode string, encoder *dialect.Encoder) []string {
	seen := make(map[string]bool)
	var unique []string
	for _, entry := range entries {
		for sessIdx, session := range entry.HaystackSessions {
			if sessIdx >= len(entry.HaystackSessionIDs) {
				continue
			}
			var userTurns []string
			for _, turnAny := range session {
				turn, _ := turnAny.(map[string]any)
				if turn == nil {
					continue
				}
				if role, ok := turn["role"].(string); ok && role == "user" {
					if content, ok := turn["content"].(string); ok {
						userTurns = append(userTurns, content)
					}
				}
			}
			if len(userTurns) == 0 {
				continue
			}
			text := strings.Join(userTurns, " ")
			if mode == "aaak" && encoder != nil {
				text = encoder.Compress(text, map[string]string{})
			}
			if !seen[text] {
				seen[text] = true
				unique = append(unique, text)
			}
		}
	}
	return unique
}

// BuildSessionCache embeds every unique session text across all entries and
// returns a map from session text → embedding vector. Uses a pool of embedders
// to parallelise the embedding work across CPU cores.
func BuildSessionCache(ctx context.Context, embs []*embedder.Embedder, entries []Entry, mode string, encoder *dialect.Encoder) (map[string][]float32, error) {
	texts := collectUniqueSessionTexts(entries, mode, encoder)
	if len(texts) == 0 {
		return nil, nil
	}

	// Split the unique texts into per-worker chunks and embed in parallel.
	// Each worker gets a contiguous slice of texts to embed.
	workers := max(len(embs), 1)

	// Build per-worker text slices (by index range, not by content).
	type textRange struct {
		start int
		end   int
	}
	ranges := make([]textRange, workers)
	chunkSize := (len(texts) + workers - 1) / workers
	for i := range ranges {
		s := i * chunkSize
		e := min(s+chunkSize, len(texts))
		ranges[i] = textRange{start: s, end: e}
	}

	// Embed each chunk with its dedicated embedder.
	vecs, err := runWorkerPoolWithResource(ranges, embs,
		func(emb *embedder.Embedder, r textRange) ([][]float32, error) {
			if r.start >= r.end {
				return nil, nil
			}
			return emb.CreateEmbeddings(ctx, texts[r.start:r.end])
		},
	)
	if err != nil {
		return nil, fmt.Errorf("cache embed: %w", err)
	}

	// Flatten results into the cache map.
	cache := make(map[string][]float32, len(texts))
	for i, chunk := range vecs {
		r := ranges[i]
		for j, v := range chunk {
			cache[texts[r.start+j]] = v
		}
	}
	return cache, nil
}
