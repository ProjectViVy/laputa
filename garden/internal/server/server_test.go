package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/dashimaki/garden/agentapi"
	"github.com/dashimaki/garden/internal/activity"
	"github.com/dashimaki/garden/internal/evolution"
	"github.com/dashimaki/garden/internal/recall"
	"github.com/dashimaki/garden/internal/runtimecore"
	"github.com/dashimaki/laputa/actmem"
	"github.com/dashimaki/mentle/facade"
)

func testServer() *Server { return &Server{Addr: ":0", ProfileID: "profile_1"} }

func evolutionTestServer(t *testing.T) *Server {
	t.Helper()
	path := filepath.Join(t.TempDir(), "evolution.db")
	store, err := evolution.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	events, err := evolution.OpenEventStore(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = events.Close() })
	return &Server{Evolution: &evolution.Service{Store: store, Events: events, Hub: evolution.DefaultHubPolicy()}, Addr: ":0"}
}

func localRequest(method, path string, body *bytes.Buffer) *http.Request {
	var reader io.Reader
	if body != nil {
		reader = body
	}
	req := httptest.NewRequest(method, path, reader)
	req.RemoteAddr = "127.0.0.1:1234"
	return req
}

func TestHandleHealth(t *testing.T) {
	rec := httptest.NewRecorder()
	testServer().handleHealth(rec, localRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestHTTPContractAddsRequestIDAndErrorEnvelope(t *testing.T) {
	req := localRequest(http.MethodPost, "/v2/recall/bootstrap", bytes.NewBufferString("not-json"))
	rec := httptest.NewRecorder()
	testServer().HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || rec.Header().Get("X-Garden-Request-ID") == "" {
		t.Fatalf("status=%d headers=%v", rec.Code, rec.Header())
	}
	var body struct {
		Code      string         `json:"code"`
		RequestID string         `json:"request_id"`
		Details   map[string]any `json:"details"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "malformed_json" || body.RequestID == "" || body.Details == nil {
		t.Fatalf("body=%+v", body)
	}
}

type fakeCardSearcher struct{}

func (fakeCardSearcher) SearchCards(context.Context, facade.CardQuery) (facade.CardPage, error) {
	return facade.CardPage{Cards: []facade.MemoryCard{{ID: "mem_1", Kind: "fact", Summary: "test card", CandidateScore: .8}}}, nil
}

func (fakeCardSearcher) ReadEvidence(context.Context, facade.EvidenceQuery) ([]facade.EvidenceFragment, error) {
	return []facade.EvidenceFragment{{CardID: "mem_1", Excerpt: "evidence text", Validity: "active"}}, nil
}

func TestFastRecallAndBootstrapEndpoints(t *testing.T) {
	srv := testServer()
	srv.FastRecall = &recall.FastService{Searcher: fakeCardSearcher{}}
	srv.AgentAPI = agentapi.NewService(&runtimecore.Garden{ProfileID: "profile_1", FastRecall: srv.FastRecall})
	for _, route := range []string{"/v2/recall/fast", "/v2/recall/bootstrap"} {
		body := `{"query":"test","budget_chars":4000}`
		if route == "/v2/recall/bootstrap" {
			body = `{"session_id":"session_1","intent":"test","budget_chars":4000}`
		}
		req := localRequest(http.MethodPost, route, bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		srv.HTTPHandler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("route=%s status=%d body=%s", route, rec.Code, rec.Body.String())
		}
		if rec.Body.Len() == 0 {
			t.Fatalf("route=%s empty response", route)
		}
	}
}

func TestBootstrapPreservesLegacyBodyWithoutClientIdentity(t *testing.T) {
	srv := testServer()
	srv.FastRecall = &recall.FastService{Searcher: fakeCardSearcher{}}
	srv.AgentAPI = agentapi.NewService(&runtimecore.Garden{ProfileID: "profile_1", FastRecall: srv.FastRecall})
	srv.Capabilities = CapabilityConfig{AgentToken: "agent-secret"}
	req := localRequest(http.MethodPost, "/v2/recall/bootstrap", bytes.NewBufferString(`{"session_id":"session_1","intent":"test","budget_chars":4000}`))
	req.Header.Set("Authorization", "Bearer agent-secret")
	req.Header.Set("X-Garden-Actor", "operator") // audit metadata, not a trusted binding
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"trace_id", "context", "frozen_core", "evidence", "degraded", "warnings"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("missing %s in %v", key, payload)
		}
	}
	if len(payload) != 6 {
		t.Fatalf("unexpected bootstrap keys: %v", payload)
	}
}

func TestBootstrapUsesBoundAgentAPIWithoutLegacyFastRecall(t *testing.T) {
	srv := testServer()
	srv.AgentAPI = agentapi.NewService(&runtimecore.Garden{ProfileID: "profile_1", FastRecall: &recall.FastService{Searcher: fakeCardSearcher{}}})
	req := localRequest(http.MethodPost, "/v2/recall/bootstrap", bytes.NewBufferString(`{"session_id":"session_1","intent":"test","budget_chars":4000}`))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestBootstrapRejectsClientIdentityClaims(t *testing.T) {
	srv := testServer()
	srv.FastRecall = &recall.FastService{Searcher: fakeCardSearcher{}}
	for _, field := range []string{
		`"binding":{"profile_id":"profile_1","agent_id":"agent_1","platform":"local","session_id":"session_1"}`,
		`"profile_id":"profile_1"`, `"agent_id":"agent_1"`, `"actor":"operator"`,
	} {
		t.Run(field, func(t *testing.T) {
			req := localRequest(http.MethodPost, "/v2/recall/bootstrap", bytes.NewBufferString(`{"session_id":"session_1",`+field+`}`))
			rec := httptest.NewRecorder()
			srv.HTTPHandler().ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestBootstrapRequiresSessionID(t *testing.T) {
	srv := testServer()
	srv.FastRecall = &recall.FastService{Searcher: fakeCardSearcher{}}
	req := localRequest(http.MethodPost, "/v2/recall/bootstrap", bytes.NewBufferString(`{"intent":"test"}`))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestBootstrapUnavailableWithoutRecall(t *testing.T) {
	srv := testServer()
	req := localRequest(http.MethodPost, "/v2/recall/bootstrap", bytes.NewBufferString(`{"session_id":"session_1"}`))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	var response struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusServiceUnavailable || response.Code != "unavailable" {
		t.Fatalf("status=%d response=%+v", rec.Code, response)
	}
}

func TestBootstrapAuthenticationIgnoresActorHeader(t *testing.T) {
	srv := testServer()
	srv.FastRecall = &recall.FastService{Searcher: fakeCardSearcher{}}
	srv.Capabilities = CapabilityConfig{AgentToken: "agent-secret"}
	req := localRequest(http.MethodPost, "/v2/recall/bootstrap", bytes.NewBufferString(`{"session_id":"session_1"}`))
	req.Header.Set("Authorization", "Bearer wrong-token")
	req.Header.Set("X-Garden-Actor", "operator")
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRetiredRoutesAreAbsent(t *testing.T) {
	srv := testServer()
	for _, path := range []string{"/v2/governance/projection", "/v2/governance/mutations", "/v2/cognitive/world", "/v2/persona/files/identity"} {
		req := localRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.HTTPHandler().ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("path=%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestActivityEndpoints(t *testing.T) {
	store, err := activity.OpenStore(filepath.Join(t.TempDir(), "garden.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	srv := testServer()
	srv.Activity = store
	req := localRequest(http.MethodPost, "/v2/activity/events", bytes.NewBufferString(`{"session_id":"sess_1","event_id":"ev_1","type":"session_start"}`))
	req.Header.Set("Authorization", "Bearer user-secret")
	req.Header.Set("Content-Type", "application/json")
	srv.Capabilities = CapabilityConfig{UserToken: "user-secret"}
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("post status=%d body=%s", rec.Code, rec.Body.String())
	}
	req = localRequest(http.MethodGet, "/v2/activity/sessions/sess_1", nil)
	req.SetPathValue("session_id", "sess_1")
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestActmemReadWriteAndExplicitOnlyRoutes(t *testing.T) {
	dir := t.TempDir()
	store := actmem.New(dir)
	srv := testServer()
	srv.Actmem = store
	srv.Capabilities = CapabilityConfig{UserToken: "user-secret", AgentToken: "agent-secret"}
	req := localRequest(http.MethodGet, "/v2/actmem", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("read status=%d body=%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(store.Root()); err == nil {
		t.Fatal("read created ACTMEM directory")
	}
	req = localRequest(http.MethodPut, "/v2/actmem", bytes.NewBufferString(`{"base_revision":0,"markdown":"---\nschema: laputa.actmem/v2\nrevision: 1\nupdated: 2030-01-01T00:00:00Z\nentries:\n  e_0000000000000000000000000000000a:\n    section: pulse\n    field: \"\"\n    scope:\n      subject_id: profile_1\n      kind: personal\n      workspace_id: \"\"\n    session_id: s-1\n    event_id: \"\"\n    occurred_at: 2030-01-01T00:00:00Z\n    sources: []\n---\n## Pulse\n<!-- actmem-entry:e_0000000000000000000000000000000a -->\nhello\n<!-- /actmem-entry:e_0000000000000000000000000000000a -->\n## Recap\n## Work\n"}`))
	req.Header.Set("Authorization", "Bearer user-secret")
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("write status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result["changed"] != true || result["revision"] != float64(1) {
		t.Fatalf("result=%v", result)
	}
}
