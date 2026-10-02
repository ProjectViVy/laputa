package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dashimaki/garden/agentapi"
	"github.com/dashimaki/garden/internal/personactx"
	"github.com/dashimaki/garden/internal/recall"
	"github.com/dashimaki/garden/internal/runtimecore"
	"github.com/dashimaki/laputa/actmem"
	"github.com/dashimaki/laputa/persona"
)

// This corpus compares the actual in-process domain service with the real
// HTTP mux. Transport-only metadata (request ID, trace ID) is not compared.
// A future MCP adapter must join these same cases before advertising them.
const (
	conformanceWorld  = "WORLD_ONLY_SENTINEL_9c6a"
	conformanceActmem = "ACTMEM_ONLY_SENTINEL_2f7b"
)

type conformanceFixture struct {
	srv        *Server
	api        *agentapi.Service
	binding    agentapi.Binding
	persona    *persona.Service
	frozen     *personactx.Store
	frozenPath string
}

func newConformanceFixture(t *testing.T) *conformanceFixture {
	t.Helper()
	dir := t.TempDir()
	person, err := persona.Open(filepath.Join(dir, "persona"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = person.Initialize(persona.Initialization{
		Identity: "identity fixture", Relationship: "relationship fixture", Redline: "redline fixture",
		User:  "## Preferences\nuser fixture\n## Observations\nprivate observation fixture",
		World: conformanceWorld,
	}, "user", persona.SourceInit, "conformance setup")
	if err != nil {
		t.Fatal(err)
	}
	activity := actmem.New(filepath.Join(dir, "activity"))
	pulse := conformanceActmem
	if _, err := activity.Put(actmem.ActmemPatch{Pulse: &pulse}); err != nil {
		t.Fatal(err)
	}
	frozenPath := filepath.Join(dir, "frozen.db")
	frozen, err := personactx.OpenStore(frozenPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = frozen.Close() })
	fast := &recall.FastService{Frozen: &personactx.SessionProvider{Store: frozen, Reader: person}}
	runtime := &runtimecore.Garden{ProfileID: "profile_1", Persona: person, Actmem: activity, FastRecall: fast}
	api := agentapi.NewService(runtime)
	srv := &Server{ProfileID: "profile_1", AgentAPI: api, Persona: person, Actmem: activity,
		Capabilities: CapabilityConfig{ReadToken: "conformance-read", UserToken: "conformance-user", AgentToken: "conformance-agent", OperatorToken: "conformance-operator"}}
	return &conformanceFixture{srv: srv, api: api, binding: agentapi.Binding{ProfileID: "profile_1", AgentID: "fixture-agent", Platform: "embedded", SessionID: "session-conformance"}, persona: person, frozen: frozen, frozenPath: frozenPath}
}

func (f *conformanceFixture) wire(t *testing.T, method, route, token string, body any, actor string) (int, []byte) {
	t.Helper()
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, route, &payload)
	req.RemoteAddr = "127.0.0.1:1234"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if actor != "" {
		req.Header.Set("X-Garden-Actor", actor)
	}
	rec := httptest.NewRecorder()
	f.srv.HTTPHandler().ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes()
}

func conformanceErrorCode(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		return ""
	}
	apiErr, ok := err.(*agentapi.Error)
	if !ok {
		t.Fatalf("unexpected embedded error type %T: %v", err, err)
	}
	return apiErr.Code
}

func conformanceWireCode(t *testing.T, payload []byte) string {
	t.Helper()
	var envelope struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Code
}

func assertToolOnlyAbsent(t *testing.T, value any) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	for _, sentinel := range []string{conformanceWorld, conformanceActmem, "private observation fixture"} {
		if bytes.Contains(payload, []byte(sentinel)) {
			t.Fatalf("automatic context leaked %q: %s", sentinel, payload)
		}
	}
}

