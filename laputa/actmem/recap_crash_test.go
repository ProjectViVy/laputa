package actmem

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"testing"
)

func TestIssue58RecapFailureBoundaries(t *testing.T) {
	for _, boundary := range []string{"before_revision", "after_revision", "before_replace", "after_replace", "before_directory_sync", "after_directory_sync"} {
		t.Run(boundary, func(t *testing.T) {
			dir := t.TempDir()
			s, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			first := issue58Commit(t, s, RecapCommit{Binding: issue58Binding, OperationID: "first", Cause: "edit", Text: "先前原文"})
			s.recapFault = func(name string) error {
				if name == boundary {
					return errors.New("injected durability failure")
				}
				return nil
			}
			_, err = s.CommitRecap(context.Background(), RecapCommit{Binding: issue58Binding, BaseRevision: 1, OperationID: "attempt", Cause: "summary", Text: "新原文", Sources: SourceRange{Through: 3, Digest: "range"}})
			afterReplace := boundary == "after_replace" || boundary == "before_directory_sync" || boundary == "after_directory_sync"
			expected := "recap_unavailable"
			if afterReplace {
				expected = "recap_commit_unknown"
			}
			if CodeOf(err) != expected {
				t.Fatalf("wrong failure semantics: %v", err)
			}
			s.recapFault = nil
			if afterReplace {
				if _, err := s.CommitRecap(context.Background(), RecapCommit{Binding: issue58Binding, BaseRevision: 2, OperationID: "blocked", Cause: "edit", Text: "blocked"}); CodeOf(err) != "recap_commit_unknown" {
					t.Fatalf("unknown owner did not freeze: %v", err)
				}
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			got, found, err := reopened.LookupRecapOperation(context.Background(), issue58Binding, "attempt")
			if err != nil || found != afterReplace {
				t.Fatalf("operation resolution: %+v %v %v", got, found, err)
			}
			current, err := reopened.ReadRecap(context.Background(), issue58Binding)
			if err != nil {
				t.Fatal(err)
			}
			if afterReplace {
				if current.Revision != 2 || current.Text != "新原文" || current.CoveredThrough != 3 {
					t.Fatalf("post replacement tore head: %+v", current)
				}
			} else {
				if !reflect.DeepEqual(current, first) {
					t.Fatal("pre replacement did not retain prior head")
				}
				// A DIFFERENT operation at the same next revision must neither reuse nor
				// be obstructed by the immutable orphan from the failed attempt.
				next := issue58Commit(t, reopened, RecapCommit{Binding: issue58Binding, BaseRevision: 1, OperationID: "different", Cause: "edit", Text: "另一个操作"})
				if next.Revision != 2 || next.Text != "另一个操作" {
					t.Fatal("orphan blocked distinct retry")
				}
				history, err := reopened.ListRecapHistory(context.Background(), issue58Binding, 0, 20)
				if err != nil || len(history) != 2 || history[0].OperationID != "different" {
					t.Fatalf("orphan promoted: %+v %v", history, err)
				}
			}
		})
	}
}

func TestIssue58RecapCancellationBeforeReplace(t *testing.T) {
	s := issue58Store(t)
	first := issue58Commit(t, s, RecapCommit{Binding: issue58Binding, OperationID: "first", Cause: "edit", Text: "保留"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.recapFault = func(name string) error {
		if name == "before_replace" {
			cancel()
		}
		return nil
	}
	_, err := s.CommitRecap(ctx, RecapCommit{Binding: issue58Binding, BaseRevision: 1, OperationID: "cancelled", Cause: "edit", Text: "丢弃"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled commit: %v", err)
	}
	s.recapFault = nil
	got, err := s.ReadRecap(context.Background(), issue58Binding)
	if err != nil || !reflect.DeepEqual(got, first) {
		t.Fatal("cancellation altered head")
	}
	if _, found, err := s.LookupRecapOperation(context.Background(), issue58Binding, "cancelled"); err != nil || found {
		t.Fatal("cancelled orphan became receipt")
	}
}

// Actual process exit skips defers and releases the kernel lock, unlike a
// returned fault. It verifies recovery at each publication/durability boundary.
func TestIssue58RecapProcessCrashBoundaries(t *testing.T) {
	for _, boundary := range []string{"before_revision", "after_revision", "before_replace", "after_replace", "before_directory_sync", "after_directory_sync"} {
		t.Run(boundary, func(t *testing.T) {
			dir := t.TempDir()
			s, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			issue58Commit(t, s, RecapCommit{Binding: issue58Binding, OperationID: "first", Cause: "edit", Text: "原文"})
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestIssue58RecapCrashSubprocess$")
			cmd.Env = append(os.Environ(), "ISSUE58_CRASH_DIR="+dir, "ISSUE58_CRASH_BOUNDARY="+boundary)
			out, err := cmd.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 77 {
				t.Fatalf("unexpected crash: %s %v", out, err)
			}
			reopened, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			got, err := reopened.ReadRecap(context.Background(), issue58Binding)
			if err != nil {
				t.Fatal(err)
			}
			afterReplace := boundary == "after_replace" || boundary == "before_directory_sync" || boundary == "after_directory_sync"
			revision := uint64(1)
			text := "原文"
			if afterReplace {
				revision = 2
				text = "崩溃前的文本"
			}
			if got.Revision != revision || got.Text != text {
				t.Fatalf("crash exposed partial state: %+v", got)
			}
			_, found, err := reopened.LookupRecapOperation(context.Background(), issue58Binding, "crash-op")
			if err != nil || found != afterReplace {
				t.Fatalf("crash receipt: %v %v", found, err)
			}
		})
	}
}
func TestIssue58RecapCrashSubprocess(t *testing.T) {
	dir := os.Getenv("ISSUE58_CRASH_DIR")
	if dir == "" {
		return
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.recapFault = func(name string) error {
		if name == os.Getenv("ISSUE58_CRASH_BOUNDARY") {
			os.Exit(77)
		}
		return nil
	}
	_, err = s.CommitRecap(context.Background(), RecapCommit{Binding: issue58Binding, BaseRevision: 1, OperationID: "crash-op", Cause: "edit", Text: "崩溃前的文本"})
	t.Fatalf("crash boundary was not reached: %v", err)
}
