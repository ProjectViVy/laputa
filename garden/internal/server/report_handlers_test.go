package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dashimaki/garden/internal/report"
	"github.com/dashimaki/mentle/facade"
)

type srvReportLister struct {
	mu    sync.RWMutex
	items []facade.Memory
}

func (l *srvReportLister) ListMemories(context.Context, facade.ListMemoryOptions) (facade.MemoryPage, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return facade.MemoryPage{Items: append([]facade.Memory(nil), l.items...)}, nil
}

func (l *srvReportLister) setItems(items []facade.Memory) {
	l.mu.Lock()
	l.items = append([]facade.Memory(nil), items...)
	l.mu.Unlock()
}

func reportTestServer(t *testing.T, items []facade.Memory) *Server {
	t.Helper()
	svc, err := report.Open(filepath.Join(t.TempDir(), "garden.db"), &srvReportLister{items: items}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	return &Server{Reports: svc, Capabilities: CapabilityConfig{UserToken: "user-secret", OperatorToken: "operator-secret"}, Addr: ":0"}
}

func reportRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if method != http.MethodGet {
		req.Header.Set("Authorization", "Bearer user-secret")
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func generateDaily(t *testing.T, srv *Server) {
	t.Helper()
	req := reportRequest(http.MethodPost, "/v2/reports/generate", `{"cadence":"daily"}`)
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

	req := reportRequest(http.MethodPost, "/v2/reports/generate", `{"cadence":"daily"}`)
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
	req = reportRequest(http.MethodPost, "/v2/reports/generate", `{"cadence":"daily"}`)
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

	req = reportRequest(http.MethodPost, "/v2/reports/generate", `{"cadence":"hourly"}`)
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
	srv := &Server{Addr: ":0", Capabilities: CapabilityConfig{UserToken: "user-secret", OperatorToken: "operator-secret"}}
	for _, route := range []struct {
		method, path, body string
	}{
		{http.MethodGet, "/v2/reports?cadence=daily", ""},
		{http.MethodGet, "/v2/reports/latest?cadence=daily", ""},
		{http.MethodPost, "/v2/reports/generate", `{"cadence":"daily"}`},
		{http.MethodGet, "/v2/reports/orientation", ""},
		{http.MethodGet, "/v2/reports/modules?kind=ambition", ""},
		{http.MethodPost, "/v2/reports/modules", `{"kind":"ambition","content":"x"}`},
		{http.MethodPatch, "/v2/reports/modules/mod_1", `{"status":"dismissed"}`},
	} {
		req := reportRequest(route.method, route.path, route.body)
		rec := httptest.NewRecorder()
		srv.HTTPHandler().ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s %s: status=%d, want 503", route.method, route.path, rec.Code)
		}
	}
}

func TestModuleEndpointsLifecycle(t *testing.T) {
	srv := reportTestServer(t, nil)

	create := func(kind, content string) (int, string) {
		req := reportRequest(http.MethodPost, "/v2/reports/modules", `{"kind":"`+kind+`","content":"`+content+`"}`)
		rec := httptest.NewRecorder()
		srv.HTTPHandler().ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}

	code, body := create("ambition", "become the best gardener")
	if code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", code, body)
	}
	var amb report.Module
	if err := json.NewDecoder(bytes.NewBufferString(body)).Decode(&amb); err != nil {
		t.Fatal(err)
	}
	if amb.ID == "" || amb.Kind != report.ModuleKindAmbition || amb.Status != report.ModuleStatusActive {
		t.Fatalf("created=%+v", amb)
	}
	if code, body := create("suggestion", "add dark theme"); code != http.StatusCreated {
		t.Fatalf("create suggestion status=%d body=%s", code, body)
	}

	list := func(status string) (int, []report.Module) {
		req := httptest.NewRequest(http.MethodGet, "/v2/reports/modules?kind=ambition&status="+status, nil)
		rec := httptest.NewRecorder()
		srv.HTTPHandler().ServeHTTP(rec, req)
		var resp struct {
			Items []report.Module `json:"items"`
			Count int             `json:"count"`
		}
		_ = json.NewDecoder(rec.Body).Decode(&resp)
		return rec.Code, resp.Items
	}

	code, items := list("active")
	if code != http.StatusOK || len(items) != 1 || items[0].ID != amb.ID {
		t.Fatalf("list active code=%d items=%+v", code, items)
	}
	code, items = list("all")
	if code != http.StatusOK || len(items) != 1 {
		t.Fatalf("list all code=%d items=%+v", code, items)
	}

	req := reportRequest(http.MethodPatch, "/v2/reports/modules/"+amb.ID, `{"content":"become the best gardener in the world"}`)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch edit status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = reportRequest(http.MethodPatch, "/v2/reports/modules/"+amb.ID, `{"status":"dismissed"}`)
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch dismiss status=%d body=%s", rec.Code, rec.Body.String())
	}
	code, items = list("active")
	if code != http.StatusOK || len(items) != 0 {
		t.Fatalf("active after dismiss=%+v", items)
	}
	code, items = list("dismissed")
	if code != http.StatusOK || len(items) != 1 || items[0].ID != amb.ID {
		t.Fatalf("dismissed=%+v", items)
	}
}

