package actmem

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Removing the v3 archive branch makes this fail with recap_unavailable after
// the real legacy append/fold and native owner transition.
func TestIssue58CapturedArchiveSurvivesV3Owner(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	pair := capturePair()
	before, err := s.AppendSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	committed, err := s.AppendCaptured(pair, before)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.FoldSession(pair[0].SessionID); err != nil {
		t.Fatal(err)
	}
	head, err := s.CommitRecap(context.Background(), RecapCommit{Binding: issue58Binding, OperationID: "v3", Cause: "edit", Text: "真实主体记忆"})
	if err != nil {
		t.Fatal(err)
	}
	files := captureAuthorityFiles(t, s.Root())
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, found, err := s.LookupCaptured(pair)
	if err != nil || !found || got.Changed || got.Revision != 0 || !reflect.DeepEqual(got.Entries, committed.Entries) {
		t.Fatalf("archive-only actual entries (no legacy head revision): %+v %v %v", got, found, err)
	}
	again, err := s.ReadRecap(context.Background(), issue58Binding)
	if err != nil || again != head {
		t.Fatalf("lookup changed C1: %+v %v", again, err)
	}
	if got := captureAuthorityFiles(t, s.Root()); !reflect.DeepEqual(got, files) {
		t.Fatal("lookup changed authority/archive bytes")
	}
	changed := capturePair()
	changed[1].Body = "changed original event"
	if _, _, err = s.LookupCaptured(changed); CodeOf(err) != "actmem_capture_conflict" {
		t.Fatalf("archive conflict lost: %v", err)
	}
}

func TestIssue58CapturedV3LookupDoesNotMaskInvalidOwner(t *testing.T) {
	for _, state := range []string{"prefix", "missing_revision", "live_unknown"} {
		t.Run(state, func(t *testing.T) {
			s := New(t.TempDir())
			defer s.Close()
			head, err := s.CommitRecap(context.Background(), RecapCommit{Binding: issue58Binding, OperationID: "v3", Cause: "edit", Text: "真实主体记忆"})
			if err != nil {
				t.Fatal(err)
			}
			switch state {
			case "prefix":
				err = os.WriteFile(s.HeadPath(), []byte("---\nschema: actmem/v3\n---\nnot a valid chain"), 0600)
			case "missing_revision":
				err = os.Remove(s.recapRevisionPath(head))
			case "live_unknown":
				s.recapFault = func(point string) error {
					if point == "after_replace" {
						return errors.New("unconfirmed replacement")
					}
					return nil
				}
				_, err = s.CommitRecap(context.Background(), RecapCommit{Binding: issue58Binding, BaseRevision: head.Revision, OperationID: "uncertain", Cause: "edit", Text: "unconfirmed"})
				if CodeOf(err) != "recap_commit_unknown" {
					t.Fatal(err)
				}
				err = nil
			}
			if err != nil {
				t.Fatal(err)
			}
			if got, found, err := s.LookupCaptured(capturePair()); err == nil || found || len(got.Entries) != 0 {
				t.Fatalf("invalid C1 became archive lookup: %+v %v %v", got, found, err)
			}
		})
	}
}

func TestIssue58CapturedV3LookupRetainsMalformedArchiveFailure(t *testing.T) {
	s := New(t.TempDir())
	defer s.Close()
	pair := capturePair()
	before, err := s.AppendSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.AppendCaptured(pair, before); err != nil {
		t.Fatal(err)
	}
	capsules, err := s.FoldSession(pair[0].SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CommitRecap(context.Background(), RecapCommit{Binding: issue58Binding, OperationID: "v3", Cause: "edit", Text: "valid owner"}); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(s.CapsulesDir(), capsules[0].Name), []byte("malformed native fold proof"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, found, err := s.LookupCaptured(pair); err == nil || found || CodeOf(err) != "actmem_malformed" {
		t.Fatalf("malformed proof must remain the native archive error: %v %v", found, err)
	}
}

func captureAuthorityFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".md" {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			files[path] = string(raw)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
