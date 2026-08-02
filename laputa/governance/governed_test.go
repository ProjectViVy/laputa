package governance

import (
	"context"
	"errors"
	"testing"
)

func newTestGoverned(t *testing.T) (*GovernedService, *FileAuditLog) {
	t.Helper()
	dir := t.TempDir()
	store, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	engine := NewEngine(store)
	audit, err := NewFileAuditLog(dir)
	if err != nil {
		t.Fatalf("create audit: %v", err)
	}
	return NewGovernedService(engine, audit), audit
}

func TestMutateAuthorized(t *testing.T) {
	g, _ := newTestGoverned(t)
	ctx := context.Background()
	err := g.Mutate(ctx, MutationRequest{
		Section: SectionCommitment,
		Action:  "write",
		Actor:   ActorUser,
		Reason:  "user edit",
		Data:    map[string]any{"red_lines": []string{"no deletion"}},
	})
	if err != nil {
		t.Fatalf("expected authorized: %v", err)
	}
	data, _ := g.GetSection(ctx, SectionCommitment)
	if data["red_lines"] == nil {
		t.Fatal("write did not persist")
	}
}

func TestMutateUnauthorized(t *testing.T) {
	g, _ := newTestGoverned(t)
	err := g.Mutate(context.Background(), MutationRequest{
		Section: SectionCommitment,
		Action:  "write",
		Actor:   ActorAgent,
		Reason:  "agent attempt",
		Data:    map[string]any{"red_lines": []string{}},
	})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}
}

func TestMutateReportSection(t *testing.T) {
	g, _ := newTestGoverned(t)
	ctx := context.Background()

	err := g.Mutate(ctx, MutationRequest{
		Section: SectionDaily,
		Action:  "write",
		Actor:   ActorAgent,
		Data:    map[string]any{"reports": []any{}},
	})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("agent should not write report section: %v", err)
	}

	err = g.Mutate(ctx, MutationRequest{
		Section: SectionDaily,
		Action:  "write",
		Actor:   ActorReportSystem,
		Reason:  "daily generation",
		Data:    map[string]any{"reports": []any{"entry"}},
	})
	if err != nil {
		t.Fatalf("report_system should write report section: %v", err)
	}
}

func TestMutateTBDSection(t *testing.T) {
	g, _ := newTestGoverned(t)
	err := g.Mutate(context.Background(), MutationRequest{
		Section: SectionProposalInbox,
		Action:  "write",
		Actor:   ActorAgent,
		Data:    map[string]any{},
	})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("agent should not write tbd section: %v", err)
	}
}

func TestAuditLogged(t *testing.T) {
	g, audit := newTestGoverned(t)
	ctx := context.Background()

	_ = g.Mutate(ctx, MutationRequest{
		Section:   SectionIdentity,
		Action:    "write",
		Actor:     ActorUser,
		Reason:    "identity update",
		RequestID: "req-1",
		Data:      map[string]any{"role": "companion"},
	})

	entries, err := audit.Recent(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Section != SectionIdentity || e.Actor != "user" || e.Reason != "identity update" {
		t.Fatalf("unexpected entry: %+v", e)
	}
	if e.Sequence != 1 {
		t.Fatalf("sequence=%d", e.Sequence)
	}
}

func TestAuditSkipsRoutineSTM(t *testing.T) {
	g, audit := newTestGoverned(t)
	ctx := context.Background()

	_ = g.Mutate(ctx, MutationRequest{
		Section: SectionMemoryMD,
		Action:  "write",
		Actor:   ActorSystem,
		Reason:  "checkpoint",
		Data:    map[string]any{"summary": "working"},
	})

	entries, _ := audit.Recent(ctx, 10)
	if len(entries) != 0 {
		t.Fatalf("routine STM write should not audit, got %d entries", len(entries))
	}
}

func TestRollbackRef(t *testing.T) {
	g, audit := newTestGoverned(t)
	ctx := context.Background()

	_ = g.Mutate(ctx, MutationRequest{
		Section: SectionCommitment,
		Action:  "write",
		Actor:   ActorUser,
		Data:    map[string]any{"red_lines": []string{"v1"}},
	})
	_ = g.Mutate(ctx, MutationRequest{
		Section: SectionCommitment,
		Action:  "write",
		Actor:   ActorUser,
		Data:    map[string]any{"red_lines": []string{"v2"}},
	})

	entries, _ := audit.Recent(ctx, 10)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[1].RollbackRef == "" {
		t.Fatal("second mutation should have rollback ref to prior state")
	}
	if entries[1].RollbackRef == entries[0].RollbackRef {
		t.Fatal("rollback refs should differ between states")
	}
}
