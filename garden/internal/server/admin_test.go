package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/dashimaki/garden/internal/activity"
	"github.com/dashimaki/garden/internal/crud"
	"github.com/dashimaki/garden/internal/ingest"
	"github.com/dashimaki/garden/internal/recall"
	"github.com/dashimaki/garden/internal/router"
	"github.com/dashimaki/laputa/governance"
)

func adminTestServer(t *testing.T) *Server {
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

	dbPath := filepath.Join(dir, "admin.db")
	traceStore, err := recall.OpenTraceStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { traceStore.Close() })

	ingestions, err := ingest.Open(dbPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ingestions.Close() })

	spool, err := activity.OpenSpool(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { spool.Close() })
	ingestions.Spool = spool

	h := &crud.Handler{
		Router: &router.Router{
			Governance: router.NewGovernanceBackend(engine),
			Mentle:     &mockBackend{name: "mentle"},
		},
	}
	return &Server{
		Handler:    h,
		TraceStore: traceStore,
		Ingestions: ingestions,
		Governed:   governed,
		Components: map[string]string{"mentle": "ok", "pipeline": "degraded"},
		Addr:       ":0",
	}
}

func TestAdminOverviewReturnsComponents(t *testing.T) {
	srv := adminTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/v2/admin/overview", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["source"] != "live" {
		t.Errorf("source=%v, want live", resp["source"])
	}
	if resp["status"] != "degraded" {
		t.Errorf("status=%v, want degraded (pipeline is degraded)", resp["status"])
	}
	components, ok := resp["components"].(map[string]any)
	if !ok {
		t.Fatal("components missing or wrong type")
	}
	if components["garden"] != "ok" || components["laputa"] != "ok" {
		t.Errorf("garden/laputa should be ok: %v", components)
	}
	if components["pipeline"] != "degraded" {
		t.Errorf("pipeline=%v, want degraded", components["pipeline"])
	}
	if _, ok := resp["ingestion"]; !ok {
		t.Error("ingestion stats missing")
	}
}

func TestAdminComponentsListsAll(t *testing.T) {
	srv := adminTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/v2/admin/components", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Components []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Source string `json:"source"`
		} `json:"components"`
		APIContract string `json:"api_contract"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.APIContract != "garden-hermes/1" {
		t.Errorf("api_contract=%q", resp.APIContract)
	}
	names := map[string]string{}
	for _, c := range resp.Components {
		names[c.Name] = c.Status
		if c.Source != "live" {
			t.Errorf("component %s source=%q, want live", c.Name, c.Source)
		}
	}
	for _, want := range []string{"garden", "laputa", "mentle", "pipeline"} {
		if _, ok := names[want]; !ok {
			t.Errorf("component %q missing", want)
		}
	}
}

func TestAdminContextManifestNotFound(t *testing.T) {
	srv := adminTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/v2/admin/context-manifest/nonexistent", nil)
	req.SetPathValue("trace_id", "nonexistent")
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404, body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminContextManifestReturnsTrace(t *testing.T) {
	srv := adminTestServer(t)
	trace := recall.RecallTrace{
		TraceID: "admin-test-trace",
		Query:   "admin verification",
		Scope:   "test",
		Steps:   []recall.TraceStep{{Step: "search", Status: "ok"}},
	}
	if err := srv.TraceStore.Save(context.Background(), trace); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v2/admin/context-manifest/admin-test-trace", nil)
	req.SetPathValue("trace_id", "admin-test-trace")
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["source"] != "live" {
		t.Errorf("source=%v, want live", resp["source"])
	}
	tr, ok := resp["trace"].(map[string]any)
	if !ok {
		t.Fatal("trace field missing")
	}
	if tr["trace_id"] != "admin-test-trace" {
		t.Errorf("trace_id=%v", tr["trace_id"])
	}
}

func TestAdminSpoolEmpty(t *testing.T) {
	srv := adminTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/v2/admin/spool", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["source"] != "live" {
		t.Errorf("source=%v, want live", resp["source"])
	}
	count, ok := resp["pending_count"].(float64)
	if !ok || count != 0 {
		t.Errorf("pending_count=%v, want 0", resp["pending_count"])
	}
	entries, ok := resp["entries"].([]any)
	if !ok || len(entries) != 0 {
		t.Errorf("entries=%v, want empty", resp["entries"])
	}
}

func TestAdminAuditReturnsEntries(t *testing.T) {
	srv := adminTestServer(t)

	mutErr := srv.Governed.Mutate(context.Background(), governance.MutationRequest{
		Section: "01-identity",
		Action:  "write",
		Actor:   governance.ActorUser,
		Reason:  "admin audit test",
		Data:    map[string]any{"test_key": "test_value"},
	})
	if mutErr != nil {
		t.Fatalf("mutation: %v", mutErr)
	}

	req := httptest.NewRequest(http.MethodGet, "/v2/admin/audit?limit=10", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Entries []governance.AuditEntry `json:"entries"`
		Count   int                     `json:"count"`
		Source  string                  `json:"source"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Source != "live" {
		t.Errorf("source=%q, want live", resp.Source)
	}
	if resp.Count == 0 {
		t.Fatal("expected at least one audit entry")
	}
	found := false
	for _, e := range resp.Entries {
		if e.Actor == string(governance.ActorUser) {
			found = true
		}
	}
	if !found {
		t.Error("audit entry with actor=user not found")
	}
}

func TestAdminEndpointsAreReadOnly(t *testing.T) {
	srv := adminTestServer(t)
	paths := []string{
		"/v2/admin/overview",
		"/v2/admin/components",
		"/v2/admin/spool",
		"/v2/admin/audit",
	}
	for _, path := range paths {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
			req := httptest.NewRequest(method, path, nil)
			rec := httptest.NewRecorder()
			srv.HTTPHandler().ServeHTTP(rec, req)
			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s %s: status=%d, want 405", method, path, rec.Code)
			}
		}
	}
}
