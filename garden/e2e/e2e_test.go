//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/dashimaki/garden/internal/evolution/hubtest"
)

const (
	startupTimeout = 15 * time.Second
	requestTimeout = 30 * time.Second
)

// TestGardenEndToEnd exercises the shipped Garden executable over its real HTTP
// interface. Its state, logs, binary, and listen address are all test-local.
func TestGardenEndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	mentleConfig := filepath.Join(tempDir, "mentle-config")
	if err := os.MkdirAll(mentleConfig, 0700); err != nil {
		t.Fatal(err)
	}
	configJSON := fmt.Sprintf(`{"palace_path":%q}`, filepath.Join(tempDir, "palace"))
	if err := os.WriteFile(filepath.Join(mentleConfig, "config.json"), []byte(configJSON), 0600); err != nil {
		t.Fatal(err)
	}
	address := freeAddress(t)
	binary := filepath.Join(tempDir, "garden")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}

	build := exec.Command(goTool(), "build", "-o", binary, ".")
	build.Dir = projectRoot(t)
	build.Env = append(os.Environ(), "GOSUMDB=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build garden: %v\n%s", err, output)
	}

	cognitiveDir := filepath.Join(tempDir, "cognitive")
	if err := os.MkdirAll(cognitiveDir, 0755); err != nil {
		t.Fatal(err)
	}
	worldSeed := `# WORLD

## [environment] E2E machine
- status: confirmed
- confidence: high
- scope: e2e
- source: user
- updated: 2026-08-03T00:00:00Z

E2E_WORLD_MARKER machine description for the end-to-end test.
`
	if err := os.WriteFile(filepath.Join(cognitiveDir, "WORLD.MD"), []byte(worldSeed), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binary)
	cmd.Env = append(os.Environ(),
		"GARDEN_ADDR="+address,
		"GARDEN_GOVERNANCE_DIR="+filepath.Join(tempDir, "governance"),
		"GARDEN_LOG_DIR="+filepath.Join(tempDir, "logs"),
		"GARDEN_MENTLE_CONFIG_DIR="+mentleConfig,
		"GARDEN_STATE_DB="+filepath.Join(tempDir, "garden.db"),
		"GARDEN_COGNITIVE_DIR="+cognitiveDir,
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start garden: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	client := &http.Client{Timeout: requestTimeout}
	baseURL := "http://" + address
	waitForHealth(t, client, baseURL)
	var health struct {
		APIContract string `json:"api_contract"`
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/health", nil, http.StatusOK, &health)
	if health.APIContract != "garden-hermes/1" {
		t.Fatalf("api contract=%q", health.APIContract)
	}

	var scopedFast struct {
		Context string `json:"context"`
	}
	writeJSON(t, client, http.MethodPost, baseURL+"/v2/recall/fast", map[string]any{"query": "machine", "scope": "e2e", "budget_chars": 4000}, http.StatusOK, &scopedFast)
	if !strings.Contains(scopedFast.Context, "E2E_WORLD_MARKER") {
		t.Errorf("fast recall context missing scoped world claim: %q", scopedFast.Context)
	}
	if strings.Contains(scopedFast.Context, "primary source of truth") || strings.Contains(scopedFast.Context, "## R") {
		t.Errorf("MEMRULES text leaked into recall context: %q", scopedFast.Context)
	}

	var bootstrap struct {
		TraceID string `json:"trace_id"`
		Context string `json:"context"`
	}
	writeJSON(t, client, http.MethodPost, baseURL+"/v2/recall/bootstrap", map[string]any{"session_id": "e2e", "intent": "role", "budget_chars": 1000}, http.StatusOK, &bootstrap)
	if bootstrap.TraceID == "" || bootstrap.Context == "" {
		t.Fatalf("bootstrap=%+v", bootstrap)
	}

	var created struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
		Status  string `json:"status"`
	}
	writeJSON(t, client, http.MethodPost, baseURL+"/v2/memories", map[string]any{"content": "Garden API v2 contract accepted", "kind": "decision", "scope": "project:garden"}, http.StatusCreated, &created)
	if created.ID == "" || created.Version != 1 || created.Status != "active" {
		t.Fatalf("created=%+v", created)
	}
	var updated struct {
		Version int    `json:"version"`
		Content string `json:"content"`
	}
	writeJSON(t, client, http.MethodPatch, baseURL+"/v2/memories/"+created.ID, map[string]any{"content": "Garden API v2 contract implemented", "expected_version": 1}, http.StatusOK, &updated)
	if updated.Version != 2 {
		t.Fatalf("updated=%+v", updated)
	}
	var page struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/v2/memories", nil, http.StatusOK, &page)
	if len(page.Items) == 0 {
		t.Fatal("canonical list is empty")
	}

	transcript := "The Garden API v2 implementation was completed in this session."
	sum := sha256.Sum256([]byte(transcript))
	var accepted struct {
		IngestionID string `json:"ingestion_id"`
	}
	writeJSON(t, client, http.MethodPost, baseURL+"/v2/ingest/sessions", map[string]any{"session_id": "sess_e2e", "event_id": "evt_e2e", "phase": "session_end", "content": transcript, "content_hash": fmt.Sprintf("sha256:%x", sum)}, http.StatusAccepted, &accepted)
	pollIngestion(t, client, baseURL, accepted.IngestionID)

	var worldProj struct {
		Claims    []map[string]any `json:"claims"`
		Total     int              `json:"total"`
		Projected int              `json:"projected"`
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/v2/cognitive/world?scope=e2e", nil, http.StatusOK, &worldProj)
	if worldProj.Total != 1 || worldProj.Projected != 1 {
		t.Fatalf("world projection=%+v, want the seeded e2e claim", worldProj)
	}

	memrulesData, err := os.ReadFile(filepath.Join(cognitiveDir, "MEMRULES.MD"))
	if err != nil {
		t.Fatalf("boot did not create default MEMRULES.MD: %v", err)
	}
	for _, id := range []string{"## R1", "## R7"} {
		if !strings.Contains(string(memrulesData), id) {
			t.Errorf("default MEMRULES.MD missing %q", id)
		}
	}
	worldData, err := os.ReadFile(filepath.Join(cognitiveDir, "WORLD.MD"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(worldData), "E2E_WORLD_MARKER") {
		t.Error("seeded WORLD.MD was overwritten at boot")
	}

	var latestV2 struct {
		Cadence   string `json:"cadence"`
		Generator string `json:"generator"`
		Revision  int    `json:"revision"`
		Scope     string `json:"scope"`
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/v2/reports/latest?cadence=daily", nil, http.StatusOK, &latestV2)
	if latestV2.Cadence != "daily" || latestV2.Generator == "" {
		t.Fatalf("v2 latest=%+v", latestV2)
	}
	var listResp struct {
		Cadence string `json:"cadence"`
		Count   int    `json:"count"`
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/v2/reports?cadence=daily", nil, http.StatusOK, &listResp)
	if listResp.Cadence != "daily" || listResp.Count == 0 {
		t.Fatalf("v2 report list=%+v", listResp)
	}
	var orientation struct {
		Note     string   `json:"note"`
		Budget   int      `json:"budget_chars"`
		Warnings []string `json:"warnings"`
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/v2/reports/orientation?budget=1500", nil, http.StatusOK, &orientation)
	if orientation.Note != "orientation only; does not replace transient spool recovery" {
		t.Fatalf("orientation note=%q", orientation.Note)
	}

	var modCreated struct {
		ID     string `json:"id"`
		Kind   string `json:"kind"`
		Status string `json:"status"`
	}
	writeJSON(t, client, http.MethodPost, baseURL+"/v2/reports/modules", map[string]any{"kind": "ambition", "content": "ship gate f end to end"}, http.StatusCreated, &modCreated)
	if modCreated.ID == "" || modCreated.Kind != "ambition" || modCreated.Status != "active" {
		t.Fatalf("created module=%+v", modCreated)
	}
	var modList struct {
		Count int `json:"count"`
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/v2/reports/modules?kind=ambition&status=all", nil, http.StatusOK, &modList)
	if modList.Count != 1 {
		t.Fatalf("module list count=%d", modList.Count)
	}
	writeJSON(t, client, http.MethodPost, baseURL+"/v2/reports/modules", map[string]any{"kind": "suggestion", "content": "make reports more readable"}, http.StatusCreated, &modCreated)
	writeJSON(t, client, http.MethodPatch, baseURL+"/v2/reports/modules/"+modCreated.ID, map[string]any{"status": "dismissed"}, http.StatusOK, &modCreated)
	if modCreated.Status != "dismissed" {
		t.Fatalf("dismissed module=%+v", modCreated)
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/v2/reports/modules?kind=suggestion&status=active", nil, http.StatusOK, &modList)
	if modList.Count != 0 {
		t.Fatalf("active suggestions after dismiss=%d", modList.Count)
	}
	var monthly struct {
		Generated bool `json:"generated"`
		Report    struct {
			Modules []string `json:"modules"`
		} `json:"report"`
	}
	writeJSON(t, client, http.MethodPost, baseURL+"/v2/reports/generate", map[string]any{"cadence": "monthly"}, http.StatusOK, &monthly)
	if !monthly.Generated {
		t.Fatal("monthly report was not generated")
	}
	if len(monthly.Report.Modules) != 1 || monthly.Report.Modules[0] != "AMBITION" {
		t.Fatalf("monthly modules=%v", monthly.Report.Modules)
	}

	var deleted struct {
		Deleted bool `json:"deleted"`
	}
	writeJSON(t, client, http.MethodDelete, baseURL+"/v2/memories/"+created.ID, nil, http.StatusOK, &deleted)
	if !deleted.Deleted {
		t.Fatal("canonical delete was not confirmed")
	}

	var pipelines struct {
		Pipelines []map[string]any `json:"pipelines"`
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/v2/pipelines", nil, http.StatusOK, &pipelines)
	if len(pipelines.Pipelines) == 0 {
		t.Fatal("pipeline status returned no pipelines")
	}

	var deepResp struct {
		Mode        string `json:"mode"`
		RecallTrace struct {
			TraceID       string `json:"trace_id"`
			TriggerReason string `json:"trigger_reason"`
		} `json:"recall_trace"`
		Assertions []any `json:"assertions"`
	}
	writeJSON(t, client, http.MethodPost, baseURL+"/v2/recall/deep", map[string]any{
		"query":          "governance",
		"trigger_reason": "e2e verification",
		"capabilities":   []string{"kg", "timeline"},
		"entities":       []string{"garden"},
		"budget_chars":   4000,
	}, http.StatusOK, &deepResp)
	if deepResp.Mode != "deep" {
		t.Fatalf("deep mode=%q", deepResp.Mode)
	}
	if deepResp.RecallTrace.TraceID == "" || deepResp.RecallTrace.TriggerReason != "e2e verification" {
		t.Fatalf("deep trace=%+v", deepResp.RecallTrace)
	}

	var trace map[string]any
	writeJSON(t, client, http.MethodGet, baseURL+"/v2/recall/traces/"+deepResp.RecallTrace.TraceID, nil, http.StatusOK, &trace)
	if trace["trigger_reason"] != "e2e verification" {
		t.Fatalf("trace=%v", trace)
	}

	var fastResp map[string]any
	writeJSON(t, client, http.MethodPost, baseURL+"/v2/recall/fast", map[string]any{"query": "test", "budget_chars": 4000}, http.StatusOK, &fastResp)
	if fastResp["assertions"] != nil || fastResp["proposals"] != nil || fastResp["recall_trace"] != nil {
		t.Fatal("fast recall must not contain deep recall fields")
	}
}

func pollIngestion(t *testing.T, client *http.Client, baseURL, id string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var status struct {
			Status    string   `json:"status"`
			MemoryIDs []string `json:"memory_ids"`
			Error     *string  `json:"error"`
		}
		writeJSON(t, client, http.MethodGet, baseURL+"/v2/ingestions/"+id, nil, http.StatusOK, &status)
		if status.Status == "completed" || status.Status == "completed_degraded" {
			if len(status.MemoryIDs) == 0 {
				t.Fatalf("ingestion=%+v", status)
			}
			return
		}
		if status.Status == "failed" {
			t.Fatalf("ingestion failed: %+v", status)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("ingestion did not complete")
}

// TestGardenEvoMapEndToEnd runs the shipped Garden executable against a
// hermetic mock of the EvoMap hub (GEP-A2A, ADR-0010): startup wiring
// (auto-registration), a full evolution run round trip over HTTP, and the
// hub status endpoint. The live hub is never touched.
func TestGardenEvoMapEndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	hub := hubtest.New()
	defer hub.Close()

	address := freeAddress(t)
	binary := filepath.Join(tempDir, "garden")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command(goTool(), "build", "-o", binary, ".")
	build.Dir = projectRoot(t)
	build.Env = append(os.Environ(), "GOSUMDB=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build garden: %v\n%s", err, output)
	}

	cmd := exec.Command(binary)
	cmd.Env = append(os.Environ(),
		"GARDEN_ADDR="+address,
		"GARDEN_GOVERNANCE_DIR="+filepath.Join(tempDir, "governance"),
		"GARDEN_LOG_DIR="+filepath.Join(tempDir, "logs"),
		"GARDEN_STATE_DB="+filepath.Join(tempDir, "garden.db"),
		"GARDEN_COGNITIVE_DIR="+filepath.Join(tempDir, "cognitive"),
		"GARDEN_EVOMAP_HUB_URL="+hub.URL,
		"GARDEN_EVOMAP_CREDS="+filepath.Join(tempDir, "evomap", "node.json"),
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start garden: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	client := &http.Client{Timeout: requestTimeout}
	baseURL := "http://" + address
	waitForHealth(t, client, baseURL)

	var health struct {
		Components map[string]string `json:"components"`
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/health", nil, http.StatusOK, &health)
	if health.Components["evolution"] != "ok" {
		t.Fatalf("evolution component=%q, want ok (mock hub + auto-register)", health.Components["evolution"])
	}

	var run struct {
		RunID    string `json:"run_id"`
		Status   string `json:"status"`
		Provider string `json:"provider"`
	}
	writeJSON(t, client, http.MethodPost, baseURL+"/v2/evolution/runs", map[string]any{
		"trigger":       "e2e discovery run",
		"outcome":       "garden e2e verifies the evomap provider round trip",
		"trace_ref":     "e2e_trace",
		"evidence_refs": []string{"e2e_trace"},
		"policy":        map[string]any{"publication_allowed": false},
	}, http.StatusAccepted, &run)
	if run.RunID == "" || run.Provider != "evomap" {
		t.Fatalf("run=%+v", run)
	}

	var got struct {
		Status string `json:"status"`
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/v2/evolution/runs/"+run.RunID, nil, http.StatusOK, &got)
	if got.Status != "completed" {
		t.Fatalf("run status=%q, want completed", got.Status)
	}

	var hubStatus struct {
		NodeID         string  `json:"node_id"`
		SurvivalStatus string  `json:"survival_status"`
		CreditBalance  float64 `json:"credit_balance"`
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/v2/evolution/hub/status", nil, http.StatusOK, &hubStatus)
	if hubStatus.NodeID != "node_test_abcd" || hubStatus.SurvivalStatus != "alive" || hubStatus.CreditBalance != 100 {
		t.Fatalf("hub status=%+v", hubStatus)
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}
	return root
}

func goTool() string {
	name := "go"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(runtime.GOROOT(), "bin", name)
}

func freeAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve local address: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release local address: %v", err)
	}
	return address
}

func waitForHealth(t *testing.T, client *http.Client, baseURL string) {
	t.Helper()
	deadline := time.Now().Add(startupTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "/health")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
			lastErr = fmt.Errorf("health returned %s", resp.Status)
		} else {
			lastErr = err
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("garden did not become healthy within %s: %v", startupTimeout, lastErr)
}

func writeJSON(t *testing.T, client *http.Client, method, url string, body any, wantStatus int, result any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode %s request: %v", method, err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, url, reader)
	if err != nil {
		t.Fatalf("create %s request: %v", method, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s response: %v", method, err)
	}
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s status = %d, want %d, body = %s", method, url, resp.StatusCode, wantStatus, responseBody)
	}
	if result != nil {
		if err := json.Unmarshal(responseBody, result); err != nil {
			t.Fatalf("decode %s response: %v; body = %s", method, err, responseBody)
		}
	}
}
