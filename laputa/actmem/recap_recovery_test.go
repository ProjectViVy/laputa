package actmem

import (
	"context"
	"errors"
	"os"
	"reflect"
	"runtime"
	"testing"
)

func issue58Unknown(t *testing.T, s *Store) (RecapCommit, RecapHead) {
	t.Helper()
	request := RecapCommit{Binding: issue58Binding, OperationID: "uncertain", Cause: "summary", Text: "必须持久确认的原文", Sources: SourceRange{Through: 3, Digest: "host-range"}}
	s.recapFault = func(point string) error {
		if point == "before_directory_sync" {
			return errors.New("injected head publication failure")
		}
		return nil
	}
	if _, err := s.CommitRecap(context.Background(), request); CodeOf(err) != "recap_commit_unknown" {
		t.Fatalf("setup uncertainty: %v", err)
	}
	raw, err := os.ReadFile(s.HeadPath())
	if err != nil {
		t.Fatal(err)
	}
	visible, err := decodeRecap(raw)
	if err != nil {
		t.Fatal(err)
	}
	s.recapFault = nil
	return request, visible // test-only observation, never a durable receipt
}

func issue58AssertUnknown(t *testing.T, s *Store, request RecapCommit) {
	t.Helper()
	ctx := context.Background()
	if h, err := s.ReadRecap(ctx, request.Binding); CodeOf(err) != "recap_commit_unknown" || h != (RecapHead{}) {
		t.Errorf("read acknowledged uncertain head: %+v %v", h, err)
	}
	for _, op := range []string{request.OperationID, "absent-operation"} {
		if h, found, err := s.LookupRecapOperation(ctx, request.Binding, op); CodeOf(err) != "recap_commit_unknown" || found || h != (RecapHead{}) {
			t.Errorf("lookup acknowledged uncertain namespace: %+v %v %v", h, found, err)
		}
	}
	if history, err := s.ListRecapHistory(ctx, request.Binding, 0, 20); CodeOf(err) != "recap_commit_unknown" || len(history) != 0 {
		t.Errorf("history acknowledged uncertain head: %+v %v", history, err)
	}
	if h, err := s.ReadRecapRevision(ctx, request.Binding, 1); CodeOf(err) != "recap_commit_unknown" || h != (RecapHead{}) {
		t.Errorf("revision acknowledged uncertain head: %+v %v", h, err)
	}
	if h, err := s.CommitRecap(ctx, request); CodeOf(err) != "recap_commit_unknown" || h != (RecapHead{}) {
		t.Errorf("replay acknowledged uncertain head: %+v %v", h, err)
	}
}

func TestIssue58RecoveryUnknownCannotAcknowledge(t *testing.T) {
	s := issue58Store(t)
	request, _ := issue58Unknown(t, s)
	issue58AssertUnknown(t, s, request)
}

func TestIssue58RecoveryNewCannotReadPausedOwner(t *testing.T) {
	dir := t.TempDir()
	writer, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	issue58Commit(t, writer, RecapCommit{Binding: issue58Binding, OperationID: "first", Cause: "edit", Text: "先前"})
	paused := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	writer.recapFault = func(point string) error {
		if point == "after_replace" {
			close(paused)
			<-release
		}
		return nil
	}
	go func() {
		_, err := writer.CommitRecap(context.Background(), RecapCommit{Binding: issue58Binding, BaseRevision: 1, OperationID: "paused", Cause: "edit", Text: "尚未持久确认"})
		done <- err
	}()
	<-paused
	reader := New(dir)
	defer reader.Close()
	func() {
		defer close(release)
		if h, err := reader.ReadRecap(context.Background(), issue58Binding); CodeOf(err) != "recap_owner_busy" || h != (RecapHead{}) {
			t.Errorf("unowned New served paused head: %+v %v", h, err)
		}
		if h, found, err := reader.LookupRecapOperation(context.Background(), issue58Binding, "paused"); CodeOf(err) != "recap_owner_busy" || found || h != (RecapHead{}) {
			t.Errorf("unowned New served paused receipt: %+v %v %v", h, found, err)
		}
	}()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestIssue58RecoveryReopenMustConfirmExactReceipt(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	request, visible := issue58Unknown(t, s)
	expectedFiles := issue58Files(t, s.Root())
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		recovering, err := Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		called := false
		recovering.recapFault = func(point string) error {
			if point == "recovery_before_replace" {
				called = true
				return errors.New("recovery still cannot confirm")
			}
			return nil
		}
		issue58AssertUnknown(t, recovering, request)
		if !called {
			t.Error("fresh owner resolved without attempting recovery publication")
		}
		if err := recovering.Close(); err != nil {
			t.Fatal(err)
		}
	}
	recovered, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.Close()
	var barriers []string
	recovered.recapFault = func(point string) error { barriers = append(barriers, point); return nil }
	receipt, err := recovered.CommitRecap(context.Background(), request)
	if err != nil || !reflect.DeepEqual(receipt, visible) {
		t.Fatalf("recovery changed original receipt: %+v %v", receipt, err)
	}
	want := []string{"recovery_before_replace", "recovery_after_replace", "recovery_before_directory_sync", "recovery_after_directory_sync"}
	if !reflect.DeepEqual(barriers, want) {
		t.Fatalf("recovery publication ordering: %v", barriers)
	}
	if !reflect.DeepEqual(expectedFiles, issue58Files(t, recovered.Root())) {
		t.Fatal("same-byte recovery changed files/revision/history")
	}
	barriers = nil
	if _, found, err := recovered.LookupRecapOperation(context.Background(), request.Binding, request.OperationID); err != nil || !found {
		t.Fatalf("confirmed receipt lookup: %v %v", found, err)
	}
	if _, err := recovered.ReadRecap(context.Background(), request.Binding); err != nil {
		t.Fatal(err)
	}
	if len(barriers) != 0 {
		t.Fatal("unchanged confirmed owner repeated recovery flush")
	}
}

