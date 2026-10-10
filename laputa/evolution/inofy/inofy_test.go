package inofy

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ProjectViVy/inofy"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/laputa/evolution/diva"
)

type fakeDomain struct{}

func (fakeDomain) Collect(_ context.Context, w evolution.Window) (evolution.EvidenceBatch, error) {
	return evolution.EvidenceBatch{Window: w}, nil
}
func (fakeDomain) Apply(context.Context, evolution.Effect) (evolution.EffectReceipt, error) {
	return evolution.EffectReceipt{Status: evolution.StatusApplied}, nil
}
func (fakeDomain) Lookup(_ context.Context, _ string) (evolution.EffectReceipt, error) {
	return evolution.EffectReceipt{}, &evolution.ContractError{Code: evolution.ErrEffectNotFound}
}

type fakeModel struct{}

func (fakeModel) Infer(_ context.Context, req evolution.ModelRequest) (evolution.ModelReply, error) {
	return evolution.ModelReply{OutputJSON: json.RawMessage(`{}`)}, nil
}

// The definition compiles on the pinned INOFY catalog.
func TestDefinitionCompiles(t *testing.T) {
	def, err := Definition()
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := inofy.NewCatalog(Descriptors())
	if err != nil {
		t.Fatal(err)
	}
	prog, diags, err := inofy.Compile(context.Background(), def, catalog, inofy.CompileOptions{})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(diags) != 0 {
		t.Fatalf("compile diagnostics: %+v", diags)
	}
	if prog == nil {
		t.Fatal("no program")
	}
}

// The executor dispatches stage calls and rejects foreign type IDs.
func TestExecutorDispatch(t *testing.T) {
	exec := NewExecutor(fakeDomain{}, fakeModel{})
	in, _ := json.Marshal(evolution.Input{
		Binding: evolution.RunBinding{SubjectID: "p", DestinationID: "actmem", PolicyRevision: "pol", MissionRevision: 1, StrategyDigest: "d"},
		Window:  evolution.Window{SourceID: "activity", After: 0, Through: 1},
	})
	callInput, _ := json.Marshal(map[string]json.RawMessage{"input": in})
	reply, err := exec.Execute(context.Background(), inofy.NodeCall{
		TypeID: evolution.NodeCollect, ImplementationID: ImplementationID,
		Input: callInput, OperationKey: "op-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Input json.RawMessage `json:"input"`
	}
	if err := json.Unmarshal(reply.Output, &out); err != nil || len(out.Input) == 0 {
		t.Fatalf("collect output = %s err=%v", reply.Output, err)
	}
	if _, err := exec.Execute(context.Background(), inofy.NodeCall{TypeID: "vivy.child-task@1", Input: callInput}); err == nil {
		t.Fatal("foreign node type dispatched")
	}
	if _, err := exec.Execute(context.Background(), inofy.NodeCall{
		TypeID: evolution.NodeCollect, ImplementationID: "other-impl", Input: callInput,
	}); err == nil {
		t.Fatal("foreign implementation dispatched")
	}
}

// A prompt/schema/implementation change moves the strategy digest.
func TestDigestMovesWithImplementation(t *testing.T) {
	digest1, err := StrategyDigest()
	if err != nil {
		t.Fatal(err)
	}
	def, _ := Definition()
	raw, _ := json.Marshal(def)
	bundle := append(append([]byte{}, diva.PromptBundle()...), 'x')
	if digest(raw, bundle) == digest1 {
		t.Fatal("prompt bundle change did not move the digest")
	}
	if digest(append(raw, 'x'), diva.PromptBundle()) == digest1 {
		t.Fatal("definition change did not move the digest")
	}
}
