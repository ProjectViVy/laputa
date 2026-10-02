package facade

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

const (
	receiptScopePersonal  = "scope/v1:personal:70726f66696c655f31"
	receiptScopeWorkspace = "scope/v1:workspace:70726f66696c655f31:77732d31"
	receiptDest           = "mentle"
)

func mutationService(t *testing.T) *Service {
	t.Helper()
	svc, err := OpenCatalogService(context.Background(), filepath.Join(t.TempDir(), "palace"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	return svc
}

func createReq(opID, scope, digest string) MutationRequest {
	return MutationRequest{
		OperationID: opID, Operation: "create", Scope: scope, DestinationID: receiptDest,
		PayloadDigest: digest, ExpectedAbsent: true,
		Body: "body", Kind: "note",
	}
}

func TestMutateCreateAndReplay(t *testing.T) {
	ctx := context.Background()
	svc := mutationService(t)
	req := createReq("op-1", receiptScopePersonal, "d1")

	r1, err := svc.Mutate(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if r1.RecordID == "" || r1.Revision != 1 {
		t.Fatalf("receipt = %+v", r1)
	}
	// Crash-after-commit: replaying the same request returns the same receipt.
	r2, err := svc.Mutate(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if r2.RecordID != r1.RecordID || r2.Revision != r1.Revision {
		t.Fatalf("replay returned %+v, want identical %+v", r2, r1)
	}
}

func TestMutateReplayConflicts(t *testing.T) {
	ctx := context.Background()
	svc := mutationService(t)
	req := createReq("op-c", receiptScopePersonal, "d1")
	if _, err := svc.Mutate(ctx, req); err != nil {
		t.Fatal(err)
	}
	// Same operation, different payload → conflict.
	bad := req
	bad.PayloadDigest = "d2"
	if _, err := svc.Mutate(ctx, bad); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed digest: %v", err)
	}
	// Same operation in another scope → conflict.
	bad = req
	bad.Scope = receiptScopeWorkspace
	if _, err := svc.Mutate(ctx, bad); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed scope: %v", err)
	}
	// Same operation, different destination → conflict.
	bad = req
	bad.DestinationID = "other"
	if _, err := svc.Mutate(ctx, bad); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed destination: %v", err)
	}
}

func TestMutateIdenticalTextSeparateScopes(t *testing.T) {
	ctx := context.Background()
	svc := mutationService(t)
	a, err := svc.Mutate(ctx, createReq("op-a", receiptScopePersonal, "same"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Mutate(ctx, createReq("op-b", receiptScopeWorkspace, "same"))
	if err != nil {
		t.Fatal(err)
	}
	if a.RecordID == b.RecordID {
		t.Fatalf("identical text in two scopes share record %q", a.RecordID)
	}
	recA, err := svc.GetMemory(ctx, a.RecordID)
	if err != nil {
		t.Fatal(err)
	}
	recB, err := svc.GetMemory(ctx, b.RecordID)
	if err != nil {
		t.Fatal(err)
	}
	if recA.Scope == recB.Scope {
		t.Fatalf("scopes collapsed to %q", recA.Scope)
	}
}

func TestMutateUpdateVersionAndScope(t *testing.T) {
	ctx := context.Background()
	svc := mutationService(t)
	created, err := svc.Mutate(ctx, createReq("op-u0", receiptScopePersonal, "d1"))
	if err != nil {
		t.Fatal(err)
	}
	upd := MutationRequest{
		OperationID: "op-u1", Operation: "update", Scope: receiptScopePersonal, DestinationID: receiptDest,
		PayloadDigest: "d2", RecordID: created.RecordID, ExpectedRevision: 1,
		Body: "new body",
	}
	r, err := svc.Mutate(ctx, upd)
	if err != nil {
		t.Fatal(err)
	}
	if r.Revision != 2 {
		t.Fatalf("update revision = %d, want 2", r.Revision)
	}
	// Replay identical → same receipt, no second bump.
	r2, err := svc.Mutate(ctx, upd)
	if err != nil || r2.Revision != 2 {
		t.Fatalf("replay update = (%+v, %v)", r2, err)
	}
	// Changed record id on replay → conflict.
	bad := upd
	bad.RecordID = "mem_other"
	if _, err := svc.Mutate(ctx, bad); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed record id: %v", err)
	}
	// Stale expected revision → version conflict.
	stale := upd
	stale.OperationID = "op-u2"
	stale.ExpectedRevision = 1
	if _, err := svc.Mutate(ctx, stale); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale revision: %v", err)
	}
	// Wrong scope → not found, no disclosure.
	wrongScope := upd
	wrongScope.OperationID = "op-u3"
	wrongScope.Scope = receiptScopeWorkspace
	if _, err := svc.Mutate(ctx, wrongScope); !errors.Is(err, ErrMutationNotFound) {
		t.Fatalf("foreign scope update: %v", err)
	}
}

func TestMutateTombstone(t *testing.T) {
	ctx := context.Background()
	svc := mutationService(t)
	created, err := svc.Mutate(ctx, createReq("op-t0", receiptScopePersonal, "d1"))
	if err != nil {
		t.Fatal(err)
	}
	tomb := MutationRequest{
		OperationID: "op-t1", Operation: "tombstone", Scope: receiptScopePersonal, DestinationID: receiptDest,
		PayloadDigest: "d3", RecordID: created.RecordID, ExpectedRevision: 1,
	}
	if _, err := svc.Mutate(ctx, tomb); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetMemory(ctx, created.RecordID); !errors.Is(err, ErrMemoryNotFound) {
		t.Fatalf("tombstoned record still readable: %v", err)
	}
	// Replaying after tombstone replays the receipt, not the operation.
	if _, err := svc.Mutate(ctx, tomb); err != nil {
		t.Fatalf("tombstone replay: %v", err)
	}
}

func TestMutationStatusBoundToScopeAndDestination(t *testing.T) {
	ctx := context.Background()
	svc := mutationService(t)
	created, err := svc.Mutate(ctx, createReq("op-s", receiptScopePersonal, "d1"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := svc.MutationStatus(ctx, "op-s", receiptScopePersonal, receiptDest)
	if err != nil || r.RecordID != created.RecordID {
		t.Fatalf("status = (%+v, %v)", r, err)
	}
	if _, err := svc.MutationStatus(ctx, "op-s", receiptScopeWorkspace, receiptDest); !errors.Is(err, ErrMutationNotFound) {
		t.Fatalf("foreign scope status: %v", err)
	}
	if _, err := svc.MutationStatus(ctx, "op-s", receiptScopePersonal, "other"); !errors.Is(err, ErrMutationNotFound) {
		t.Fatalf("foreign destination status: %v", err)
	}
}

func TestMutateValidatesRequest(t *testing.T) {
	ctx := context.Background()
	svc := mutationService(t)
	// Update without record id.
	if _, err := svc.Mutate(ctx, MutationRequest{OperationID: "x", Operation: "update", Scope: receiptScopePersonal, DestinationID: receiptDest, PayloadDigest: "d", ExpectedRevision: 1}); err == nil {
		t.Fatal("update without record id accepted")
	}
	// Create claiming an expected revision.
	bad := createReq("x2", receiptScopePersonal, "d")
	bad.ExpectedRevision = 3
	if _, err := svc.Mutate(ctx, bad); err == nil {
		t.Fatal("create with expected revision accepted")
	}
	// Unknown operation.
	if _, err := svc.Mutate(ctx, MutationRequest{OperationID: "x3", Operation: "nuke", Scope: receiptScopePersonal, DestinationID: receiptDest, PayloadDigest: "d"}); err == nil {
		t.Fatal("unknown operation accepted")
	}
}

func TestMutationReceiptSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "palace")
	svc, err := OpenCatalogService(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	created, err := svc.Mutate(ctx, createReq("op-r", receiptScopePersonal, "d1"))
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Close(); err != nil {
		t.Fatal(err)
	}
	svc2, err := OpenCatalogService(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = svc2.Close() }()
	r, err := svc2.Mutate(ctx, createReq("op-r", receiptScopePersonal, "d1"))
	if err != nil {
		t.Fatal(err)
	}
	if r.RecordID != created.RecordID || r.Revision != created.Revision {
		t.Fatalf("post-restart replay = %+v want %+v", r, created)
	}
}
