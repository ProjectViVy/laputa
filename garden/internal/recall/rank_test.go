package recall

import (
	"testing"

	"github.com/dashimaki/garden/internal/authority"
	"github.com/dashimaki/mentle/facade"
)

func TestFilterCardsDeniedSource(t *testing.T) {
	cards := []facade.MemoryCard{
		{ID: "mem_1", Kind: "fact", Collection: "trusted"},
		{ID: "mem_2", Kind: "fact", Collection: "untrusted"},
		{ID: "mem_3", Kind: "note", SourceRef: "blocked"},
	}
	proj := authority.GovernanceProjection{DeniedSources: []string{"untrusted", "blocked"}}
	out := FilterCards(cards, proj)
	if len(out) != 1 || out[0].ID != "mem_1" {
		t.Fatalf("out=%+v", out)
	}
}

func TestFilterCardsAllowedKinds(t *testing.T) {
	cards := []facade.MemoryCard{
		{ID: "mem_1", Kind: "fact"},
		{ID: "mem_2", Kind: "note"},
	}
	proj := authority.GovernanceProjection{AllowedKinds: []string{"fact", "decision"}}
	out := FilterCards(cards, proj)
	if len(out) != 1 || out[0].ID != "mem_1" {
		t.Fatalf("out=%+v", out)
	}
}

func TestRankCardsOrder(t *testing.T) {
	cards := []facade.MemoryCard{
		{ID: "low", CandidateScore: 0.1, HeatScore: 0.1, Scope: "other"},
		{ID: "high", CandidateScore: 0.9, HeatScore: 0.8, Scope: "project:garden"},
		{ID: "mid", CandidateScore: 0.5, HeatScore: 0.5, Scope: "project:garden"},
	}
	sorted := RankCards(cards, "project:garden")
	if sorted[0].ID != "high" {
		t.Fatalf("first=%s", sorted[0].ID)
	}
	if sorted[2].ID != "low" {
		t.Fatalf("last=%s", sorted[2].ID)
	}
}

func TestDeduplicateCards(t *testing.T) {
	cards := []facade.MemoryCard{
		{ID: "mem_1", Summary: "first"},
		{ID: "mem_1", Summary: "dup"},
		{ID: "mem_2", Summary: "second"},
	}
	out := DeduplicateCards(cards)
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].Summary != "first" {
		t.Fatalf("first-seen should win, got %q", out[0].Summary)
	}
}
