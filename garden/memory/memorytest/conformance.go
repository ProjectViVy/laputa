// Package memorytest provides the shared backend-conformance suite and a
// minimal in-memory reference backend. Import it from adapter tests under
// garden/backends/<name>/ to verify a primary backend against
// contracts.md section 6.
package memorytest

import (
	"context"
	"strings"
	"testing"

	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/evolution"
)

// BackendFactory returns a backend bound to the given scope and
// destination. Factories for shared-store backends must connect the
// returned instance to the same underlying store as previous calls so
// cross-scope isolation is exercised against real stored data.
type BackendFactory func(t testing.TB, bound evolution.Scope, destination string) memory.Backend

// BackendConformance is the shared contract suite every primary backend
// must satisfy (contracts.md section 6): bound-writer mutations, scope
// isolation on reads, atomic receipts, and opaque cross-binding lookup.
func BackendConformance(t testing.TB, factory BackendFactory) {
	t.Helper()
	ctx := context.Background()
	subject := "subject-a"
	personal := evolution.Scope{SubjectID: subject, Kind: evolution.ScopePersonal}
	workspaceA := evolution.Scope{SubjectID: subject, Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-a"}
	workspaceB := evolution.Scope{SubjectID: subject, Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-b"}
	const dest = "test-destination"

	b := factory(t, workspaceA, dest)
	bP := factory(t, personal, dest)

	mustCreate := func(backend memory.Backend, scope evolution.Scope, op, body string) memory.MutationReceipt {
		t.Helper()
		receipt, err := backend.Mutate(ctx, memory.AuthorizedMutation{
			Scope:          scope,
			DestinationID:  dest,
			OperationID:    op,
			PayloadDigest:  "digest-" + op,
			Operation:      evolution.MutationCreate,
			RecordID:       "rec-" + op,
			ExpectedAbsent: true,
			Body:           body,
			Inference:      evolution.InferenceObserved,
		})
		if err != nil {
			t.Fatalf("create %s: %v", op, err)
		}
		return receipt
	}

	// Writes are bound: a mutation for another scope or destination is denied.
	if _, err := b.Mutate(ctx, memory.AuthorizedMutation{
		Scope:          personal,
		DestinationID:  dest,
		OperationID:    "op-foreign-scope",
		PayloadDigest:  "d-foreign-scope",
		Operation:      evolution.MutationCreate,
		RecordID:       "rec-foreign-scope",
		ExpectedAbsent: true,
		Body:           "foreign scope",
		Inference:      evolution.InferenceObserved,
	}); err == nil {
		t.Fatal("mutation under a foreign scope accepted")
	}
	if _, err := b.Mutate(ctx, memory.AuthorizedMutation{
		Scope:          workspaceA,
		DestinationID:  "other-backend",
		OperationID:    "op-foreign-dest",
		PayloadDigest:  "d-foreign-dest",
		Operation:      evolution.MutationCreate,
		RecordID:       "rec-foreign-dest",
		ExpectedAbsent: true,
		Body:           "foreign destination",
		Inference:      evolution.InferenceObserved,
	}); err == nil {
		t.Fatal("mutation under a foreign destination accepted")
	}

	r1 := mustCreate(b, workspaceA, "op-a", "alpha project needle")
	mustCreate(bP, personal, "op-p", "personal preference needle")

	// Search scoped to ws-a must not reveal personal records.
	page, err := b.Search(ctx, memory.AuthorizedSearch{Scopes: []evolution.Scope{workspaceA}, Query: "needle", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != r1.TargetRef {
		t.Fatalf("scoped search exposed out-of-scope data: %+v", page.Items)
	}

	// Identical text in different scopes stays separate.
	r3 := mustCreate(b, workspaceA, "op-a2", "shared sentence identical")
	r4 := mustCreate(bP, personal, "op-p2", "shared sentence identical")
	if r3.TargetRef == r4.TargetRef {
		t.Fatal("same body in two scopes collapsed into one record")
	}

	// Expansion under the wrong scope or a stale revision is refused.
	if _, err = b.Expand(ctx, memory.AuthorizedExpansion{Scopes: []evolution.Scope{workspaceB}, CardID: r1.TargetRef, ExpectedRevision: r1.Revision}); err == nil {
		t.Fatal("expansion across workspaces accepted")
	}
	if _, err = b.Expand(ctx, memory.AuthorizedExpansion{Scopes: []evolution.Scope{workspaceA}, CardID: r1.TargetRef, ExpectedRevision: r1.Revision + 1}); err == nil {
		t.Fatal("expansion with a stale revision accepted")
	}
	evidence, err := b.Expand(ctx, memory.AuthorizedExpansion{Scopes: []evolution.Scope{workspaceA}, CardID: r1.TargetRef, ExpectedRevision: r1.Revision, BudgetChars: 400})
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence.Items) == 0 {
		t.Fatal("scoped expansion returned no evidence")
	}

	// Receipt replay: same operation id + digest returns the committed
	// receipt; a changed digest conflicts.
	replay, err := b.Mutate(ctx, memory.AuthorizedMutation{
		Scope:          workspaceA,
		DestinationID:  dest,
		OperationID:    "op-a",
		PayloadDigest:  "digest-op-a",
		Operation:      evolution.MutationCreate,
		RecordID:       r1.TargetRef,
		ExpectedAbsent: true,
		Body:           "alpha project needle",
		Inference:      evolution.InferenceObserved,
	})
	if err != nil {
		t.Fatal(err)
	}
	if replay.Revision != r1.Revision {
		t.Fatalf("replay receipt = %+v want %d", replay, r1.Revision)
	}
	if _, err = b.Mutate(ctx, memory.AuthorizedMutation{
		Scope:          workspaceA,
		DestinationID:  dest,
		OperationID:    "op-a",
		PayloadDigest:  "DIFFERENT",
		Operation:      evolution.MutationCreate,
		RecordID:       r1.TargetRef,
		ExpectedAbsent: true,
		Body:           "alpha project needle",
		Inference:      evolution.InferenceObserved,
	}); err == nil {
		t.Fatal("changed digest replay accepted")
	}
	if _, err = b.MutationStatus(ctx, "op-a"); err != nil {
		t.Fatal(err)
	}
	if _, err = b.MutationStatus(ctx, "op-p"); err == nil {
		t.Fatal("foreign-scope receipt visible to this binding")
	}
}

// FakeBackend is the in-memory reference implementation run through the
// same conformance suite; it is intentionally minimal — a real backend
// keeps receipts in its canonical store, this one keeps them in a map.
type FakeBackend struct {
	store *FakeStore
	scope evolution.Scope
	dest  string
}

type fakeRecord struct {
	scope, body, status string
	revision            uint64
}

// FakeStore is shared by every FakeBackend from one factory so
// cross-scope reads see real cross-scope data.
type FakeStore struct {
	records  map[string]*fakeRecord
	receipts map[string]memory.MutationReceipt
	digests  map[string]string
	closed   bool
}

// NewFakeStore allocates a shared fake canonical store.
func NewFakeStore() *FakeStore {
	return &FakeStore{records: map[string]*fakeRecord{}, receipts: map[string]memory.MutationReceipt{}, digests: map[string]string{}}
}

// NewFakeBackend binds a fake backend to one scope/destination.
func NewFakeBackend(store *FakeStore, bound evolution.Scope, dest string) *FakeBackend {
	return &FakeBackend{store: store, scope: bound, dest: dest}
}

// NewFakeFactory returns a BackendFactory over one shared store.
func NewFakeFactory(store *FakeStore) BackendFactory {
	return func(t testing.TB, bound evolution.Scope, dest string) memory.Backend {
		return NewFakeBackend(store, bound, dest)
	}
}

func (f *FakeBackend) Capabilities() memory.Capabilities {
	return memory.Capabilities{Search: true, Expand: true, Mutate: true, MutationLookup: true}
}

func (f *FakeBackend) Search(_ context.Context, req memory.AuthorizedSearch) (memory.CardPage, error) {
	if err := req.Validate(); err != nil {
		return memory.CardPage{}, err
	}
	out := memory.CardPage{}
	for id, r := range f.store.records {
		scope, err := memory.DecodeScope(r.scope)
		if err != nil || !scopeIn(req.Scopes, scope) {
			continue
		}
		if r.status != "active" || !strings.Contains(r.body, req.Query) {
			continue
		}
		out.Items = append(out.Items, memory.MemoryCard{ID: id, Scope: r.scope, Summary: r.body, Revision: int(r.revision), Status: r.status})
	}
	return out, nil
}

func (f *FakeBackend) Expand(_ context.Context, req memory.AuthorizedExpansion) (memory.EvidencePage, error) {
	if err := req.Validate(); err != nil {
		return memory.EvidencePage{}, err
	}
	r, ok := f.store.records[req.CardID]
	if !ok || !scopeIn(req.Scopes, decodeOrZero(r.scope)) || r.status != "active" {
		return memory.EvidencePage{}, &evolution.ContractError{Code: evolution.ErrAuthorityDenied, Message: "cross-scope record access"}
	}
	if r.revision != req.ExpectedRevision {
		return memory.EvidencePage{}, &evolution.ContractError{Code: evolution.ErrRevisionConflict, Message: "record revision conflict"}
	}
	return memory.EvidencePage{Items: []memory.EvidenceFragment{{
		CardID: req.CardID, Scope: r.scope, Revision: r.revision, Status: r.status, Excerpt: r.body,
	}}}, nil
}

func (f *FakeBackend) Mutate(_ context.Context, req memory.AuthorizedMutation) (memory.MutationReceipt, error) {
	if err := req.Validate(); err != nil {
		return memory.MutationReceipt{}, err
	}
	if err := req.MatchesWriter(f.scope, f.dest); err != nil {
		return memory.MutationReceipt{}, err
	}
	key := memory.EncodeScope(req.Scope) + "|" + req.DestinationID + "|" + req.OperationID
	if stored, ok := f.store.receipts[key]; ok {
		if f.store.digests[key] != req.PayloadDigest {
			return memory.MutationReceipt{}, &evolution.ContractError{Code: evolution.ErrIdempotencyConflict, Message: "operation replayed with a different payload"}
		}
		return stored, nil
	}
	r := &fakeRecord{scope: memory.EncodeScope(req.Scope), body: req.Body, status: "active", revision: 1}
	f.store.records[req.RecordID] = r
	receipt := memory.MutationReceipt{
		EffectReceipt:   evolution.EffectReceipt{OperationID: req.OperationID, PayloadDigest: req.PayloadDigest, Status: evolution.StatusApplied, TargetRef: req.RecordID, Revision: r.revision},
		CanonicalStatus: memory.CanonicalCompleted,
		IndexStatus:     memory.IndexNotRequired,
	}
	f.store.receipts[key] = receipt
	f.store.digests[key] = req.PayloadDigest
	return receipt, nil
}

func (f *FakeBackend) MutationStatus(_ context.Context, operationID string) (memory.MutationReceipt, error) {
	key := memory.EncodeScope(f.scope) + "|" + f.dest + "|" + operationID
	if r, ok := f.store.receipts[key]; ok {
		return r, nil
	}
	return memory.MutationReceipt{}, &evolution.ContractError{Code: evolution.ErrEffectNotFound, Message: "receipt not found"}
}

func (f *FakeBackend) Health(context.Context) (memory.Health, error) {
	if f.store.closed {
		return memory.Health{Status: memory.HealthUnavailable, ReasonCode: "closed"}, nil
	}
	return memory.Health{Status: memory.HealthAvailable}, nil
}

func (f *FakeBackend) Close() error {
	f.store.closed = true
	return nil
}

func scopeIn(scopes []evolution.Scope, s evolution.Scope) bool {
	for _, c := range scopes {
		if c.Equal(s) {
			return true
		}
	}
	return false
}

func decodeOrZero(raw string) evolution.Scope {
	s, err := memory.DecodeScope(raw)
	if err != nil {
		return evolution.Scope{}
	}
	return s
}