func TestConformanceAutomaticContextEmbeddedAndWire(t *testing.T) {
	f := newConformanceFixture(t)
	ctx := context.Background()
	for _, tc := range []struct {
		name, route string
		bootstrap   bool
	}{
		{"C01 bootstrap six-slot frozen-only", "/v2/recall/bootstrap", true},
		{"C02 fast Mentle unavailable", "/v2/recall/fast", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var embeddedCore agentapi.FrozenCore
			var embeddedContext string
			var embeddedDegraded bool
			var embeddedWarnings []string
			var embeddedEvidence []agentapi.EvidenceFragment
			var body any
			if tc.bootstrap {
				got, err := f.api.Bootstrap(ctx, agentapi.PrincipalAgent, agentapi.BootstrapRequest{Binding: f.binding, Intent: "fixture", BudgetChars: 2048})
				if err != nil {
					t.Fatal(err)
				}
				embeddedCore, embeddedContext, embeddedDegraded, embeddedWarnings, embeddedEvidence = got.FrozenCore, got.Context, got.Degraded, got.Warnings, got.Evidence
				body = map[string]any{"session_id": f.binding.SessionID, "intent": "fixture", "budget_chars": 2048}
			} else {
				got, err := f.api.FastRecall(ctx, agentapi.PrincipalAgent, agentapi.FastRecallRequest{Binding: f.binding, Query: "fixture", BudgetChars: 2048})
				if err != nil {
					t.Fatal(err)
				}
				embeddedCore, embeddedContext, embeddedDegraded, embeddedWarnings, embeddedEvidence = got.FrozenCore, got.Context, got.Degraded, got.Warnings, got.Evidence
				if len(got.Cards) != 0 || got.Mode != "fast" {
					t.Fatalf("unavailable Mentle returned cards: %+v", got)
				}
				body = map[string]any{"session_id": f.binding.SessionID, "query": "fixture", "budget_chars": 2048}
			}
			status, raw := f.wire(t, http.MethodPost, tc.route, "conformance-agent", body, "operator")
			if status != http.StatusOK {
				t.Fatalf("HTTP status=%d body=%s", status, raw)
			}
			var wire struct {
				FrozenCore agentapi.FrozenCore         `json:"frozen_core"`
				Context    string                      `json:"context"`
				Degraded   bool                        `json:"degraded"`
				Warnings   []string                    `json:"warnings"`
				Evidence   []agentapi.EvidenceFragment `json:"evidence"`
			}
			if err := json.Unmarshal(raw, &wire); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(wire.FrozenCore, embeddedCore) || wire.Context != embeddedContext || wire.Degraded != embeddedDegraded || !reflect.DeepEqual(wire.Warnings, embeddedWarnings) || !reflect.DeepEqual(wire.Evidence, embeddedEvidence) {
				t.Fatalf("embedded/wire context differs: embedded=%+v/%q/%v/%v/%v wire=%+v", embeddedCore, embeddedContext, embeddedDegraded, embeddedWarnings, embeddedEvidence, wire)
			}
			if len(wire.FrozenCore.Sections) != 7 || wire.FrozenCore.SessionID != f.binding.SessionID || wire.FrozenCore.Sections[1].Content != "identity fixture" || !wire.Degraded || len(wire.Evidence) != 0 || len(wire.Warnings) == 0 || !strings.Contains(wire.Warnings[0], "mentle unavailable") || len([]rune(wire.Context)) > 2048 {
				t.Fatalf("invalid bounded frozen-only response: %+v", wire)
			}
			assertToolOnlyAbsent(t, raw)
			assertToolOnlyAbsent(t, embeddedCore)
		})
	}
}

func TestConformanceExplicitWorldAndACTMEM(t *testing.T) {
	f := newConformanceFixture(t)
	world, err := f.api.ReadPersona(context.Background(), agentapi.PrincipalRead, f.binding, "world")
	if err != nil || world.Content != conformanceWorld {
		t.Fatalf("embedded WORLD: %+v %v", world, err)
	}
	status, raw := f.wire(t, http.MethodGet, "/v2/persona/documents/world", "conformance-read", nil, "")
	if status != http.StatusOK {
		t.Fatalf("WORLD status=%d body=%s", status, raw)
	}
	var wireWorld agentapi.PersonaDocument
	if err := json.Unmarshal(raw, &wireWorld); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(wireWorld, world) {
		t.Fatalf("WORLD differs: embedded=%+v wire=%+v", world, wireWorld)
	}
	activity, err := f.api.ReadACTMEM(context.Background(), agentapi.PrincipalRead, f.binding)
	if err != nil {
		t.Fatal(err)
	}
	status, raw = f.wire(t, http.MethodGet, "/v2/actmem", "conformance-read", nil, "")
	if status != http.StatusOK {
		t.Fatalf("ACTMEM status=%d body=%s", status, raw)
	}
	var wireActivity struct {
		Revision uint64 `json:"revision"`
		Markdown string `json:"markdown"`
	}
	if err := json.Unmarshal(raw, &wireActivity); err != nil {
		t.Fatal(err)
	}
	if wireActivity.Revision != activity.Revision || wireActivity.Markdown != activity.Markdown {
		t.Fatalf("ACTMEM differs: embedded=%+v wire=%+v", activity, wireActivity)
	}
}

