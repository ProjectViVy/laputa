package actmem

import (
	"context"
	"testing"
)

// Canceling the first Recap read after startup, while recovery is still before
// any head replacement, must not wedge the owner: the next call recovers.
func TestRecapRecoveryCancelBeforeReplaceDoesNotPoisonOwner(t *testing.T) {
	dir := t.TempDir()
	binding := RecapBinding{SubjectID: "subject", DomainID: "domain"}
	first, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.CommitRecap(context.Background(), RecapCommit{Binding: binding, OperationID: "op-1", Text: "kept", Cause: "edit"}); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	ctx, cancel := context.WithCancel(context.Background())
	reopened.recapFault = func(boundary string) error {
		if boundary == "recovery_before_replace" {
			cancel()
		}
		return nil
	}
	if _, err := reopened.ReadRecap(ctx, binding); err == nil {
		t.Fatal("canceled recovery reported success")
	} else if CodeOf(err) == "recap_commit_unknown" {
		t.Fatalf("cancel before replacement was reported as an unknown commit: %v", err)
	}
	reopened.recapFault = nil
	head, err := reopened.ReadRecap(context.Background(), binding)
	if err != nil || head.Text != "kept" || head.Revision != 1 {
		t.Fatalf("owner stayed wedged after a harmless cancel: %+v %v", head, err)
	}
	// A real commit still works afterwards.
	if _, err := reopened.CommitRecap(context.Background(), RecapCommit{Binding: binding, BaseRevision: 1, OperationID: "op-2", Text: "next", Cause: "edit"}); err != nil {
		t.Fatalf("commit after harmless cancel: %v", err)
	}
}
