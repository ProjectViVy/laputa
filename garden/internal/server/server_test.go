package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/dashimaki/garden/internal/activity"
	"github.com/dashimaki/garden/internal/arbiter"
	"github.com/dashimaki/garden/internal/authority"
	"github.com/dashimaki/garden/internal/crud"
	"github.com/dashimaki/garden/internal/evolution"
	"github.com/dashimaki/garden/internal/rag"
	"github.com/dashimaki/garden/internal/recall"
	"github.com/dashimaki/garden/internal/router"
	"github.com/dashimaki/laputa/governance"
	"github.com/dashimaki/mentle/facade"
)

type mockBackend struct {
	name string
}

type mockResolver struct{}

func (mockResolver) Resolve(_ context.Context, request rag.ResolveRequest) (rag.ContextPackage, error) {
	return rag.ContextPackage{TraceID: "run_test", Context: request.Intent, Evidence: []rag.Evidence{}, Warnings: []string{}}, nil
}

func (m *mockBackend) Write(ctx context.Context, key, value string, meta map[string]any) (string, error) {
	return key, nil
}

func (m *mockBackend) Read(ctx context.Context, key string) (map[string]any, error) {
	return map[string]any{"key": key, "value": map[string]any{"agent": "matsumoto"}, "backend": m.name}, nil
}

func (m *mockBackend) List(ctx context.Context, prefix string, limit int) ([]map[string]any, error) {
	return []map[string]any{{"key": prefix + "01-identity", "backend": m.name}}, nil
}

func (m *mockBackend) Forget(ctx context.Context, key string) (bool, error) {
	return true, nil
}

func testServer() *Server {
	h := &crud.Handler{
		Router: &router.Router{
			Governance: &mockBackend{name: "governance"},
			Mentle:     &mockBackend{name: "mentle"},
		},
	}
	return &Server{Handler: h, Addr: ":0"}
}

func TestHandleWrite(t *testing.T) {
	srv := testServer()
	body := `{"key":"section:01-identity","value":"{\"agent\":\"matsumoto\"}"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/memories", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleWrite(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["id"] != "section:01-identity" {
		t.Errorf("id = %q, want section:01-identity", resp["id"])
	}
}

func TestHandleRead(t *testing.T) {
	srv := testServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/memories/section:01-identity", nil)
	req.SetPathValue("key", "section:01-identity")
	rec := httptest.NewRecorder()

	srv.handleRead(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleListDefaultsToSectionPrefix(t *testing.T) {
	srv := testServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/memories", nil)
	rec := httptest.NewRecorder()

	srv.handleList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var resp struct {
		Records []map[string]any `json:"records"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Records) != 1 {
		t.Fatalf("records = %v, want 1", resp.Records)
	}
}

