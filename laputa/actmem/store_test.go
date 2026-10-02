package actmem

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dashimaki/laputa/evolution"
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

func TestAppendCASAndRestart(t *testing.T) {
	temp := t.TempDir()
	clock := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	store := New(temp)
	store.SetClock(func() time.Time { return clock })
	first, err := store.AppendEntry(evolution.Entry{Section: evolution.SectionPulse, Scope: evolution.Scope{SubjectID: "sub", Kind: evolution.ScopePersonal}, SessionID: "gui:one", Body: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed || first.Revision != 1 {
		t.Fatalf("first = %#v", first)
	}
	restarted := New(temp)
	document, err := restarted.Read()
	if err != nil || document.Revision != 1 || !strings.Contains(document.Pulse, "hello") {
		t.Fatalf("restart = %#v, %v", document, err)
	}
	if document.Unclassified || len(document.Entries()) != 1 {
		t.Fatalf("restart document = %#v", document)
	}
}

func TestCapsuleLifecycle(t *testing.T) {
	temp := t.TempDir()
	store := New(temp)
	if _, err := store.AppendEntry(evolution.Entry{Section: evolution.SectionPulse, Scope: evolution.Scope{SubjectID: "sub", Kind: evolution.ScopeWorkspace, WorkspaceID: "w"}, SessionID: "s-1", Body: "folded line"}); err != nil {
		t.Fatal(err)
	}
	capsules, err := store.FoldSession("s-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(capsules) != 1 {
		t.Fatalf("capsules = %#v", capsules)
	}
	listed, err := store.ListCapsules()
	if err != nil || len(listed) != 1 {
		t.Fatalf("list = %#v, %v", listed, err)
	}
	full, err := store.ReadCapsule(capsules[0].Name)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(full.Markdown, "folded line") || full.SessionKey != "s-1" {
		t.Fatalf("capsule = %#v", full)
	}
	if err := store.DeleteCapsule(capsules[0].Name); err != nil {
		t.Fatal(err)
	}
	if listed, _ := store.ListCapsules(); len(listed) != 0 {
		t.Fatalf("delete failed: %#v", listed)
	}
}

func TestQueryRespectsScope(t *testing.T) {
	store := New(t.TempDir())
	for _, e := range []evolution.Entry{
		{Section: evolution.SectionPulse, Scope: evolution.Scope{SubjectID: "sub", Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-x"}, SessionID: "s", Body: "x scoped needle"},
		{Section: evolution.SectionPulse, Scope: evolution.Scope{SubjectID: "sub", Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-y"}, SessionID: "s", Body: "y scoped needle"},
	} {
		if _, err := store.AppendEntry(e); err != nil {
			t.Fatal(err)
		}
	}
	caller := evolution.Scope{SubjectID: "sub", Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-x"}
	result, err := store.Query(caller, QueryOptions{Query: "needle"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || !strings.Contains(result.Items[0].Excerpt, "x scoped") {
		t.Fatalf("query leaked or missed: %+v", result.Items)
	}
	if _, err := store.Query(caller, QueryOptions{Query: "needle", Sections: []string{"bogus"}}); err == nil {
		t.Fatal("unknown section accepted")
	}
}

func TestRecapFromFinalResponse(t *testing.T) {
	got := RecapFromFinalResponse("```go\nignored\n```\n\n完成了主路径。\n\n后文")
	if !strings.Contains(got, "完成了主路径") || strings.Contains(got, "ignored") {
		t.Fatalf("recap = %q", got)
	}
}
