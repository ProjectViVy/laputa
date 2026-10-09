//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
)

const (
	startupTimeout = 20 * time.Second
	requestTimeout = 30 * time.Second
	userToken      = "e2e-user-capability"
	agentToken     = "e2e-agent-capability"
	readToken      = "e2e-read-capability"
)

// TestGardenCleanBreakEndToEnd proves the clean-break vertical slice through
// the shipped Garden process. Every stateful resource is test-local and the
// process is restarted in the middle of the assertions.
func TestGardenCleanBreakEndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	profileDir := filepath.Join(tempDir, "profile")
	legacySections := filepath.Join(profileDir, ".laputa", "sections")
	if err := os.MkdirAll(legacySections, 0o700); err != nil {
		t.Fatal(err)
	}
	legacyMarker := []byte(`{"content":"LEGACY_JSON_MUST_BE_IGNORED"}`)
	if err := os.WriteFile(filepath.Join(legacySections, "01-identity.json"), legacyMarker, 0o600); err != nil {
		t.Fatal(err)
	}

	mentleConfig := filepath.Join(tempDir, "mentle-config")
	if err := os.MkdirAll(mentleConfig, 0o700); err != nil {
		t.Fatal(err)
	}
	modelsDir, err := filepath.Abs(filepath.Join("..", "..", "mentle", "models"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(modelsDir, "onnx", "model.onnx")); err != nil {
		t.Fatal(err)
	}
	configJSON := fmt.Sprintf(`{"palace_path":%q,"models_dir":%q}`, filepath.Join(tempDir, "palace"), modelsDir)
	if err := os.WriteFile(filepath.Join(mentleConfig, "config.json"), []byte(configJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	binary := buildGarden(t, tempDir)
	address := freeAddress(t)
	env := map[string]string{
		"GARDEN_ADDR":                      address,
		"GARDEN_PERSONA_DIR":               profileDir,
		"GARDEN_MENTLE_CONFIG_DIR":         mentleConfig,
		"GARDEN_STATE_DB":                  filepath.Join(tempDir, "garden.db"),
		"GARDEN_LOG_DIR":                   filepath.Join(tempDir, "logs"),
		"GARDEN_CAPABILITY_READ_TOKEN":     readToken,
		"GARDEN_CAPABILITY_USER_TOKEN":     userToken,
		"GARDEN_CAPABILITY_AGENT_TOKEN":    agentToken,
		"GARDEN_CAPABILITY_OPERATOR_TOKEN": "e2e-operator-capability",
	}

	process := startGarden(t, binary, env)
	client := &http.Client{Timeout: requestTimeout}
	baseURL := "http://" + address
	waitForHealth(t, client, baseURL)
	defer func() { process.stop(t) }()

	var beforeInit struct {
		Status    string `json:"status"`
		Documents []struct {
			Kind   string `json:"kind"`
			Exists bool   `json:"exists"`
		} `json:"documents"`
	}
	requestJSON(t, client, http.MethodGet, baseURL+"/v2/persona/documents", nil, readToken, http.StatusOK, "", &beforeInit)
	if beforeInit.Status != "uninitialized" || len(beforeInit.Documents) != 8 {
		t.Fatalf("initial Persona status=%+v", beforeInit)
	}

	for _, path := range []string{
		"/v2/persona/files/identity",
		"/v2/persona/requests",
		"/v2/governance/projection",
		"/v2/governance/mutations",
		"/v2/cognitive/world",
	} {
		requestJSON(t, client, http.MethodGet, baseURL+path, nil, "", http.StatusNotFound, "", nil)
	}

	worldMarker := "E2E_WORLD_EXPLICIT_ONLY"
	initialize := map[string]any{
		"identity":     "# Identity\nE2E identity before review",
		"relationship": "# Relationship\nE2E relationship",
		"redline":      "# Redline\nE2E redline",
		"user":         "E2E preferences",
		"world":        "# WORLD\n\n## [environment] E2E\n- status: confirmed\n- source: user\n\n" + worldMarker,
	}
	var initialized struct {
		Status    string           `json:"status"`
		Documents []map[string]any `json:"documents"`
	}
	requestJSON(t, client, http.MethodPost, baseURL+"/v2/persona/initialize", initialize, userToken, http.StatusCreated, "", &initialized)
	if initialized.Status != "ready" || len(initialized.Documents) != 8 {
		t.Fatalf("initialized Persona=%+v", initialized)
	}

	var world struct {
		Content string `json:"content"`
		Kind    string `json:"kind"`
	}
	requestJSON(t, client, http.MethodGet, baseURL+"/v2/persona/documents/world", nil, readToken, http.StatusOK, "", &world)
	if world.Kind != "world" || !strings.Contains(world.Content, worldMarker) {
		t.Fatalf("explicit WORLD read=%+v", world)
	}

	var initialIdentity struct {
		ContentHash string `json:"content_hash"`
		Revision    uint64 `json:"revision"`
	}
	requestJSON(t, client, http.MethodGet, baseURL+"/v2/persona/documents/identity", nil, readToken, http.StatusOK, "", &initialIdentity)
	if initialIdentity.Revision != 1 || initialIdentity.ContentHash == "" {
		t.Fatalf("initial identity=%+v", initialIdentity)
	}

	var frozenBefore struct {
		Context    string `json:"context"`
		FrozenCore struct {
			Sections []struct {
				Content string `json:"content"`
			} `json:"sections"`
		} `json:"frozen_core"`
	}
	requestJSON(t, client, http.MethodPost, baseURL+"/v2/recall/fast", map[string]any{
		"query": "E2E identity", "session_id": "frozen-session", "budget_chars": 4000,
	}, "", http.StatusOK, "", &frozenBefore)
	if len(frozenBefore.FrozenCore.Sections) != 7 {
		t.Fatalf("Frozen Core sections=%d, want 7", len(frozenBefore.FrozenCore.Sections))
	}
	assertAutomaticContextOmits(t, frozenBefore.Context, "before Persona review")

	// A forged actor label cannot turn an agent into a user direct writer.
	requestJSON(t, client, http.MethodPut, baseURL+"/v2/persona/documents/identity", map[string]any{
		"base_revision": 1, "content": "# Identity\nforged direct write", "reason": "forged actor",
	}, agentToken, http.StatusBadRequest, "user", nil)

	var review struct {
		ID           string `json:"id"`
		State        string `json:"state"`
		BaseRevision uint64 `json:"base_revision"`
	}
	requestJSON(t, client, http.MethodPost, baseURL+"/v2/persona/reviews", map[string]any{
		"kind": "identity", "base_revision": initialIdentity.Revision,
		"base_hash":         initialIdentity.ContentHash,
		"proposed_markdown": "# Identity\nE2E identity after approved review",
		"reason":            "clean-break e2e review",
	}, agentToken, http.StatusCreated, "", &review)
	if review.ID == "" || review.State != "pending" || review.BaseRevision != 1 {
		t.Fatalf("review=%+v", review)
	}
	var acceptedReview struct {
		State string `json:"state"`
	}
	requestJSON(t, client, http.MethodPost, baseURL+"/v2/persona/reviews/"+review.ID+"/approve", map[string]any{
		"decision_reason": "approved in clean-break e2e",
	}, userToken, http.StatusOK, "", &acceptedReview)
	if acceptedReview.State != "accepted" {
		t.Fatalf("accepted review=%+v", acceptedReview)
	}

	var frozenSameSession struct {
		Context    string `json:"context"`
		FrozenCore struct {
			Sections []struct {
				Content string `json:"content"`
			} `json:"sections"`
		} `json:"frozen_core"`
	}
	requestJSON(t, client, http.MethodPost, baseURL+"/v2/recall/fast", map[string]any{
		"query": "approved review", "session_id": "frozen-session", "budget_chars": 4000,
	}, "", http.StatusOK, "", &frozenSameSession)
	if len(frozenSameSession.FrozenCore.Sections) != 7 || frozenSameSession.FrozenCore.Sections[1].Content != frozenBefore.FrozenCore.Sections[1].Content {
		t.Fatalf("Frozen Core drifted after Persona edit: before=%+v after=%+v", frozenBefore.FrozenCore.Sections[1], frozenSameSession.FrozenCore.Sections[1])
	}
	assertAutomaticContextOmits(t, frozenSameSession.Context, "same-session recall")

	var history struct {
		Items []struct {
			Revision uint64 `json:"revision"`
		} `json:"items"`
	}
	requestJSON(t, client, http.MethodGet, baseURL+"/v2/persona/history/identity?limit=10", nil, readToken, http.StatusOK, "", &history)
	if len(history.Items) < 2 || history.Items[0].Revision != 2 || history.Items[1].Revision != 1 {
		t.Fatalf("identity history=%+v", history)
	}

	var actmemBefore struct {
		Revision uint64 `json:"revision"`
		Markdown string `json:"markdown"`
	}
	requestJSON(t, client, http.MethodGet, baseURL+"/v2/actmem", nil, readToken, http.StatusOK, "", &actmemBefore)
	if actmemBefore.Revision != 0 {
		t.Fatalf("missing ACTMEM=%+v", actmemBefore)
	}
	actmemMarker := "ACTMEM_EXPLICIT_ONLY"
	var actmemWrite struct {
		Result struct {
			Changed  bool   `json:"changed"`
			Revision uint64 `json:"revision"`
		} `json:"result"`
	}
	requestJSON(t, client, http.MethodPost, baseURL+"/v2/actmem/maintenance", map[string]any{
		"operation": "system_append",
		"entry":     map[string]any{"section": "pulse", "session_id": "e2e-session", "body": actmemMarker},
	}, agentToken, http.StatusOK, "", &actmemWrite)
	if !actmemWrite.Result.Changed || actmemWrite.Result.Revision != 1 {
		t.Fatalf("ACTMEM write=%+v", actmemWrite)
	}
	var actmemQuery struct {
		Items []map[string]any `json:"items"`
	}
	requestJSON(t, client, http.MethodPost, baseURL+"/v2/actmem/query", map[string]any{
		"query": actmemMarker, "max_hits": 5, "max_chars": 1200,
	}, readToken, http.StatusOK, "", &actmemQuery)
	if len(actmemQuery.Items) == 0 {
		t.Fatal("explicit ACTMEM query returned no marker")
	}
	if _, err := os.Stat(filepath.Join(profileDir, "actmem", "ACTMEM.MD")); err != nil {
		t.Fatalf("ACTMEM was not persisted: %v", err)
	}
	assertAutomaticContextOmits(t, frozenSameSession.Context, "ACTMEM after explicit write")

	var created struct {
		Memory struct {
			ID      string `json:"id"`
			Version int    `json:"version"`
		} `json:"memory"`
		IndexState string  `json:"index_state"`
		IndexJobID *string `json:"index_job_id"`
	}
	memoryBody := map[string]any{
		"content": "canonical e2e memory survives restart",
		"kind":    "decision",
		"scope":   "project:garden",
		"source":  map[string]any{"type": "user"},
	}
	requestJSONWithHeaders(t, client, http.MethodPost, baseURL+"/v2/memories", memoryBody, userToken, map[string]string{"Idempotency-Key": "e2e-memory-1"}, http.StatusCreated, &created)
	if created.Memory.ID == "" || created.Memory.Version != 1 || (created.IndexState != "applied" && created.IndexState != "index_pending") {
		t.Fatalf("created memory=%+v", created)
	}
	var duplicate struct {
		Memory struct {
			ID string `json:"id"`
		} `json:"memory"`
	}
	requestJSONWithHeaders(t, client, http.MethodPost, baseURL+"/v2/memories", memoryBody, userToken, map[string]string{"Idempotency-Key": "e2e-memory-1"}, http.StatusCreated, &duplicate)
	if duplicate.Memory.ID != created.Memory.ID {
		t.Fatalf("idempotent create returned %q, want %q", duplicate.Memory.ID, created.Memory.ID)
	}

	transcript := "canonical session ingestion also survives a process boundary"
	hash := sha256.Sum256([]byte(transcript))
	var accepted struct {
		IngestionID string `json:"ingestion_id"`
	}
	requestJSON(t, client, http.MethodPost, baseURL+"/v2/ingest/sessions", map[string]any{
		"session_id": "ingest-session", "event_id": "ingest-event", "phase": "session_end",
		"content": transcript, "content_hash": "sha256:" + hex.EncodeToString(hash[:]),
	}, userToken, http.StatusAccepted, "", &accepted)
	if accepted.IngestionID == "" {
		t.Fatal("ingestion was not accepted")
	}
	pollIngestion(t, client, baseURL, accepted.IngestionID)
	var duplicateIngest struct {
		IngestionID string `json:"ingestion_id"`
	}
	requestJSON(t, client, http.MethodPost, baseURL+"/v2/ingest/sessions", map[string]any{
		"session_id": "ingest-session", "event_id": "ingest-event", "phase": "session_end",
		"content": transcript, "content_hash": "sha256:" + hex.EncodeToString(hash[:]),
	}, userToken, http.StatusAccepted, "", &duplicateIngest)
	if duplicateIngest.IngestionID != accepted.IngestionID {
		t.Fatalf("duplicate ingestion=%q, want %q", duplicateIngest.IngestionID, accepted.IngestionID)
	}

	var health struct {
		Status               string         `json:"status"`
		CanonicalActiveCount int            `json:"canonical_active_count"`
		ExpectedIdentity     any            `json:"expected_embedding_identity"`
		LastRebuild          map[string]any `json:"last_rebuild"`
	}
	requestJSON(t, client, http.MethodGet, baseURL+"/v2/admin/index-health", nil, readToken, http.StatusOK, "", &health)
	if health.Status == "unavailable" || health.CanonicalActiveCount < 1 || health.ExpectedIdentity == nil {
		t.Fatalf("live IndexHealth=%+v", health)
	}

	// Restart the real process and verify both authority and derived recovery
	// boundaries. The old JSON descriptor remains inert evidence on disk.
	process.stop(t)
	process = startGarden(t, binary, env)
	waitForHealth(t, client, baseURL)

	var afterRestart struct {
		Status string `json:"status"`
	}
	requestJSON(t, client, http.MethodGet, baseURL+"/v2/persona/documents", nil, readToken, http.StatusOK, "", &afterRestart)
	if afterRestart.Status != "ready" {
		t.Fatalf("Persona status after restart=%q", afterRestart.Status)
	}
	var actmemAfter struct {
		Revision uint64 `json:"revision"`
		Markdown string `json:"markdown"`
	}
	requestJSON(t, client, http.MethodGet, baseURL+"/v2/actmem", nil, readToken, http.StatusOK, "", &actmemAfter)
	if actmemAfter.Revision != 1 || !strings.Contains(actmemAfter.Markdown, actmemMarker) {
		t.Fatalf("ACTMEM after restart=%+v", actmemAfter)
	}

	var frozenAfterRestart struct {
		FrozenCore struct {
			Sections []struct {
				Content string `json:"content"`
			} `json:"sections"`
		} `json:"frozen_core"`
	}
	requestJSON(t, client, http.MethodPost, baseURL+"/v2/recall/bootstrap", map[string]any{
		"session_id": "frozen-session", "intent": "restart proof", "budget_chars": 4000,
	}, "", http.StatusOK, "", &frozenAfterRestart)
	if len(frozenAfterRestart.FrozenCore.Sections) != 7 || frozenAfterRestart.FrozenCore.Sections[1].Content != frozenBefore.FrozenCore.Sections[1].Content {
		t.Fatalf("restart changed Frozen Core=%+v", frozenAfterRestart.FrozenCore)
	}

	var newSession struct {
		FrozenCore struct {
			Sections []struct {
				Content string `json:"content"`
			} `json:"sections"`
		} `json:"frozen_core"`
	}
	requestJSON(t, client, http.MethodPost, baseURL+"/v2/recall/fast", map[string]any{
		"query": "approved review", "session_id": "new-session", "budget_chars": 4000,
	}, "", http.StatusOK, "", &newSession)
	if len(newSession.FrozenCore.Sections) != 7 || !strings.Contains(newSession.FrozenCore.Sections[1].Content, "after approved review") {
		t.Fatalf("new session did not capture current Persona=%+v", newSession.FrozenCore)
	}

	var persistedMemory struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	requestJSON(t, client, http.MethodGet, baseURL+"/v2/memories/"+created.Memory.ID, nil, readToken, http.StatusOK, "", &persistedMemory)
	if persistedMemory.ID != created.Memory.ID || persistedMemory.Version != 1 {
		t.Fatalf("canonical memory after restart=%+v", persistedMemory)
	}
	requestJSON(t, client, http.MethodGet, baseURL+"/v2/cognitive/world", nil, "", http.StatusNotFound, "", nil)

	legacy, err := os.ReadFile(filepath.Join(legacySections, "01-identity.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(legacy, legacyMarker) {
		t.Fatal("legacy JSON descriptor was modified")
	}
}

func assertAutomaticContextOmits(t *testing.T, contextValue, label string) {
	t.Helper()
	if strings.Contains(contextValue, "E2E_WORLD_EXPLICIT_ONLY") || strings.Contains(contextValue, "ACTMEM_EXPLICIT_ONLY") || strings.Contains(contextValue, "LEGACY_JSON_MUST_BE_IGNORED") {
		t.Fatalf("%s leaked tool-only or legacy material: %q", label, contextValue)
	}
}

type gardenProcess struct {
	cmd *exec.Cmd
}

func (p *gardenProcess) stop(t *testing.T) {
	t.Helper()
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return
	}
	if err := p.cmd.Process.Kill(); err != nil && !strings.Contains(strings.ToLower(err.Error()), "already") {
		t.Logf("stop Garden: %v", err)
	}
	_ = p.cmd.Wait()
	p.cmd.Process = nil
}

func buildGarden(t *testing.T, tempDir string) string {
	t.Helper()
	binary := filepath.Join(tempDir, "garden")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command(goTool(), "build", "-o", binary, ".")
	build.Dir = projectRoot(t)
	build.Env = environment(map[string]string{"GOSUMDB": "off"})
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build garden: %v\n%s", err, output)
	}
	return binary
}

func startGarden(t *testing.T, binary string, overrides map[string]string) *gardenProcess {
	t.Helper()
	cmd := exec.Command(binary)
	cmd.Env = environment(overrides)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start Garden: %v", err)
	}
	return &gardenProcess{cmd: cmd}
}