func TestHandleForget(t *testing.T) {
	srv := testServer()
	req := httptest.NewRequest(http.MethodDelete, "/v1/memories/section:01-identity", nil)
	req.SetPathValue("key", "section:01-identity")
	rec := httptest.NewRecorder()

	srv.handleForget(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleHealth(t *testing.T) {
	srv := testServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHTTPContractAddsRequestIDAndErrorEnvelope(t *testing.T) {
	srv := testServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/memories", bytes.NewBufferString("not-json"))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || rec.Header().Get("X-Garden-Request-ID") == "" {
		t.Fatalf("status=%d headers=%v", rec.Code, rec.Header())
	}
	var body struct {
		Error     string         `json:"error"`
		Code      string         `json:"code"`
		RequestID string         `json:"request_id"`
		Retryable bool           `json:"retryable"`
		Details   map[string]any `json:"details"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "invalid_request" || body.RequestID == "" || body.Retryable || body.Details == nil {
		t.Fatalf("body=%+v", body)
	}
}

func TestHealthAdvertisesFrozenContract(t *testing.T) {
	srv := testServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["api_contract"] != "garden-hermes/1" {
		t.Fatalf("body=%v", body)
	}
}

func TestHandleWriteBadRequest(t *testing.T) {
	srv := testServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/memories", bytes.NewBufferString("not-json"))
	rec := httptest.NewRecorder()

	srv.handleWrite(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleResolveContext(t *testing.T) {
	srv := testServer()
	srv.Resolver = mockResolver{}
	req := httptest.NewRequest(http.MethodPost, "/v1/context/resolve", bytes.NewBufferString(`{"intent":"garden"}`))
	rec := httptest.NewRecorder()
	srv.handleResolveContext(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response rag.ContextPackage
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.TraceID != "run_test" {
		t.Fatalf("response=%+v", response)
	}
}

func TestHandleResolveContextRequiresIntent(t *testing.T) {
	srv := testServer()
	srv.Resolver = mockResolver{}
	req := httptest.NewRequest(http.MethodPost, "/v1/context/resolve", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()
	srv.handleResolveContext(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

type fakeGovReader struct{}

func (fakeGovReader) GetSection(_ context.Context, _ governance.SectionName) (map[string]any, error) {
	return map[string]any{"_meta": map[string]any{"version": "test"}}, nil
}

type fakeCardSearcher struct{}

func (fakeCardSearcher) SearchCards(_ context.Context, _ facade.CardQuery) (facade.CardPage, error) {
	return facade.CardPage{Cards: []facade.MemoryCard{{ID: "mem_1", Kind: "fact", Summary: "test card", CandidateScore: 0.8}}}, nil
}

func (fakeCardSearcher) ReadEvidence(_ context.Context, _ facade.EvidenceQuery) ([]facade.EvidenceFragment, error) {
	return []facade.EvidenceFragment{{CardID: "mem_1", Excerpt: "evidence text", Validity: "active"}}, nil
}

func TestFastRecallEndpoint(t *testing.T) {
	srv := testServer()
	srv.FastRecall = &recall.FastService{Gov: fakeGovReader{}, Searcher: fakeCardSearcher{}}
	req := httptest.NewRequest(http.MethodPost, "/v2/recall/fast", bytes.NewBufferString(`{"query":"governance","budget_chars":6000}`))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var view recall.ContextView
	if err := json.NewDecoder(rec.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if view.Mode != "fast" || len(view.Cards) != 1 || view.Context == "" {
		t.Fatalf("view=%+v", view)
	}
}

func TestFastRecallEndpointUnavailable(t *testing.T) {
	srv := testServer()
	srv.FastRecall = nil
	req := httptest.NewRequest(http.MethodPost, "/v2/recall/fast", bytes.NewBufferString(`{"query":"test"}`))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestBootstrapRoutesThroughFastRecall(t *testing.T) {
	srv := testServer()
	srv.FastRecall = &recall.FastService{Gov: fakeGovReader{}, Searcher: fakeCardSearcher{}}
	req := httptest.NewRequest(http.MethodPost, "/v1/context/bootstrap", bytes.NewBufferString(`{"intent":"bootstrap","budget_chars":4000}`))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	traceID, _ := resp["trace_id"].(string)
	if traceID == "" || resp["context"] == "" {
		t.Fatalf("resp=%v", resp)
	}
}

func TestActivityEventsEndpoint(t *testing.T) {
	srv := testServer()
	store, err := activity.OpenStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	srv.Activity = store
	body := `{"session_id":"sess_1","event_id":"ev_1","type":"code_edit"}`
	req := httptest.NewRequest(http.MethodPost, "/v2/activity/events", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "accepted" || resp["event_id"] != "ev_1" {
		t.Fatalf("resp=%v", resp)
	}
}

func TestActivityEventsIdempotent(t *testing.T) {
	srv := testServer()
	store, err := activity.OpenStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	srv.Activity = store
	body := `{"session_id":"sess_1","event_id":"ev_1","type":"session_end"}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v2/activity/events", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		srv.HTTPHandler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("attempt %d: status=%d", i, rec.Code)
		}
	}
	events, _ := store.SessionEvents(context.Background(), "sess_1", 10)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestSessionActivityEndpoint(t *testing.T) {
	srv := testServer()
	store, err := activity.OpenStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	srv.Activity = store
	_ = store.Append(context.Background(), activity.Event{ID: "ev_1", SessionID: "sess_1", Type: "session_start"})
	_ = store.Append(context.Background(), activity.Event{ID: "ev_2", SessionID: "sess_1", Type: "session_end"})
	req := httptest.NewRequest(http.MethodGet, "/v2/activity/sessions/sess_1", nil)
	req.SetPathValue("session_id", "sess_1")
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		SessionID string           `json:"session_id"`
		Events    []activity.Event `json:"events"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.SessionID != "sess_1" || len(resp.Events) != 2 {
		t.Fatalf("resp=%+v", resp)
	}
}

func governedTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	store, err := governance.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	engine := governance.NewEngine(store)
	_ = engine.Initialize(context.Background())
	audit, err := governance.NewFileAuditLog(dir)
	if err != nil {
		t.Fatal(err)
	}
	governed := governance.NewGovernedService(engine, audit)
	govBackend := router.NewGovernanceBackend(engine)
	govBackend.Governed = governed
	h := &crud.Handler{
		Gov: engine,
		Router: &router.Router{
			Governance: govBackend,
			Mentle:     &mockBackend{name: "mentle"},
		},
	}
	return &Server{
		Handler:        h,
		Governed:       governed,
		GovernedWriter: &authority.GovernedWriter{Gov: governed},
		Addr:           ":0",
	}
}

func TestGovernanceProjectionEndpoint(t *testing.T) {
	srv := governedTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/v2/governance/projection", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var proj map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&proj); err != nil {
		t.Fatal(err)
	}
	if proj["projection_version"] == nil {
		t.Fatalf("proj=%v", proj)
	}
}

func TestGovernanceMutationAuthorized(t *testing.T) {
	srv := governedTestServer(t)
	body := `{"section":"03-commitment","action":"write","reason":"user edit","data":{"red_lines":["no delete"]}}`
	req := httptest.NewRequest(http.MethodPost, "/v2/governance/mutations", bytes.NewBufferString(body))
	req.Header.Set("X-Garden-Actor", "user_request")
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGovernanceMutationUnauthorized(t *testing.T) {
	srv := governedTestServer(t)
	body := `{"section":"03-commitment","action":"write","reason":"agent attempt","data":{"red_lines":[]}}`
	req := httptest.NewRequest(http.MethodPost, "/v2/governance/mutations", bytes.NewBufferString(body))
	req.Header.Set("X-Garden-Actor", "agent")
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGovernanceAuditEndpoint(t *testing.T) {
	srv := governedTestServer(t)
	body := `{"section":"01-identity","action":"write","reason":"identity change","data":{"role":"companion"}}`
	req := httptest.NewRequest(http.MethodPost, "/v2/governance/mutations", bytes.NewBufferString(body))
	req.Header.Set("X-Garden-Actor", "user_request")
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("mutation status=%d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v2/governance/audit?limit=10", nil)
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("audit status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Entries []governance.AuditEntry `json:"entries"`
		Count   int                     `json:"count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Count != 1 || resp.Entries[0].Section != "01-identity" {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestLegacyCRUDGovernanceBlocked(t *testing.T) {
	srv := governedTestServer(t)
	body := `{"key":"section:03-commitment","value":"{\"red_lines\":[]}"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/memories", bytes.NewBufferString(body))
	req.Header.Set("X-Garden-Actor", "agent")
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", rec.Code, rec.Body.String())
	}
}

type deepMockGraph struct {
	facts []facade.GraphFact
}

func (m deepMockGraph) QueryEntity(_ context.Context, _ string, _ string, _ string) ([]facade.GraphFact, error) {
	return m.facts, nil
}

func (m deepMockGraph) Timeline(_ context.Context, _ string) ([]facade.TimelineEvent, error) {
	return nil, nil
}

func deepRecallTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	store, err := governance.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	engine := governance.NewEngine(store)
	_ = engine.Initialize(context.Background())

	traceStore, err := recall.OpenTraceStore(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { traceStore.Close() })

	fast := &recall.FastService{Gov: engine, Searcher: fakeCardSearcher{}}
	deep := &recall.DeepService{
		Fast:    fast,
		Graph:   deepMockGraph{facts: []facade.GraphFact{{Predicate: "leads", Object: "alice", Confidence: 0.9}}},
		Arbiter: arbiter.New(),
		Traces:  traceStore,
	}
	h := &crud.Handler{
		Router: &router.Router{
			Governance: router.NewGovernanceBackend(engine),
			Mentle:     &mockBackend{name: "mentle"},
		},
	}
	return &Server{
		Handler:    h,
		FastRecall: fast,
		DeepRecall: deep,
		TraceStore: traceStore,
		Addr:       ":0",
	}
}

func TestDeepRecallEndpoint(t *testing.T) {
	srv := deepRecallTestServer(t)
	body := `{"query":"governance","trigger_reason":"test","capabilities":["kg"],"entities":["garden"],"budget_chars":4000}`
	req := httptest.NewRequest(http.MethodPost, "/v2/recall/deep", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["recall_trace"] == nil {
		t.Fatal("missing recall_trace")
	}
	trace := resp["recall_trace"].(map[string]any)
	if trace["trace_id"] == "" || trace["trigger_reason"] != "test" {
		t.Fatalf("trace=%v", trace)
	}
	if resp["mode"] != "deep" {
		t.Fatalf("mode=%v", resp["mode"])
	}
}

func TestDeepRecallEndpointUnavailable(t *testing.T) {
	srv := testServer()
	srv.DeepRecall = nil
	req := httptest.NewRequest(http.MethodPost, "/v2/recall/deep", bytes.NewBufferString(`{"query":"test","trigger_reason":"x"}`))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestDeepRecallTraceEndpoint(t *testing.T) {
	srv := deepRecallTestServer(t)
	body := `{"query":"trace test","trigger_reason":"verification","budget_chars":4000}`
	req := httptest.NewRequest(http.MethodPost, "/v2/recall/deep", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("deep status=%d body=%s", rec.Code, rec.Body.String())
	}
	var deepResp struct {
		Trace struct {
			TraceID string `json:"trace_id"`
		} `json:"recall_trace"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&deepResp); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodGet, "/v2/recall/traces/"+deepResp.Trace.TraceID, nil)
	req.SetPathValue("trace_id", deepResp.Trace.TraceID)
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("trace status=%d body=%s", rec.Code, rec.Body.String())
	}
	var trace map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&trace); err != nil {
		t.Fatal(err)
	}
	if trace["trigger_reason"] != "verification" {
		t.Fatalf("trace=%v", trace)
	}
}

