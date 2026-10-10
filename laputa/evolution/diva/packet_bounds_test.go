package diva

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/ProjectViVy/laputa/laputa/evolution"
)

func TestReflectionPacketKeepsEffectsWithinPacketBound(t *testing.T) {
	batch := testBatch()
	body := strings.Repeat("长", 1500)
	batch.Entries[0].Body = body
	candidate, err := json.Marshal(map[string]any{
		"kind": "memory_mutation",
		"memory_mutation": map[string]any{
			"operation": "create", "record_id": "bounded-memory", "expected_absent": true,
			"body": body, "inference": "observed", "sources": batch.Entries[0].Sources,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	d := &fakeDomain{}
	m := &fakeModel{replies: map[evolution.ModelStage]json.RawMessage{evolution.StageReflect: reflectReply(candidate)}}
	s := New(d, m)
	raw, err := marshal(chainDoc{Input: testInput(), Batch: &batch})
	if err != nil {
		t.Fatal(err)
	}
	out, err := s.reflect(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) > 8<<10 {
		t.Fatalf("reflection packet repeats evidence beside its effect: %d bytes", len(out))
	}
	var doc chainDoc
	if err := decodeStrict(out, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Candidates) != 1 || doc.Candidates[0].MemoryMutation.Body != body || !reflect.DeepEqual(doc.Candidates[0].MemoryMutation.Sources, batch.Entries[0].Sources) {
		t.Fatal("bounded packet changed the memory effect or its source references")
	}
	effects, err := s.effects(context.Background(), out, "bounded-effects")
	if err != nil {
		t.Fatal(err)
	}
	finished, err := s.finish(context.Background(), effects)
	if err != nil {
		t.Fatal(err)
	}
	var result Outcome
	if err := json.Unmarshal(finished, &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != OutcomeApplied || result.Window != testInput().Window || len(result.Receipts) != 1 || len(d.applied) != 1 {
		t.Fatalf("bounded effect lost its terminal receipt or window: %+v", result)
	}
}
