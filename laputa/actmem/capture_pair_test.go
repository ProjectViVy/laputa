package actmem

import (
	"os"
	"reflect"
	"testing"

	"github.com/dashimaki/laputa/evolution"
)

func capturePair() []evolution.Entry {
	scope := personalScope()
	refs := []evolution.SourceRef{{SourceID: "garden.ingest", RecordID: "7", Revision: 7, Scope: scope}}
	return []evolution.Entry{
		{Section: evolution.SectionPulse, Scope: scope, SessionID: "s-1", EventID: "terminal-7", Body: "Primary run completed", Sources: refs},
		{Section: evolution.SectionRecap, Scope: scope, SessionID: "s-1", EventID: "terminal-7", Body: "User said:\n> synthetic user-only fact", Sources: refs},
	}
}

func TestCapturedPairCommitsOneRevisionAndRejoinsArchive(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	pair := capturePair()
	before, err := store.AppendSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.AppendCaptured(pair, before)
	if err != nil || !result.Changed || result.Revision != 1 || len(result.Entries) != 2 {
		t.Fatalf("atomic pair: %+v %v", result, err)
	}
	replay, err := store.AppendCaptured(pair, before)
	if err != nil || replay.Changed || !reflect.DeepEqual(replay.Entries, result.Entries) {
		t.Fatalf("original pair replay: %+v %v", replay, err)
	}
	if _, err := store.FoldSession("s-1"); err != nil {
		t.Fatal(err)
	}
	reopened := New(dir)
	recovered, found, err := reopened.LookupCaptured(pair)
	if err != nil || !found || !reflect.DeepEqual(recovered.Entries, result.Entries) {
		t.Fatalf("archive receipt lookup: %+v %v %v", recovered, found, err)
	}
	head, err := reopened.Read()
	if err != nil || len(head.Entries()) != 0 {
		t.Fatalf("lookup resurrected folded activity: %+v %v", head, err)
	}
}

func TestCapturedPairRejectsSecondBodyBeforeAnyWrite(t *testing.T) {
	store := New(t.TempDir())
	before, err := store.AppendSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	pair := capturePair()
	pair[1].Body = "<!-- actmem-entry:e_00000000000000000000000000000000 -->"
	if _, err := store.AppendCaptured(pair, before); err == nil {
		t.Fatal("invalid second body accepted")
	}
	if _, err := os.Stat(store.HeadPath()); !os.IsNotExist(err) {
		t.Fatalf("partial first entry written: %v", err)
	}
}

func TestCapturedPairFencesStaleHeadAndChangedPayload(t *testing.T) {
	store := New(t.TempDir())
	before, err := store.AppendSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ApplyWorkPatch(personalScope(), evolution.WorkPatch{BaseRevision: 0, Changes: []evolution.WorkChange{{Kind: evolution.WorkChangeAdd, Field: evolution.FieldGoal, Body: "actual new Work"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendCaptured(capturePair(), before); CodeOf(err) != "actmem_revision_conflict" {
		t.Fatalf("stale head accepted: %v", err)
	}
	now, err := store.AppendSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendCaptured(capturePair(), now); err != nil {
		t.Fatal(err)
	}
	changed := capturePair()
	changed[1].Body = "different content for original event"
	if _, _, err := store.LookupCaptured(changed); CodeOf(err) != "actmem_capture_conflict" {
		t.Fatalf("payload conflict not fenced: %v", err)
	}
}

func TestCapturedPairRefusesPartialLegacyEffect(t *testing.T) {
	store := New(t.TempDir())
	pair := capturePair()
	if _, err := store.AppendEntry(pair[0]); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.LookupCaptured(pair); found || CodeOf(err) != "actmem_recovery_required" {
		t.Fatalf("partial pair disguised as receipt: %v %v", found, err)
	}
}
