// Package inofy adapts the diva/v1 strategy to the INOFY workflow runtime:
// it owns the fixed graph definition, the laputa.evolution.*@1 node
// descriptors and the NodeExecutor that dispatches stage calls to the
// bound strategy. It imports INOFY's public API only.
package inofy

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"

	"github.com/ProjectViVy/inofy"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/laputa/evolution/diva"
)

// ImplementationID is the catalog identity of this strategy build. It moves
// whenever prompts, schemas or stage code change.
const ImplementationID = "laputa-diva/" + diva.ImplementationRevision

var stageOrder = []string{
	diva.StageCollectID,
	diva.StagePrepareID,
	diva.StageReconcileID,
	diva.StageReflectID,
	diva.StageEffectsID,
	diva.StageFinishID,
}

var nodeTypes = []string{
	evolution.NodeCollect,
	evolution.NodePrepare,
	evolution.NodeReconcile,
	evolution.NodeReflect,
	evolution.NodeEffects,
	evolution.NodeFinish,
}

// Definition returns the fixed six-stage graph. Nodes run in the declared
// edge order; each stage's input is the previous stage's whole packet.
func Definition() (inofy.Definition, error) {
	nodes := make([]inofy.Node, 0, len(stageOrder))
	for i, id := range stageOrder {
		node := inofy.Node{ID: id, Kind: inofy.NodeKindCall, Type: nodeTypes[i]}
		source := "input"
		if i > 0 {
			source = stageOrder[i-1]
		}
		node.Inputs = map[string]inofy.Binding{
			"input": {Source: source},
		}
		nodes = append(nodes, node)
	}
	edges := make([]inofy.Edge, 0, len(stageOrder)-1)
	for i := 1; i < len(stageOrder); i++ {
		edges = append(edges, inofy.Edge{From: stageOrder[i-1], To: stageOrder[i]})
	}
	return inofy.Definition{
		SchemaVersion: inofy.SchemaVersionV1,
		Graph: inofy.Graph{
			Nodes: nodes,
			Edges: edges,
			Exits: []string{diva.StageFinishID},
			Outputs: map[string]inofy.Binding{
				"outcome": {Source: diva.StageFinishID},
			},
		},
	}, nil
}

// Descriptors registers the six fixed node types under this build's
// implementation identity.
func Descriptors() []inofy.NodeDescriptor {
	out := make([]inofy.NodeDescriptor, 0, len(nodeTypes))
	for i, typeID := range nodeTypes {
		out = append(out, inofy.NodeDescriptor{
			TypeID:           typeID,
			ImplementationID: ImplementationID,
			Display:          inofy.DisplayMeta{Title: "laputa evolution " + stageOrder[i]},
		})
	}
	return out
}

// StrategyDigest covers definition bytes, the prompt/schema bundle and the
// implementation identity — a change to any of them moves the digest.
func StrategyDigest() (string, error) {
	def, err := Definition()
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(def)
	if err != nil {
		return "", err
	}
	return digest(raw, diva.PromptBundle()), nil
}

func digest(definition, bundle []byte) string {
	h := sha256.New()
	h.Write(definition)
	h.Write([]byte{0})
	h.Write(bundle)
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
