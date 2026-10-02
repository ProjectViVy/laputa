package evolution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dashimaki/garden/internal/ingest"
	"github.com/dashimaki/garden/memory"
	"github.com/dashimaki/laputa/actmem"
	laputaevolution "github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/laputa/persona"
	"github.com/dashimaki/mentle/facade"
)

func testScope() laputaevolution.Scope {
	return laputaevolution.Scope{SubjectID: "profile_a", Kind: laputaevolution.ScopePersonal}
}

// fakeWriter satisfies ingest.MemoryWriter; CreateMemory is deduped by its
// idempotency key like the real facade.
type fakeWriter struct {
	mu    sync.Mutex
	memos map[string]facade.Memory
	seq   int
}

func (f *fakeWriter) CreateMemory(_ context.Context, req facade.CreateMemoryRequest, key, _ string) (facade.Memory, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.memos == nil {
		f.memos = map[string]facade.Memory{}
	}
	if m, ok := f.memos[key]; ok {
		return m, nil
	}
	f.seq++
	m := facade.Memory{ID: fmt.Sprintf("mem_%d", f.seq), Kind: req.Kind, Content: req.Content}
	f.memos[key] = m
	return m, nil
}

func (f *fakeWriter) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.memos)
}

func openIngest(t *testing.T, writer *fakeWriter) *ingest.Service {
	t.Helper()
	svc, err := ingest.Open(filepath.Join(t.TempDir(), "ingest.db"), writer)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	return svc
}

