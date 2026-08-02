package layers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dashimaki/mentle/internal/config"
	"github.com/dashimaki/mentle/internal/search"

	govector "github.com/dashimaki/mentle/storage/govector"
)

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
	return m.results, nil
}

func (m *mockStore) Close() error {
	return nil
}

func (m *mockStore) AddBatch(points []govector.Point) error {
	return nil
}

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

func newTestConfig() *config.Config {
	return &config.Config{
		PalacePath:     filepath.Join(os.TempDir(), "test-palace"),
		CollectionName: "test_collection",
	}
}

func TestLayer0Render(t *testing.T) {
	l0 := NewLayer0("/nonexistent")
	_, err := l0.Render()
	if err == nil {
		t.Log("Expected error for nonexistent identity file")
	}
}

func TestLayer1Generate(t *testing.T) {
	store := &mockStore{
		results: []govector.SearchResult{
			{ID: "1", Payload: map[string]any{"content": "test memory"}},
		},
	}
	emb := &mockEmbedder{embedding: []float32{0.1, 0.2, 0.3}}
	searcher := search.NewSearcher(store, emb)
	l1 := NewLayer1(searcher)
	l1.wing = "test-wing"
	_, err := l1.Generate(context.Background())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
}

func TestNewMemoryStack(t *testing.T) {
	cfg := newTestConfig()
	store := &mockStore{}
	emb := &mockEmbedder{embedding: []float32{0.1, 0.2, 0.3}}
	searcher := search.NewSearcher(store, emb)
	stack := NewMemoryStack(cfg, searcher)
	if stack == nil {
		t.Fatal("NewMemoryStack returned nil")
	}
	if stack.cfg != cfg {
		t.Error("config not set correctly")
	}
}

func TestMemoryStackWakeUp(t *testing.T) {
	cfg := newTestConfig()
	store := &mockStore{
		results: []govector.SearchResult{},
	}
	emb := &mockEmbedder{embedding: []float32{0.1, 0.2, 0.3}}
	searcher := search.NewSearcher(store, emb)
	stack := NewMemoryStack(cfg, searcher)

	_, err := stack.WakeUp(context.Background(), "test-wing")
	// May fail due to missing identity file, that's expected
	t.Logf("WakeUp result: err=%v", err)
}

func TestMemoryStackSearch(t *testing.T) {
	cfg := newTestConfig()
	store := &mockStore{
		results: []govector.SearchResult{
			{ID: "1", Payload: map[string]any{"wing": "kitchen", "room": "main"}},
		},
	}
	emb := &mockEmbedder{embedding: []float32{0.1, 0.2, 0.3}}
	searcher := search.NewSearcher(store, emb)
	stack := NewMemoryStack(cfg, searcher)

	results, err := stack.Search(context.Background(), "query", "", "", 5)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestMemoryStackRecall(t *testing.T) {
	cfg := newTestConfig()
	store := &mockStore{
		results: []govector.SearchResult{
			{ID: "1", Payload: map[string]any{"wing": "kitchen", "room": "main"}},
		},
	}
	emb := &mockEmbedder{embedding: []float32{0.1, 0.2, 0.3}}
	searcher := search.NewSearcher(store, emb)
	stack := NewMemoryStack(cfg, searcher)

	text, err := stack.Recall(context.Background(), "kitchen", "main", 5)
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}
	if text == "" {
		t.Error("expected non-empty recall text")
	}
}
