package search

import (
	"context"
	"testing"

	govector "github.com/dashimaki/mentle/storage/govector"
)

type mockEmbedder struct {
	embedding []float32
	err       error
}

func (m *mockEmbedder) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.embedding, nil
}

type mockStore struct {
	results []govector.SearchResult
	err     error
}

func (m *mockStore) Add(id string, vector []float32, payload map[string]any) error {
	return nil
}

func (m *mockStore) Search(query []float32, limit int, filter map[string]any) ([]govector.SearchResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func (m *mockStore) Delete(id string) error {
	return nil
}

func (m *mockStore) ListAll(limit int) ([]govector.SearchResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func (m *mockStore) Close() error {
	return nil
}

func (m *mockStore) AddBatch(points []govector.Point) error {
	return nil
}

func (m *mockEmbedder) CreateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = m.embedding
	}
	return result, nil
}

func TestNewSearcher(t *testing.T) {
	store := &mockStore{}
	llama := &mockEmbedder{}
	searcher := NewSearcher(store, llama)
	if searcher == nil {
		t.Fatal("NewSearcher returned nil")
	}
	if searcher.store != store {
		t.Error("store not set correctly")
	}
	if searcher.embedder != llama {
		t.Error("llama not set correctly")
	}
}

func TestSearcherSearch(t *testing.T) {
	store := &mockStore{
		results: []govector.SearchResult{
			{
				ID:    "test-id",
				Score: 0.85,
				Payload: map[string]any{
					"wing": "kitchen",
					"room": "main",
				},
			},
		},
	}
	llama := &mockEmbedder{
		embedding: []float32{0.1, 0.2, 0.3, 0.4},
	}
	searcher := NewSearcher(store, llama)

	drawers, err := searcher.Search(context.Background(), "test query", "kitchen", "main", 5)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(drawers) != 1 {
		t.Fatalf("expected 1 drawer, got %d", len(drawers))
	}
	if drawers[0].ID != "test-id" {
		t.Errorf("expected ID 'test-id', got '%s'", drawers[0].ID)
	}
	if drawers[0].Wing != "kitchen" {
		t.Errorf("expected Wing 'kitchen', got '%s'", drawers[0].Wing)
	}
	if drawers[0].Room != "main" {
		t.Errorf("expected Room 'main', got '%s'", drawers[0].Room)
	}
}

func TestSearcherSearchWithEmptyFilters(t *testing.T) {
	store := &mockStore{
		results: []govector.SearchResult{},
	}
	llama := &mockEmbedder{
		embedding: []float32{0.1, 0.2, 0.3, 0.4},
	}
	searcher := NewSearcher(store, llama)

	drawers, err := searcher.Search(context.Background(), "test", "", "", 5)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(drawers) != 0 {
		t.Errorf("expected 0 drawers, got %d", len(drawers))
	}
}

func TestSearchResultFields(t *testing.T) {
	result := govector.SearchResult{
		ID:    "id-123",
		Score: 0.95,
		Payload: map[string]any{
			"wing": "bedroom",
			"room": "north",
		},
	}
	if result.ID != "id-123" {
		t.Errorf("expected ID 'id-123', got '%s'", result.ID)
	}
	if result.Score != 0.95 {
		t.Errorf("expected Score 0.95, got %f", result.Score)
	}
	if result.Payload["wing"] != "bedroom" {
		t.Errorf("expected wing 'bedroom', got '%v'", result.Payload["wing"])
	}
}

func TestDrawerMetadataPopulation(t *testing.T) {
	store := &mockStore{
		results: []govector.SearchResult{
			{
				ID:    "drawer-1",
				Score: 0.9,
				Payload: map[string]any{
					"wing": "office",
					"room": "east",
				},
			},
		},
	}
	llama := &mockEmbedder{embedding: []float32{0.5, 0.5, 0.5}}
	searcher := NewSearcher(store, llama)

	drawers, err := searcher.Search(context.Background(), "query", "", "", 1)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	d := drawers[0]
	if d.Metadata["wing"] != "office" {
		t.Errorf("expected metadata wing 'office', got '%s'", d.Metadata["wing"])
	}
	if d.Metadata["room"] != "east" {
		t.Errorf("expected metadata room 'east', got '%s'", d.Metadata["room"])
	}
	if _, ok := d.Metadata["non_existent"]; ok {
		t.Error("unexpected metadata key found")
	}
}

type mockStoreWithErr struct {
	err error
}

func (m *mockStoreWithErr) Add(id string, vector []float32, payload map[string]any) error {
	return nil
}

func (m *mockStoreWithErr) Search(query []float32, limit int, filter map[string]any) ([]govector.SearchResult, error) {
	return nil, m.err
}

func (m *mockStoreWithErr) Delete(id string) error {
	return m.err
}

func (m *mockStoreWithErr) ListAll(limit int) ([]govector.SearchResult, error) {
	return nil, m.err
}

func (m *mockStoreWithErr) Close() error {
	return nil
}

func (m *mockStoreWithErr) AddBatch(points []govector.Point) error {
	return m.err
}

func TestSearcherSearchStoreError(t *testing.T) {
	store := &mockStoreWithErr{err: context.DeadlineExceeded}
	llama := &mockEmbedder{embedding: []float32{0.1}}
	searcher := NewSearcher(store, llama)

	_, err := searcher.Search(context.Background(), "query", "", "", 5)
	if err == nil {
		t.Fatal("expected error from store")
	}
}

func TestSearcherSearchLlamaError(t *testing.T) {
	store := &mockStore{}
	llama := &mockEmbedder{err: context.DeadlineExceeded}
	searcher := NewSearcher(store, llama)

	_, err := searcher.Search(context.Background(), "query", "", "", 5)
	if err == nil {
		t.Fatal("expected error from llama")
	}
}
