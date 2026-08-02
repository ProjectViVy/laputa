package activity

import (
	"fmt"
	"testing"
)

func TestWorkingSetUpdateAndGet(t *testing.T) {
	ws := NewWorkingSet()
	ws.Update("project:garden", []string{"mem_1", "mem_2"}, []string{"ref_a"})
	snap := ws.Get("project:garden")
	if len(snap.ActiveCardIDs) != 2 || snap.ActiveCardIDs[0] != "mem_1" {
		t.Fatalf("cards=%v", snap.ActiveCardIDs)
	}
	if len(snap.EvidenceRefs) != 1 || snap.EvidenceRefs[0] != "ref_a" {
		t.Fatalf("refs=%v", snap.EvidenceRefs)
	}
}

func TestWorkingSetBounded(t *testing.T) {
	ws := NewWorkingSet()
	cards := make([]string, 60)
	for i := range cards {
		cards[i] = fmt.Sprintf("mem_%03d", i)
	}
	ws.Update("scope", cards, nil)
	snap := ws.Get("scope")
	if len(snap.ActiveCardIDs) != maxCardIDs {
		t.Fatalf("expected %d cards, got %d", maxCardIDs, len(snap.ActiveCardIDs))
	}
	if snap.ActiveCardIDs[0] != "mem_000" {
		t.Fatalf("newest should be first: %s", snap.ActiveCardIDs[0])
	}
}

func TestWorkingSetRestore(t *testing.T) {
	ws := NewWorkingSet()
	snap := ScopeSnapshot{
		ActiveCardIDs: []string{"mem_a", "mem_b"},
		EvidenceRefs:  []string{"ref_x"},
		PendingLeads:  []string{"lead_1"},
		UpdatedAt:     "2026-08-01T00:00:00Z",
	}
	ws.Restore("scope", snap)
	got := ws.Get("scope")
	if len(got.ActiveCardIDs) != 2 || got.ActiveCardIDs[0] != "mem_a" {
		t.Fatalf("restored=%v", got)
	}
	if got.UpdatedAt != "2026-08-01T00:00:00Z" {
		t.Fatalf("updatedAt=%q", got.UpdatedAt)
	}
}

func TestWorkingSetEmptyScope(t *testing.T) {
	ws := NewWorkingSet()
	snap := ws.Get("nonexistent")
	if len(snap.ActiveCardIDs) != 0 || len(snap.EvidenceRefs) != 0 {
		t.Fatalf("expected empty snapshot, got %v", snap)
	}
}

func TestWorkingSetMergeDeduplicates(t *testing.T) {
	ws := NewWorkingSet()
	ws.Update("s", []string{"a", "b"}, nil)
	ws.Update("s", []string{"b", "c"}, nil)
	snap := ws.Get("s")
	if len(snap.ActiveCardIDs) != 3 {
		t.Fatalf("expected 3 unique, got %v", snap.ActiveCardIDs)
	}
}