func environment(overrides map[string]string) []string {
	result := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if _, replace := overrides[key]; !replace {
			result = append(result, entry)
		}
	}
	for key, value := range overrides {
		result = append(result, key+"="+value)
	}
	return result
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
	t.Fatalf("Garden did not become healthy within %s: %v", startupTimeout, lastErr)
}

func requestJSON(t *testing.T, client *http.Client, method, url string, body any, token string, wantStatus int, actor string, result any) {
	t.Helper()
	headers := map[string]string{}
	if actor != "" {
		headers["X-Garden-Actor"] = actor
	}
	requestJSONWithHeaders(t, client, method, url, body, token, headers, wantStatus, result)
}

func requestJSONWithHeaders(t *testing.T, client *http.Client, method, url string, body any, token string, headers map[string]string, wantStatus int, result any) {
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
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
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
		t.Fatalf("%s %s status=%d want=%d body=%s", method, url, resp.StatusCode, wantStatus, responseBody)
	}
	if result != nil {
		if err := json.Unmarshal(responseBody, result); err != nil {
			t.Fatalf("decode %s response: %v; body=%s", method, err, responseBody)
		}
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
		requestJSON(t, client, http.MethodGet, baseURL+"/v2/ingestions/"+id, nil, "", http.StatusOK, "", &status)
		if status.Status == "completed" || status.Status == "completed_degraded" {
			if len(status.MemoryIDs) == 0 {
				t.Fatalf("ingestion completed without canonical memory: %+v", status)
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
