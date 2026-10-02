package persona

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeAndVisibleLen(t *testing.T) {
	if got := NormalizeMarkdown("  e\u0301\r\n猫  "); got != "é\n猫" {
		t.Fatalf("normalize = %q", got)
	}
	if got := VisibleLen("  e\u0301\r\n猫  "); got != 2 {
		t.Fatalf("visible_len = %d", got)
	}
}

func TestSectionExtractAndReplace(t *testing.T) {
	input := "# User\n\n## Preferences\nshort\n\n## Observations\nold"
	if body, ok := ExtractMarkdownSection(input, "Preferences"); !ok || body != "short" {
		t.Fatalf("extract = %q ok=%v", body, ok)
	}
	replaced := ReplaceMarkdownSection(input, "Observations", "new")
	if want := "# User\n\n## Preferences\nshort\n\n## Observations\nnew"; replaced != want {
		t.Fatalf("replace = %q", replaced)
	}
}

func TestKindContract(t *testing.T) {
	cases := []struct {
		kind         Kind
		file         string
		contentLimit int
		frozen       int
		frozenOK     bool
	}{
		{KindIdentity, "IDENTITY.MD", 800, 200, true},
		{KindRelationship, "RELATIONSHIP.MD", 600, 120, true},
		{KindRedline, "REDLINE.MD", 400, 200, true},
		{KindUser, "USER.MD", 800, 160, true},
		{KindWorld, "WORLD.MD", 1000, 0, false},
		{KindDream, "DREAM.MD", 40, 10, true},
		{KindDark, "DARK.MD", 300, 60, true},
	}
	for _, tc := range cases {
		if tc.kind.FileName() != tc.file {
			t.Errorf("%s file = %s", tc.kind, tc.kind.FileName())
		}
		if tc.kind.ContentLimit() != tc.contentLimit {
			t.Errorf("%s content limit = %d", tc.kind, tc.kind.ContentLimit())
		}
		fl, ok := tc.kind.FrozenLimit()
		if ok != tc.frozenOK || fl != tc.frozen {
			t.Errorf("%s frozen = %d,%v", tc.kind, fl, ok)
		}
	}
	if len(AllKinds) != 8 || len(RequiredKinds) != 5 || len(FrozenKinds) != 7 {
		t.Fatal("kind set sizes wrong")
	}
}

func writePersonaTree(t *testing.T, files map[string]string, history map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, "persona")
	if err := os.MkdirAll(filepath.Join(root, "history"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "requests"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for rel, content := range history {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func identityContent() string {
	body := "一个安静、专注、以证据为先的人格。"
	return body
}

func TestStatusUninitializedWhenAllRequiredAbsent(t *testing.T) {
	dir := writePersonaTree(t, map[string]string{"DREAM.MD": "留白"}, nil)
	svc, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	view, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != StatusUninitialized {
		t.Fatalf("status = %s", view.Status)
	}
	if !view.Files["dream"].Exists || view.Files["dream"].Valid {
		t.Fatal("optional dream should exist and be valid")
	}
	if view.Files["identity"].Valid {
		t.Fatal("missing required identity must be invalid with reason")
	}
	if view.Files["identity"].Reason == nil || *view.Files["identity"].Reason != "missing" {
		t.Fatal("identity reason should be missing")
	}
}

func TestStatusHistoryMismatch(t *testing.T) {
	content := identityContent()
	dir := writePersonaTree(t, map[string]string{"IDENTITY.MD": content}, nil)
	svc, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	view, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}
	state := view.Files["identity"]
	if state.Exists && state.Valid {
		t.Fatal("file without history must be flagged history_mismatch")
	}
	if state.Reason == nil || *state.Reason != "history_mismatch" {
		t.Fatalf("reason = %v", state.Reason)
	}
}

func TestStatusReadyWithMatchingHistory(t *testing.T) {
	content := identityContent()
	hash := ContentHash(content)
	dir := writePersonaTree(t, map[string]string{"IDENTITY.MD": content}, map[string]string{
		"history/IDENTITY.MD/1.md": content,
		"history/IDENTITY.MD/log.jsonl": `{"revision":1,"content_hash":"` + hash + `","snapshot":"1.md","diff":"1.diff","actor":"user","source":"init","reason":"init","base_revision":0,"created_at":"2026-08-14T00:00:00Z"}` + "\n",
	})
	svc, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	view, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}
	state := view.Files["identity"]
	if !state.Exists || !state.Valid {
		t.Fatalf("identity should be valid, got %+v", state)
	}
	if state.Revision != 1 {
		t.Fatalf("revision = %d", state.Revision)
	}
}

func TestGetDocumentAndHistory(t *testing.T) {
	content := identityContent()
	hash := ContentHash(content)
	dir := writePersonaTree(t, map[string]string{"IDENTITY.MD": content}, map[string]string{
		"history/IDENTITY.MD/1.md":  content,
		"history/IDENTITY.MD/1.diff": "--- IDENTITY.MD\n+++ IDENTITY.MD\n@@ -1 +1 @@\n+new\n",
		"history/IDENTITY.MD/log.jsonl": `{"revision":1,"content_hash":"` + hash + `","snapshot":"1.md","diff":"1.diff","actor":"user","source":"init","reason":"init","base_revision":0,"created_at":"2026-08-14T00:00:00Z"}` + "\n",
	})
	svc, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := svc.GetDocument(KindIdentity)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Content != content || doc.Revision != 1 || doc.ContentHash != hash {
		t.Fatalf("doc = %+v", doc)
	}
	rev, err := svc.ReadHistory(KindIdentity, 1)
	if err != nil {
		t.Fatal(err)
	}
	if rev.Content != content || rev.UnifiedDiff == "" || rev.Actor != "user" {
		t.Fatalf("history revision = %+v", rev)
	}
	if _, err := svc.ReadHistory(KindIdentity, 9); err == nil {
		t.Fatal("unknown revision should error")
	}
}

func TestGetDocumentUninitialized(t *testing.T) {
	dir := writePersonaTree(t, nil, nil)
	svc, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetDocument(KindIdentity); CodeOf(err) != "persona_uninitialized" {
		t.Fatalf("err = %v code = %s", err, CodeOf(err))
	}
}

func TestCapExceeded(t *testing.T) {
	dir := writePersonaTree(t, map[string]string{"DREAM.MD": "这是一个远远超过四十个可见字符上限的梦境描述，用来验证字数上限校验是否生效，继续加长以确保超过"}, nil)
	svc, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	// DIVA-verified behavior: cap violations propagate out of status().
	_, err = svc.Status()
	if err == nil {
		t.Fatal("oversized dream should fail status")
	}
	if CodeOf(err) != "persona_cap_exceeded" {
		t.Fatalf("code = %s err = %v", CodeOf(err), err)
	}
}
