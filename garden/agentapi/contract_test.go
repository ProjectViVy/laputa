package agentapi

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dashimaki/laputa/evolution"
)

func TestBootstrapWireRoundTrip(t *testing.T) {
	wire := `{"trace_id":"t1","context":"bounded","frozen_core":{"schema_version":"laputa.frozen-core/v2","session_id":"s1","captured_at":"2026-09-27T00:00:00Z","mission_status":"unassigned","sections":[{"kind":"mission","content":"","source_revision":0,"source_hash":""},{"kind":"identity","content":"id","source_revision":1,"source_hash":"h"},{"kind":"relationship","content":"rel","source_revision":1,"source_hash":"h"},{"kind":"redline","content":"red","source_revision":1,"source_hash":"h"},{"kind":"user","content":"user","source_revision":1,"source_hash":"h"},{"kind":"dream","content":"","source_revision":1,"source_hash":"h"},{"kind":"dark","content":"","source_revision":1,"source_hash":"h"}]},"evidence":[],"degraded":false,"warnings":[]}`
	var response BootstrapResponse
	if err := json.Unmarshal([]byte(wire), &response); err != nil {
		t.Fatal(err)
	}
	if response.FrozenCore.Sections[5].Kind != evolution.FrozenKindDream || response.FrozenCore.CapturedAt.IsZero() {
		t.Fatalf("frozen core lost: %+v", response.FrozenCore)
	}
	assertSameJSON(t, wire, response)
}

func TestPublicContextViewWireAndNoAutomaticAuthorityFields(t *testing.T) {
	view := ContextView{TraceID: "t", Scope: "", Mode: "fast", FrozenCore: FrozenCore{SchemaVersion: evolution.FrozenCoreV2SchemaVersion, MissionStatus: evolution.MissionUnassigned, Sections: []evolution.FrozenSectionV2{{Kind: evolution.FrozenKindMission}, {Kind: evolution.FrozenKindIdentity}, {Kind: evolution.FrozenKindRelationship}, {Kind: evolution.FrozenKindRedline}, {Kind: evolution.FrozenKindUser}, {Kind: evolution.FrozenKindDream}, {Kind: evolution.FrozenKindDark}}}, Cards: []MemoryCard{}, Evidence: []EvidenceFragment{}, Context: "bounded", BudgetChars: 8000, Warnings: []string{}}
	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var shape map[string]any
	if err := json.Unmarshal(encoded, &shape); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"world", "actmem", "history", "profile_path", "raw_persona"} {
		if _, present := shape[forbidden]; present {
			t.Fatalf("automatic authority field %s", forbidden)
		}
	}
	for _, required := range []string{"trace_id", "scope", "mode", "frozen_core", "cards", "evidence", "context", "budget_chars", "degraded", "warnings"} {
		if _, present := shape[required]; !present {
			t.Errorf("missing %s", required)
		}
	}
}

func TestCaptureHostTerminalDTOAndWireReceipt(t *testing.T) {
	request := CaptureRequest{Binding: Binding{ProfileID: "default", AgentID: "vivy", Platform: "vivy", SessionID: "s", TurnID: "turn", EventID: "event"}, Phase: CaptureCompleted, Content: "bounded", ContentHash: "hash", Provenance: CaptureProvenance{Source: "journal", RunID: "run", EventSeq: 7}, OccurredAt: time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{`"turn_id"`, `"event_id"`, `"profile_id"`, `"phase":"completed"`, `"provenance"`} {
		if !strings.Contains(string(encoded), name) {
			t.Errorf("missing %s in %s", name, encoded)
		}
	}
	assertSameJSON(t, `{"ingestion_id":"i","session_id":"s","event_id":"event","status":"accepted","seq":3}`, CaptureReceipt{IngestionID: "i", SessionID: "s", EventID: "event", Status: "accepted", Seq: 3})
}

func TestTrustedWorkspaceBinding(t *testing.T) {
	bound := &BoundClient{binding: Binding{ProfileID: "default", AgentID: "vivy", Platform: "vivy", SessionID: "s", WorkspaceID: "ws-x"}}
	if _, err := bound.requestBinding(Binding{ProfileID: "default", AgentID: "vivy", Platform: "vivy", SessionID: "s", WorkspaceID: "ws-y"}); err == nil {
		t.Fatalf("request-supplied workspace must not override the host binding")
	}
	got, err := bound.requestBinding(Binding{})
	if err != nil || got.WorkspaceID != "ws-x" {
		t.Fatalf("host workspace binding lost: %+v %v", got, err)
	}
	scope, err := got.TrustedScope()
	if err != nil || scope.WorkspaceID != "ws-x" || scope.SubjectID != "default" || scope.Kind == "" {
		t.Fatalf("trusted scope derivation failed: %+v %v", scope, err)
	}
	personal, err := (Binding{ProfileID: "default", AgentID: "vivy", Platform: "vivy"}).TrustedScope()
	if err != nil || personal.Kind == "" || personal.WorkspaceID != "" {
		t.Fatalf("empty workspace must derive the implicit personal scope: %+v", personal)
	}
	wire, err := json.Marshal(Binding{ProfileID: "default", AgentID: "vivy", Platform: "vivy", SessionID: "s", WorkspaceID: "ws-x"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(wire), `"workspace_id":"ws-x"`) {
		t.Fatalf("workspace_id missing from binding wire shape: %s", wire)
	}
}

func TestPolicyErrorMatchesHTTPEnvelope(t *testing.T) {
	err := Authorize("default", Binding{ProfileID: "default"}, PrincipalAgent, OpPersonaReview)
	assertSameJSON(t, `{"code":"principal_forbidden","message":"principal is not permitted for this operation","error":"principal is not permitted for this operation","retryable":false,"request_id":"","details":{}}`, err)
}

func TestNormalizedErrorMatchesHTTPEnvelope(t *testing.T) {
	assertSameJSON(t, `{"code":"principal_forbidden","message":"denied","error":"denied","retryable":false,"request_id":"req","details":{}}`, Error{Code: "principal_forbidden", Message: "denied", LegacyError: "denied", Retryable: false, RequestID: "req", Details: map[string]any{}})
}

func assertSameJSON(t *testing.T, wire string, value any) {
	t.Helper()
	var want, got any
	if err := json.Unmarshal([]byte(wire), &want); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("wire mismatch\nwant: %s\ngot: %s", wire, encoded)
	}
}