func TestModuleEndpointsValidation(t *testing.T) {
	srv := reportTestServer(t, nil)
	req := reportRequest(http.MethodPost, "/v2/reports/modules", `{"kind":"ambition","content":"exists"}`)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("seed module status=%d", rec.Code)
	}
	var seed report.Module
	_ = json.NewDecoder(rec.Body).Decode(&seed)

	cases := []struct {
		method, path, body string
		want               int
	}{
		{http.MethodPost, "/v2/reports/modules", `{"kind":"wish","content":"x"}`, http.StatusBadRequest},
		{http.MethodPost, "/v2/reports/modules", `{"kind":"ambition","content":""}`, http.StatusBadRequest},
		{http.MethodPost, "/v2/reports/modules", `{"kind":"ambition","content":"   "}`, http.StatusBadRequest},
		{http.MethodPost, "/v2/reports/modules", `{"kind":"ambition","content":"` + longModuleContent() + `"}`, http.StatusBadRequest},
		{http.MethodPost, "/v2/reports/modules", `not-json`, http.StatusBadRequest},
		{http.MethodGet, "/v2/reports/modules?kind=ambition&status=bogus", "", http.StatusBadRequest},
		{http.MethodGet, "/v2/reports/modules?kind=wish", "", http.StatusBadRequest},
		{http.MethodPatch, "/v2/reports/modules/mod_nope", `{"status":"dismissed"}`, http.StatusNotFound},
		{http.MethodPatch, "/v2/reports/modules/" + seed.ID, `{}`, http.StatusBadRequest},
		{http.MethodPatch, "/v2/reports/modules/" + seed.ID, `{"status":"bogus"}`, http.StatusBadRequest},
	}
	for _, c := range cases {
		req := reportRequest(c.method, c.path, c.body)
		rec := httptest.NewRecorder()
		srv.HTTPHandler().ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s %s body=%q: status=%d, want %d", c.method, c.path, c.body, rec.Code, c.want)
		}
	}
}

func TestMonthlyReportModulesViaHTTP(t *testing.T) {
	// Open with an empty lister so the startup loop inserts no report;
	// the lister is populated only after the module exists, so the
	// generate call performs a fresh insert with modules attached.
	lister := &srvReportLister{}
	path := filepath.Join(t.TempDir(), "garden.db")
	svc, err := report.Open(path, lister, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	svc.SetClock(func() time.Time { return now })
	t.Cleanup(func() { _ = svc.Close() })
	srv := &Server{
		Reports:      svc,
		Capabilities: CapabilityConfig{UserToken: "user-secret", OperatorToken: "operator-secret"},
		Addr:         ":0",
		now: func() time.Time {
			return now
		},
	}

	req := reportRequest(http.MethodPost, "/v2/reports/modules", `{"kind":"ambition","content":"ship gate f"}`)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	lister.setItems([]facade.Memory{{ID: "mem_1", Content: "august work", UpdatedAt: now}})

	req = reportRequest(http.MethodPost, "/v2/reports/generate", `{"cadence":"monthly"}`)
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("generate status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Generated bool          `json:"generated"`
		Report    report.Report `json:"report"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Generated || len(resp.Report.Modules) != 1 || resp.Report.Modules[0] != "AMBITION" {
		t.Fatalf("resp=%+v", resp)
	}
}

func longModuleContent() string {
	runes := make([]byte, 2001)
	for i := range runes {
		runes[i] = 'x'
	}
	return string(runes)
}