func TestDeepRecallTraceNotFound(t *testing.T) {
	srv := deepRecallTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/v2/recall/traces/nonexistent", nil)
	req.SetPathValue("trace_id", "nonexistent")
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestFastRecallDoesNotTriggerKG(t *testing.T) {
	srv := deepRecallTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/v2/recall/fast", bytes.NewBufferString(`{"query":"test","budget_chars":4000}`))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["assertions"] != nil || resp["proposals"] != nil {
		t.Fatal("fast recall must not contain assertions or proposals")
	}
	if resp["recall_trace"] != nil {
		t.Fatal("fast recall must not contain recall_trace")
	}
}

func evolutionTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	store, err := governance.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	engine := governance.NewEngine(store)
	_ = engine.Initialize(context.Background())

	evoStore, err := evolution.OpenStore(filepath.Join(dir, "evo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { evoStore.Close() })
	evoEvents, err := evolution.OpenEventStore(filepath.Join(dir, "evo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { evoEvents.Close() })

	evoService := &evolution.Service{Provider: nil, Store: evoStore, Events: evoEvents, Hub: evolution.DefaultHubPolicy()}
	h := &crud.Handler{
		Router: &router.Router{
			Governance: router.NewGovernanceBackend(engine),
			Mentle:     &mockBackend{name: "mentle"},
		},
	}
	return &Server{
		Handler:    h,
		FastRecall: &recall.FastService{Gov: engine, Searcher: fakeCardSearcher{}},
		Evolution:  evoService,
		Addr:       ":0",
	}
}

func TestEvolutionStartRunUnavailable(t *testing.T) {
	srv := evolutionTestServer(t)
	body := `{"trigger":"test","evidence_refs":["ref_1"]}`
	req := httptest.NewRequest(http.MethodPost, "/v2/evolution/runs", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestEvolutionGetRunNotFound(t *testing.T) {
	srv := evolutionTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/v2/evolution/runs/nonexistent", nil)
	req.SetPathValue("run_id", "nonexistent")
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestEvolutionDoesNotAffectFastRecall(t *testing.T) {
	srv := evolutionTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/v2/recall/fast", bytes.NewBufferString(`{"query":"test","budget_chars":4000}`))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("fast recall status=%d body=%s", rec.Code, rec.Body.String())
	}
}
