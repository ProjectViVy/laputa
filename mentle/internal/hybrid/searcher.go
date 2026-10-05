// Package hybrid provides hybrid search combining BM25 lexical search
// with vector similarity search for improved recall.
package hybrid

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/ProjectViVy/laputa/mentle/internal/bm25"
	"github.com/ProjectViVy/laputa/mentle/internal/palace"
	"github.com/ProjectViVy/laputa/mentle/internal/search"
	govector "github.com/ProjectViVy/laputa/mentle/storage/govector"
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
	bm25Mu   sync.RWMutex

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
// With no embedder it searches only the BM25 projection and returns BM25 scores.
func (s *Searcher) SearchScored(ctx context.Context, query string, wing, room string, nResults int) ([]ScoredDrawer, error) {
	if s.embedder == nil {
		filter := map[string]any{}
		if wing != "" {
			filter["wing"] = wing
		}
		if room != "" {
			filter["room"] = room
		}
		// Filter before limiting so out-of-scope matches cannot crowd out
		// valid lexical matches. This path never touches the vector store.
		matches := s.filterBM25Results(s.BM25Search(query, 0), filter)
		if nResults > 0 && len(matches) > nResults {
			matches = matches[:nResults]
		}
		lexical := make([]fusedDoc, 0, len(matches))
		for _, match := range matches {
			lexical = append(lexical, fusedDoc{
				ID: match.ID, Score: match.Score, Payload: match.Payload, Channels: []string{"bm25"},
			})
		}
		return scoredDrawers(lexical), nil
	}

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
	bm25Results := s.BM25Search(query, nResults*3)

	// 3. Apply BM25 filters (wing/room) if specified.
	if wing != "" || room != "" {
		bm25Results = s.filterBM25Results(bm25Results, filter)
	}

	// 4. Fuse scores using Reciprocal Rank Fusion (RRF).
	fused := s.fuseScores(vectorResults, bm25Results, nResults)

	return scoredDrawers(fused), nil
}

// scoredDrawers preserves payload fields and score provenance across retrieval paths.
func scoredDrawers(fused []fusedDoc) []ScoredDrawer {
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

	return drawers
}

// Store adds a drawer to both the vector store and BM25 index.
func (s *Searcher) Store(ctx context.Context, drawer palace.Drawer) error {
	if s == nil || s.embedder == nil || s.store == nil {
		return fmt.Errorf("hybrid: vector write unavailable without an embedder and store")
	}
	vector, err := s.embedder.CreateEmbedding(ctx, drawer.Content)
	if err != nil {
		return err
	}

	payload := drawerPayload(drawer)

	if err := s.store.Add(drawer.ID, vector, payload); err != nil {
		return err
	}

	// Index content in BM25 with payload for filtering.
	s.IndexBM25(drawer.ID, drawer.Content, payload)
	return nil
}

// Lexical reports whether this searcher has no vector path (embedder or
// store unavailable) — writes then index BM25 directly.
func (s *Searcher) Lexical() bool {
	return s == nil || s.embedder == nil || s.store == nil
}

// IndexBM25Drawer indexes one drawer into BM25 only; used when the vector
// store is unavailable.
func (s *Searcher) IndexBM25Drawer(drawer palace.Drawer) {
	s.IndexBM25(drawer.ID, drawer.Content, drawerPayload(drawer))
}

// RemoveBM25 drops one physical id from BM25; used when the vector store
// is unavailable.
func (s *Searcher) RemoveBM25(id string) {
	s.bm25Mu.Lock()
	s.bm25.Remove(id)
	s.bm25Mu.Unlock()
}

func drawerPayload(drawer palace.Drawer) map[string]any {
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
	return payload
}

// StoreVectors stores pre-computed embeddings and indexes content in BM25.
func (s *Searcher) StoreVectors(ids []string, vectors [][]float32, payloads []map[string]any) error {
	if s == nil || s.embedder == nil || s.store == nil {
		return fmt.Errorf("hybrid: vector write unavailable without an embedder and store")
	}
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
			s.IndexBM25(ids[i], content, payloads[i])
		}
	}

	return s.store.AddBatch(points)
}

// Delete removes a document from both stores.
func (s *Searcher) Delete(ctx context.Context, id string) error {
	if err := s.store.Delete(id); err != nil {
		return err
	}
	s.bm25Mu.Lock()
	s.bm25.Remove(id)
	s.bm25Mu.Unlock()
	return nil
}

// PruneCanonicalRevisions removes physical version-qualified projections for
// one canonical memory, retaining only keepID when it is non-empty. This is
// part of derived-index maintenance; it never changes canonical SQLite.
func (s *Searcher) PruneCanonicalRevisions(ctx context.Context, canonicalID, keepID string) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("vector store unavailable")
	}
	physical, err := s.store.ListAll(1_000_000)
	if err != nil {
		return err
	}
	prefix := canonicalID + "@v"
	for _, point := range physical {
		if !strings.HasPrefix(point.ID, prefix) || point.ID == keepID {
			continue
		}
		if err := s.Delete(ctx, point.ID); err != nil {
			return err
		}
	}
	return nil
}

