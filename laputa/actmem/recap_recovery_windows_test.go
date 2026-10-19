//go:build windows

package actmem

import (
	"context"
	"os"
	"testing"
)

// Windows has no qualified missing-head namespace barrier in this owner. A zero
// view is uninitialized, never a durable not-found receipt or retirement signal.
func TestIssue58RecoveryWindowsInitialViewAndPositiveCommit(t *testing.T) {
	s := New(t.TempDir())
	defer s.Close()
	ctx := context.Background()
	h, err := s.ReadRecap(ctx, issue58Binding)
	if err != nil || h.Revision != 0 || h.CoveredThrough != 0 || h.OperationID != "" {
		t.Fatalf("uninitialized current view: %+v %v", h, err)
	}
	if got, found, err := s.LookupRecapOperation(ctx, issue58Binding, "pending-exact"); CodeOf(err) != "recap_commit_unknown" || found || got != (RecapHead{}) {
		t.Fatalf("unsupported negative resolution: %+v %v %v", got, found, err)
	}
	if history, err := s.ListRecapHistory(ctx, issue58Binding, 0, 20); CodeOf(err) != "recap_commit_unknown" || len(history) != 0 {
		t.Fatalf("unconfirmed empty history: %+v %v", history, err)
	}
	if got, err := s.ReadRecapRevision(ctx, issue58Binding, 1); CodeOf(err) != "recap_commit_unknown" || got != (RecapHead{}) {
		t.Fatalf("unconfirmed missing revision: %+v %v", got, err)
	}
	if _, err := os.Stat(s.HeadPath()); !os.IsNotExist(err) {
		t.Fatal("uninitialized view created a head")
	}
	request := RecapCommit{Binding: issue58Binding, OperationID: "pending-exact", Cause: "summary", Text: "实际首次持久提交", Sources: SourceRange{Through: 3, Digest: "host-range"}}
	receipt, err := s.CommitRecap(ctx, request)
	if err != nil || receipt.Revision != 1 || receipt.Text != request.Text || receipt.CoveredThrough != 3 {
		t.Fatalf("first actual publication disabled: %+v %v", receipt, err)
	}
	if got, found, err := s.LookupRecapOperation(ctx, issue58Binding, request.OperationID); err != nil || !found || got != receipt {
		t.Fatalf("positive durable receipt: %+v %v %v", got, found, err)
	}
}

func TestIssue58RecoveryWindowsOrphanDoesNotResolveMissingHead(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	request := RecapCommit{Binding: issue58Binding, OperationID: "initial-exact-retry", Cause: "edit", Text: "尚未提交的原文"}
	s.recapFault = func(point string) error {
		if point == "after_revision" {
			return os.ErrPermission
		}
		return nil
	}
	if _, err := s.CommitRecap(context.Background(), request); CodeOf(err) != "recap_unavailable" {
		t.Fatalf("orphan setup: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if h, err := reopened.ReadRecap(context.Background(), request.Binding); err != nil || h.Revision != 0 || h.CoveredThrough != 0 {
		t.Fatalf("orphan promoted to current: %+v %v", h, err)
	}
	if h, found, err := reopened.LookupRecapOperation(context.Background(), request.Binding, request.OperationID); CodeOf(err) != "recap_commit_unknown" || found || h != (RecapHead{}) {
		t.Fatalf("orphan/missing lookup acknowledged: %+v %v %v", h, found, err)
	}
	receipt, err := reopened.CommitRecap(context.Background(), request)
	if err != nil || receipt.Revision != 1 || receipt.Text != request.Text {
		t.Fatalf("exact first retry disabled: %+v %v", receipt, err)
	}
	history, err := reopened.ListRecapHistory(context.Background(), request.Binding, 0, 20)
	if err != nil || len(history) != 1 || history[0] != receipt {
		t.Fatalf("orphan appeared in history: %+v %v", history, err)
	}
}
