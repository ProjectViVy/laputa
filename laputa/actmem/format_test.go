package actmem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProjectViVy/laputa/laputa/evolution"
)

func readFixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// TestV2GrammarRoundTrip: multiline bodies carrying headings and fences
// round-trip verbatim through the store kernel.
func TestV2GrammarRoundTrip(t *testing.T) {
	temp := t.TempDir()
	store := New(temp)
	writeHead(t, store, readFixture(t, "v2_multiline.md"))
	document, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	if document.Revision != 2 || document.Unclassified {
		t.Fatalf("document=%+v", document)
	}
	entries := document.Entries()
	if len(entries) != 1 || entries[0].ID != "e_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("entries=%+v", entries)
	}
	body := entries[0].Body
	for _, want := range []string{"First line.", "## A heading inside the body", "```go", `fmt.Println("fenced")`, "Last line."} {
		if !strings.Contains(body, want) {
			t.Fatalf("body lost %q: %q", want, body)
		}
	}
	// Re-render through the kernel and parse again: identical structure.
	parsed, err := evolution.ParseActmemDocument(document.Markdown)
	if err != nil {
		t.Fatalf("store markdown is not valid v2: %v", err)
	}
	got := parsed.Sections[evolution.SectionWork]
	if len(got) != 1 || got[0].Body != body {
		t.Fatalf("render round-trip body = %q", got[0].Body)
	}
}

// TestLegacyHeadIsUnclassified: a v1 head remains owner-readable but is
// excluded from every scoped projection until the owner re-saves v2.
func TestLegacyHeadIsUnclassified(t *testing.T) {
	temp := t.TempDir()
	store := New(temp)
	writeHead(t, store, readFixture(t, "v1_legacy.md"))
	document, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	if !document.Unclassified {
		t.Fatal("legacy head must be marked unclassified")
	}
	if !strings.Contains(document.Pulse, "legacy pulse") || !strings.Contains(document.Markdown, "legacy open item") {
		t.Fatalf("owner projection lost legacy content: %+v", document)
	}
	result, err := store.ReadScoped(evolution.Scope{SubjectID: "sub", Kind: evolution.ScopePersonal}, evolution.ReadRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Revision != 7 || len(result.Entries) != 0 {
		t.Fatalf("unclassified entries leaked: %+v", result)
	}
}

// TestScopedProjectionIsolatesScope: a scoped reader sees only personal plus
// their workspace — no text, ids or counts of hidden scope entries.
func TestScopedProjectionIsolatesScope(t *testing.T) {
	temp := t.TempDir()
	store := New(temp)
	writeHead(t, store, readFixture(t, "v2_two_scopes.md"))

	wx := evolution.Scope{SubjectID: "sub", Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-x"}
	result, err := store.ReadScoped(wx, evolution.ReadRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 3 {
		t.Fatalf("entries=%+v", result.Entries)
	}
	for _, entry := range result.Entries {
		if strings.Contains(entry.Body, "ws-y") || entry.ID == "e_22222222222222222222222222222222" {
			t.Fatalf("ws-y content leaked to ws-x: %+v", entry)
		}
	}

	// Personal-only caller sees only personal entries.
	personal := evolution.Scope{SubjectID: "sub", Kind: evolution.ScopePersonal}
	result, err = store.ReadScoped(personal, evolution.ReadRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 1 || result.Entries[0].ID != "e_44444444444444444444444444444444" {
		t.Fatalf("personal entries=%+v", result.Entries)
	}
}

// TestInvalidV2Rejected: orphan header metadata, duplicate and misplaced ids
// are invalid input — never silently reinterpreted.
func TestInvalidV2Rejected(t *testing.T) {
	temp := t.TempDir()
	store := New(temp)
	writeHead(t, store, readFixture(t, "invalid_orphan_meta.md"))
	if _, err := store.Read(); err == nil {
		t.Fatal("orphan header metadata accepted")
	}
}

func writeHead(t *testing.T, store *Store, markdown string) {
	t.Helper()
	if err := os.MkdirAll(store.Root(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.HeadPath(), []byte(markdown), 0o600); err != nil {
		t.Fatal(err)
	}
}