// RebuildBM25Index rebuilds the BM25 index from the disposable vector-store
// projection. Production recovery should use the canonical Facade snapshot
// path so deleted/tombstoned rows cannot be mistaken for live authority.
// This compatibility helper has no fixed 50k truncation; callers migrating
// from the standalone hybrid example should move to Facade rebuilds.
func (s *Searcher) RebuildBM25Index(ctx context.Context) error {
	results, err := s.store.ListAll(1_000_000)
	if err != nil {
		return err
	}

	index := bm25.New(bm25.DefaultK1, bm25.DefaultB)
	for _, r := range results {
		if content, ok := r.Payload["content"].(string); ok {
			index.AddWithPayload(r.ID, content, r.Payload)
		}
	}
	s.bm25Mu.Lock()
	s.bm25 = index
	s.bm25Mu.Unlock()

	return nil
}

// BM25Search executes a lexical search against the disposable index.
func (s *Searcher) BM25Search(query string, limit int) []bm25.ScoredDoc {
	s.bm25Mu.RLock()
	defer s.bm25Mu.RUnlock()
	return s.bm25.Search(query, limit)
}

// BM25Count returns the number of documents in the disposable lexical index.
func (s *Searcher) BM25Count() int {
	s.bm25Mu.RLock()
	defer s.bm25Mu.RUnlock()
	return s.bm25.Count()
}

// IndexBM25 upserts one document into the disposable lexical index.
func (s *Searcher) IndexBM25(id, content string, payload map[string]any) {
	s.bm25Mu.Lock()
	defer s.bm25Mu.Unlock()
	s.bm25.AddWithPayload(id, content, payload)
}

// RebuildBM25FromDrawers replaces BM25 from an explicit canonical snapshot.
// The snapshot must already be bounded to active current revisions.
func (s *Searcher) RebuildBM25FromDrawers(drawers []search.Drawer) {
	index := bm25.New(bm25.DefaultK1, bm25.DefaultB)
	for _, drawer := range drawers {
		payload := make(map[string]any, len(drawer.Metadata)+3)
		for key, value := range drawer.Metadata {
			payload[key] = value
		}
		if drawer.Wing != "" {
			payload["wing"] = drawer.Wing
		}
		if drawer.Room != "" {
			payload["room"] = drawer.Room
		}
		payload["content"] = drawer.Content
		index.AddWithPayload(drawer.ID, drawer.Content, payload)
	}
	s.bm25Mu.Lock()
	s.bm25 = index
	s.bm25Mu.Unlock()
}

// RebuildVectorIndex prepares every embedding before mutating the current
// disposable vector store. It then removes physical points that are absent
// from the canonical snapshot. Canonical SQLite is never touched here.
func (s *Searcher) RebuildVectorIndex(ctx context.Context, drawers []search.Drawer) error {
	if s == nil || s.store == nil || s.embedder == nil {
		return fmt.Errorf("vector rebuild dependencies unavailable")
	}
	texts := make([]string, len(drawers))
	for i, drawer := range drawers {
		texts[i] = drawer.Content
	}
	vectors, err := s.embedder.CreateEmbeddings(ctx, texts)
	if err != nil {
		return err
	}
	if len(vectors) != len(drawers) {
		return fmt.Errorf("embedding count mismatch: got %d want %d", len(vectors), len(drawers))
	}
	points := make([]govector.Point, len(drawers))
	keep := make(map[string]struct{}, len(drawers))
	for i, drawer := range drawers {
		payload := make(map[string]any, len(drawer.Metadata)+3)
		for key, value := range drawer.Metadata {
			payload[key] = value
		}
		if drawer.Wing != "" {
			payload["wing"] = drawer.Wing
		}
		if drawer.Room != "" {
			payload["room"] = drawer.Room
		}
		payload["content"] = drawer.Content
		points[i] = govector.Point{ID: drawer.ID, Vector: vectors[i], Payload: payload}
		keep[drawer.ID] = struct{}{}
	}
	if atomicStore, ok := s.store.(interface {
		RebuildAtomically([]govector.Point) error
	}); ok {
		return atomicStore.RebuildAtomically(points)
	}
	if len(points) > 0 {
		if err := s.store.AddBatch(points); err != nil {
			return err
		}
	}
	existing, err := s.store.ListAll(1_000_000)
	if err != nil {
		return err
	}
	for _, point := range existing {
		if _, ok := keep[point.ID]; ok {
			continue
		}
		if err := s.store.Delete(point.ID); err != nil {
			return err
		}
	}
	return nil
}

// VectorPhysicalCount probes the disposable vector backend.
func (s *Searcher) VectorPhysicalCount(ctx context.Context) (int, error) {
	if counter, ok := s.store.(interface{ Count() int }); ok {
		return counter.Count(), nil
	}
	results, err := s.store.ListAll(1_000_000)
	if err != nil {
		return 0, err
	}
	return len(results), nil
}

// ListAll returns a snapshot of physical vector documents for diagnostics and
// rebuild verification. It does not grant callers mutation authority.
func (s *Searcher) ListAll(ctx context.Context, limit int) ([]search.Drawer, error) {
	if limit <= 0 {
		limit = 1_000_000
	}
	results, err := s.store.ListAll(limit)
	if err != nil {
		return nil, err
	}
	drawers := make([]search.Drawer, 0, len(results))
	for _, result := range results {
		drawer := search.Drawer{ID: result.ID, Metadata: map[string]string{}}
		for key, value := range result.Payload {
			if text, ok := value.(string); ok {
				drawer.Metadata[key] = text
			}
		}
		drawer.Wing, _ = result.Payload["wing"].(string)
		drawer.Room, _ = result.Payload["room"].(string)
		drawer.Content, _ = result.Payload["content"].(string)
		drawers = append(drawers, drawer)
	}
	return drawers, nil
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
