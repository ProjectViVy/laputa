// Package hybrid provides hybrid search combining BM25 lexical search
// with vector similarity search for improved recall.
package hybrid

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/dashimaki/mentle/internal/bm25"
	"github.com/dashimaki/mentle/internal/palace"
	"github.com/dashimaki/mentle/internal/search"
	govector "github.com/dashimaki/mentle/storage/govector"
)

// Store defines the interface for hybrid search combining
// vector similarity (govector) and lexical matching (BM25).
type Store interface {
	search.Store
	BM25Search(query string, limit int) ([]bm25.ScoredDoc, error)
	BM25Index(docID, content string)
	BM25Remove(docID string)
}

// Searcher wraps a vector store and BM25 index to provide
// hybrid search with score fusion.
type Searcher struct {
	store    search.Store
	embedder search.Embedder
	bm25     *bm25.Index

	// alpha controls the weight of vector similarity (0.0-1.0).
	// alpha=1.0: pure vector search, alpha=0.0: pure BM25.
	alpha float64
}

// ScoredDrawer preserves the fused score and retrieval channels used by
// higher-level orchestration without exposing storage implementation types.
type ScoredDrawer struct {
	Drawer   search.Drawer
	Score    float64
	Channels []string
}

// NewSearcher creates a hybrid searcher.
// alpha controls vector search weight (1.0 = pure vector, 0.0 = pure BM25).
func NewSearcher(store search.Store, embedder search.Embedder, alpha float64) *Searcher {
	if alpha < 0 || alpha > 1 {
		alpha = 0.7 // Default: 70% vector, 30% BM25.
	}
	return &Searcher{
		store:    store,
		embedder: embedder,
		bm25:     bm25.New(bm25.DefaultK1, bm25.DefaultB),
		alpha:    alpha,
	}
}

// BM25 returns the underlying BM25 index for direct access.
func (s *Searcher) BM25() *bm25.Index {
	return s.bm25
}

// Search performs hybrid search combining vector similarity and BM25.
func (s *Searcher) Search(ctx context.Context, query string, wing, room string, nResults int) ([]search.Drawer, error) {
	scored, err := s.SearchScored(ctx, query, wing, room, nResults)
	if err != nil {
		return nil, err
	}
	drawers := make([]search.Drawer, 0, len(scored))
	for _, result := range scored {
		drawers = append(drawers, result.Drawer)
	}
	return drawers, nil
}

// SearchScored performs hybrid search and retains RRF score provenance.
func (s *Searcher) SearchScored(ctx context.Context, query string, wing, room string, nResults int) ([]ScoredDrawer, error) {
	// 1. Vector search.
	vector, err := s.embedder.CreateEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}

	filter := map[string]any{}
	if wing != "" {
		filter["wing"] = wing
	}
	if room != "" {
		filter["room"] = room
	}

	vectorResults, err := s.store.Search(vector, nResults*3, filter) // Fetch more for fusion.
	if err != nil {
		return nil, err
	}

	// 2. BM25 search (post-filter by wing/room).
	bm25Results := s.bm25.Search(query, nResults*3)

	// 3. Apply BM25 filters (wing/room) if specified.
	if wing != "" || room != "" {
		bm25Results = s.filterBM25Results(bm25Results, filter)
	}

	// 4. Fuse scores using Reciprocal Rank Fusion (RRF).
	fused := s.fuseScores(vectorResults, bm25Results, nResults)

	// 5. Convert to drawers.
	drawers := make([]ScoredDrawer, 0, len(fused))
	for _, f := range fused {
		d := search.Drawer{
			ID:       f.ID,
			Metadata: map[string]string{},
		}
		for k, v := range f.Payload {
			if strVal, ok := v.(string); ok {
				d.Metadata[k] = strVal
			}
		}
		if wingVal, ok := f.Payload["wing"].(string); ok {
			d.Wing = wingVal
		}
		if roomVal, ok := f.Payload["room"].(string); ok {
			d.Room = roomVal
		}
		if contentVal, ok := f.Payload["content"].(string); ok {
			d.Content = contentVal
		}
		drawers = append(drawers, ScoredDrawer{Drawer: d, Score: f.Score, Channels: f.Channels})
	}

	return drawers, nil
}

