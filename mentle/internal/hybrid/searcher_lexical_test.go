package hybrid

import (
	"context"
	"reflect"
	"testing"

	"github.com/dashimaki/mentle/internal/palace"
	"github.com/dashimaki/mentle/internal/search"
	"github.com/dashimaki/mentle/storage/govector"
)

type vectorSearchProbe struct {
	search.Store
	called bool
}

func (p *vectorSearchProbe) Search(vector []float32, limit int, filter map[string]any) ([]govector.SearchResult, error) {
	p.called = true
	return []govector.SearchResult{{ID: "vector", Payload: map[string]any{"content": "vector document"}}}, nil
}

type vectorTieSearchProbe struct {
	search.Store
	results []govector.SearchResult
}

func (p *vectorTieSearchProbe) Search([]float32, int, map[string]any) ([]govector.SearchResult, error) {
	return append([]govector.SearchResult(nil), p.results...), nil
}

type embeddingProbe struct{ search.Embedder }

func TestLexicalOnlyRejectsBothVectorWritePaths(t *testing.T) {
	s := NewSearcher(nil, nil, 1)
	if err := s.Store(context.Background(), palace.Drawer{Content: "must not write"}); err == nil {
		t.Fatal("lexical-only Store accepted a vector write")
	}
	if err := s.StoreVectors([]string{"id"}, [][]float32{{1}}, []map[string]any{{"content": "must not write"}}); err == nil {
		t.Fatal("lexical-only StoreVectors accepted a vector write")
	}
}

func (embeddingProbe) CreateEmbedding(context.Context, string) ([]float32, error) {
	return []float32{1}, nil
}

func TestSearchScoredWithoutEmbedderUsesOnlyBM25(t *testing.T) {
	// No vector store exists: lexical retrieval must use the canonical BM25
	// projection without embedding or accessing a vector backend.
	s := NewSearcher(nil, nil, 1)
	s.RebuildBM25FromDrawers([]search.Drawer{
		{ID: "first", Wing: "work", Room: "notes", Content: "orchid orchid", Metadata: map[string]string{"source": "one"}},
		{ID: "second", Wing: "work", Room: "notes", Content: "orchid", Metadata: map[string]string{"source": "two"}},
		{ID: "other", Wing: "home", Room: "notes", Content: "orchid orchid orchid"},
		{ID: "unmatched", Wing: "home", Room: "notes", Content: "daisy"},
	})
	want := s.BM25Search("orchid", 0)
	got, err := s.SearchScored(context.Background(), "orchid", "", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d results, want 2: %+v", len(got), got)
	}
	for i, result := range got {
		if result.Drawer.ID != want[i].ID || result.Score != want[i].Score {
			t.Errorf("result %d = %s (score %g), want %s (BM25 score %g)", i, result.Drawer.ID, result.Score, want[i].ID, want[i].Score)
		}
		if !reflect.DeepEqual(result.Channels, []string{"bm25"}) {
			t.Errorf("channels = %v, want [bm25]", result.Channels)
		}
		if result.Drawer.Content == "" || result.Drawer.Metadata["content"] != result.Drawer.Content {
			t.Errorf("missing canonical content in result: %+v", result.Drawer)
		}
	}
}

func TestSearchScoredWithoutEmbedderBreaksScoreTiesByID(t *testing.T) {
	s := NewSearcher(nil, nil, 1)
	s.RebuildBM25FromDrawers([]search.Drawer{
		{ID: "id-c", Content: "orchid tie"},
		{ID: "id-a", Content: "orchid tie"},
		{ID: "id-d", Content: "orchid tie"},
		{ID: "id-b", Content: "orchid tie"},
	})
	for attempt := 0; attempt < 20; attempt++ {
		got, err := s.SearchScored(context.Background(), "orchid", "", "", 4)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 4 {
			t.Fatalf("attempt %d returned %d results, want 4", attempt, len(got))
		}
		for i, want := range []string{"id-a", "id-b", "id-c", "id-d"} {
			if got[i].Drawer.ID != want {
				t.Fatalf("attempt %d result %d = %q, want deterministic score tie order %q", attempt, i, got[i].Drawer.ID, want)
			}
		}
	}
}

func TestSearchScoredWithEmbedderStillUsesVectorFusion(t *testing.T) {
	store := &vectorSearchProbe{}
	s := NewSearcher(store, embeddingProbe{}, 0.7)
	s.RebuildBM25FromDrawers([]search.Drawer{{ID: "lexical", Content: "orchid"}})
	got, err := s.SearchScored(context.Background(), "orchid", "", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !store.called || len(got) != 2 || got[0].Drawer.ID != "vector" || !reflect.DeepEqual(got[0].Channels, []string{"vector"}) || got[1].Drawer.ID != "lexical" || !reflect.DeepEqual(got[1].Channels, []string{"bm25"}) {
		t.Fatalf("hybrid results = %+v, vector called = %v", got, store.called)
	}
}

func TestSearchScoredWithEmbedderBreaksVectorScoreTiesByID(t *testing.T) {
	store := &vectorTieSearchProbe{results: []govector.SearchResult{
		{ID: "id-c", Score: 0.8, Payload: map[string]any{"content": "orchid tie"}},
		{ID: "id-a", Score: 0.8, Payload: map[string]any{"content": "orchid tie"}},
		{ID: "id-b", Score: 0.8, Payload: map[string]any{"content": "orchid tie"}},
	}}
	s := NewSearcher(store, embeddingProbe{}, 0.7)
	s.RebuildBM25FromDrawers([]search.Drawer{
		{ID: "id-c", Content: "orchid tie"},
		{ID: "id-a", Content: "orchid tie"},
		{ID: "id-b", Content: "orchid tie"},
	})
	got, err := s.SearchScored(context.Background(), "orchid", "", "", 3)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"id-a", "id-b", "id-c"} {
		if got[i].Drawer.ID != want {
			t.Fatalf("result %d = %q, want deterministic vector tie order %q; all=%+v", i, got[i].Drawer.ID, want, got)
		}
	}
}

func TestSearchScoredWithoutEmbedderFiltersBeforeLimit(t *testing.T) {
	s := NewSearcher(nil, nil, 0.7)
	s.RebuildBM25FromDrawers([]search.Drawer{
		{ID: "outside-1", Wing: "outside", Room: "notes", Content: "orchid orchid orchid orchid"},
		{ID: "outside-2", Wing: "outside", Room: "notes", Content: "orchid orchid orchid"},
		{ID: "outside-3", Wing: "outside", Room: "notes", Content: "orchid orchid"},
		{ID: "outside-4", Wing: "outside", Room: "notes", Content: "orchid orchid"},
		{ID: "inside", Wing: "work", Room: "notes", Content: "orchid"},
		{ID: "wrong-room", Wing: "work", Room: "other", Content: "orchid"},
	})
	got, err := s.SearchScored(context.Background(), "orchid", "work", "notes", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Drawer.ID != "inside" || got[0].Drawer.Wing != "work" || got[0].Drawer.Room != "notes" {
		t.Fatalf("filtered result = %+v, want only inside", got)
	}
}
