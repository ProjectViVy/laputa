package actmem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProjectViVy/laputa/laputa/evolution"
)

func appendRing(t *testing.T, store *Store, section evolution.EntrySection, session, body string) evolution.Entry {
	t.Helper()
	result, err := store.AppendEntry(evolution.Entry{
		Section:   section,
		Scope:     wsScope("ws-x"),
		SessionID: session,
		EventID:   "ev-" + body,
		Body:      body,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.Entries[len(result.Entries)-1]
}

// TestFoldSessionArchivesAndRemoves: folding captures the session's ring
// entries into a deterministic capsule and removes them from the head.
func TestFoldSessionArchivesAndRemoves(t *testing.T) {
	store := New(t.TempDir())
	appendRing(t, store, evolution.SectionPulse, "s-1", "one")
	appendRing(t, store, evolution.SectionPulse, "s-2", "other session")
	appendRing(t, store, evolution.SectionRecap, "s-1", "two")

	summaries, err := store.FoldSession("s-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) == 0 {
		t.Fatal("no capsule written")
	}
	doc, _ := store.Read()
	if strings.Contains(doc.Pulse, "one") || strings.Contains(doc.Recap, "two") {
		t.Fatalf("folded entries still in head: %s", doc.Markdown)
	}
	if !strings.Contains(doc.Pulse, "other session") {
		t.Fatal("foreign session entry removed by fold")
	}
	var archived []byte
	for _, summary := range summaries {
		raw, err := os.ReadFile(filepath.Join(store.CapsulesDir(), summary.Name))
		if err != nil {
			t.Fatal(err)
		}
		archived = append(archived, raw...)
	}
	if !strings.Contains(string(archived), "one") || !strings.Contains(string(archived), "two") {
		t.Fatal("capsules lost folded bodies")
	}
	// Folding again is a no-op — no second capsule.
	again, err := store.FoldSession("s-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 {
		t.Fatalf("refold produced %+v", again)
	}
}

// TestFoldCrashRecovery: a crash after the capsule write leaves duplicate
// state; the retry recognizes the matching capsule and finishes removal
// without data loss. Entries appended meanwhile are never consumed.
func TestFoldCrashRecovery(t *testing.T) {
	store := New(t.TempDir())
	appendRing(t, store, evolution.SectionPulse, "s-1", "crashable")
	store.crashBeforeHeadCommit = true
	if _, err := store.FoldSession("s-1"); err == nil {
		t.Fatal("expected injected crash")
	}
	store.crashBeforeHeadCommit = false
	// New activity lands between the crash and the retry.
	appendRing(t, store, evolution.SectionPulse, "s-1", "post-crash")
	summaries, err := store.FoldSession("s-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) == 0 {
		t.Fatal("retry produced no capsule")
	}
	doc, _ := store.Read()
	if strings.Contains(doc.Pulse, "crashable") {
		t.Fatal("head removal never completed")
	}
	if !strings.Contains(doc.Pulse, "post-crash") {
		t.Fatal("stale fold consumed a newer entry")
	}
	entries, _ := os.ReadDir(store.CapsulesDir())
	if len(entries) != len(summaries) {
		t.Fatalf("duplicate capsules: %d files vs %d summaries", len(entries), len(summaries))
	}
}

// TestFoldMismatchedCapsuleConflicts: a same-named capsule with different
// content is an explicit conflict, never deletion.
func TestFoldMismatchedCapsuleConflicts(t *testing.T) {
	temp := t.TempDir()
	store := New(temp)
	appendRing(t, store, evolution.SectionPulse, "s-1", "real body")
	// Pre-seed the capsule slot the fold would claim with foreign bytes.
	doc, _ := store.Read()
	var sources []evolution.Entry
	for _, e := range doc.Entries() {
		if e.SessionID == "s-1" {
			sources = append(sources, e)
		}
	}
	name := foldCapsuleName(sources)
	if err := os.MkdirAll(store.CapsulesDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.CapsulesDir(), name), []byte("forged capsule"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.FoldSession("s-1"); err == nil {
		t.Fatal("mismatched capsule silently accepted")
	}
	doc, _ = store.Read()
	if !strings.Contains(doc.Pulse, "real body") {
		t.Fatal("conflict deleted head entries")
	}
}
