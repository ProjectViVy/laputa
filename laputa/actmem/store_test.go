package actmem

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestMissingHeadIsEmptyWithoutCreatingDirectories(t *testing.T) {
	temp := t.TempDir()
	store := New(temp)
	document, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	if document.Revision != 0 || document.UpdatedAt.Unix() != 0 {
		t.Fatalf("empty document = %#v", document)
	}
	if _, err := os.Stat(store.Root()); !os.IsNotExist(err) {
		t.Fatalf("read created ACTMEM root: %v", err)
	}
}

func TestAppendCASNoopAndRestart(t *testing.T) {
	temp := t.TempDir()
	clock := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	store := New(temp)
	store.SetClock(func() time.Time { return clock })
	first, err := store.AppendPulse("gui:one", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed || first.Document.Revision != 1 {
		t.Fatalf("first = %#v", first)
	}
	second, err := store.Put(ActmemPatch{Pulse: &first.Document.Pulse, BaseRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed || second.Document.Revision != 1 {
		t.Fatalf("no-op = %#v", second)
	}
	if _, err := store.Put(ActmemPatch{BaseRevision: 0}); CodeOf(err) != "actmem_revision_conflict" {
		t.Fatalf("CAS error = %v", err)
	}
	restarted := New(temp)
	document, err := restarted.Read()
	if err != nil || document.Revision != 1 || !strings.Contains(document.Pulse, "hello") {
		t.Fatalf("restart = %#v, %v", document, err)
	}
}

func TestWorkMaintenanceAndCapsuleLifecycle(t *testing.T) {
	temp := t.TempDir()
	store := New(temp)
	updated, err := store.EditWork("Open", "- pending", 0)
	if err != nil {
		t.Fatal(err)
	}
	updated, err = store.CompleteOpenItem(0, updated.Document.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(updated.Document.Work, "### Open") {
		t.Fatal("canonical work heading missing")
	}
	if _, err := store.EditWork("Unknown", "x", updated.Document.Revision); CodeOf(err) != "actmem_invalid_edit" {
		t.Fatalf("unknown section = %v", err)
	}
	tooLarge := "### Goal\n" + strings.Repeat("x", ACTMEMWorkCapChars+1)
	if _, err := store.Put(ActmemPatch{Work: &tooLarge, BaseRevision: updated.Document.Revision}); CodeOf(err) != "actmem_cap_exceeded" {
		t.Fatalf("cap error = %v", err)
	}
	if _, err := store.AppendPulse("session/one", "fold me"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendRecap("other", "keep me"); err != nil {
		t.Fatal(err)
	}
	capsules, err := store.FoldSession("session/one")
	if err != nil || len(capsules) != 1 {
		t.Fatalf("fold = %#v, %v", capsules, err)
	}
	if capsules[0].Chars > ACTMEMCapsuleCap {
		t.Fatalf("capsule exceeds cap: %#v", capsules[0])
	}
	head, _ := store.Read()
	if strings.Contains(head.Pulse, "fold me") || !strings.Contains(head.Recap, "keep me") {
		t.Fatalf("fold head = %#v", head)
	}
	read, err := store.ReadCapsule(capsules[0].Name)
	if err != nil || read.SessionKey != "session/one" {
		t.Fatalf("capsule = %#v, %v", read, err)
	}
	if err := store.DeleteCapsule("../escape.md"); CodeOf(err) != "actmem_capsule_invalid" {
		t.Fatalf("traversal = %v", err)
	}
	if err := store.DeleteCapsule(capsules[0].Name); err != nil {
		t.Fatal(err)
	}
}

func TestRecapFromFinalResponse(t *testing.T) {
	got := RecapFromFinalResponse("```go\nignored\n```\n\n完成了主路径。\n\n后文")
	if got != "完成了主路径。" {
		t.Fatalf("recap = %q", got)
	}
}

func TestQueryIsBoundedAndLiteral(t *testing.T) {
	store := New(t.TempDir())
	if _, err := store.AppendPulse("session", "Alpha decision"); err != nil {
		t.Fatal(err)
	}
	result, err := store.Query(QueryOptions{Query: "DECISION", MaxChars: 6})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.ReturnedChars > 6 || !result.Truncated {
		t.Fatalf("query = %#v", result)
	}
}
