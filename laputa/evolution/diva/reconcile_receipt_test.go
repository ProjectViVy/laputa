package diva

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dashimaki/laputa/evolution"
)

func TestUnresolvedWorkReceiptStopsLaterReflection(t *testing.T) {
	for _, status := range []evolution.EffectStatus{evolution.StatusUnknown, evolution.StatusRejected} {
		t.Run(string(status), func(t *testing.T) {
			batch := testBatch()
			d := &fakeDomain{applyOut: map[string]evolution.EffectReceipt{"work:work": {OperationID: "work:work", Status: status}}}
			m := &fakeModel{replies: map[evolution.ModelStage]json.RawMessage{evolution.StageReconcile: workPatchReply(), evolution.StageReflect: reflectReply(memoryCandidate())}}
			s := New(d, m)
			raw, err := marshal(chainDoc{Input: testInput(), Batch: &batch})
			if err != nil {
				t.Fatal(err)
			}
			raw, err = s.reconcile(context.Background(), raw, "work")
			if err != nil {
				t.Fatal(err)
			}
			raw, err = s.reflect(context.Background(), raw)
			if err != nil {
				t.Fatal(err)
			}
			raw, err = s.effects(context.Background(), raw, "effects")
			if err != nil {
				t.Fatal(err)
			}
			raw, err = s.finish(context.Background(), raw)
			if err != nil {
				t.Fatal(err)
			}
			var out Outcome
			if err := json.Unmarshal(raw, &out); err != nil {
				t.Fatal(err)
			}
			want := OutcomePartial
			if status == evolution.StatusUnknown {
				want = OutcomeRecoveryRequired
			}
			if out.Status != want || len(d.applied) != 1 || len(m.calls) != 1 || len(out.Receipts) != 1 || out.Receipts[0].Status != status {
				t.Fatalf("unresolved work continued: outcome=%+v applied=%d model_calls=%v", out, len(d.applied), m.calls)
			}
		})
	}
}