func submitCapture(t *testing.T, svc *ingest.Service, sessionID, eventID, workspace, body string) ingest.Accepted {
	t.Helper()
	sum := sha256.Sum256([]byte(body))
	accepted, err := svc.Submit(context.Background(), ingest.SubmitRequest{
		SessionID: sessionID, EventID: eventID, Phase: "session_end",
		Content: body, ContentHash: "sha256:" + hex.EncodeToString(sum[:]),
		Workspace: workspace, OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return accepted
}

func openPersona(t *testing.T) *persona.Service {
	t.Helper()
	svc, err := persona.Open(filepath.Join(t.TempDir(), "persona"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Initialize(persona.Initialization{
		Identity: "id", Relationship: "rel", Redline: "red", User: "user", World: "world",
	}, "test", persona.SourceInit, "init"); err != nil {
		t.Fatal(err)
	}
	return svc
}

// fakeBackend satisfies memory.Backend with an operation-id keyed receipt map.
type fakeBackend struct {
	mu       sync.Mutex
	mutated  []memory.AuthorizedMutation
	receipts map[string]memory.MutationReceipt
	fail     error
}

func (b *fakeBackend) Capabilities() memory.Capabilities { return memory.Capabilities{} }
func (b *fakeBackend) Search(context.Context, memory.AuthorizedSearch) (memory.CardPage, error) {
	return memory.CardPage{}, nil
}
func (b *fakeBackend) Expand(context.Context, memory.AuthorizedExpansion) (memory.EvidencePage, error) {
	return memory.EvidencePage{}, nil
}
func (b *fakeBackend) Mutate(_ context.Context, m memory.AuthorizedMutation) (memory.MutationReceipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.fail != nil {
		return memory.MutationReceipt{}, b.fail
	}
	if b.receipts == nil {
		b.receipts = map[string]memory.MutationReceipt{}
	}
	if r, ok := b.receipts[m.OperationID]; ok {
		return r, nil
	}
	r := memory.MutationReceipt{EffectReceipt: laputaevolution.EffectReceipt{
		OperationID: m.OperationID, PayloadDigest: m.PayloadDigest,
		Status: laputaevolution.StatusApplied, TargetRef: "memory:" + m.RecordID, Revision: 1,
	}, CanonicalStatus: memory.CanonicalCompleted, IndexStatus: memory.IndexReady}
	b.receipts[m.OperationID] = r
	b.mutated = append(b.mutated, m)
	return r, nil
}
func (b *fakeBackend) MutationStatus(_ context.Context, opID string) (memory.MutationReceipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if r, ok := b.receipts[opID]; ok {
		return r, nil
	}
	return memory.MutationReceipt{}, &laputaevolution.ContractError{Code: laputaevolution.ErrEffectNotFound, Message: "unknown operation"}
}
func (b *fakeBackend) Health(context.Context) (memory.Health, error) { return memory.Health{}, nil }
func (b *fakeBackend) Close() error                                  { return nil }

type fakeProposer struct {
	got []ProposedCapability
}

func (p *fakeProposer) SubmitCapabilityProposal(_ context.Context, in ProposedCapability) (string, error) {
	p.got = append(p.got, in)
	return "evomap:run_" + in.Name, nil
}

func newDomain(t *testing.T, deps Deps) *Domain {
	t.Helper()
	if deps.Dir == "" {
		deps.Dir = t.TempDir()
	}
	d, err := NewDomain(deps)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestCollectProjectsCommittedWindow(t *testing.T) {
	writer := &fakeWriter{}
	ing := openIngest(t, writer)
	submitCapture(t, ing, "sess_1", "run_1:3", "", "first exchange")
	submitCapture(t, ing, "sess_1", "run_2:4", "", "second exchange")
	submitCapture(t, ing, "sess_1", "run_3:2", "ws_other", "other workspace")

	d := newDomain(t, Deps{
		Scope: testScope(), DestinationID: "dest_a",
		Actmem: actmem.New(t.TempDir()), Persona: openPersona(t), Activity: ing,
	})
	batch, err := d.Collect(context.Background(), laputaevolution.Window{SourceID: "activity", After: 0, Through: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Entries) != 2 {
		t.Fatalf("expected 2 scope-visible entries, got %d", len(batch.Entries))
	}
	if batch.Entries[0].EventID != "run_1:3" || batch.Entries[0].Body != "first exchange" {
		t.Fatalf("unexpected first entry: %+v", batch.Entries[0])
	}
	if len(batch.Persona) < 5 {
		t.Fatalf("expected authority views, got %d", len(batch.Persona))
	}
	// (2,3] selects only the middle row.
	narrow, err := d.Collect(context.Background(), laputaevolution.Window{SourceID: "activity", After: 1, Through: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(narrow.Entries) != 1 || narrow.Entries[0].EventID != "run_2:4" {
		t.Fatalf("window bounds wrong: %+v", narrow.Entries)
	}
}

func TestLookupReconcilesCommittedEffects(t *testing.T) {
	mem := &fakeBackend{}
	d := newDomain(t, Deps{Scope: testScope(), DestinationID: "dest_a", Memory: mem})
	scope := testScope()
	effect, err := laputaevolution.NewEffect("op:e0", laputaevolution.KindMemoryMutation,
		&laputaevolution.MemoryMutationPayload{
			Operation: laputaevolution.MutationCreate, RecordID: "rec_1",
			ExpectedAbsent: true, Body: "remembered", Inference: laputaevolution.InferenceInferred,
		}, scope, "dest_a")
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := d.Apply(context.Background(), effect)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != laputaevolution.StatusApplied || receipt.TargetRef != "memory:rec_1" {
		t.Fatalf("unexpected receipt %+v", receipt)
	}
	if len(mem.mutated) != 1 {
		t.Fatalf("expected one backend mutation, got %d", len(mem.mutated))
	}
	// Fresh adapter instance (restart) must find the committed receipt.
	d2 := newDomain(t, Deps{Scope: scope, DestinationID: "dest_a", Memory: mem, Dir: d.deps.Dir})
	again, err := d2.Lookup(context.Background(), "op:e0")
	if err != nil {
		t.Fatal(err)
	}
	if again.Status != laputaevolution.StatusApplied || again.PayloadDigest != effect.PayloadDigest {
		t.Fatalf("restart lookup lost receipt: %+v", again)
	}
	// A backend-committed mutation that never reached the ledger still
	// reconciles through the backend's own receipt.
	fx, err := laputaevolution.NewEffect("op:e9", laputaevolution.KindMemoryMutation,
		&laputaevolution.MemoryMutationPayload{
			Operation: laputaevolution.MutationCreate, RecordID: "rec_9",
			ExpectedAbsent: true, Body: "x", Inference: laputaevolution.InferenceObserved,
		}, scope, "dest_a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mem.Mutate(context.Background(), memory.AuthorizedMutation{
		Scope: scope, DestinationID: "dest_a", OperationID: fx.OperationID, PayloadDigest: fx.PayloadDigest,
		Operation: laputaevolution.MutationCreate, RecordID: "rec_9", ExpectedAbsent: true, Body: "x",
		Inference: laputaevolution.InferenceObserved,
	}); err != nil {
		t.Fatal(err)
	}
	backendOnly, err := d2.Lookup(context.Background(), "op:e9")
	if err != nil {
		t.Fatal(err)
	}
	if backendOnly.Status != laputaevolution.StatusApplied {
		t.Fatalf("backend receipt not reconciled: %+v", backendOnly)
	}
}

func TestLookupUnknownIsNotFound(t *testing.T) {
	d := newDomain(t, Deps{Scope: testScope(), DestinationID: "dest_a"})
	_, err := d.Lookup(context.Background(), "op:nope")
	if laputaevolution.CodeOf(err) != laputaevolution.ErrEffectNotFound {
		t.Fatalf("expected effect_not_found, got %v", err)
	}
}

func TestWorkPatchApplyAndReplay(t *testing.T) {
	store := actmem.New(t.TempDir())
	d := newDomain(t, Deps{Scope: testScope(), DestinationID: "dest_a", Actmem: store})
	patch := &laputaevolution.WorkPatch{Changes: []laputaevolution.WorkChange{{
		Kind: laputaevolution.WorkChangeAdd, Field: laputaevolution.FieldGoal, Body: "ship S07",
	}}}
	effect, err := laputaevolution.NewEffect("op:work", laputaevolution.KindWorkPatch, patch, testScope(), "dest_a")
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := d.Apply(context.Background(), effect)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != laputaevolution.StatusApplied || receipt.Revision == 0 {
		t.Fatalf("expected applied receipt with revision, got %+v", receipt)
	}
	replayed, err := d.Lookup(context.Background(), "op:work")
	if err != nil {
		t.Fatal(err)
	}
	if replayed != receipt {
		t.Fatalf("replay diverged: %+v vs %+v", replayed, receipt)
	}
}

func TestPersonaRequestSubmitsReview(t *testing.T) {
	psvc := openPersona(t)
	d := newDomain(t, Deps{Scope: testScope(), DestinationID: "dest_a", Persona: psvc})
	doc, err := psvc.GetDocument(persona.KindIdentity)
	if err != nil {
		t.Fatal(err)
	}
	effect, err := laputaevolution.NewEffect("op:e0", laputaevolution.KindPersonaRequest,
		&laputaevolution.PersonaRequestPayload{
			Kind: laputaevolution.PersonaRequestIdentity, BaseRevision: doc.Revision,
			ProposedMarkdown: "new identity", Reason: "observed shift",
		}, testScope(), "dest_a")
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := d.Apply(context.Background(), effect)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != laputaevolution.StatusSubmitted {
		t.Fatalf("expected submitted, got %+v", receipt)
	}
	reqs, err := psvc.ListRequests(nil)
	if err != nil || len(reqs) != 1 {
		t.Fatalf("expected one pending review, got %v %v", reqs, err)
	}
	if reqs[0].Actor != persona.ActorAgent || reqs[0].State != persona.RequestPending {
		t.Fatalf("unexpected review: %+v", reqs[0])
	}
	// Stale base revision rejects instead of submitting.
	stale, err := laputaevolution.NewEffect("op:e1", laputaevolution.KindPersonaRequest,
		&laputaevolution.PersonaRequestPayload{
			Kind: laputaevolution.PersonaRequestIdentity, BaseRevision: doc.Revision + 9,
			ProposedMarkdown: "other", Reason: "r",
		}, testScope(), "dest_a")
	if err != nil {
		t.Fatal(err)
	}
	staleReceipt, err := d.Apply(context.Background(), stale)
	if err != nil {
		t.Fatal(err)
	}
	if staleReceipt.Status != laputaevolution.StatusRejected || staleReceipt.ErrorCode != "persona_revision_mismatch" {
		t.Fatalf("stale revision not rejected: %+v", staleReceipt)
	}
}

func TestReflectionNoteIsDurableButNotEvidence(t *testing.T) {
	ing := openIngest(t, &fakeWriter{})
	dir := t.TempDir()
	d := newDomain(t, Deps{Scope: testScope(), DestinationID: "dest_a", Activity: ing, Dir: dir})
	effect, err := laputaevolution.NewEffect("op:e0", laputaevolution.KindReflectionNote,
		&laputaevolution.ReflectionNotePayload{Body: "learned X"}, testScope(), "dest_a")
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := d.Apply(context.Background(), effect)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != laputaevolution.StatusApplied {
		t.Fatalf("note receipt: %+v", receipt)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.jsonl")); err != nil {
		t.Fatalf("note not persisted: %v", err)
	}
	// Notes never enter the committed-activity window as evidence.
	batch, err := d.Collect(context.Background(), laputaevolution.Window{SourceID: "activity", After: 0, Through: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Entries) != 0 {
		t.Fatalf("reflection note leaked into evidence: %+v", batch.Entries)
	}
}

func TestCapabilityProposalReachesProposerOnly(t *testing.T) {
	prop := &fakeProposer{}
	d := newDomain(t, Deps{Scope: testScope(), DestinationID: "dest_a", Proposals: prop})
	effect, err := laputaevolution.NewEffect("op:e0", laputaevolution.KindCapabilityProposal,
		&laputaevolution.CapabilityProposalPayload{Name: "summarizer", ProposedArtifact: "skill body"}, testScope(), "dest_a")
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := d.Apply(context.Background(), effect)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != laputaevolution.StatusSubmitted || len(prop.got) != 1 {
		t.Fatalf("proposal not submitted: %+v", receipt)
	}
}

func TestUnboundPortsReject(t *testing.T) {
	d := newDomain(t, Deps{Scope: testScope(), DestinationID: "dest_a"})
	for _, effect := range []laputaevolution.Effect{
		mustEffect(t, laputaevolution.KindWorkPatch, &laputaevolution.WorkPatch{Changes: []laputaevolution.WorkChange{{Kind: laputaevolution.WorkChangeAdd, Field: laputaevolution.FieldGoal, Body: "g"}}}),
		mustEffect(t, laputaevolution.KindMemoryMutation, &laputaevolution.MemoryMutationPayload{Operation: laputaevolution.MutationCreate, RecordID: "r", ExpectedAbsent: true, Body: "b", Inference: laputaevolution.InferenceObserved}),
		mustEffect(t, laputaevolution.KindPersonaRequest, &laputaevolution.PersonaRequestPayload{Kind: laputaevolution.PersonaRequestWorld, ProposedMarkdown: "m", Reason: "r"}),
		mustEffect(t, laputaevolution.KindCapabilityProposal, &laputaevolution.CapabilityProposalPayload{Name: "n", ProposedArtifact: "a"}),
	} {
		receipt, err := d.Apply(context.Background(), effect)
		if err != nil {
			t.Fatal(err)
		}
		if receipt.Status != laputaevolution.StatusRejected {
			t.Fatalf("%s should reject unbound, got %+v", effect.Kind, receipt)
		}
	}
}

func mustEffect(t *testing.T, kind laputaevolution.EffectKind, payload any) laputaevolution.Effect {
	t.Helper()
	e, err := laputaevolution.NewEffect("op:"+string(kind), kind, payload, testScope(), "dest_a")
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestCaptureReceiptCarriesLedgerSeq(t *testing.T) {
	writer := &fakeWriter{}
	ing := openIngest(t, writer)
	first := submitCapture(t, ing, "sess_1", "run_1:3", "", "exchange one")
	second := submitCapture(t, ing, "sess_1", "run_2:4", "", "exchange two")
	if first.Seq == 0 || second.Seq <= first.Seq {
		t.Fatalf("seqs not monotonic: %d %d", first.Seq, second.Seq)
	}
	// Repeated delivery returns the original acceptance and no new memory.
	dup := submitCapture(t, ing, "sess_1", "run_1:3", "", "exchange one")
	if dup.Seq != first.Seq || dup.IngestionID != first.IngestionID {
		t.Fatalf("redelivery changed receipt: %+v", dup)
	}
	if writer.count() > 2 {
		t.Fatalf("redelivery duplicated memory: %d", writer.count())
	}
}
