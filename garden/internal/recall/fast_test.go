package recall

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dashimaki/laputa/governance"
	"github.com/dashimaki/mentle/facade"
)

type fakeGov struct{}

func (fakeGov) GetSection(_ context.Context, _ governance.SectionName) (map[string]any, error) {
	return map[string]any{"_meta": map[string]any{"version": "test-v1"}}, nil
}

type fakeSearcher struct {
	cards    []facade.MemoryCard
	evidence []facade.EvidenceFragment
	cardErr  error
	evErr    error
}

func (f *fakeSearcher) SearchCards(_ context.Context, _ facade.CardQuery) (facade.CardPage, error) {
	if f.cardErr != nil {
		return facade.CardPage{}, f.cardErr
	}
	return facade.CardPage{Cards: f.cards}, nil
}

func (f *fakeSearcher) ReadEvidence(_ context.Context, _ facade.EvidenceQuery) ([]facade.EvidenceFragment, error) {
	if f.evErr != nil {
		return nil, f.evErr
	}
	return f.evidence, nil
}

func TestFastRecallWithSearcher(t *testing.T) {
	searcher := &fakeSearcher{
		cards: []facade.MemoryCard{
			{ID: "mem_1", Kind: "fact", Summary: "garden architecture", CandidateScore: 0.9, HeatScore: 0.5},
		},
		evidence: []facade.EvidenceFragment{
			{CardID: "mem_1", Excerpt: "Garden uses three modules", Validity: "active"},
		},
	}
	svc := &FastService{Gov: fakeGov{}, Searcher: searcher}
	view, err := svc.Recall(context.Background(), FastRequest{Query: "architecture", BudgetChars: 6000})
	if err != nil {
		t.Fatal(err)
	}
	if view.Mode != "fast" {
		t.Fatalf("mode=%q", view.Mode)
	}
	if len(view.Cards) != 1 || view.Cards[0].ID != "mem_1" {
		t.Fatalf("cards=%+v", view.Cards)
	}
	if len(view.Evidence) != 1 {
		t.Fatalf("evidence=%+v", view.Evidence)
	}
	if !strings.Contains(view.Context, "three modules") {
		t.Fatalf("context=%q", view.Context)
	}
	if view.Degraded {
		t.Fatalf("should not be degraded: %v", view.Warnings)
	}
}

func TestFastRecallMentleDegraded(t *testing.T) {
	svc := &FastService{Gov: fakeGov{}, Searcher: nil}
	view, err := svc.Recall(context.Background(), FastRequest{Query: "test", BudgetChars: 6000})
	if err != nil {
		t.Fatal(err)
	}
	if !view.Degraded {
		t.Fatal("should be degraded")
	}
	if len(view.Warnings) == 0 || !strings.Contains(view.Warnings[0], "mentle unavailable") {
		t.Fatalf("warnings=%v", view.Warnings)
	}
	if len(view.Cards) != 0 {
		t.Fatalf("cards=%+v", view.Cards)
	}
	if view.Context == "" {
		t.Fatal("governance context should not be empty")
	}
}

func TestFastRecallSearchError(t *testing.T) {
	searcher := &fakeSearcher{cardErr: errors.New("connection refused")}
	svc := &FastService{Gov: fakeGov{}, Searcher: searcher}
	view, err := svc.Recall(context.Background(), FastRequest{Query: "test", BudgetChars: 6000})
	if err != nil {
		t.Fatal(err)
	}
	if !view.Degraded {
		t.Fatal("should be degraded on search error")
	}
	if len(view.Warnings) == 0 {
		t.Fatal("should have warning")
	}
}

func TestFastRecallBudgetEnforced(t *testing.T) {
	longExcerpt := strings.Repeat("x", 5000)
	searcher := &fakeSearcher{
		cards:    []facade.MemoryCard{{ID: "mem_1", Kind: "fact", CandidateScore: 0.9}},
		evidence: []facade.EvidenceFragment{{CardID: "mem_1", Excerpt: longExcerpt, Validity: "active"}},
	}
	svc := &FastService{Gov: fakeGov{}, Searcher: searcher}
	view, err := svc.Recall(context.Background(), FastRequest{Query: "test", BudgetChars: 500})
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(view.Context)) > 500 {
		t.Fatalf("context len=%d exceeds budget", len([]rune(view.Context)))
	}
}

func TestFastRecallValidation(t *testing.T) {
	svc := &FastService{Gov: fakeGov{}}
	_, err := svc.Recall(context.Background(), FastRequest{Query: "", BudgetChars: 6000})
	if err == nil {
		t.Fatal("empty query should error")
	}
	_, err = svc.Recall(context.Background(), FastRequest{Query: "test", BudgetChars: 10})
	if err == nil {
		t.Fatal("budget below min should error")
	}
}
