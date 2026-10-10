package actmem

import (
	"strings"
	"testing"

	"github.com/ProjectViVy/laputa/laputa/evolution"
)

func TestReadScopedEnforcesAggregateBodyBudget(t *testing.T) {
	store := New(t.TempDir())
	for i, section := range []evolution.EntrySection{evolution.SectionPulse, evolution.SectionRecap} {
		if _, err := store.AppendEntry(evolution.Entry{Section: section, Scope: personalScope(), Body: strings.Repeat("长", 20), EventID: string(rune('a' + i))}); err != nil {
			t.Fatal(err)
		}
	}
	if result, err := store.ReadScoped(personalScope(), evolution.ReadRequest{MaxChars: 32}); CodeOf(err) != string(evolution.ErrActmemCapExceeded) || len(result.Entries) != 0 {
		t.Fatalf("read exceeded aggregate budget without bounded error: entries=%d err=%v", len(result.Entries), err)
	}
	result, err := store.ReadScoped(personalScope(), evolution.ReadRequest{MaxChars: 40})
	if err != nil || len(result.Entries) != 2 || result.Entries[0].Body != strings.Repeat("长", 20) {
		t.Fatalf("Unicode code-point boundary: %+v %v", result, err)
	}
	if _, err := store.AppendEntry(evolution.Entry{Section: evolution.SectionPulse, Scope: wsScope("hidden"), Body: strings.Repeat("外", 200)}); err != nil {
		t.Fatal(err)
	}
	if result, err := store.ReadScoped(personalScope(), evolution.ReadRequest{MaxChars: 40}); err != nil || len(result.Entries) != 2 {
		t.Fatalf("hidden scope consumed visible budget: %+v %v", result, err)
	}
}

func TestReadScopedDefaultBudgetIsBounded(t *testing.T) {
	store := New(t.TempDir())
	if _, err := store.ApplyWorkPatch(personalScope(), evolution.WorkPatch{BaseRevision: 0, Changes: []evolution.WorkChange{{Kind: evolution.WorkChangeAdd, Field: evolution.FieldGoal, Body: strings.Repeat("x", 1201)}}}); err != nil {
		t.Fatal(err)
	}
	if result, err := store.ReadScoped(personalScope(), evolution.ReadRequest{}); CodeOf(err) != string(evolution.ErrActmemCapExceeded) || len(result.Entries) != 0 {
		t.Fatalf("zero budget returned unbounded body: entries=%d err=%v", len(result.Entries), err)
	}
}
