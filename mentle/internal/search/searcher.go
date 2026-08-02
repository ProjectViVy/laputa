// Package search provides vector similarity search for memory retrieval.
// It uses embeddings to find relevant drawers in the palace.
package search

import (
	"context"
	"fmt"

	"github.com/dashimaki/mentle/internal/palace"
	"github.com/dashimaki/mentle/storage/govector"
)

type Store interface {
	Search(query []float32, limit int, filter map[string]any) ([]govector.SearchResult, error)
	Add(id string, vector []float32, payload map[string]any) error
	AddBatch(points []govector.Point) error
	Delete(id string) error
	ListAll(limit int) ([]govector.SearchResult, error)
	Close() error
}

type Embedder interface {
	CreateEmbedding(ctx context.Context, text string) ([]float32, error)
	CreateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

type LlamaClient interface {
	CreateEmbedding(ctx context.Context, text string) ([]float32, error)
}

type Drawer struct {
	ID       string
	Wing     string
	Room     string
	Content  string
	Metadata map[string]string
}

type WingInfo struct {
	Name        string `json:"name"`
	DrawerCount int    `json:"drawer_count"`
}

type RoomInfo struct {
	Name        string `json:"name"`
	Wing        string `json:"wing"`
	DrawerCount int    `json:"drawer_count"`
}

type TaxonomyNode struct {
	Name  string                   `json:"name"`
	Rooms map[string]*TaxonomyNode `json:"rooms,omitempty"`
	Count int                      `json:"count"`
}

type Searcher struct {
	store    Store
	embedder Embedder
}

func NewSearcher(store Store, embedder Embedder) *Searcher {
	return &Searcher{store: store, embedder: embedder}
}

func (s *Searcher) Search(ctx context.Context, query string, wing, room string, nResults int) ([]Drawer, error) {
	filter := map[string]any{}
	if wing != "" {
		filter["wing"] = wing
	}
	if room != "" {
		filter["room"] = room
	}
	return s.SearchWithFilter(ctx, query, filter, nResults)
}

// SearchOptions provides advanced filtering with $in/$nin operators.
type SearchOptions struct {
	// Wings filters by wing — single value for exact match, slice for $in.
	Wings []string
	// Rooms filters by room — single value for exact match, slice for $in.
	Rooms []string
	// ExcludeWings applies $nin on the wing field.
	ExcludeWings []string
	// ExcludeRooms applies $nin on the room field.
	ExcludeRooms []string
}

// SearchWithOptions performs search with advanced filter options.
func (s *Searcher) SearchWithOptions(ctx context.Context, query string, opts SearchOptions, nResults int) ([]Drawer, error) {
	filter := buildFilterFromOptions(opts)
	return s.SearchWithFilter(ctx, query, filter, nResults)
}

// SearchWithFilter performs search with a raw filter map supporting $in/$nin.
func (s *Searcher) SearchWithFilter(ctx context.Context, query string, filter map[string]any, nResults int) ([]Drawer, error) {
	vector, err := s.embedder.CreateEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}

	results, err := s.store.Search(vector, nResults, filter)
	if err != nil {
		return nil, err
	}

	var drawers []Drawer
	for _, r := range results {
		d := Drawer{
			ID:       r.ID,
			Metadata: map[string]string{},
		}
		// Copy all string payload fields into Metadata so callers can access
		// any metadata stored alongside the vector (e.g. corpus_id, session_id).
		for k, v := range r.Payload {
			if strVal, ok := v.(string); ok {
				d.Metadata[k] = strVal
			}
		}
		if wingVal, ok := r.Payload["wing"].(string); ok {
			d.Wing = wingVal
		}
		if roomVal, ok := r.Payload["room"].(string); ok {
			d.Room = roomVal
		}
		if contentVal, ok := r.Payload["content"].(string); ok {
			d.Content = contentVal
		}
		drawers = append(drawers, d)
	}
	return drawers, nil
}

