package actmem

import (
	"context"
	"errors"
	"testing"
)

func TestRecapRecoveryCancellationKeepsReplacementBoundary(t *testing.T) {
	for _, tc := range []struct {
		name     string
		boundary string
		ioFault  bool
		wantSafe bool
	}{
		{"before_replace", "recovery_before_replace", false, true},
		{"after_replace", "recovery_after_replace", false, false},
		{"io_failure_with_canceled_context", "recovery_before_replace", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binding := RecapBinding{SubjectID: "subject", DomainID: "domain"}
			dir := t.TempDir()
			first, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := first.CommitRecap(context.Background(), RecapCommit{Binding: binding, OperationID: "one", Text: "kept", Cause: "edit"}); err != nil {
				t.Fatal(err)
			}
			if err := first.Close(); err != nil {
				t.Fatal(err)
			}
			owner, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = owner.Close() })
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fired := false
			fault := errors.New("recovery I/O failure")
			owner.recapFault = func(boundary string) error {
				if boundary == tc.boundary {
					fired = true
					cancel()
					if tc.ioFault {
						return fault
					}
					if boundary == "recovery_after_replace" {
						return ctx.Err()
					}
				}
				return nil
			}
			_, err = owner.ReadRecap(ctx, binding)
			owner.recapFault = nil
			if !fired {
				t.Fatal("recovery boundary was not reached")
			}
			if tc.wantSafe {
				if !errors.Is(err, context.Canceled) || CodeOf(err) == "recap_commit_unknown" {
					t.Errorf("harmless cancellation: %v", err)
				}
			} else {
				if CodeOf(err) != "recap_commit_unknown" {
					t.Fatalf("uncertain recovery: %v", err)
				}
				if tc.ioFault && !errors.Is(err, fault) {
					t.Fatalf("I/O cause lost: %v", err)
				}
				if _, err := owner.ReadRecap(context.Background(), binding); CodeOf(err) != "recap_commit_unknown" {
					t.Fatalf("uncertain owner served read: %v", err)
				}
				if _, err := owner.CommitRecap(context.Background(), RecapCommit{Binding: binding, OperationID: "two", BaseRevision: 1, Text: "next", Cause: "edit"}); CodeOf(err) != "recap_commit_unknown" {
					t.Fatalf("uncertain owner accepted write: %v", err)
				}
				if err := owner.Close(); err != nil {
					t.Fatal(err)
				}
				owner, err = Open(dir)
				if err != nil {
					t.Fatal(err)
				}
			}
			head, err := owner.ReadRecap(context.Background(), binding)
			if err != nil || head.Revision != 1 || head.Text != "kept" {
				t.Fatalf("retry: %+v %v", head, err)
			}
			head, err = owner.CommitRecap(context.Background(), RecapCommit{Binding: binding, OperationID: "two", BaseRevision: 1, Text: "next", Cause: "edit"})
			if err != nil || head.Revision != 2 || head.Text != "next" {
				t.Fatalf("write after recovery: %+v %v", head, err)
			}
		})
	}
}
