package inofy

import (
	"context"
	"encoding/json"

	"github.com/ProjectViVy/inofy"
	"github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/laputa/evolution/diva"
)

// stageByTypeID maps the fixed node types to stage identities.
var stageByTypeID = map[string]string{
	evolution.NodeCollect:   diva.StageCollectID,
	evolution.NodePrepare:   diva.StagePrepareID,
	evolution.NodeReconcile: diva.StageReconcileID,
	evolution.NodeReflect:   diva.StageReflectID,
	evolution.NodeEffects:   diva.StageEffectsID,
	evolution.NodeFinish:    diva.StageFinishID,
}

// executor is the host's trusted node boundary for the bound strategy.
// It holds no authority of its own: the Domain and Model were bound at
// construction.
type executor struct {
	strategy *diva.Strategy
}

// NewExecutor returns the NodeExecutor for the bound Domain and Model.
func NewExecutor(domain evolution.Domain, model evolution.Model) inofy.NodeExecutor {
	return &executor{strategy: diva.New(domain, model)}
}

// Execute dispatches one trusted node call to its stage. The call's input
// envelope carries a single "input" member bound by the definition.
func (e *executor) Execute(ctx context.Context, call inofy.NodeCall) (inofy.NodeReply, error) {
	stage, ok := stageByTypeID[call.TypeID]
	if !ok {
		return inofy.NodeReply{}, &inofy.Error{Code: inofy.ErrInvalidDefinition, Message: "unknown node type " + call.TypeID}
	}
	if call.ImplementationID != "" && call.ImplementationID != ImplementationID {
		return inofy.NodeReply{}, &inofy.Error{Code: inofy.ErrInvalidDefinition, Message: "implementation " + call.ImplementationID + " does not match " + ImplementationID}
	}
	var envelope struct {
		Input json.RawMessage `json:"input"`
	}
	if err := json.Unmarshal(call.Input, &envelope); err != nil {
		return inofy.NodeReply{}, err
	}
	out, err := e.strategy.RunStage(ctx, stage, envelope.Input, call.OperationKey)
	if err != nil {
		return inofy.NodeReply{}, err
	}
	return inofy.NodeReply{Output: out}, nil
}
