package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	laputa "github.com/dashimaki/laputa/governance"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	store, err := laputa.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	engine := laputa.NewEngine(store)
	if err := engine.Initialize(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}
	srv, err := New(engine, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("server: %v", err)
	}
	return srv
}

func TestServer_Healthz(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.handleHealthz(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["ok"] != true {
		t.Errorf("expected ok=true, got %v", body["ok"])
	}
}

func TestServer_IndexPage_ListsSections(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	srv.handleIndex(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Laputa Governance") {
		t.Errorf("expected dashboard header")
	}
	for _, name := range []string{"01-identity", "07-daily", "14-aaak_summaries"} {
		if !strings.Contains(body, name) {
			t.Errorf("expected section %s in dashboard", name)
		}
	}
	// Status styles should appear
	if !strings.Contains(body, "status-stable") && !strings.Contains(body, "status-tbd") {
		t.Errorf("expected at least one status class")
	}
}

func TestServer_DetailPage_RendersJSON(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("GET", "/s/01-identity", nil)
	rec := httptest.NewRecorder()
	srv.handleDetail(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "01-identity") {
		t.Errorf("expected section name in HTML")
	}
	if !strings.Contains(body, "role") {
		t.Errorf("expected default 'role' field in JSON view")
	}
}

func TestServer_DetailPage_UnknownSection404(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("GET", "/s/99-notexist", nil)
	rec := httptest.NewRecorder()
	srv.handleDetail(rec, req)
	if rec.Code != 404 {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestServer_SectionsJSON(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/sections", nil)
	rec := httptest.NewRecorder()
	srv.handleSectionsJSON(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body struct {
		Sections []sectionInfoJSON `json:"sections"`
		Count    int               `json:"count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Count != 14 {
		t.Errorf("expected 14 sections, got %d", body.Count)
	}
}

func TestServer_SectionJSON(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/sections/07-daily", nil)
	rec := httptest.NewRecorder()
	srv.handleSectionJSON(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body struct {
		Name string         `json:"name"`
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Name != "07-daily" {
		t.Errorf("expected name 07-daily, got %s", body.Name)
	}
	if _, ok := body.Data["reports"]; !ok {
		t.Errorf("expected reports field")
	}
}

func TestServer_SnapshotJSON(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/snapshot", nil)
	rec := httptest.NewRecorder()
	srv.handleSnapshotJSON(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(body), "schema_version") {
		t.Errorf("expected schema_version in snapshot")
	}
}

func TestServer_DetailPage_NoMeta_RendersOK(t *testing.T) {
	store, err := laputa.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	engine := laputa.NewEngine(store)
	if err := engine.Initialize(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}
	// Strip _meta to simulate backends that don't write metadata.
	_ = engine.SetSection(context.Background(), laputa.SectionIdentity, map[string]any{
		"role":         "agent",
		"capabilities": []string{"x"},
	})

	srv, err := New(engine, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("server: %v", err)
	}
	req := httptest.NewRequest("GET", "/s/01-identity", nil)
	rec := httptest.NewRecorder()
	srv.handleDetail(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200 even without _meta, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestServer_LiveRequest_CancelsCleanly(t *testing.T) {
	srv := newTestServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := srv.ListenAndServe(ctx)
	if err != nil && err != context.Canceled {
		t.Fatalf("expected clean shutdown, got %v", err)
	}
}
