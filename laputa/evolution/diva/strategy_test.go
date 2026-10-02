package diva

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dashimaki/laputa/evolution"
)

// --- fakes (host-side Domain/Model bound on construction) ---

type fakeDomain struct {
	batch    evolution.EvidenceBatch
	applied  []evolution.Effect
	receipts map[string]evolution.EffectReceipt
	applyErr map[string]error
	applyOut map[string]evolution.EffectReceipt
}

func (f *fakeDomain) Collect(_ context.Context, _ evolution.Window) (evolution.EvidenceBatch, error) {
	return f.batch, nil
}

func (f *fakeDomain) Apply(_ context.Context, e evolution.Effect) (evolution.EffectReceipt, error) {
	f.applied = append(f.applied, e)
	if err, ok := f.applyErr[e.OperationID]; ok {
		return evolution.EffectReceipt{}, err
	}
	if r, ok := f.applyOut[e.OperationID]; ok {
		return r, nil
	}
	return evolution.EffectReceipt{OperationID: e.OperationID, PayloadDigest: e.PayloadDigest, Status: evolution.StatusApplied, Revision: 1}, nil
}

func (f *fakeDomain) Lookup(_ context.Context, opID string) (evolution.EffectReceipt, error) {
	if r, ok := f.receipts[opID]; ok {
		return r, nil
	}
	return evolution.EffectReceipt{}, &evolution.ContractError{Code: evolution.ErrEffectNotFound, Message: "no receipt"}
}

type fakeModel struct {
	calls   []evolution.ModelStage
	replies map[evolution.ModelStage]json.RawMessage
}

func (f *fakeModel) Infer(_ context.Context, req evolution.ModelRequest) (evolution.ModelReply, error) {
	f.calls = append(f.calls, req.Stage)
	out, ok := f.replies[req.Stage]
	if !ok {
		out = json.RawMessage(`{}`)
	}
	return evolution.ModelReply{OutputJSON: out}, nil
}

// --- helpers ---

func testInput() evolution.Input {
	return evolution.Input{
		Binding: evolution.RunBinding{
			SubjectID:       "profile_1",
			DestinationID:   "actmem",
			PolicyRevision:  "pol-1",
			MissionRevision: 1,
			StrategyDigest:  "dig",
		},
		Window: evolution.Window{SourceID: "activity", After: 0, Through: 5},
	}
}

func testBatch() evolution.EvidenceBatch {
	return evolution.EvidenceBatch{
		Window:           evolution.Window{SourceID: "activity", After: 0, Through: 5},
		ActivityRevision: 5,
		Entries: []evolution.Entry{{
			ID: "e1", Section: "pulse", Scope: evolution.Scope{SubjectID: "profile_1", Kind: evolution.ScopePersonal},
			SessionID: "s", EventID: "e", OccurredAt: "2026-10-02T00:00:00Z", Body: "did things",
			Sources: []evolution.SourceRef{{SourceID: "activity", RecordID: "r1", Revision: 1, Scope: evolution.Scope{SubjectID: "profile_1", Kind: evolution.ScopePersonal}}},
		}},
	}
}

func workPatchReply() json.RawMessage {
	p := evolution.WorkPatch{
		BaseRevision: 1,
		Changes: []evolution.WorkChange{
			{Kind: evolution.WorkChangeAdd, Field: evolution.FieldNext, Body: "ship it"},
		},
	}
	raw, _ := json.Marshal(p)
	return raw
}

func reflectReply(candidates ...json.RawMessage) json.RawMessage {
	out := struct {
		Candidates     []json.RawMessage `json:"candidates"`
		NoChangeReason string            `json:"no_change_reason,omitempty"`
	}{Candidates: candidates}
	raw, _ := json.Marshal(out)
	return raw
}

func memoryCandidate() json.RawMessage {
	c := map[string]any{
		"kind": "memory_mutation",
		"memory_mutation": map[string]any{
			"operation": "create", "record_id": "obs-1", "expected_absent": true,
			"body": "observed preference", "inference": "observed",
		},
	}
	raw, _ := json.Marshal(c)
	return raw
}

func personaCandidate() json.RawMessage {
	c := map[string]any{
		"kind": "persona_request",
		"persona_request": map[string]any{
			"kind": "user_observations", "proposed_markdown": "- likes tea",
			"reason": "observed",
		},
	}
	raw, _ := json.Marshal(c)
	return raw
}

func capabilityCandidate() json.RawMessage {
	c := map[string]any{
		"kind": "capability_proposal",
		"capability_proposal": map[string]any{
			"name": "tidy", "proposed_artifact": "skill body",
		},
	}
	raw, _ := json.Marshal(c)
	return raw
}

