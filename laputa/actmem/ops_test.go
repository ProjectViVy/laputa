package actmem

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dashimaki/laputa/evolution"
)

func wsScope(ws string) evolution.Scope {
	return evolution.Scope{SubjectID: "sub", Kind: evolution.ScopeWorkspace, WorkspaceID: ws}
}

func personalScope() evolution.Scope {
	return evolution.Scope{SubjectID: "sub", Kind: evolution.ScopePersonal}
}

// TestSystemAppendEntry: the host appends a scoped ring entry; the store
// allocates the id and stamps the time; missing head starts as v2.
func TestSystemAppendEntry(t *testing.T) {
	temp := t.TempDir()
	store := New(temp)
	store.SetClock(func() time.Time { return time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC) })
	result, err := store.AppendEntry(evolution.Entry{
		Section:   evolution.SectionPulse,
		Scope:     wsScope("ws-x"),
		SessionID: "s-1",
		EventID:   "ev-1",
		Body:      "first pulse",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Revision != 1 || len(result.Entries) != 1 {
		t.Fatalf("result=%+v", result)
	}
	entry := result.Entries[0]
	if !strings.HasPrefix(entry.ID, "e_") || entry.OccurredAt == "" || entry.Body != "first pulse" {
		t.Fatalf("entry=%+v", entry)
	}
	doc, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	if doc.Revision != 1 || doc.Unclassified || !strings.Contains(doc.Pulse, "first pulse") {
		t.Fatalf("doc=%+v", doc)
	}
	if _, err := evolution.ParseActmemDocument(doc.Markdown); err != nil {
		t.Fatalf("head is not v2: %v", err)
	}
}

// TestSystemAppendDedupesEventID: a retained duplicate event id is a no-op;
// the ring is not a delivery ledger.
func TestSystemAppendDedupesEventID(t *testing.T) {
	store := New(t.TempDir())
	first := evolution.Entry{Section: evolution.SectionPulse, Scope: personalScope(), SessionID: "s-1", EventID: "ev-1", Body: "x"}
	if _, err := store.AppendEntry(first); err != nil {
		t.Fatal(err)
	}
	second, err := store.AppendEntry(first)
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed || second.Revision != 1 || len(second.Entries) != 1 {
		t.Fatalf("dedupe result=%+v", second)
	}
}

// TestRingEvictsOldest: a ring over its 1600-char budget evicts the oldest
// entries, never the newest.
func TestRingEvictsOldest(t *testing.T) {
	store := New(t.TempDir())
	body := strings.Repeat("y", 300)
	for i := 0; i < 7; i++ {
		if _, err := store.AppendEntry(evolution.Entry{Section: evolution.SectionPulse, Scope: personalScope(), SessionID: "s", Body: body}); err != nil {
			t.Fatal(err)
		}
	}
	doc, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	entries := doc.Entries()
	sectionChars := 0
	for _, e := range entries {
		if e.Section != evolution.SectionPulse {
			continue
		}
		sectionChars += len([]rune(e.Body))
	}
	if sectionChars > ACTMEMRingCapChars || len(entries) == 7 {
		t.Fatalf("no eviction: %d entries %d chars", len(entries), sectionChars)
	}
}

// TestWorkPatchLifecycle: add allocates ids in the caller's scope; replace/
// complete/drop act on visible ids; a scoped patch preserves other
// workspaces' entries.
func TestWorkPatchLifecycle(t *testing.T) {
	store := New(t.TempDir())
	// Seed a ws-y work entry + a ws-x pulse.
	if _, err := store.AppendEntry(evolution.Entry{Section: evolution.SectionPulse, Scope: wsScope("ws-x"), SessionID: "s", Body: "x pulse"}); err != nil {
		t.Fatal(err)
	}
	patch := evolution.WorkPatch{BaseRevision: 1, Changes: []evolution.WorkChange{
		{Kind: evolution.WorkChangeAdd, Field: evolution.FieldOpen, Body: "ws-y open item"},
	}}
	if _, err := store.ApplyWorkPatch(wsScope("ws-y"), patch); err != nil {
		t.Fatal(err)
	}
	result, err := store.ApplyWorkPatch(wsScope("ws-x"), evolution.WorkPatch{BaseRevision: 2, Changes: []evolution.WorkChange{
		{Kind: evolution.WorkChangeAdd, Field: evolution.FieldGoal, Body: "ws-x goal"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Revision != 3 {
		t.Fatalf("result=%+v", result)
	}
	doc, _ := store.Read()
	var goal, yOpen string
	for _, e := range doc.Entries() {
		switch {
		case e.Body == "ws-x goal":
			goal = e.ID
		case e.Body == "ws-y open item":
			yOpen = e.ID
		}
	}
	if goal == "" || yOpen == "" {
		t.Fatalf("entries=%+v", doc.Entries())
	}
	// ws-x cannot touch the ws-y entry (invisible → not editable).
	_, err = store.ApplyWorkPatch(wsScope("ws-x"), evolution.WorkPatch{BaseRevision: 3, Changes: []evolution.WorkChange{
		{Kind: evolution.WorkChangeDrop, Field: evolution.FieldOpen, EntryID: yOpen},
	}})
	if err == nil {
		t.Fatal("ws-x dropped a ws-y entry")
	}
	// ws-y drops its own entry.
	if _, err := store.ApplyWorkPatch(wsScope("ws-y"), evolution.WorkPatch{BaseRevision: 3, Changes: []evolution.WorkChange{
		{Kind: evolution.WorkChangeDrop, Field: evolution.FieldOpen, EntryID: yOpen},
	}}); err != nil {
		t.Fatal(err)
	}
	// ws-x's entries survived every ws-y patch.
	doc, _ = store.Read()
	var found bool
	for _, e := range doc.Entries() {
		if e.ID == goal {
			found = true
		}
	}
	if !found {
		t.Fatal("ws-x goal lost to a foreign-scope patch")
	}
	// Stale base revision → conflict.
	if _, err := store.ApplyWorkPatch(wsScope("ws-x"), evolution.WorkPatch{BaseRevision: 3, Changes: []evolution.WorkChange{
		{Kind: evolution.WorkChangeAdd, Field: evolution.FieldNext, Body: "late"},
	}}); CodeOf(err) != "actmem_revision_conflict" {
		t.Fatalf("stale patch err=%v", err)
	}
}

// TestWorkOverflowRejects: a Work patch that would exceed the section cap is
// rejected without deleting other entries; bytes+revision preserved.
func TestWorkOverflowRejects(t *testing.T) {
	store := New(t.TempDir())
	big := strings.Repeat("z", 1200)
	if _, err := store.ApplyWorkPatch(personalScope(), evolution.WorkPatch{Changes: []evolution.WorkChange{
		{Kind: evolution.WorkChangeAdd, Field: evolution.FieldOpen, Body: big},
	}}); err != nil {
		t.Fatal(err)
	}
	headBefore, _ := os.ReadFile(store.HeadPath())
	_, err := store.ApplyWorkPatch(personalScope(), evolution.WorkPatch{BaseRevision: 1, Changes: []evolution.WorkChange{
		{Kind: evolution.WorkChangeAdd, Field: evolution.FieldGoal, Body: strings.Repeat("w", 800)},
	}})
	if CodeOf(err) != "actmem_cap_exceeded" {
		t.Fatalf("overflow err=%v", err)
	}
	headAfter, _ := os.ReadFile(store.HeadPath())
	if string(headBefore) != string(headAfter) {
		t.Fatal("rejected write mutated the head")
	}
}

// TestOwnerSave: whole-document save validates v2 grammar + caps; stale
// bases conflict; invalid input leaves the file byte-identical.
func TestOwnerSave(t *testing.T) {
	temp := t.TempDir()
	store := New(temp)
	valid := readFixture(t, "v2_two_scopes.md")
	result, err := store.Save(valid, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Revision != 1 {
		t.Fatalf("save result=%+v", result)
	}
	if _, err := store.Save(valid, 4); CodeOf(err) != "actmem_revision_conflict" {
		t.Fatalf("stale save err=%v", err)
	}
	before, _ := os.ReadFile(store.HeadPath())
	if _, err := store.Save(readFixture(t, "invalid_orphan_meta.md"), 1); err == nil {
		t.Fatal("invalid save accepted")
	}
	after, _ := os.ReadFile(store.HeadPath())
	if string(before) != string(after) {
		t.Fatal("invalid save mutated the head")
	}
}

// TestV1OpsRemovedOnClassifiedHead: scoped ops refuse a legacy head; writes
// do not silently reclassify.
func TestScopedOpsRejectLegacyHead(t *testing.T) {
	temp := t.TempDir()
	store := New(temp)
	writeHead(t, store, readFixture(t, "v1_legacy.md"))
	if _, err := store.AppendEntry(evolution.Entry{Section: evolution.SectionPulse, Scope: personalScope(), Body: "x"}); err == nil {
		t.Fatal("append on unclassified head")
	}
	if _, err := store.ApplyWorkPatch(personalScope(), evolution.WorkPatch{BaseRevision: 7}); err == nil {
		t.Fatal("patch on unclassified head")
	}
}
