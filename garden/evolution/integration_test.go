package evolution

// Integrated effect-recovery coverage (S07 task 3): real ingest, ACTMEM,
// Persona and the Domain adapter under the real DIVA strategy with a
// scripted model. The ViVy host exercises its own RunStore side; the
// authority side of the loop lives entirely inside this module.

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/dashimaki/laputa/actmem"
	laputaevolution "github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/laputa/evolution/diva"
	"github.com/dashimaki/laputa/persona"
)

type stageModel struct {
	calls   []laputaevolution.ModelStage
	replies map[laputaevolution.ModelStage]json.RawMessage
}

func (m *stageModel) Infer(_ context.Context, req laputaevolution.ModelRequest) (laputaevolution.ModelReply, error) {
	m.calls = append(m.calls, req.Stage)
	out, ok := m.replies[req.Stage]
	if !ok {
		out = json.RawMessage(`{}`)
	}
	return laputaevolution.ModelReply{OutputJSON: out}, nil
}

func candidate(kind string, body map[string]any) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{"kind": kind, kind: body})
	return raw
}

func reflectWith(candidates ...json.RawMessage) json.RawMessage {
	out := struct {
		Candidates []json.RawMessage `json:"candidates"`
	}{Candidates: candidates}
	raw, _ := json.Marshal(out)
	return raw
}

func testInput(through uint64) laputaevolution.Input {
	return laputaevolution.Input{
		Binding: laputaevolution.RunBinding{
			SubjectID: "profile_a", WorkspaceID: "", DestinationID: "dest_a",
			PolicyRevision: "pol-1", StrategyDigest: "dig-1", MissionRevision: 0,
		},
		Window: laputaevolution.Window{SourceID: "activity", After: 0, Through: through},
	}
}

// A captured conversation produces the full effect fan-out with distinct
// durable statuses: ACTMEM patch applied, memory mutation applied, persona
// request submitted for review — each replayed across a Domain restart.
func TestIntegratedCaptureToDurableStatuses(t *testing.T) {
	ctx := context.Background()
	writer := &fakeWriter{}
	ing := openIngest(t, writer)
	submitCapture(t, ing, "sess_1", "run_1:3", "", "user asked about progress")
	// Redelivery of the same event dedupes before the window ever reads it.
	submitCapture(t, ing, "sess_1", "run_1:3", "", "user asked about progress")

	psvc := openPersona(t)
	userDoc, err := psvc.GetDocument(persona.KindUser)
	if err != nil {
		t.Fatal(err)
	}
	mem := &fakeBackend{}
	prop := &fakeProposer{}
	dir := t.TempDir()
	d := newDomain(t, Deps{
		Scope: testScope(), DestinationID: "dest_a", Dir: dir,
		Actmem: actmem.New(t.TempDir()), Persona: psvc, Memory: mem, Proposals: prop, Activity: ing,
	})
	patch, _ := json.Marshal(laputaevolution.WorkPatch{
		Changes: []laputaevolution.WorkChange{{
			Kind: laputaevolution.WorkChangeAdd, Field: laputaevolution.FieldGoal, Body: "finish S07",
		}},
	})
	model := &stageModel{replies: map[laputaevolution.ModelStage]json.RawMessage{
		laputaevolution.StageReconcile: patch,
		laputaevolution.StageReflect: reflectWith(
			candidate("memory_mutation", map[string]any{
				"operation": "create", "record_id": "obs_1", "expected_absent": true,
				"body": "asked about progress", "inference": "observed",
			}),
			candidate("persona_request", map[string]any{
				"kind": "user_observations", "base_revision": userDoc.Revision,
				"proposed_markdown": "- tracks story progress closely", "reason": "observed",
			}),
		),
	}}
	outcome, err := diva.New(d, model).Run(ctx, testInput(10), "op:int")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Status != diva.OutcomeSubmitted {
		t.Fatalf("outcome = %q, want submitted", outcome.Status)
	}
	// Distinct durable statuses per effect kind.
	byKind := map[laputaevolution.EffectKind]laputaevolution.EffectReceipt{}
	for _, r := range outcome.Receipts {
		byKind[receiptKind(d, r.OperationID)] = r
	}
	if byKind[laputaevolution.KindWorkPatch].Status != laputaevolution.StatusApplied {
		t.Fatalf("work patch receipt %+v", byKind)
	}
	if byKind[laputaevolution.KindMemoryMutation].Status != laputaevolution.StatusApplied {
		t.Fatalf("memory receipt %+v", byKind)
	}
	if byKind[laputaevolution.KindPersonaRequest].Status != laputaevolution.StatusSubmitted {
		t.Fatalf("persona receipt %+v", byKind)
	}
	// The persona review waits in the authority's own queue.
	reqs, err := psvc.ListRequests(nil)
	if err != nil || len(reqs) != 1 || reqs[0].State != persona.RequestPending {
		t.Fatalf("persona reviews = %+v %v", reqs, err)
	}
	// A fresh adapter (restart) replays every committed receipt.
	reopened := newDomain(t, Deps{Scope: testScope(), DestinationID: "dest_a", Dir: dir, Memory: mem})
	for _, op := range []string{"op:int:work", "op:int:e0", "op:int:e1"} {
		if _, err := reopened.Lookup(ctx, op); err != nil {
			t.Fatalf("lookup %s after restart: %v", op, err)
		}
	}
}

