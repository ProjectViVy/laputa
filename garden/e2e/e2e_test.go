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
	"testing"
	"time"
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

	cmd := exec.Command(binary)
	cmd.Env = append(os.Environ(),
		"GARDEN_ADDR="+address,
		"GARDEN_GOVERNANCE_DIR="+filepath.Join(tempDir, "governance"),
		"GARDEN_LOG_DIR="+filepath.Join(tempDir, "logs"),
		"GARDEN_MENTLE_CONFIG_DIR="+mentleConfig,
		"GARDEN_STATE_DB="+filepath.Join(tempDir, "garden.db"),
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

	key := "section:01-identity"
	value := `{"agent":"garden-e2e","role":"integration-test"}`
	writeJSON(t, client, http.MethodPost, baseURL+"/v1/memories", map[string]any{
		"key":   key,
		"value": value,
	}, http.StatusOK, nil)

	var read struct {
		Key   string         `json:"key"`
		Value map[string]any `json:"value"`
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/v1/memories/"+key, nil, http.StatusOK, &read)
	if read.Key != key || read.Value["agent"] != "garden-e2e" || read.Value["role"] != "integration-test" {
		t.Fatalf("read response = %#v, want persisted section data", read)
	}

	var list struct {
		Records []struct {
			Key string `json:"key"`
		} `json:"records"`
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/v1/memories?prefix=section:01-identity", nil, http.StatusOK, &list)
	if len(list.Records) != 1 || list.Records[0].Key != key {
		t.Fatalf("list response = %#v, want one record for %q", list, key)
	}

	var contextPackage struct {
		TraceID  string `json:"trace_id"`
		Context  string `json:"context"`
		Evidence []struct {
			Source  string `json:"source"`
			Locator string `json:"locator"`
		} `json:"evidence"`
		Degraded bool `json:"degraded"`
	}
	writeJSON(t, client, http.MethodPost, baseURL+"/v1/context/resolve", map[string]any{"intent": "What is this agent's role?", "session_id": "e2e", "mode": "basic"}, http.StatusOK, &contextPackage)
	if contextPackage.TraceID == "" || contextPackage.Context == "" || len(contextPackage.Evidence) == 0 {
		t.Fatalf("context response = %#v, want governed evidence", contextPackage)
	}
	foundGovernance := false
	for _, evidence := range contextPackage.Evidence {
		if evidence.Source == "governance" && evidence.Locator == "01-identity" {
			foundGovernance = true
		}
	}
	if !foundGovernance {
		t.Fatalf("context evidence = %#v, want identity governance evidence", contextPackage.Evidence)
	}
	var bootstrap struct {
		TraceID string `json:"trace_id"`
		Context string `json:"context"`
	}
	writeJSON(t, client, http.MethodPost, baseURL+"/v1/context/bootstrap", map[string]any{"session_id": "e2e", "intent": "role", "budget_chars": 1000}, http.StatusOK, &bootstrap)
	if bootstrap.TraceID == "" || bootstrap.Context == "" {
		t.Fatalf("bootstrap=%+v", bootstrap)
	}

	var created struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
		Status  string `json:"status"`
	}
	writeJSON(t, client, http.MethodPost, baseURL+"/v1/memories", map[string]any{"content": "Garden API v1 contract accepted", "kind": "decision", "scope": "project:garden"}, http.StatusCreated, &created)
	if created.ID == "" || created.Version != 1 || created.Status != "active" {
		t.Fatalf("created=%+v", created)
	}
	var updated struct {
		Version int    `json:"version"`
		Content string `json:"content"`
	}
	writeJSON(t, client, http.MethodPatch, baseURL+"/v1/memories/"+created.ID, map[string]any{"content": "Garden API v1 contract implemented", "expected_version": 1}, http.StatusOK, &updated)
	if updated.Version != 2 {
		t.Fatalf("updated=%+v", updated)
	}
	var page struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/v1/memories?view=canonical", nil, http.StatusOK, &page)
	if len(page.Items) == 0 {
		t.Fatal("canonical list is empty")
	}

	transcript := "The Garden API v1 implementation was completed in this session."
	sum := sha256.Sum256([]byte(transcript))
	var accepted struct {
		IngestionID string `json:"ingestion_id"`
	}
	writeJSON(t, client, http.MethodPost, baseURL+"/v1/sessions", map[string]any{"session_id": "sess_e2e", "event_id": "evt_e2e", "phase": "session_end", "content": transcript, "content_hash": fmt.Sprintf("sha256:%x", sum)}, http.StatusAccepted, &accepted)
	pollIngestion(t, client, baseURL, accepted.IngestionID)
	var daily struct {
		Cadence   string   `json:"cadence"`
		SourceIDs []string `json:"source_ids"`
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/v1/reports/latest?cadence=daily", nil, http.StatusOK, &daily)
	if daily.Cadence != "daily" || len(daily.SourceIDs) == 0 {
		t.Fatalf("daily=%+v", daily)
	}
	var deleted struct {
		Deleted bool `json:"deleted"`
	}
	writeJSON(t, client, http.MethodDelete, baseURL+"/v1/memories/"+created.ID, nil, http.StatusOK, &deleted)
	if !deleted.Deleted {
		t.Fatal("canonical delete was not confirmed")
	}

	var pipelines struct {
		Pipelines []map[string]any `json:"pipelines"`
	}
	writeJSON(t, client, http.MethodGet, baseURL+"/v1/pipelines", nil, http.StatusOK, &pipelines)
	if len(pipelines.Pipelines) == 0 {
		t.Fatal("pipeline status returned no pipelines")
	}

	var deepResp struct {
		Mode       string `json:"mode"`
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

	var forgotten struct {
		OK bool `json:"ok"`
	}
	writeJSON(t, client, http.MethodDelete, baseURL+"/v1/memories/"+key, nil, http.StatusOK, &forgotten)
	if !forgotten.OK {
		t.Fatal("forget response did not confirm success")
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
		writeJSON(t, client, http.MethodGet, baseURL+"/v1/ingestions/"+id, nil, http.StatusOK, &status)
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
