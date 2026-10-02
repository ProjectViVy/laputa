package recall

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dashimaki/garden/internal/personactx"
	"github.com/dashimaki/mentle/facade"
)

func TestFrozenCoreInContextAndDTO(t *testing.T) {
	core := testCore()
	core.Sections[2] = personactx.FrozenSection{Kind: personactx.SectionRelationship, Content: "relationship"}
	searcher := &fakeSearcher{cards: []facade.MemoryCard{{ID: "card", Kind: "fact"}}, evidence: []facade.EvidenceFragment{{CardID: "card", Excerpt: "evidence"}}}
	view, err := (&FastService{Frozen: staticFrozen{core: core}, Searcher: searcher}).Recall(context.Background(), FastRequest{Query: "context", SessionID: "session-1", BudgetChars: 6000})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(view.Context, "Frozen Core") || !strings.Contains(view.Context, "identity") || !strings.Contains(view.Context, "evidence") {
		t.Fatalf("context=%q", view.Context)
	}
	payload, _ := json.Marshal(view)
	if strings.Contains(string(payload), "governance") || strings.Contains(string(payload), "world") || strings.Contains(string(payload), "actmem") {
		t.Fatalf("retired automatic context field leaked: %s", payload)
	}
}

func TestNoFrozenCoreFallbackToRetiredAuthority(t *testing.T) {
	view, err := (&FastService{}).Recall(context.Background(), FastRequest{Query: "bootstrap", BudgetChars: 6000})
	if err != nil {
		t.Fatal(err)
	}
	if view.Context != "" || strings.Contains(view.Context, "projection") {
		t.Fatalf("unexpected fallback context=%q", view.Context)
	}
}

func TestContextBudgetIncludesFrozenAndEvidence(t *testing.T) {
	core := testCore()
	core.Sections[0].Content = strings.Repeat("i", 500)
	searcher := &fakeSearcher{cards: []facade.MemoryCard{{ID: "card", Kind: "fact"}}, evidence: []facade.EvidenceFragment{{CardID: "card", Excerpt: strings.Repeat("e", 500)}}}
	view, err := (&FastService{Frozen: staticFrozen{core: core}, Searcher: searcher}).Recall(context.Background(), FastRequest{Query: "bounded", SessionID: "session-1", BudgetChars: 256})
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(view.Context)) > 256 {
		t.Fatalf("context exceeds budget: %d", len([]rune(view.Context)))
	}
}
