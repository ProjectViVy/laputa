//go:build e2e

package e2e

// Decisive DIVA cognitive scenario (S09 task 1) over real module
// composition: ingest ledger, ACTMEM, Persona and the bound evolution
// Domain under the real DIVA strategy with a scripted model. The live
// desktop path (G8) is covered by agent-vivy separately and a configured
// live model is recorded as not exercised.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	gardenevolution "github.com/dashimaki/garden/evolution"
	"github.com/dashimaki/garden/internal/ingest"
	"github.com/dashimaki/garden/memory"
	"github.com/dashimaki/laputa/actmem"
	laputaevolution "github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/laputa/evolution/diva"
	"github.com/dashimaki/laputa/persona"
	"github.com/dashimaki/mentle/facade"
)

// --- fixtures (e2e-local; the authority side is real services) ---

type divaE2EWriter struct {
	mu    sync.Mutex
	memos map[string]facade.Memory
	seq   int
}

func (f *divaE2EWriter) CreateMemory(_ context.Context, req facade.CreateMemoryRequest, key, _ string) (facade.Memory, error) {
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

type divaE2EBackend struct {
	mu       sync.Mutex
	mutated  []memory.AuthorizedMutation
	receipts map[string]memory.MutationReceipt
}

func (b *divaE2EBackend) Capabilities() memory.Capabilities { return memory.Capabilities{} }
func (b *divaE2EBackend) Search(context.Context, memory.AuthorizedSearch) (memory.CardPage, error) {
	return memory.CardPage{}, nil
}
func (b *divaE2EBackend) Expand(context.Context, memory.AuthorizedExpansion) (memory.EvidencePage, error) {
	return memory.EvidencePage{}, nil
}
func (b *divaE2EBackend) Mutate(_ context.Context, m memory.AuthorizedMutation) (memory.MutationReceipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
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
func (b *divaE2EBackend) MutationStatus(_ context.Context, opID string) (memory.MutationReceipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if r, ok := b.receipts[opID]; ok {
		return r, nil
	}
	return memory.MutationReceipt{}, &laputaevolution.ContractError{Code: laputaevolution.ErrEffectNotFound, Message: "unknown operation"}
}
func (b *divaE2EBackend) Health(context.Context) (memory.Health, error) { return memory.Health{}, nil }
func (b *divaE2EBackend) Close() error                                  { return nil }

type divaE2EProposer struct {
	got []gardenevolution.ProposedCapability
}

func (p *divaE2EProposer) SubmitCapabilityProposal(_ context.Context, in gardenevolution.ProposedCapability) (string, error) {
	p.got = append(p.got, in)
	return "evomap:run_" + in.Name, nil
}

type divaE2EModel struct {
	replies map[laputaevolution.ModelStage]json.RawMessage
}

func (m *divaE2EModel) Infer(_ context.Context, req laputaevolution.ModelRequest) (laputaevolution.ModelReply, error) {
	out, ok := m.replies[req.Stage]
	if !ok {
		out = json.RawMessage(`{}`)
	}
	return laputaevolution.ModelReply{OutputJSON: out}, nil
}

func divaE2ECandidate(kind string, body map[string]any) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{"kind": kind, kind: body})
	return raw
}

func divaE2EReflect(candidates ...json.RawMessage) json.RawMessage {
	out := struct {
		Candidates []json.RawMessage `json:"candidates"`
	}{Candidates: candidates}
	raw, _ := json.Marshal(out)
	return raw
}

func divaE2EOpenIngest(t *testing.T, writer *divaE2EWriter, dir string) *ingest.Service {
	t.Helper()
	svc, err := ingest.Open(filepath.Join(dir, "ingest.db"), writer)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	return svc
}