// --- Task 1: fixed chain, zero model calls on empty input ---

func TestStrategyEmptyBatchNoModelCalls(t *testing.T) {
	d := &fakeDomain{batch: evolution.EvidenceBatch{Window: evolution.Window{SourceID: "activity", After: 0, Through: 5}}}
	m := &fakeModel{}
	s := New(d, m)
	out, err := s.Run(context.Background(), testInput(), "op-1")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != OutcomeNoChange {
		t.Fatalf("status = %q, want no_change", out.Status)
	}
	if len(m.calls) != 0 {
		t.Fatalf("model calls = %v, want none", m.calls)
	}
	if len(d.applied) != 0 {
		t.Fatalf("applied = %d, want none", len(d.applied))
	}
}

// --- Task 2: typed outputs reach Domain in stable order ---

func TestStrategyRoutesTypedEffectsInOrder(t *testing.T) {
	d := &fakeDomain{batch: testBatch()}
	m := &fakeModel{replies: map[evolution.ModelStage]json.RawMessage{
		evolution.StageReconcile: workPatchReply(),
		evolution.StageReflect:   reflectReply(memoryCandidate(), personaCandidate(), capabilityCandidate()),
	}}
	s := New(d, m)
	out, err := s.Run(context.Background(), testInput(), "op-9")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.applied) != 4 {
		t.Fatalf("applied = %d, want 4 (work patch + 3 candidates)", len(d.applied))
	}
	ids := []string{d.applied[0].OperationID, d.applied[1].OperationID, d.applied[2].OperationID, d.applied[3].OperationID}
	want := []string{"op-9:work", "op-9:e0", "op-9:e1", "op-9:e2"}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("op[%d] = %q want %q", i, ids[i], want[i])
		}
	}
	if d.applied[0].Kind != evolution.KindWorkPatch || d.applied[1].Kind != evolution.KindMemoryMutation ||
		d.applied[2].Kind != evolution.KindPersonaRequest || d.applied[3].Kind != evolution.KindCapabilityProposal {
		t.Fatalf("kinds = %v", ids)
	}
	for _, e := range d.applied {
		if e.PayloadDigest == "" {
			t.Fatal("effect missing payload digest")
		}
	}
	if out.Status != OutcomeApplied && out.Status != OutcomeSubmitted {
		t.Fatalf("status = %q", out.Status)
	}
}

func TestStrategyRejectsForbiddenAndInvalidOutput(t *testing.T) {
	for _, raw := range []json.RawMessage{
		json.RawMessage(`{"candidates":[{"kind":"mission_write","mission_write":{"body":"x"}}]}`),
		json.RawMessage(`{"candidates":[{"kind":"dream_write","dream_write":{"body":"x"}}]}`),
		json.RawMessage(`{"candidates":[{"kind":"bogus","bogus":{}}]}`),
		json.RawMessage(`{"candidates":[{}],"no_change_reason":"both"}`),
		json.RawMessage(`{`),
	} {
		d := &fakeDomain{batch: testBatch()}
		m := &fakeModel{replies: map[evolution.ModelStage]json.RawMessage{
			evolution.StageReconcile: json.RawMessage(`{"base_revision":1,"changes":[]}`),
			evolution.StageReflect:   raw,
		}}
		s := New(d, m)
		if _, err := s.Run(context.Background(), testInput(), "op-bad"); err == nil {
			t.Fatalf("invalid reflect output %s accepted", raw)
		}
		for _, e := range d.applied {
			if e.Kind != evolution.KindWorkPatch {
				t.Fatalf("forbidden write applied: %s", e.Kind)
			}
		}
	}
}

func TestStrategyNoChangeIsValid(t *testing.T) {
	d := &fakeDomain{batch: testBatch()}
	m := &fakeModel{replies: map[evolution.ModelStage]json.RawMessage{
		evolution.StageReconcile: json.RawMessage(`{"base_revision":1,"changes":[]}`),
		evolution.StageReflect:   reflectReply(),
	}}
	m.replies[evolution.StageReflect] = json.RawMessage(`{"no_change_reason":"nothing useful"}`)
	s := New(d, m)
	out, err := s.Run(context.Background(), testInput(), "op-nc")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != OutcomeNoChange {
		t.Fatalf("status = %q", out.Status)
	}
	if len(d.applied) != 0 {
		t.Fatalf("applied = %d", len(d.applied))
	}
}

// --- Task 3: partial and unknown outcomes ---

