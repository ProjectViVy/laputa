package actmem

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestIssue58RecapOwnerExclusive(t *testing.T) {
	dir := t.TempDir()
	first, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if closer, ok := any(first).(interface{ Close() error }); ok {
		t.Cleanup(func() { _ = closer.Close() })
	}
	second, err := Open(dir)
	if err == nil {
		if closer, ok := any(second).(interface{ Close() error }); ok {
			_ = closer.Close()
		}
		t.Fatal("second owner acquired the same authority directory; want recap_owner_busy")
	}
	if CodeOf(err) != "recap_owner_busy" {
		t.Fatalf("got %v", err)
	}
}

func TestIssue58RecapOwnerAliasAndProcess(t *testing.T) {
	dir := t.TempDir()
	first, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = first.Close() })
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(dir, alias); err == nil {
		if second, err := Open(alias); CodeOf(err) != "recap_owner_busy" {
			if second != nil {
				_ = second.Close()
			}
			t.Fatalf("alias acquired owner: %v", err)
		}
	} else {
		t.Logf("symlink unavailable: %v", err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestIssue58OwnerSubprocess$")
	cmd.Env = append(os.Environ(), "ISSUE58_LOCK_DIR="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("process owner: %s %v", out, err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := Open(dir)
	if err != nil {
		t.Fatalf("close did not release owner: %v", err)
	}
	_ = second.Close()
}

func TestIssue58OwnerSubprocess(t *testing.T) {
	dir := os.Getenv("ISSUE58_LOCK_DIR")
	if dir == "" {
		return
	}
	second, err := Open(dir)
	if second != nil {
		_ = second.Close()
	}
	if CodeOf(err) != "recap_owner_busy" {
		t.Fatalf("subprocess acquired existing owner: %v", err)
	}
}

func TestIssue58LazyOwnerFencesLegacyMutators(t *testing.T) {
	dir := t.TempDir()
	first, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second := New(dir)
	defer second.Close()
	mutations := []func() error{
		func() error { _, e := second.AppendEntry(evolution.Entry{}); return e },
		func() error { _, e := second.Save("invalid", 0); return e },
		func() error { _, e := second.ApplyWorkPatch(personalScope(), evolution.WorkPatch{}); return e },
		func() error { _, e := second.FoldSession("session"); return e },
		func() error { _, e := second.AppendCaptured(capturePair(), AppendSnapshot{}); return e },
		func() error { return second.DeleteCapsule("missing.md") },
	}
	for i, mutate := range mutations {
		if err := mutate(); CodeOf(err) != "recap_owner_busy" {
			t.Fatalf("mutator %d bypassed owner: %v", i, err)
		}
	}
	entries, _ := os.ReadDir(first.Root())
	if len(entries) != 1 || entries[0].Name() != ".owner.lock" {
		t.Fatalf("denied writer changed files: %v", entries)
	}
}

var issue58Binding = RecapBinding{SubjectID: "主体-甲", DomainID: "个人"}

func issue58Store(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func issue58Commit(t *testing.T, s *Store, c RecapCommit) RecapHead {
	t.Helper()
	h, err := s.CommitRecap(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestIssue58RecapRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	text := "  用户纠正：日期是十月十八日。\n\n承诺：先核实，再回复。\n## 自由标题\n---\n```go\n已验证：测试通过。\n```\n\n"
	h := issue58Commit(t, s, RecapCommit{Binding: issue58Binding, OperationID: "summary-1", Cause: "summary", Text: text, Sources: SourceRange{Through: 3, Digest: "host-verified-完整区间"}})
	if h.Text != text || h.Revision != 1 || h.CoveredThrough != 3 {
		t.Fatalf("non-atomic/exact recap: %+v", h)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err := reopened.ReadRecap(context.Background(), issue58Binding)
	if err != nil || !reflect.DeepEqual(got, h) {
		t.Fatalf("restart lost exact head: %+v %v", got, err)
	}
	raw, _ := os.ReadFile(reopened.HeadPath())
	if strings.Contains(string(raw), "## Pulse\n") || strings.Contains(string(raw), "## Work\n") {
		t.Fatal("v3 serialized mandatory legacy sections")
	}
}

func TestIssue58RecapCAS(t *testing.T) {
	s := issue58Store(t)
	first := issue58Commit(t, s, RecapCommit{Binding: issue58Binding, OperationID: "first", Cause: "edit", Text: "原文"})
	before, _ := os.ReadFile(s.HeadPath())
	_, err := s.CommitRecap(context.Background(), RecapCommit{Binding: issue58Binding, OperationID: "stale", Cause: "edit", Text: "覆盖"})
	if CodeOf(err) != "recap_conflict" {
		t.Fatalf("stale overwrite: %v", err)
	}
	after, _ := os.ReadFile(s.HeadPath())
	if !bytes.Equal(before, after) {
		t.Fatal("conflict changed head")
	}
	for _, b := range []RecapBinding{{SubjectID: "other", DomainID: issue58Binding.DomainID}, {SubjectID: issue58Binding.SubjectID, DomainID: "other"}, {}} {
		if _, err := s.ReadRecap(context.Background(), b); CodeOf(err) != "recap_scope_denied" {
			t.Fatalf("read scope leaked: %v", err)
		}
		if _, err := s.CommitRecap(context.Background(), RecapCommit{Binding: b, BaseRevision: first.Revision, OperationID: "alien", Cause: "clear"}); CodeOf(err) != "recap_scope_denied" {
			t.Fatalf("write scope leaked: %v", err)
		}
		if _, _, err := s.LookupRecapOperation(context.Background(), b, "first"); CodeOf(err) != "recap_scope_denied" {
			t.Fatalf("operation scope leaked: %v", err)
		}
		if _, err := s.ListRecapHistory(context.Background(), b, 0, 20); CodeOf(err) != "recap_scope_denied" {
			t.Fatalf("history scope leaked: %v", err)
		}
		if _, err := s.ReadRecapRevision(context.Background(), b, 1); CodeOf(err) != "recap_scope_denied" {
			t.Fatalf("revision scope leaked: %v", err)
		}
	}
}

func TestIssue58RecapOperationReplay(t *testing.T) {
	s := issue58Store(t)
	req := RecapCommit{Binding: issue58Binding, OperationID: "op-1", Cause: "edit", Text: "原始收据"}
	original := issue58Commit(t, s, req)
	issue58Commit(t, s, RecapCommit{Binding: issue58Binding, BaseRevision: 1, OperationID: "op-2", Cause: "edit", Text: "较新内容"})
	replay, err := s.CommitRecap(context.Background(), req)
	if err != nil || !reflect.DeepEqual(replay, original) {
		t.Fatalf("replay did not return original: %+v %v", replay, err)
	}
	found, ok, err := s.LookupRecapOperation(context.Background(), issue58Binding, "op-1")
	if err != nil || !ok || !reflect.DeepEqual(found, original) {
		t.Fatalf("lookup original: %+v %v %v", found, ok, err)
	}
	req.Text = "变更载荷"
	if _, err := s.CommitRecap(context.Background(), req); CodeOf(err) != "recap_operation_conflict" {
		t.Fatalf("payload reuse accepted: %v", err)
	}
	req.Text = original.Text
	req.BaseRevision = 2
	if _, err := s.CommitRecap(context.Background(), req); CodeOf(err) != "recap_operation_conflict" {
		t.Fatalf("base reuse accepted: %v", err)
	}
}

func TestIssue58RecapClearRestoreAndRange(t *testing.T) {
	s := issue58Store(t)
	first := issue58Commit(t, s, RecapCommit{Binding: issue58Binding, OperationID: "s1", Cause: "summary", Text: "未完成的工作", Sources: SourceRange{Through: 5, Digest: "range-1"}})
	for i, r := range []SourceRange{{After: 4, Through: 6, Digest: "d"}, {After: 5, Through: 5, Digest: "d"}, {After: 5, Through: 6}} {
		_, err := s.CommitRecap(context.Background(), RecapCommit{Binding: issue58Binding, BaseRevision: 1, OperationID: fmt.Sprint("bad", i), Cause: "summary", Text: "x", Sources: r})
		if CodeOf(err) != "recap_invalid_range" {
			t.Fatalf("bad range accepted: %+v %v", r, err)
		}
	}
	clear := issue58Commit(t, s, RecapCommit{Binding: issue58Binding, BaseRevision: 1, OperationID: "clear", Cause: "clear"})
	if clear.Text != "" || clear.CoveredThrough != 5 || clear.InvalidationEpoch != 1 {
		t.Fatalf("clear semantics: %+v", clear)
	}
	emptyEdit := issue58Commit(t, s, RecapCommit{Binding: issue58Binding, BaseRevision: 2, OperationID: "empty-edit", Cause: "edit"})
	if emptyEdit.InvalidationEpoch != 1 || emptyEdit.CoveredThrough != 5 {
		t.Fatal("empty edit invalidated or rewound coverage")
	}
	summary := issue58Commit(t, s, RecapCommit{Binding: issue58Binding, BaseRevision: 3, OperationID: "s2", Cause: "summary", Text: "后来核实", Sources: SourceRange{After: 5, Through: 8, Digest: "range-2"}})
	restored := issue58Commit(t, s, RecapCommit{Binding: issue58Binding, BaseRevision: 4, OperationID: "restore", Cause: "restore", RestoredFrom: 1})
	if restored.Text != first.Text || restored.CoveredThrough != summary.CoveredThrough || restored.InvalidationEpoch != 1 {
		t.Fatalf("restore rewound state: %+v", restored)
	}
	_, err := s.CommitRecap(context.Background(), RecapCommit{Binding: issue58Binding, BaseRevision: 5, OperationID: "bad-restore", Cause: "restore", RestoredFrom: 1, Text: "caller text"})
	if CodeOf(err) != "recap_invalid_range" {
		t.Fatalf("restore trusted caller text: %v", err)
	}
	history, err := s.ListRecapHistory(context.Background(), issue58Binding, 0, 2)
	if err != nil || len(history) != 2 || history[0].Revision != 5 || history[1].Revision != 4 {
		t.Fatalf("history page: %+v %v", history, err)
	}
	next, err := s.ListRecapHistory(context.Background(), issue58Binding, history[1].Revision, 20)
	if err != nil || len(next) != 3 || next[0].Revision != 3 || next[2].Revision != 1 {
		t.Fatalf("history cursor: %+v %v", next, err)
	}
	old, err := s.ReadRecapRevision(context.Background(), issue58Binding, 1)
	if err != nil || !reflect.DeepEqual(old, first) {
		t.Fatalf("history text changed: %+v %v", old, err)
	}
}

func TestIssue58LegacyNonemptyPreserved(t *testing.T) {
	for _, raw := range []string{readFixture(t, "v2_multiline.md"), render(ActmemDocument{Revision: 3, UpdatedAt: time.Now(), Pulse: "keep", Work: emptyWork()})} {
		t.Run(fmt.Sprint(len(raw)), func(t *testing.T) {
			s := issue58Store(t)
			writeHead(t, s, raw)
			before, _ := os.ReadFile(s.HeadPath())
			if _, err := s.CommitRecap(context.Background(), RecapCommit{Binding: issue58Binding, OperationID: "no-migrate", Cause: "edit", Text: "new"}); CodeOf(err) != "recap_legacy_nonempty" {
				t.Fatalf("legacy overwrite: %v", err)
			}
			after, _ := os.ReadFile(s.HeadPath())
			if !bytes.Equal(before, after) {
				t.Fatal("legacy bytes changed")
			}
			if _, err := os.Stat(filepath.Join(s.Root(), "recap-history")); !os.IsNotExist(err) {
				t.Fatal("denied conversion wrote history")
			}
		})
	}
}

func TestIssue58LegacyWriterRejectsV3(t *testing.T) {
	s := issue58Store(t)
	issue58Commit(t, s, RecapCommit{Binding: issue58Binding, OperationID: "v3", Cause: "edit", Text: "自由文本"})
	before := issue58Files(t, s.Root())
	mutations := []func() error{
		func() error {
			_, err := s.AppendEntry(evolution.Entry{Section: evolution.SectionPulse, Scope: personalScope(), Body: "legacy"})
			return err
		},
		func() error { _, err := s.Save(emptyHead().markdown, 1); return err },
		func() error {
			_, err := s.ApplyWorkPatch(personalScope(), evolution.WorkPatch{BaseRevision: 1})
			return err
		},
		func() error { _, err := s.AppendCaptured(capturePair(), AppendSnapshot{}); return err },
		func() error { _, err := s.FoldSession("s-1"); return err },
		func() error { return s.DeleteCapsule("missing.md") },
	}
	for i, mutate := range mutations {
		if err := mutate(); CodeOf(err) != "recap_unavailable" {
			t.Fatalf("legacy mutator %d did not fail closed: %v", i, err)
		}
		if !reflect.DeepEqual(before, issue58Files(t, s.Root())) {
			t.Fatalf("legacy mutator %d changed files", i)
		}
	}
}

// issue58Files snapshots the owner-visible ACTMem content that must not change.
// The lifetime owner lock is ownership metadata without a content contract:
// Windows holds it under an exclusive byte-range lock for the whole owner
// lifetime, so reading it is neither possible nor meaningful. Excluding it keeps
// the durable-content assertion identical on every platform.
func issue58Files(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			if path == filepath.Join(root, ".owner.lock") {
				// Preserve presence/type evidence without reading the exclusively
				// locked bytes. Same-named nested files remain ordinary data.
				out[".owner.lock"] = "coordination:" + d.Type().String()
				return nil
			}
			raw, e := os.ReadFile(path)
			if e != nil {
				return e
			}
			rel, _ := filepath.Rel(root, path)
			out[rel] = string(raw)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestIssue58RecapEmptyLegacyAndExactBinding(t *testing.T) {
	for _, raw := range []string{render(ActmemDocument{Revision: 9, UpdatedAt: time.Now(), Work: emptyWork()}), emptyHead().markdown} {
		s := issue58Store(t)
		writeHead(t, s, raw)
		before, _ := os.ReadFile(s.HeadPath())
		view, err := s.ReadRecap(context.Background(), issue58Binding)
		if err != nil || view.Revision != 0 || view.Text != "" {
			t.Fatalf("empty legacy view: %+v %v", view, err)
		}
		after, _ := os.ReadFile(s.HeadPath())
		if !bytes.Equal(before, after) {
			t.Fatal("read converted legacy")
		}
		h := issue58Commit(t, s, RecapCommit{Binding: issue58Binding, OperationID: "init", Cause: "edit", Text: "新的自由文本\r\n\t尾行\n"})
		if h.Revision != 1 || h.Text != "新的自由文本\r\n\t尾行\n" {
			t.Fatalf("empty legacy init: %+v", h)
		}
		if _, err := s.ReadRecap(context.Background(), RecapBinding{SubjectID: issue58Binding.SubjectID, DomainID: "workspace"}); CodeOf(err) != "recap_scope_denied" {
			t.Fatalf("binding was not fixed: %v", err)
		}
	}
}

func TestIssue58RecapRejectsCorruptHeadAndHistory(t *testing.T) {
	for _, target := range []string{"head_text", "head_metadata", "history_text", "missing_history"} {
		t.Run(target, func(t *testing.T) {
			s := issue58Store(t)
			first := issue58Commit(t, s, RecapCommit{Binding: issue58Binding, OperationID: "first", Cause: "edit", Text: "承诺文本"})
			second := issue58Commit(t, s, RecapCommit{Binding: issue58Binding, BaseRevision: 1, OperationID: "second", Cause: "edit", Text: "当前文本"})
			path := s.HeadPath()
			if target == "history_text" || target == "missing_history" {
				path = s.recapRevisionPath(first)
			}
			raw, _ := os.ReadFile(path)
			switch target {
			case "head_text":
				raw = append(raw, []byte("未授权篡改")...)
			case "head_metadata":
				raw = bytes.Replace(raw, []byte("schema: actmem/v3\n"), []byte("schema: actmem/v3\nunknown: true\n"), 1)
			case "history_text":
				raw = append(raw, []byte("归档篡改")...)
			case "missing_history":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
			if target != "missing_history" {
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			before := issue58Files(t, s.Root())
			got, err := s.ReadRecap(context.Background(), issue58Binding)
			if CodeOf(err) != "recap_unavailable" {
				t.Fatalf("corrupt state exposed: %+v %v", got, err)
			}
			if _, err := s.CommitRecap(context.Background(), RecapCommit{Binding: issue58Binding, BaseRevision: second.Revision, OperationID: "overwrite", Cause: "edit", Text: "replacement"}); CodeOf(err) != "recap_unavailable" {
				t.Fatalf("corrupt state overwritten: %v", err)
			}
			if !reflect.DeepEqual(before, issue58Files(t, s.Root())) {
				t.Fatal("validation changed files")
			}
		})
	}
}

func TestIssue58RecapConcurrentCAS(t *testing.T) {
	s := issue58Store(t)
	var wg sync.WaitGroup
	var successes atomic.Int32
	errorsOut := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := s.CommitRecap(context.Background(), RecapCommit{Binding: issue58Binding, OperationID: fmt.Sprint("op", i), Cause: "edit", Text: fmt.Sprint("text", i)})
			if err == nil {
				successes.Add(1)
			} else if CodeOf(err) != "recap_conflict" {
				errorsOut <- err
			}
		}(i)
	}
	wg.Wait()
	close(errorsOut)
	for err := range errorsOut {
		t.Errorf("CAS error: %v", err)
	}
	if successes.Load() != 1 {
		t.Fatalf("CAS committed %d competing edits", successes.Load())
	}
	h, err := s.ReadRecap(context.Background(), issue58Binding)
	if err != nil || h.Revision != 1 {
		t.Fatalf("concurrent head: %+v %v", h, err)
	}
}

func TestIssue58RecapInvalidPayloadPreservesFiles(t *testing.T) {
	s := issue58Store(t)
	issue58Commit(t, s, RecapCommit{Binding: issue58Binding, OperationID: "first", Cause: "edit", Text: "保留"})
	before := issue58Files(t, s.Root())
	for _, req := range []RecapCommit{
		{Cause: "edit", Text: string([]byte{0xff})},
		{Cause: "clear", Text: "must not silently discard"},
		{Cause: "edit", Sources: SourceRange{Through: 9, Digest: "range"}},
		{Cause: "restore", RestoredFrom: 999},
		{Cause: "restore", RestoredFrom: 1, Text: "must not trust this"},
		{Cause: "edit", RestoredFrom: 1},
		{Cause: "summary", Sources: SourceRange{Through: 1, Digest: " "}},
		{Cause: "unknown"},
	} {
		req.Binding = issue58Binding
		req.BaseRevision = 1
		req.OperationID = "invalid"
		if _, err := s.CommitRecap(context.Background(), req); err == nil {
			t.Fatalf("invalid request accepted: %+v", req)
		}
		if !reflect.DeepEqual(before, issue58Files(t, s.Root())) {
			t.Fatal("invalid payload changed files")
		}
	}
}