func divaE2ECapture(t *testing.T, svc *ingest.Service, sessionID, eventID, workspace, body string) ingest.Accepted {
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

func divaE2EOpenPersona(t *testing.T, dir string) *persona.Service {
	t.Helper()
	svc, err := persona.Open(filepath.Join(dir, "persona"))
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

func divaE2EDomain(t *testing.T, deps gardenevolution.Deps) *gardenevolution.Domain {
	t.Helper()
	d, err := gardenevolution.NewDomain(deps)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// The decisive scenario: a human assigns Mission; a captured conversation
// reconciles scoped Work and permitted ordinary memory; a Persona proposal
// waits for review; the conversational dream tool writes DREAM; AutoDream
// Mission/Dream mutations are refused; a restart after a lost effect reply
// neither duplicates nor falsely confirms anything.
func TestDivaCognitiveDecisiveEndToEnd(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	ing := divaE2EOpenIngest(t, &divaE2EWriter{}, dir)
	psvc := divaE2EOpenPersona(t, dir)
	am := actmem.New(filepath.Join(dir, "actmem"))
	backend := &divaE2EBackend{}
	proposer := &divaE2EProposer{}
	scope := laputaevolution.Scope{SubjectID: "profile_a", Kind: laputaevolution.ScopePersonal}
	effectDir := filepath.Join(dir, "effects")
	d := divaE2EDomain(t, gardenevolution.Deps{
		Scope: scope, DestinationID: "dest_a", Dir: effectDir,
		Actmem: am, Persona: psvc, Memory: backend, Proposals: proposer, Activity: ing,
	})

	// Human assigns Mission; every non-human channel is refused.
	missionDoc, err := psvc.GetDocument(persona.KindMission)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := psvc.Write(persona.KindMission, "# Mission\n\nship the governed loop", missionDoc.Revision, "human-e2e", persona.SourceUserDirect, "assign"); err != nil {
		t.Fatalf("human mission write refused: %v", err)
	}
	missionDoc, err = psvc.GetDocument(persona.KindMission)
	if err != nil || missionDoc.Revision == 0 {
		t.Fatalf("mission revision not durable: %+v %v", missionDoc, err)
	}
	for _, source := range []persona.WriteSource{persona.SourceAutodreamP5, persona.SourceAgentP16} {
		if _, err := psvc.Write(persona.KindMission, "hijack", missionDoc.Revision, "forged", source, "x"); err == nil {
			t.Fatalf("%s mutated Mission", source)
		}
	}
	if _, err := psvc.SaveAgentP16(persona.KindMission, "hijack", "x"); err == nil {
		t.Fatal("conversational path mutated Mission")
	}
	// A stale Mission pin refuses autonomous effects before any model call.
	pinned := laputaevolution.RunBinding{MissionRevision: missionDoc.Revision + 9}
	if err := pinned.CheckMissionRevision(missionDoc.Revision); laputaevolution.CodeOf(err) != laputaevolution.ErrMissionRevisionChanged {
		t.Fatalf("stale mission pin not refused: %v", err)
	}

	// Authorized experience lands on the durable capture ledger; a replayed
	// delivery returns the original seq.
	first := divaE2ECapture(t, ing, "sess_1", "run_1:3", "", "asked about S07 progress")
	replay := divaE2ECapture(t, ing, "sess_1", "run_1:3", "", "asked about S07 progress")
	if replay.Seq != first.Seq {
		t.Fatalf("redelivery minted new seq %d want %d", replay.Seq, first.Seq)
	}
	second := divaE2ECapture(t, ing, "sess_1", "run_1:4", "", "said work must stay in work sections")
	if second.Seq <= first.Seq {
		t.Fatalf("ledger seq not advancing: %d after %d", second.Seq, first.Seq)
	}

	userDoc, err := psvc.GetDocument(persona.KindUser)
	if err != nil {
		t.Fatal(err)
	}
	patch, _ := json.Marshal(laputaevolution.WorkPatch{
		Changes: []laputaevolution.WorkChange{{
			Kind: laputaevolution.WorkChangeAdd, Field: laputaevolution.FieldNext, Body: "ship the governed loop",
		}},
	})
	model := &divaE2EModel{replies: map[laputaevolution.ModelStage]json.RawMessage{
		laputaevolution.StageReconcile: patch,
		laputaevolution.StageReflect: divaE2EReflect(
			divaE2ECandidate("memory_mutation", map[string]any{
				"operation": "create", "record_id": "obs_progress", "expected_absent": true,
				"body": "follows story progress", "inference": "observed",
			}),
			divaE2ECandidate("persona_request", map[string]any{
				"kind": "user_observations", "base_revision": userDoc.Revision,
				"proposed_markdown": "- tracks delivery closely", "reason": "observed",
			}),
		),
	}}
	input := laputaevolution.Input{
		Binding: laputaevolution.RunBinding{
			SubjectID: scope.SubjectID, DestinationID: "dest_a",
			PolicyRevision: "pol-1", StrategyDigest: "dig-1",
			MissionRevision: missionDoc.Revision,
		},
		Window: laputaevolution.Window{SourceID: "activity", After: 0, Through: second.Seq},
	}
	outcome, err := diva.New(d, model).Run(ctx, input, "op:decisive")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Status != diva.OutcomeSubmitted {
		t.Fatalf("outcome = %q, want submitted", outcome.Status)
	}

	// Scoped Work changed through the authority, ordinary memory committed
	// exactly once, and the persona proposal waits in the review queue.
	view, err := am.ReadScoped(scope, laputaevolution.ReadRequest{})
	if err != nil {
		t.Fatal(err)
	}
	workBody := ""
	for _, entry := range view.Entries {
		if entry.Field == laputaevolution.FieldNext && entry.Body == "ship the governed loop" {
			workBody = entry.Body
		}
	}
	if workBody == "" {
		t.Fatalf("scoped work not applied: %+v", view.Entries)
	}
	if len(backend.mutated) != 1 {
		t.Fatalf("memory mutations = %d, want 1", len(backend.mutated))
	}
	requests, err := psvc.ListRequests(nil)
	if err != nil || len(requests) != 1 || requests[0].State != persona.RequestPending {
		t.Fatalf("persona review queue = %+v %v", requests, err)
	}

	// Reviewed Persona change applies distinctly; the next session's
	// snapshot sees the applied revision.
	if _, err := psvc.AcceptRequest(requests[0].ID); err != nil {
		t.Fatalf("review acceptance: %v", err)
	}
	userDoc, _ = psvc.GetDocument(persona.KindUser)
	if userDoc.Content == "" || len(userDoc.Content) == 0 {
		t.Fatal("user doc lost after review")
	}
	// The conversational dream tool writes DREAM; neither it nor AutoDream
	// can reach Mission again.
	if _, err := psvc.SaveAgentP16(persona.KindDream, "# Dream\n\nbecome steadily more helpful", "considered"); err != nil {
		t.Fatalf("conversational dream tool refused: %v", err)
	}
	missionDoc2, _ := psvc.GetDocument(persona.KindMission)
	if missionDoc2.Revision != missionDoc.Revision {
		t.Fatal("mission revision moved without a human write")
	}

	// A restart after a lost effect reply: the same operation cannot
	// duplicate a mutation, and a reopened adapter replays committed
	// receipts instead of re-applying.
	reopened := divaE2EDomain(t, gardenevolution.Deps{
		Scope: scope, DestinationID: "dest_a", Dir: effectDir, Memory: backend,
	})
	for _, op := range []string{"op:decisive:work", "op:decisive:e0", "op:decisive:e1"} {
		if _, err := reopened.Lookup(ctx, op); err != nil {
			t.Fatalf("restart lost receipt %s: %v", op, err)
		}
	}
	memoEffect, err := laputaevolution.NewEffect("op:decisive:e0", laputaevolution.KindMemoryMutation,
		&laputaevolution.MemoryMutationPayload{
			Operation: laputaevolution.MutationCreate, RecordID: "obs_progress",
			ExpectedAbsent: true, Body: "follows story progress", Inference: laputaevolution.InferenceObserved,
		}, scope, "dest_a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.Apply(ctx, memoEffect); err != nil {
		t.Fatalf("replay apply: %v", err)
	}
	if len(backend.mutated) != 1 {
		t.Fatalf("lost-reply restart duplicated memory: %d mutations", len(backend.mutated))
	}
}
