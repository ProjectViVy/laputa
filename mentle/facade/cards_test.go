package facade

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dashimaki/mentle/internal/hybrid"
	"github.com/dashimaki/mentle/internal/search"
	"github.com/dashimaki/mentle/storage/govector"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	store := &fakeStore{points: map[string]govector.SearchResult{}}
	embedder := fakeEmbedder{}
	catalog, err := OpenCatalog(filepath.Join(t.TempDir(), "canonical.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { catalog.Close() })
	return &Service{
		Searcher: search.NewSearcher(store, embedder),
		Hybrid:   hybrid.NewSearcher(store, embedder, .7),
		Catalog:  catalog,
	}
}

func TestSearchCardsNoFullContent(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	longContent := strings.Repeat("detailed memory content ", 50)
	created, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: longContent, Kind: "fact"}, "k1", "h1")
	if err != nil {
		t.Fatal(err)
	}
	page, err := svc.SearchCards(ctx, CardQuery{Text: "detailed", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(page.Cards))
	}
	card := page.Cards[0]
	if card.ID != created.ID {
		t.Fatalf("card.ID=%q want %q", card.ID, created.ID)
	}
	if len([]rune(card.Summary)) > summaryBudget+1 {
		t.Fatalf("summary too long: %d runes", len([]rune(card.Summary)))
	}
	if card.Summary == longContent {
		t.Fatal("card summary must not equal full content")
	}
	if card.CandidateScore <= 0 {
		t.Fatalf("candidate score should be positive, got %f", card.CandidateScore)
	}
}

func TestSearchCardsStatusFilter(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	active, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "active memory", Kind: "fact"}, "k1", "h1")
	if err != nil {
		t.Fatal(err)
	}
	deleted, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "deleted memory", Kind: "fact"}, "k2", "h2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DeleteMemory(ctx, deleted.ID, "test", "req1"); err != nil {
		t.Fatal(err)
	}
	page, err := svc.SearchCards(ctx, CardQuery{Text: "memory", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, card := range page.Cards {
		if card.ID == deleted.ID {
			t.Fatal("deleted memory must not appear in card results")
		}
	}
	found := false
	for _, card := range page.Cards {
		if card.ID == active.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("active memory should appear in card results")
	}
}

func TestSearchCardsSupersededFilter(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	old, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "old version", Kind: "fact"}, "k1", "h1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreateMemory(ctx, CreateMemoryRequest{Content: "new version", Kind: "fact", Supersedes: []string{old.ID}}, "k2", "h2")
	if err != nil {
		t.Fatal(err)
	}
	page, err := svc.SearchCards(ctx, CardQuery{Text: "version", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, card := range page.Cards {
		if card.ID == old.ID {
			t.Fatal("superseded memory must not appear in card results")
		}
		if card.SupersededBy != nil {
			t.Fatalf("card %s has SupersededBy set", card.ID)
		}
	}
}

func TestSearchCardsScopeFilter(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	_, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "scoped memory", Kind: "fact", Scope: "project-a"}, "k1", "h1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreateMemory(ctx, CreateMemoryRequest{Content: "other scoped memory", Kind: "fact", Scope: "project-b"}, "k2", "h2")
	if err != nil {
		t.Fatal(err)
	}
	page, err := svc.SearchCards(ctx, CardQuery{Text: "scoped", Scope: "project-a", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, card := range page.Cards {
		if card.Scope != "project-a" {
			t.Fatalf("card scope=%q, want project-a", card.Scope)
		}
	}
}

func TestSearchCardsLimitAndDedup(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "memory item", Kind: "note"}, "k"+string(rune('a'+i)), "h"+string(rune('a'+i)))
		if err != nil {
			t.Fatal(err)
		}
	}
	page, err := svc.SearchCards(ctx, CardQuery{Text: "memory", Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Cards) > 3 {
		t.Fatalf("expected at most 3 cards, got %d", len(page.Cards))
	}
	seen := map[string]bool{}
	for _, card := range page.Cards {
		if seen[card.ID] {
			t.Fatalf("duplicate card ID: %s", card.ID)
		}
		seen[card.ID] = true
	}
}

func TestSearchCardsUnavailable(t *testing.T) {
	svc := &Service{}
	if _, err := svc.SearchCards(context.Background(), CardQuery{Text: "test"}); err == nil {
		t.Fatal("expected error on uninitialized service")
	}
}
