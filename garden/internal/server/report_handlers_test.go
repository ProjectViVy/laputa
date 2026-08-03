package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/dashimaki/garden/internal/report"
	"github.com/dashimaki/mentle/facade"
)

type srvReportLister struct{ items []facade.Memory }

func (l srvReportLister) ListMemories(context.Context, facade.ListMemoryOptions) (facade.MemoryPage, error) {
	return facade.MemoryPage{Items: l.items}, nil
}

func reportTestServer(t *testing.T, items []facade.Memory) *Server {
	t.Helper()
	svc, err := report.Open(filepath.Join(t.TempDir(), "garden.db"), srvReportLister{items}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	return &Server{Reports: svc, Addr: ":0"}
}

func generateDaily(t *testing.T, srv *Server) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v2/reports/generate", bytes.NewBufferString(`{"cadence":"daily"}`))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("generate status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestReportGenerateEndpoint(t *testing.T) {
	now := time.Now().UTC()
	srv := reportTestServer(t, []facade.Memory{{ID: "mem_1", Kind: "fact", Content: "did report work", UpdatedAt: now}})
	generateDaily(t, srv)

	req := httptest.NewRequest(http.MethodPost, "/v2/reports/generate", bytes.NewBufferString(`{"cadence":"daily"}`))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Generated bool          `json:"generated"`
		Report    report.Report `json:"report"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Generated || resp.Report.Generator != report.GeneratorDeterministic || resp.Report.Revision < 1 {
		t.Fatalf("resp=%+v", resp)
	}

	empty := reportTestServer(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/v2/reports/generate", bytes.NewBufferString(`{"cadence":"daily"}`))
	rec = httptest.NewRecorder()
	empty.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("empty window status=%d", rec.Code)
	}
	var emptyResp struct {
		Generated bool `json:"generated"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&emptyResp)
	if emptyResp.Generated {
		t.Fatal("empty window must not generate")
	}

	req = httptest.NewRequest(http.MethodPost, "/v2/reports/generate", bytes.NewBufferString(`{"cadence":"hourly"}`))
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid cadence status=%d, want 400", rec.Code)
	}
}

func TestReportsListEndpoint(t *testing.T) {
	now := time.Now().UTC()
	srv := reportTestServer(t, []facade.Memory{{ID: "mem_1", Content: "work", UpdatedAt: now}})
	generateDaily(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/v2/reports?cadence=daily", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Items []report.Report `json:"items"`
		Count int             `json:"count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Count == 0 || len(resp.Items) == 0 {
		t.Fatalf("resp=%+v", resp)
	}
	item := resp.Items[0]
	if item.Cadence != "daily" || item.Generator != report.GeneratorDeterministic || item.Scope != "mentle_active" || item.Revision < 1 {
		t.Fatalf("item=%+v", item)
	}

	req = httptest.NewRequest(http.MethodGet, "/v2/reports?cadence=hourly", nil)
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid cadence status=%d, want 400", rec.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "/v2/reports?cadence=daily&limit=abc", nil)
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid limit status=%d, want 400", rec.Code)
	}
}

func TestLatestReportV2LazyGenerate(t *testing.T) {
	now := time.Now().UTC()
	srv := reportTestServer(t, []facade.Memory{{ID: "mem_1", Content: "lazy generation", UpdatedAt: now}})
	req := httptest.NewRequest(http.MethodGet, "/v2/reports/latest?cadence=daily", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var r report.Report
	if err := json.NewDecoder(rec.Body).Decode(&r); err != nil {
		t.Fatal(err)
	}
	if len(r.SourceIDs) == 0 || r.Generator == "" || r.Revision < 1 {
		t.Fatalf("report=%+v", r)
	}
}

func TestReportOrientationEndpoint(t *testing.T) {
	now := time.Now().UTC()
	srv := reportTestServer(t, []facade.Memory{{ID: "mem_1", Content: "orientation source material", UpdatedAt: now}})
	generateDaily(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/v2/reports/orientation?budget=150", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var view report.OrientationView
	if err := json.NewDecoder(rec.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if view.Note != report.OrientationNote {
		t.Fatalf("note=%q", view.Note)
	}
	if runes := len([]rune(view.Orientation)); runes == 0 || runes > 150 {
		t.Fatalf("orientation runes=%d", runes)
	}
	if view.BudgetChars != 150 {
		t.Fatalf("budget=%d", view.BudgetChars)
	}

	req = httptest.NewRequest(http.MethodGet, "/v2/reports/orientation?budget=-5", nil)
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid budget status=%d, want 400", rec.Code)
	}

	empty := reportTestServer(t, nil)
	req = httptest.NewRequest(http.MethodGet, "/v2/reports/orientation", nil)
	rec = httptest.NewRecorder()
	empty.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("no-report status=%d", rec.Code)
	}
	var emptyView report.OrientationView
	_ = json.NewDecoder(rec.Body).Decode(&emptyView)
	if emptyView.Note != report.OrientationNote || len(emptyView.Warnings) == 0 {
		t.Fatalf("empty view=%+v", emptyView)
	}
}

func TestReportEndpointsUnavailable(t *testing.T) {
	srv := &Server{Addr: ":0"}
	for _, route := range []struct {
		method, path string
	}{
		{http.MethodGet, "/v2/reports?cadence=daily"},
		{http.MethodGet, "/v2/reports/latest?cadence=daily"},
		{http.MethodPost, "/v2/reports/generate"},
		{http.MethodGet, "/v2/reports/orientation"},
	} {
		var body *bytes.Buffer
		if route.method == http.MethodPost {
			body = bytes.NewBufferString(`{"cadence":"daily"}`)
		} else {
			body = bytes.NewBuffer(nil)
		}
		req := httptest.NewRequest(route.method, route.path, body)
		rec := httptest.NewRecorder()
		srv.HTTPHandler().ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s %s: status=%d, want 503", route.method, route.path, rec.Code)
		}
	}
}