// buildFilterFromOptions converts SearchOptions into a filter map.
func buildFilterFromOptions(opts SearchOptions) map[string]any {
	filter := map[string]any{}

	if len(opts.Wings) == 1 {
		filter["wing"] = opts.Wings[0]
	} else if len(opts.Wings) > 1 {
		vals := make([]any, len(opts.Wings))
		for i, v := range opts.Wings {
			vals[i] = v
		}
		filter["wing"] = map[string]any{"$in": vals}
	}

	if len(opts.Rooms) == 1 {
		filter["room"] = opts.Rooms[0]
	} else if len(opts.Rooms) > 1 {
		vals := make([]any, len(opts.Rooms))
		for i, v := range opts.Rooms {
			vals[i] = v
		}
		filter["room"] = map[string]any{"$in": vals}
	}

	if len(opts.ExcludeWings) > 0 {
		vals := make([]any, len(opts.ExcludeWings))
		for i, v := range opts.ExcludeWings {
			vals[i] = v
		}
		filter["wing"] = mergeWithNin(filter["wing"], vals)
	}

	if len(opts.ExcludeRooms) > 0 {
		vals := make([]any, len(opts.ExcludeRooms))
		for i, v := range opts.ExcludeRooms {
			vals[i] = v
		}
		filter["room"] = mergeWithNin(filter["room"], vals)
	}

	return filter
}

// mergeWithNin adds $nin to an existing filter value (exact or $in).
func mergeWithNin(existing any, ninVals []any) map[string]any {
	result := map[string]any{}

	switch v := existing.(type) {
	case string:
		result["$in"] = []any{v}
	case map[string]any:
		if inVals, ok := v["$in"]; ok {
			result["$in"] = inVals
		}
	}

	result["$nin"] = ninVals
	return result
}

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

	return s.store.Add(drawer.ID, vector, payload)
}

// StoreVectors stores pre-computed embeddings with payloads.
func (s *Searcher) StoreVectors(ids []string, vectors [][]float32, payloads []map[string]any) error {
	if len(ids) != len(vectors) || len(ids) != len(payloads) {
		return fmt.Errorf("mismatched lengths: ids=%d, vectors=%d, payloads=%d",
			len(ids), len(vectors), len(payloads))
	}
	points := make([]govector.Point, len(ids))
	for i := range ids {
		points[i] = govector.Point{
			ID:      ids[i],
			Vector:  vectors[i],
			Payload: payloads[i],
		}
	}
	return s.store.AddBatch(points)
}

func (s *Searcher) ListWings(ctx context.Context) ([]WingInfo, error) {
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

	var wings []WingInfo
	for wing, count := range wingCounts {
		wings = append(wings, WingInfo{Name: wing, DrawerCount: count})
	}
	return wings, nil
}

func (s *Searcher) ListRooms(ctx context.Context, wingFilter string) ([]RoomInfo, error) {
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

	var rooms []RoomInfo
	for key, count := range roomCounts {
		rooms = append(rooms, RoomInfo{
			Name:        key.Room,
			Wing:        key.Wing,
			DrawerCount: count,
		})
	}
	return rooms, nil
}

func (s *Searcher) GetTaxonomy(ctx context.Context) (map[string]*TaxonomyNode, error) {
	results, err := s.store.ListAll(10000)
	if err != nil {
		return nil, err
	}

	taxonomy := make(map[string]*TaxonomyNode)
	for _, r := range results {
		wing, wingOk := r.Payload["wing"].(string)
		room, roomOk := r.Payload["room"].(string)
		if !wingOk || !roomOk {
			continue
		}

		if _, exists := taxonomy[wing]; !exists {
			taxonomy[wing] = &TaxonomyNode{
				Name:  wing,
				Rooms: make(map[string]*TaxonomyNode),
			}
		}

		if _, exists := taxonomy[wing].Rooms[room]; !exists {
			taxonomy[wing].Rooms[room] = &TaxonomyNode{
				Name:  room,
				Count: 0,
			}
		}
		taxonomy[wing].Rooms[room].Count++
		taxonomy[wing].Count++
	}

	return taxonomy, nil
}

func (s *Searcher) Delete(ctx context.Context, id string) error {
	return s.store.Delete(id)
}

func (s *Searcher) Close() error { return s.store.Close() }

// ListAll returns stored drawers without performing semantic retrieval.
func (s *Searcher) ListAll(ctx context.Context, limit int) ([]Drawer, error) {
	if limit <= 0 {
		limit = 10000
	}
	results, err := s.store.ListAll(limit)
	if err != nil {
		return nil, err
	}
	drawers := make([]Drawer, 0, len(results))
	for _, r := range results {
		d := Drawer{ID: r.ID, Metadata: map[string]string{}}
		for k, v := range r.Payload {
			if value, ok := v.(string); ok {
				d.Metadata[k] = value
			}
		}
		d.Wing, _ = r.Payload["wing"].(string)
		d.Room, _ = r.Payload["room"].(string)
		d.Content, _ = r.Payload["content"].(string)
		drawers = append(drawers, d)
	}
	return drawers, nil
}