// Store adds a drawer to both the vector store and BM25 index.
func (s *Searcher) Store(ctx context.Context, drawer palace.Drawer) error {
	vector, err := s.embedder.CreateEmbedding(ctx, drawer.Content)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"wing":    drawer.Wing,
		"room":    drawer.Room,
		"source":  drawer.SourceFile,
		"content": drawer.Content,
	}
	for key, value := range drawer.Metadata {
		if _, reserved := payload[key]; !reserved {
			payload[key] = value
		}
	}

	if err := s.store.Add(drawer.ID, vector, payload); err != nil {
		return err
	}

	// Index content in BM25 with payload for filtering.
	s.bm25.AddWithPayload(drawer.ID, drawer.Content, payload)
	return nil
}

// StoreVectors stores pre-computed embeddings and indexes content in BM25.
func (s *Searcher) StoreVectors(ids []string, vectors [][]float32, payloads []map[string]any) error {
	if len(ids) != len(vectors) || len(ids) != len(payloads) {
		return fmt.Errorf("mismatched lengths: ids=%d vectors=%d payloads=%d", len(ids), len(vectors), len(payloads))
	}

	points := make([]govector.Point, len(ids))
	for i := range ids {
		points[i] = govector.Point{
			ID:      ids[i],
			Vector:  vectors[i],
			Payload: payloads[i],
		}
		// Index content in BM25 with payload for filtering.
		if content, ok := payloads[i]["content"].(string); ok {
			s.bm25.AddWithPayload(ids[i], content, payloads[i])
		}
	}

	return s.store.AddBatch(points)
}

// Delete removes a document from both stores.
func (s *Searcher) Delete(ctx context.Context, id string) error {
	if err := s.store.Delete(id); err != nil {
		return err
	}
	s.bm25.Remove(id)
	return nil
}

// RebuildBM25Index rebuilds the BM25 index from all documents in the vector store.
// Call this when initializing hybrid search on an existing database.
func (s *Searcher) RebuildBM25Index(ctx context.Context) error {
	results, err := s.store.ListAll(50000)
	if err != nil {
		return err
	}

	for _, r := range results {
		if content, ok := r.Payload["content"].(string); ok {
			s.bm25.AddWithPayload(r.ID, content, r.Payload)
		}
	}

	return nil
}

// ListWings delegates to the underlying store.
func (s *Searcher) ListWings(ctx context.Context) ([]search.WingInfo, error) {
	results, err := s.store.ListAll(10000)
	if err != nil {
		return nil, err
	}

	wingCounts := make(map[string]int)
	for _, r := range results {
		if wing, ok := r.Payload["wing"].(string); ok {
			wingCounts[wing]++
		}
	}

	var wings []search.WingInfo
	for wing, count := range wingCounts {
		wings = append(wings, search.WingInfo{Name: wing, DrawerCount: count})
	}
	return wings, nil
}

// ListRooms delegates to the underlying store.
func (s *Searcher) ListRooms(ctx context.Context, wingFilter string) ([]search.RoomInfo, error) {
	results, err := s.store.ListAll(10000)
	if err != nil {
		return nil, err
	}

	type roomKey struct {
		Wing string
		Room string
	}
	roomCounts := make(map[roomKey]int)
	for _, r := range results {
		wing, wingOk := r.Payload["wing"].(string)
		room, roomOk := r.Payload["room"].(string)
		if wingOk && roomOk {
			if wingFilter == "" || wing == wingFilter {
				roomCounts[roomKey{Wing: wing, Room: room}]++
			}
		}
	}

	var rooms []search.RoomInfo
	for key, count := range roomCounts {
		rooms = append(rooms, search.RoomInfo{
			Name:        key.Room,
			Wing:        key.Wing,
			DrawerCount: count,
		})
	}
	return rooms, nil
}

// GetTaxonomy delegates to the underlying store.
func (s *Searcher) GetTaxonomy(ctx context.Context) (map[string]*search.TaxonomyNode, error) {
	results, err := s.store.ListAll(10000)
	if err != nil {
		return nil, err
	}

	taxonomy := make(map[string]*search.TaxonomyNode)
	for _, r := range results {
		wing, wingOk := r.Payload["wing"].(string)
		room, roomOk := r.Payload["room"].(string)
		if !wingOk || !roomOk {
			continue
		}

		if _, exists := taxonomy[wing]; !exists {
			taxonomy[wing] = &search.TaxonomyNode{
				Name:  wing,
				Rooms: make(map[string]*search.TaxonomyNode),
			}
		}

		if _, exists := taxonomy[wing].Rooms[room]; !exists {
			taxonomy[wing].Rooms[room] = &search.TaxonomyNode{
				Name:  room,
				Count: 0,
			}
		}
		taxonomy[wing].Rooms[room].Count++
		taxonomy[wing].Count++
	}

	return taxonomy, nil
}

