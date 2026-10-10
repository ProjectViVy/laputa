package agentapi

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/garden/memory/memorytest"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/laputa/persona"
)

func TestOwnedMissionPinRejectsZeroAndAssignedDrift(t *testing.T) {
	for _, assigned := range []bool{false, true} {
		name := "unassigned"
		if assigned {
			name = "assigned"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			cfg := userEmbeddedConfig(t)
			initAuthority(t, cfg)
			scope := evolution.Scope{SubjectID: cfg.ProfileID, Kind: evolution.ScopePersonal}
			cfg.BackendID, cfg.DestinationID = "fake", "fake-dest"
			cfg.Backends = map[string]memory.Backend{memory.EncodeScope(scope): memorytest.NewFakeBackend(memorytest.NewFakeStore(), scope, cfg.DestinationID)}
			client, err := Open(ctx, cfg)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			human, err := client.BindHumanSession("s1", "")
			if err != nil {
				t.Fatal(err)
			}
			var revision uint64
			if assigned {
				if _, err := human.SavePersona(ctx, persona.KindMission, "initial mission", 0, "initial"); err != nil {
					t.Fatal(err)
				}
				revision = 1
			}
			ports, err := client.BindEvolution(scope, cfg.DestinationID)
			if err != nil {
				t.Fatal(err)
			}
			pinned, err := ports.WithMissionRevision(revision)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := human.SavePersona(ctx, persona.KindMission, "changed mission", revision, "changed"); err != nil {
				t.Fatal(err)
			}
			patch := &evolution.WorkPatch{BaseRevision: 0, Changes: []evolution.WorkChange{{Kind: evolution.WorkChangeAdd, Field: evolution.FieldGoal, Body: "new goal"}}}
			effect := mustEffect(t, "old-pin", evolution.KindWorkPatch, patch, scope, cfg.DestinationID)
			if _, err := pinned.Apply(ctx, effect); evolution.CodeOf(err) != evolution.ErrMissionRevisionChanged {
				t.Fatalf("stale effect: %v", err)
			}
			if _, err := pinned.Lookup(ctx, effect.OperationID); evolution.CodeOf(err) != evolution.ErrMissionRevisionChanged {
				t.Fatalf("stale recovery: %v", err)
			}
			view, err := human.ReadActivity(ctx, ReadRequest{Sections: []evolution.EntrySection{evolution.SectionWork}})
			if err != nil || len(view.Entries) != 0 || view.Revision != 0 {
				t.Fatalf("old pin changed authority: %+v %v", view, err)
			}
			fresh, err := ports.WithMissionRevision(revision + 1)
			if err != nil {
				t.Fatal(err)
			}
			effect = mustEffect(t, "fresh-pin", evolution.KindWorkPatch, patch, scope, cfg.DestinationID)
			if receipt, err := fresh.Apply(ctx, effect); err != nil || receipt.Status != evolution.StatusApplied {
				t.Fatalf("fresh pin: %+v %v", receipt, err)
			}
		})
	}
}

type gatedMissionBackend struct {
	memory.Backend
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (b *gatedMissionBackend) Mutate(ctx context.Context, req memory.AuthorizedMutation) (memory.MutationReceipt, error) {
	b.once.Do(func() { close(b.entered) })
	select {
	case <-b.release:
	case <-ctx.Done():
		return memory.MutationReceipt{}, ctx.Err()
	}
	return b.Backend.Mutate(ctx, req)
}

func TestHumanMissionSaveWaitsForPinnedEffectCommit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cfg := userEmbeddedConfig(t)
	initAuthority(t, cfg)
	scope := evolution.Scope{SubjectID: cfg.ProfileID, Kind: evolution.ScopePersonal}
	cfg.BackendID, cfg.DestinationID = "fake", "fake-dest"
	backend := &gatedMissionBackend{Backend: memorytest.NewFakeBackend(memorytest.NewFakeStore(), scope, cfg.DestinationID), entered: make(chan struct{}), release: make(chan struct{})}
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(backend.release) }) }
	defer release()
	cfg.Backends = map[string]memory.Backend{memory.EncodeScope(scope): backend}
	client, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { release(); _ = client.Close() }()
	human, err := client.BindHumanSession("s1", "")
	if err != nil {
		t.Fatal(err)
	}
	ports, err := client.BindEvolution(scope, cfg.DestinationID)
	if err != nil {
		t.Fatal(err)
	}
	pinned, err := ports.WithMissionRevision(0)
	if err != nil {
		t.Fatal(err)
	}
	effect := mustEffect(t, "commit-before-edit", evolution.KindMemoryMutation, &evolution.MemoryMutationPayload{Operation: evolution.MutationCreate, RecordID: "test-record", ExpectedAbsent: true, Body: "test body", Inference: evolution.InferenceObserved}, scope, cfg.DestinationID)
	effectDone := make(chan error, 1)
	go func() { _, err := pinned.Apply(ctx, effect); effectDone <- err }()
	select {
	case <-backend.entered:
	case <-ctx.Done():
		t.Fatal("effect did not reach actual backend boundary")
	}
	editStarted, editDone := make(chan struct{}), make(chan error, 1)
	go func() {
		close(editStarted)
		_, err := human.SavePersona(ctx, persona.KindMission, "new mission", 0, "concurrent edit")
		editDone <- err
	}()
	<-editStarted
	select {
	case err := <-editDone:
		release()
		t.Fatalf("human Mission passed an uncommitted pinned effect: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	release()
	select {
	case err := <-effectDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	select {
	case err := <-editDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if _, err := pinned.Lookup(ctx, effect.OperationID); evolution.CodeOf(err) != evolution.ErrMissionRevisionChanged {
		t.Fatalf("old pin recovered after edit: %v", err)
	}
}