func TestIssue58RecoveryMissingHeadCannotResolveWithoutAbsenceBarrier(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("qualified Windows missing-head behavior has a dedicated platform test")
	}
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.recapFault = func(point string) error {
		if point == "after_revision" {
			return errors.New("orphan only")
		}
		return nil
	}
	request := RecapCommit{Binding: issue58Binding, OperationID: "orphan", Cause: "edit", Text: "不能提升的孤儿"}
	if _, err := s.CommitRecap(context.Background(), request); CodeOf(err) != "recap_unavailable" {
		t.Fatalf("orphan setup: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	failed, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	failed.recapFault = func(point string) error {
		if point == "recovery_before_absence_sync" {
			called = true
			return errors.New("absence unconfirmed")
		}
		return nil
	}
	issue58AssertUnknown(t, failed, request)
	if !called {
		t.Error("missing-head resolution omitted namespace absence barrier")
	}
	if err := failed.Close(); err != nil {
		t.Fatal(err)
	}
	recovered, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.Close()
	confirmations := 0
	recovered.recapFault = func(point string) error {
		if point == "recovery_after_absence_sync" {
			confirmations++
		}
		return nil
	}
	if h, found, err := recovered.LookupRecapOperation(context.Background(), issue58Binding, "orphan"); err != nil || found || h != (RecapHead{}) {
		t.Fatalf("orphan/absence recovery: %+v %v %v", h, found, err)
	}
	if confirmations != 1 {
		t.Fatalf("negative resolution had %d absence barriers", confirmations)
	}
	if _, err := os.Stat(recovered.HeadPath()); !os.IsNotExist(err) {
		t.Fatal("recovery promoted orphan or created head")
	}
	if h, err := recovered.ReadRecap(context.Background(), issue58Binding); err != nil || h.Revision != 0 {
		t.Fatalf("confirmed empty read: %+v %v", h, err)
	}
	if confirmations != 1 {
		t.Fatal("confirmed unchanged absence repeated full barrier")
	}
}

func TestIssue58RecoveryPublicationFailuresStayUnknown(t *testing.T) {
	for _, point := range []string{"recovery_before_replace", "recovery_after_replace", "recovery_before_directory_sync", "recovery_after_directory_sync"} {
		t.Run(point, func(t *testing.T) {
			dir := t.TempDir()
			s, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			request, visible := issue58Unknown(t, s)
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			failed, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			failed.recapFault = func(boundary string) error {
				if boundary == point {
					return errors.New("recovery publication failure")
				}
				return nil
			}
			issue58AssertUnknown(t, failed, request)
			if err := failed.Close(); err != nil {
				t.Fatal(err)
			}
			recovered := New(dir)
			defer recovered.Close()
			got, err := recovered.ReadRecap(context.Background(), request.Binding)
			if err != nil || !reflect.DeepEqual(got, visible) {
				t.Fatalf("new lazy owner failed exact recovery: %+v %v", got, err)
			}
			competitor, err := Open(dir)
			if competitor != nil {
				_ = competitor.Close()
			}
			if CodeOf(err) != "recap_owner_busy" {
				t.Fatalf("lazy authoritative read did not retain same owner: %v", err)
			}
		})
	}
}

func TestIssue58RecoveryFirstServiceMustCrossBarrier(t *testing.T) {
	for _, service := range []string{"read", "positive_lookup", "negative_lookup", "history", "revision", "replay"} {
		t.Run(service, func(t *testing.T) {
			dir := t.TempDir()
			s, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			request := RecapCommit{Binding: issue58Binding, OperationID: "confirmed-original", Cause: "edit", Text: "完整原文"}
			issue58Commit(t, s, request)
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			recovering, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer recovering.Close()
			called := false
			recovering.recapFault = func(point string) error {
				if point == "recovery_before_directory_sync" {
					called = true
					return errors.New("fresh owner confirmation failed")
				}
				return nil
			}
			ctx := context.Background()
			switch service {
			case "read":
				h, e := recovering.ReadRecap(ctx, request.Binding)
				err = e
				if h != (RecapHead{}) {
					t.Error("returned unconfirmed head")
				}
			case "positive_lookup", "negative_lookup":
				op := request.OperationID
				if service == "negative_lookup" {
					op = "missing"
				}
				h, found, e := recovering.LookupRecapOperation(ctx, request.Binding, op)
				err = e
				if found || h != (RecapHead{}) {
					t.Error("returned unconfirmed lookup data")
				}
			case "history":
				history, e := recovering.ListRecapHistory(ctx, request.Binding, 0, 20)
				err = e
				if len(history) != 0 {
					t.Error("returned unconfirmed history")
				}
			case "revision":
				h, e := recovering.ReadRecapRevision(ctx, request.Binding, 1)
				err = e
				if h != (RecapHead{}) {
					t.Error("returned unconfirmed revision")
				}
			case "replay":
				h, e := recovering.CommitRecap(ctx, request)
				err = e
				if h != (RecapHead{}) {
					t.Error("returned unconfirmed replay")
				}
			}
			if !called || CodeOf(err) != "recap_commit_unknown" {
				t.Fatalf("first %s bypassed durability barrier: called=%v err=%v", service, called, err)
			}
		})
	}
}