// fuseScores combines vector and BM25 results using Reciprocal Rank Fusion.
// RRF is robust to different score scales: RRF(d) = Σ 1 / (k + rank(d))
func (s *Searcher) fuseScores(
	vectorResults []govector.SearchResult,
	bm25Results []bm25.ScoredDoc,
	limit int,
) []fusedDoc {
	const rrfK = 60.0 // Standard RRF constant.

	type entry struct {
		payload    map[string]any
		vectorRank int
		bm25Rank   int
	}

	// Track all unique docs.
	entries := make(map[string]*entry)

	// Add vector results.
	for i, r := range vectorResults {
		e := &entry{payload: r.Payload, vectorRank: i + 1, bm25Rank: 0}
		entries[r.ID] = e
	}

	// Add BM25 results.
	for i, r := range bm25Results {
		e, exists := entries[r.ID]
		if exists {
			e.bm25Rank = i + 1
		} else {
			entries[r.ID] = &entry{payload: r.Payload, vectorRank: 0, bm25Rank: i + 1}
		}
	}

	// Compute RRF scores.
	docs := make([]fusedDoc, 0, len(entries))
	for id, e := range entries {
		var vectorScore, bm25Score float64
		if e.vectorRank > 0 {
			vectorScore = 1.0 / (rrfK + float64(e.vectorRank))
		}
		if e.bm25Rank > 0 {
			bm25Score = 1.0 / (rrfK + float64(e.bm25Rank))
		}

		// Weighted combination.
		finalScore := s.alpha*vectorScore + (1.0-s.alpha)*bm25Score

		channels := make([]string, 0, 2)
		if e.vectorRank > 0 {
			channels = append(channels, "vector")
		}
		if e.bm25Rank > 0 {
			channels = append(channels, "bm25")
		}
		docs = append(docs, fusedDoc{
			ID:       id,
			Score:    finalScore,
			Payload:  e.payload,
			Channels: channels,
		})
	}

	// Sort by final score.
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].Score > docs[j].Score
	})

	if limit > 0 && len(docs) > limit {
		docs = docs[:limit]
	}

	return docs
}

type fusedDoc struct {
	ID       string
	Score    float64
	Payload  map[string]any
	Channels []string
}

// filterBM25Results applies wing/room filters to BM25 results by looking up
// payloads from the vector store. Since BM25 only knows docIDs, we need
// to cross-reference with the vector results.
func (s *Searcher) filterBM25Results(results []bm25.ScoredDoc, filter map[string]any) []bm25.ScoredDoc {
	if len(filter) == 0 {
		return results
	}

	// Build a lookup of docID → payload from vector results for fast access.
	// Note: This only works for docs that also appeared in vector results.
	// BM25-only matches can't be filtered by payload fields.

	var filtered []bm25.ScoredDoc
	for _, r := range results {
		if matchesFilter(r, filter) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// matchesFilter checks if a BM25 scored doc matches the filter criteria.
func matchesFilter(doc bm25.ScoredDoc, filter map[string]any) bool {
	payload := doc.Payload
	if payload == nil {
		payload = map[string]any{}
	}

	for key, val := range filter {
		condMap, ok := val.(map[string]any)
		if !ok {
			// Simple exact match.
			if docVal, exists := payload[key]; !exists || docVal != val {
				return false
			}
			continue
		}

		// Check $in.
		if inVals, has := condMap["$in"].([]any); has && len(inVals) > 0 {
			docVal, exists := payload[key]
			if !exists {
				return false
			}
			found := slices.Contains(inVals, docVal)
			if !found {
				return false
			}
		}

		// Check $nin.
		if ninVals, has := condMap["$nin"].([]any); has {
			docVal, exists := payload[key]
			if exists {
				if slices.Contains(ninVals, docVal) {
					return false
				}
			}
		}
	}
	return true
}
