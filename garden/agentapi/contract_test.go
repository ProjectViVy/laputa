package agentapi

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestBootstrapWireRoundTrip(t *testing.T) {
	wire := `{"trace_id":"t1","context":"bounded","frozen_core":{"session_id":"s1","captured_at":"2026-09-27T00:00:00Z","sections":[{"section":0,"content":"id","source_revision":1,"source_hash":"h"},{"section":1,"content":"rel","source_revision":1,"source_hash":"h"},{"section":2,"content":"red","source_revision":1,"source_hash":"h"},{"section":3,"content":"user","source_revision":1,"source_hash":"h"},{"section":4,"content":"dream","source_revision":1,"source_hash":"h"},{"section":5,"content":"dark","source_revision":1,"source_hash":"h"}]},"evidence":[],"degraded":false,"warnings":[]}`
	var response BootstrapResponse
	if err := json.Unmarshal([]byte(wire), &response); err != nil {
		t.Fatal(err)
	}
	if response.FrozenCore.Sections[5].Section != 5 || response.FrozenCore.CapturedAt.IsZero() {
		t.Fatalf("frozen core lost: %+v", response.FrozenCore)
	}
	assertSameJSON(t, wire, response)
}

func TestPublicContextViewWireAndNoAutomaticAuthorityFields(t *testing.T) {
	view := ContextView{TraceID: "t", Scope: "", Mode: "fast", FrozenCore: FrozenCore{Sections: [6]FrozenSection{}}, Cards: []MemoryCard{}, Evidence: []EvidenceFragment{}, Context: "bounded", BudgetChars: 8000, Warnings: []string{}}
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
	assertSameJSON(t, `{"ingestion_id":"i","session_id":"s","event_id":"event","status":"accepted"}`, CaptureReceipt{IngestionID: "i", SessionID: "s", EventID: "event", Status: "accepted"})
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
