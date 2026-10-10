package agentapi

import (
	"encoding/json"
	"testing"

	"github.com/ProjectViVy/laputa/laputa/evolution"
)

// TestFrozenCoreWireIsV2: the bootstrap/context wire carries the v2 envelope —
// schema marker, mission first of exactly seven named slots, mission_status.
func TestFrozenCoreWireIsV2(t *testing.T) {
	core := evolution.FrozenCoreV2{
		SchemaVersion: evolution.FrozenCoreV2SchemaVersion,
		SessionID:     "s1",
		MissionStatus: evolution.MissionAssigned,
		Sections: []evolution.FrozenSectionV2{
			{Kind: evolution.FrozenKindMission, Content: "mission", SourceRevision: 3, SourceHash: "h"},
			{Kind: evolution.FrozenKindIdentity, Content: "identity", SourceRevision: 1},
			{Kind: evolution.FrozenKindRelationship, SourceRevision: 1},
			{Kind: evolution.FrozenKindRedline, SourceRevision: 1},
			{Kind: evolution.FrozenKindUser, SourceRevision: 1},
			{Kind: evolution.FrozenKindDream},
			{Kind: evolution.FrozenKindDark},
		},
	}
	response := BootstrapResponse{TraceID: "t", FrozenCore: core}
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var probe struct {
		FrozenCore struct {
			SchemaVersion string `json:"schema_version"`
			MissionStatus string `json:"mission_status"`
			Sections      []struct {
				Kind string `json:"kind"`
			} `json:"sections"`
		} `json:"frozen_core"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		t.Fatal(err)
	}
	if probe.FrozenCore.SchemaVersion != "laputa.frozen-core/v2" || probe.FrozenCore.MissionStatus != "assigned" {
		t.Fatalf("wire envelope=%+v", probe.FrozenCore)
	}
	if len(probe.FrozenCore.Sections) != 7 || probe.FrozenCore.Sections[0].Kind != "mission" {
		t.Fatalf("wire sections=%+v", probe.FrozenCore.Sections)
	}
}

// TestV1AdmissionCannotMasquerade: a numeric six-slot v1 payload is rejected
// by the v2 decoder; it is never silently relabelled.
func TestV1AdmissionCannotMasquerade(t *testing.T) {
	v1 := []byte(`{"session_id":"s","captured_at":"2026-08-14T00:00:00Z","sections":[{"section":0,"content":"i","source_revision":1,"source_hash":"h"},{"section":1},{"section":2},{"section":3},{"section":4},{"section":5}]}`)
	if _, err := evolution.DecodeFrozenCoreV2(v1); err == nil {
		t.Fatal("v1 payload decoded as v2")
	}
}

// TestMissionPinRejectsStaleRevision: an autonomous run bound to a mission
// revision is blocked by mission_revision_changed after the human edits it.
func TestMissionPinRejectsStaleRevision(t *testing.T) {
	binding := evolution.RunBinding{
		SubjectID: "sub", DestinationID: "dest", PolicyRevision: "p",
		StrategyDigest: "d", MissionRevision: 2,
	}
	if err := binding.CheckMissionRevision(2); err != nil {
		t.Fatalf("matching revision must pass: %v", err)
	}
	if err := binding.CheckMissionRevision(3); evolution.CodeOf(err) != evolution.ErrMissionRevisionChanged {
		t.Fatalf("err=%v want mission_revision_changed", err)
	}
	// A run pinned while unassigned may not proceed after a human assigns.
	unpinned := binding
	unpinned.MissionRevision = 0
	if err := unpinned.CheckMissionRevision(4); evolution.CodeOf(err) != evolution.ErrMissionRevisionChanged {
		t.Fatalf("unassigned->assigned err=%v", err)
	}
}