func TestConformanceFrozenCoreDoesNotDriftAfterPersonaWriteOrStoreReopen(t *testing.T) {
	f := newConformanceFixture(t)
	ctx := context.Background()
	first, err := f.api.Bootstrap(ctx, agentapi.PrincipalAgent, agentapi.BootstrapRequest{Binding: f.binding, Intent: "fixture", BudgetChars: 2048})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.persona.Write(persona.KindIdentity, "changed identity", 1, "user", persona.SourceUserDirect, "conformance drift check"); err != nil {
		t.Fatal(err)
	}
	second, err := f.api.Bootstrap(ctx, agentapi.PrincipalAgent, agentapi.BootstrapRequest{Binding: f.binding, Intent: "fixture", BudgetChars: 2048})
	if err != nil || !reflect.DeepEqual(first.FrozenCore, second.FrozenCore) || first.Context != second.Context {
		t.Fatalf("embedded frozen session drifted: first=%+v second=%+v err=%v", first, second, err)
	}
	status, raw := f.wire(t, http.MethodPost, "/v2/recall/bootstrap", "conformance-agent", map[string]any{"session_id": f.binding.SessionID, "intent": "fixture", "budget_chars": 2048}, "")
	if status != http.StatusOK {
		t.Fatalf("wire bootstrap status=%d body=%s", status, raw)
	}
	var wire agentapi.BootstrapResponse
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.FrozenCore, wire.FrozenCore) || first.Context != wire.Context {
		t.Fatalf("wire frozen session drifted: first=%+v wire=%+v", first, wire)
	}
	// Reopen the actual SQLite snapshot rather than relying on an in-memory fake.
	path := filepath.Join(f.frozenPath)
	if err := f.frozen.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := personactx.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	reloaded, err := reopened.Capture(ctx, f.binding.SessionID, f.persona)
	if err != nil || reloaded.Sections[1].Content != "identity fixture" || reloaded.Sections[1].SourceRevision != 1 {
		t.Fatalf("reopened frozen session drifted: %+v err=%v", reloaded, err)
	}
}

func TestConformanceInvalidRecallBudget(t *testing.T) {
	f := newConformanceFixture(t)
	_, err := f.api.FastRecall(context.Background(), agentapi.PrincipalAgent, agentapi.FastRecallRequest{Binding: f.binding, Query: "fixture", BudgetChars: 1})
	status, raw := f.wire(t, http.MethodPost, "/v2/recall/fast", "conformance-agent", map[string]any{"query": "fixture", "session_id": f.binding.SessionID, "budget_chars": 1}, "")
	if status != http.StatusBadRequest || conformanceErrorCode(t, err) != "invalid_request" || conformanceWireCode(t, raw) != "invalid_request" {
		t.Fatalf("invalid budget embedded=%v wire status=%d body=%s", err, status, raw)
	}
}

func TestConformanceTrustedPrincipalNotActorHeader(t *testing.T) {
	f := newConformanceFixture(t)
	for _, tc := range []struct {
		name               string
		principal          agentapi.Principal
		token, actor, want string
		status             int
	}{
		{"read denied", agentapi.PrincipalRead, "conformance-read", "user", "principal_forbidden", http.StatusForbidden},
		{"agent denied", agentapi.PrincipalAgent, "conformance-agent", "user", "principal_forbidden", http.StatusForbidden},
		{"operator denied", agentapi.PrincipalOperator, "conformance-operator", "user", "principal_forbidden", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			embedded := conformanceErrorCode(t, agentapi.Authorize(f.binding.ProfileID, f.binding, tc.principal, agentapi.OpPersonaReview))
			// An unknown review ID still exercises the authenticated approval gate.
			status, raw := f.wire(t, http.MethodPost, "/v2/persona/reviews/missing/approve", tc.token, map[string]any{}, tc.actor)
			if embedded != tc.want || status != tc.status {
				t.Fatalf("principal=%s embedded=%s wire status=%d body=%s", tc.principal, embedded, status, raw)
			}
			if tc.want != "" && conformanceWireCode(t, raw) != embedded {
				t.Fatalf("wire code=%s embedded=%s", conformanceWireCode(t, raw), embedded)
			}
		})
	}
	if err := agentapi.Authorize(f.binding.ProfileID, f.binding, agentapi.PrincipalUser, agentapi.OpPersonaReview); err != nil {
		t.Fatalf("embedded user review authorization: %v", err)
	}
	status, raw := f.wire(t, http.MethodPost, "/v2/persona/reviews", "conformance-agent", map[string]any{
		"kind": "identity", "base_revision": 1, "base_hash": persona.ContentHash("identity fixture"),
		"proposed_markdown": "reviewed identity", "reason": "conformance review",
	}, "user")
	if status != http.StatusCreated {
		t.Fatalf("create review status=%d body=%s", status, raw)
	}
	var review persona.ChangeRequest
	if err := json.Unmarshal(raw, &review); err != nil {
		t.Fatal(err)
	}
	if review.Actor != persona.ActorAgent {
		t.Fatalf("actor header escalated review identity: %+v", review)
	}
	status, raw = f.wire(t, http.MethodPost, "/v2/persona/reviews/"+review.ID+"/approve", "conformance-user", map[string]any{}, "agent")
	if status != http.StatusOK {
		t.Fatalf("trusted user could not approve: status=%d body=%s", status, raw)
	}
}