func TestStrategyFailureAfterFirstEffectPreservesReceipt(t *testing.T) {
	d := &fakeDomain{
		batch:    testBatch(),
		applyErr: map[string]error{"op-p:e1": &evolution.ContractError{Code: evolution.ErrAuthorityDenied}},
	}
	m := &fakeModel{replies: map[evolution.ModelStage]json.RawMessage{
		evolution.StageReconcile: json.RawMessage(`{"base_revision":1,"changes":[]}`),
		evolution.StageReflect:   reflectReply(memoryCandidate(), personaCandidate(), capabilityCandidate()),
	}}
	s := New(d, m)
	out, err := s.Run(context.Background(), testInput(), "op-p")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.applied) != 2 {
		t.Fatalf("applied = %d, want 2 (e0 ok, e1 failed, e2 never attempted)", len(d.applied))
	}
	if out.Status != OutcomePartial {
		t.Fatalf("status = %q, want partial", out.Status)
	}
	if len(out.Receipts) != 1 {
		t.Fatalf("receipts = %+v, want the single committed receipt", out.Receipts)
	}
	if out.Receipts[0].OperationID != "op-p:e0" {
		t.Fatalf("receipt = %+v", out.Receipts[0])
	}
}

func TestStrategyUnknownOutcomeStopsAndMarksRecovery(t *testing.T) {
	d := &fakeDomain{
		batch:    testBatch(),
		applyOut: map[string]evolution.EffectReceipt{"op-u:e0": {OperationID: "op-u:e0", Status: evolution.StatusUnknown}},
	}
	m := &fakeModel{replies: map[evolution.ModelStage]json.RawMessage{
		evolution.StageReconcile: json.RawMessage(`{"base_revision":1,"changes":[]}`),
		evolution.StageReflect:   reflectReply(memoryCandidate(), personaCandidate()),
	}}
	s := New(d, m)
	out, err := s.Run(context.Background(), testInput(), "op-u")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != OutcomeRecoveryRequired {
		t.Fatalf("status = %q, want recovery_required", out.Status)
	}
	if len(d.applied) != 1 {
		t.Fatalf("applied = %d, later effects must not run", len(d.applied))
	}
}

func TestStrategyReplayResolvesCommittedEffect(t *testing.T) {
	d := &fakeDomain{batch: testBatch()}
	// Compute the digest the first run would have produced for e0.
	cand := mustCandidate(t, memoryCandidate())
	eff, err := evolution.NewEffect("op-r:e0", evolution.KindMemoryMutation, cand.Payload(),
		evolution.Scope{SubjectID: "profile_1", Kind: evolution.ScopePersonal}, "actmem")
	if err != nil {
		t.Fatal(err)
	}
	d.receipts = map[string]evolution.EffectReceipt{
		"op-r:e0": {OperationID: "op-r:e0", PayloadDigest: eff.PayloadDigest, Status: evolution.StatusApplied, TargetRef: "mem_x", Revision: 2},
	}
	m := &fakeModel{replies: map[evolution.ModelStage]json.RawMessage{
		evolution.StageReconcile: json.RawMessage(`{"base_revision":1,"changes":[]}`),
		evolution.StageReflect:   reflectReply(memoryCandidate(), personaCandidate()),
	}}
	s := New(d, m)
	out, err := s.Run(context.Background(), testInput(), "op-r")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.applied) != 1 || d.applied[0].OperationID != "op-r:e1" {
		t.Fatalf("applied = %+v, want only e1 (e0 resolved from receipt)", d.applied)
	}
	if len(out.Receipts) != 2 || out.Receipts[0].TargetRef != "mem_x" {
		t.Fatalf("receipts = %+v", out.Receipts)
	}
}

func TestStrategyReplayConflictFails(t *testing.T) {
	d := &fakeDomain{
		batch: testBatch(),
		receipts: map[string]evolution.EffectReceipt{
			"op-c:e0": {OperationID: "op-c:e0", PayloadDigest: "different", Status: evolution.StatusApplied},
		},
	}
	m := &fakeModel{replies: map[evolution.ModelStage]json.RawMessage{
		evolution.StageReconcile: json.RawMessage(`{"base_revision":1,"changes":[]}`),
		evolution.StageReflect:   reflectReply(memoryCandidate()),
	}}
	s := New(d, m)
	if _, err := s.Run(context.Background(), testInput(), "op-c"); err == nil {
		t.Fatal("changed payload under the same operation id must fail")
	}
	if len(d.applied) != 0 {
		t.Fatalf("applied = %d", len(d.applied))
	}
}

func mustCandidate(t *testing.T, raw json.RawMessage) evolution.Candidate {
	t.Helper()
	c, err := evolution.DecodeCandidate(raw)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
