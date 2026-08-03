package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dashimaki/garden/internal/recall"
	"github.com/dashimaki/laputa/governance"
	"github.com/dashimaki/laputa/governance/cognitive"
)

func TestLegacySectionsNeverInContext(t *testing.T) {
	dir := t.TempDir()
	store, err := governance.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	engine := governance.NewEngine(store)
	if err := engine.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	markers := map[string]string{
		"06-history_md":     "LEGACY_HISTORY_MARKER",
		"13-report_indexes": "LEGACY_REPORT_INDEX_MARKER",
		"14-aaak_summaries": "LEGACY_AAAK_MARKER",
	}
	for section, marker := range markers {
		if err := engine.SetSection(ctx, governance.SectionName(section), map[string]any{"content": marker}); err != nil {
			t.Fatalf("seed %s: %v", section, err)
		}
	}

	srv := &Server{
		FastRecall: &recall.FastService{Gov: engine},
		Addr:       ":0",
	}

	fastReq := httptest.NewRequest(http.MethodPost, "/v2/recall/fast", bytes.NewBufferString(`{"query":"legacy data","budget_chars":6000}`))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, fastReq)
	if rec.Code != http.StatusOK {
		t.Fatalf("fast recall status=%d body=%s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); containsAny(body, markers) {
		t.Errorf("legacy section data leaked into fast recall: %s", body)
	}

	bootReq := httptest.NewRequest(http.MethodPost, "/v2/recall/bootstrap", bytes.NewBufferString(`{"intent":"bootstrap","budget_chars":6000}`))
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, bootReq)
	if rec.Code != http.StatusOK {
		t.Fatalf("bootstrap status=%d body=%s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); containsAny(body, markers) {
		t.Errorf("legacy section data leaked into bootstrap: %s", body)
	}
}

func containsAny(haystack string, markers map[string]string) bool {
	for _, marker := range markers {
		if strings.Contains(haystack, marker) {
			return true
		}
	}
	return false
}

func TestCognitiveWorldEndpoint(t *testing.T) {
	world := loadWorldFixture(t)

	srv := testServer()
	srv.Cognitive = world

	req := httptest.NewRequest(http.MethodGet, "/v2/cognitive/world?scope=dev&budget=2000", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Claims    []cognitive.WorldClaim `json:"claims"`
		Total     int                    `json:"total"`
		Projected int                    `json:"projected"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Total != 3 || resp.Projected != 2 || len(resp.Claims) != 2 {
		t.Fatalf("resp=%+v", resp)
	}

	req = httptest.NewRequest(http.MethodGet, "/v2/cognitive/world?budget=20000", nil)
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("budget>16000: status=%d, want 400", rec.Code)
	}

	srv2 := testServer()
	req = httptest.NewRequest(http.MethodGet, "/v2/cognitive/world", nil)
	rec = httptest.NewRecorder()
	srv2.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("nil Cognitive: status=%d, want 503", rec.Code)
	}
}

func loadWorldFixture(t *testing.T) *cognitive.WorldStore {
	t.Helper()
	worldText := `# WORLD

## [environment] Development machine
- status: confirmed
- confidence: high
- scope: dev, infra
- source: user
- updated: 2026-08-01T10:00:00Z

Windows 11, 64GB RAM, Go 1.26, Node 24.

## [project] LAPUTA repository
- status: observed
- confidence: medium
- scope: dev, project
- source: mem_abc123
- updated: 2026-08-02T14:30:00Z

Monorepo with three Go modules: laputa, mentle, garden.

## [people] Team
- status: inferred
- confidence: low
- scope: org
- source: autodream
- updated: 2026-08-02T16:00:00Z

Single developer working on the project.
`
	path := filepath.Join(t.TempDir(), "WORLD.MD")
	if err := os.WriteFile(path, []byte(worldText), 0644); err != nil {
		t.Fatal(err)
	}
	w, err := cognitive.LoadWorld(path)
	if err != nil {
		t.Fatal(err)
	}
	return w
}