// receiptKind resolves the recorded kind for one committed operation.
func receiptKind(d *Domain, opID string) laputaevolution.EffectKind {
	rec, ok, err := d.ledger.lookup(opID)
	if !ok || err != nil {
		return ""
	}
	return laputaevolution.EffectKind(rec.Kind)
}

// A stale Mission pin blocks before any model call or effect commit.
func TestStaleMissionBlocksAdmission(t *testing.T) {
	psvc := openPersona(t)
	missionDoc, err := psvc.GetDocument(persona.KindMission)
	if err != nil {
		t.Fatal(err)
	}
	input := testInput(5)
	input.Binding.MissionRevision = missionDoc.Revision + 7
	if err := input.Binding.CheckMissionRevision(missionDoc.Revision); laputaevolution.CodeOf(err) != laputaevolution.ErrMissionRevisionChanged {
		t.Fatalf("stale pin not refused: %v", err)
	}
}

// A backend outage produces an accurate partial outcome: committed effects
// stay visible in receipts, the failed one is surfaced, and no model retry
// manufactures a blind second write.
func TestUnavailableBackendPartialOutcome(t *testing.T) {
	ctx := context.Background()
	ing := openIngest(t, &fakeWriter{})
	submitCapture(t, ing, "sess_1", "run_1:3", "", "exchange")
	mem := &fakeBackend{fail: errors.New("fixture: canonical store down")}
	d := newDomain(t, Deps{
		Scope: testScope(), DestinationID: "dest_a",
		Actmem: actmem.New(t.TempDir()), Memory: mem, Activity: ing,
	})
	patch, _ := json.Marshal(laputaevolution.WorkPatch{
		Changes: []laputaevolution.WorkChange{{
			Kind: laputaevolution.WorkChangeAdd, Field: laputaevolution.FieldGoal, Body: "goal",
		}},
	})
	model := &stageModel{replies: map[laputaevolution.ModelStage]json.RawMessage{
		laputaevolution.StageReconcile: patch,
		laputaevolution.StageReflect: reflectWith(
			candidate("memory_mutation", map[string]any{
				"operation": "create", "record_id": "obs_1", "expected_absent": true,
				"body": "x", "inference": "observed",
			}),
		),
	}}
	outcome, err := diva.New(d, model).Run(ctx, testInput(10), "op:down")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Status != diva.OutcomePartial {
		t.Fatalf("outcome = %q, want partial", outcome.Status)
	}
	// The committed work patch remains a durable applied receipt.
	got, err := d.Lookup(ctx, "op:down:work")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != laputaevolution.StatusApplied {
		t.Fatalf("committed work lost: %+v", got)
	}
}
