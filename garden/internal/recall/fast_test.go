package recall

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dashimaki/garden/internal/personactx"
	"github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/mentle/facade"
)

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

type staticFrozen struct {
	core personactx.FrozenCore
	err  error
}

func (f staticFrozen) Get(context.Context, string) (personactx.FrozenCore, error) {
	return f.core, f.err
}

func testCore() personactx.FrozenCore {
	return personactx.FrozenCore{
		SchemaVersion: evolution.FrozenCoreV2SchemaVersion,
		SessionID:     "session-1",
		MissionStatus: evolution.MissionUnassigned,
		Sections: []personactx.FrozenSection{
			{Kind: personactx.SectionMission},
			{Kind: personactx.SectionIdentity, Content: "identity"},
			{Kind: personactx.SectionRelationship},
			{Kind: personactx.SectionRedline},
			{Kind: personactx.SectionUser},
			{Kind: personactx.SectionDream},
			{Kind: personactx.SectionDark},
		},
	}
}

func TestFastRecallLexicalOnlyReturnsCardsAndReportsDegraded(t *testing.T) {
	svc := &FastService{Searcher: &fakeSearcher{cards: []facade.MemoryCard{{ID: "mem_lexical", Kind: "note", Summary: "offline evidence"}}, evidence: []facade.EvidenceFragment{{CardID: "mem_lexical", Excerpt: "offline evidence"}}}, LexicalOnly: true}
	view, err := svc.Recall(context.Background(), FastRequest{Query: "offline", BudgetChars: 6000})
	if err != nil || len(view.Cards) != 1 || !strings.Contains(view.Context, "offline evidence") || !view.Degraded || !strings.Contains(strings.Join(view.Warnings, " "), "lexical") {
		t.Fatalf("lexical view=%+v err=%v", view, err)
	}
}

func TestFastRecallWithSearcher(t *testing.T) {
	searcher := &fakeSearcher{
		cards:    []facade.MemoryCard{{ID: "mem_1", Kind: "fact", Summary: "garden architecture", CandidateScore: 0.9, HeatScore: 0.5}},
		evidence: []facade.EvidenceFragment{{CardID: "mem_1", Excerpt: "Garden uses three modules", Validity: "active"}},
	}
	svc := &FastService{Searcher: searcher}
	view, err := svc.Recall(context.Background(), FastRequest{Query: "architecture", BudgetChars: 6000})
	if err != nil {
		t.Fatal(err)
	}
	if view.Mode != "fast" || len(view.Cards) != 1 || view.Cards[0].ID != "mem_1" {
		t.Fatalf("view=%+v", view)
	}
	if len(view.Evidence) != 1 || !strings.Contains(view.Context, "three modules") || view.Degraded {
		t.Fatalf("view=%+v", view)
	}
}

func TestFastRecallMentleDegradedKeepsFrozenCore(t *testing.T) {
	svc := &FastService{Frozen: staticFrozen{core: testCore()}}
	view, err := svc.Recall(context.Background(), FastRequest{Query: "test", SessionID: "session-1", BudgetChars: 6000})
	if err != nil {
		t.Fatal(err)
	}
	if !view.Degraded || !strings.Contains(view.Context, "identity") {
		t.Fatalf("view=%+v", view)
	}
	if len(view.Cards) != 0 || len(view.Evidence) != 0 {
		t.Fatalf("unexpected retrieval output: %+v", view)
	}
}

func TestFastRecallSearchError(t *testing.T) {
	svc := &FastService{Frozen: staticFrozen{core: testCore()}, Searcher: &fakeSearcher{cardErr: errors.New("connection refused")}}
	view, err := svc.Recall(context.Background(), FastRequest{Query: "test", SessionID: "session-1", BudgetChars: 6000})
	if err != nil || !view.Degraded || len(view.Warnings) == 0 {
		t.Fatalf("view=%+v err=%v", view, err)
	}
}

func TestFastRecallFrozenCoreProviderIsSessionScoped(t *testing.T) {
	provider := staticFrozen{core: testCore()}
	svc := &FastService{Frozen: provider}
	view, err := svc.Recall(context.Background(), FastRequest{Query: "test", SessionID: "session-1", BudgetChars: 6000})
	if err != nil {
		t.Fatal(err)
	}
	if view.FrozenCore.SessionID != "session-1" {
		t.Fatalf("frozen core=%+v", view.FrozenCore)
	}
}

func TestFastRecallBudgetEnforced(t *testing.T) {
	searcher := &fakeSearcher{
		cards:    []facade.MemoryCard{{ID: "mem_1", Kind: "fact", CandidateScore: 0.9}},
		evidence: []facade.EvidenceFragment{{CardID: "mem_1", Excerpt: strings.Repeat("x", 5000), Validity: "active"}},
	}
	view, err := (&FastService{Searcher: searcher}).Recall(context.Background(), FastRequest{Query: "test", BudgetChars: 500})
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(view.Context)) > 500 {
		t.Fatalf("context len=%d exceeds budget", len([]rune(view.Context)))
	}
}

func TestFastRecallValidation(t *testing.T) {
	svc := &FastService{}
	if _, err := svc.Recall(context.Background(), FastRequest{Query: "", BudgetChars: 6000}); err == nil {
		t.Fatal("empty query should error")
	}
	if _, err := svc.Recall(context.Background(), FastRequest{Query: "test", BudgetChars: 10}); err == nil {
		t.Fatal("budget below min should error")
	}
}
