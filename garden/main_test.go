package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ProjectViVy/laputa/garden/agentapi"
	"github.com/ProjectViVy/laputa/garden/internal/server"
	"github.com/ProjectViVy/laputa/laputa/persona"
	"github.com/ProjectViVy/laputa/mentle/facade"
)

func TestRuntimeConfigNormalizesLegacyRelativePaths(t *testing.T) {
	cfg := runtimeConfig("persona", "palace", "models", "state.db", "profile-a")
	for name, path := range map[string]string{
		"persona": cfg.PersonaDir, "palace": cfg.PalacePath,
		"models": cfg.ModelsDir, "state": cfg.StateDB,
	} {
		if !filepath.IsAbs(path) {
			t.Errorf("%s path remains relative: %q", name, path)
		}
	}
	if cfg.ProfileID != "profile-a" {
		t.Fatalf("profile changed: %q", cfg.ProfileID)
	}
}

func TestOpenAppSharesOneRuntimeAcrossAdapters(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GARDEN_PIPELINE_CONFIG", filepath.Join(root, "missing-pipelines.yaml"))
	t.Setenv("GARDEN_EVOMAP_CREDS", filepath.Join(root, "missing-creds"))
	cfg := runtimeConfig(root, filepath.Join(root, "palace"), filepath.Join(root, "models"), filepath.Join(root, "state.db"), "profile-a")
	app, err := openApp(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if app.runtime == nil || app.server.AgentAPI == nil {
		t.Fatal("missing shared runtime or agent adapter")
	}
	if app.server.ProfileID != "profile-a" {
		t.Fatalf("fixed profile = %q", app.server.ProfileID)
	}
	if app.server.Persona != app.runtime.Persona || app.server.Actmem != app.runtime.Actmem || app.server.Facade != app.runtime.Mentle || app.server.FastRecall != app.runtime.FastRecall || app.server.Ingestions != app.runtime.Ingest || app.server.Activity != app.runtime.Activity || app.server.TraceStore != app.runtime.Trace {
		t.Fatal("HTTP adapter has a duplicate domain service")
	}
	if app.server.Components["mentle"] != "degraded" {
		t.Fatalf("optional local model state: %v", app.server.Components)
	}
	if app.server.Checkpointer == nil || app.server.Reports == nil || app.server.Evolution == nil || app.server.Mailbox == nil || app.server.DeepRecall == nil {
		t.Fatal("app-level management was lost")
	}
	if app.server.DeepRecall.Fast != app.runtime.FastRecall {
		t.Fatal("deep recall uses another runtime")
	}
	if _, err := app.runtime.Persona.Initialize(persona.Initialization{Identity: "identity", Relationship: "relationship", Redline: "redline", User: "user", World: "world"}, "user", persona.SourceInit, "init"); err != nil {
		t.Fatal(err)
	}
	binding := agentapi.Binding{ProfileID: "profile-a", AgentID: "agent", Platform: "test", SessionID: "session"}
	if _, err := app.server.AgentAPI.FastRecall(context.Background(), agentapi.PrincipalAgent, agentapi.FastRecallRequest{Binding: binding, Query: "test", BudgetChars: 1000}); err != nil {
		t.Fatalf("bound agent recall: %v", err)
	}
	binding.ProfileID = "other-profile"
	if _, err := app.server.AgentAPI.FastRecall(context.Background(), agentapi.PrincipalAgent, agentapi.FastRecallRequest{Binding: binding, Query: "test", BudgetChars: 1000}); err == nil {
		t.Fatal("agent could select another profile")
	}
}

func TestOpenAppWithExistingCanonicalAndNoModelStaysLexicalReadOnly(t *testing.T) {
	root := t.TempDir()
	palace := filepath.Join(root, "palace")
	if err := os.MkdirAll(palace, 0700); err != nil {
		t.Fatal(err)
	}
	catalog, err := facade.OpenCatalog(filepath.Join(palace, "canonical.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.Close(); err != nil {
		t.Fatal(err)
	}
	cfg := runtimeConfig(filepath.Join(root, "persona"), palace, filepath.Join(root, "absent-model"), filepath.Join(root, "state.db"), "profile-local")
	app, err := openApp(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if !app.runtime.LexicalOnly || app.server.Components["mentle"] != "degraded" || app.server.DeepRecall.Graph != nil {
		t.Fatalf("lexical startup enabled full-model capability: lexical=%v components=%v graph=%v", app.runtime.LexicalOnly, app.server.Components, app.server.DeepRecall.Graph)
	}
	app.server.Capabilities = server.CapabilityConfig{UserToken: "fixture-user"}
	req := httptest.NewRequest(http.MethodPost, "/v2/memories", bytes.NewBufferString(`{"content":"must-not-write","kind":"note"}`))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer fixture-user")
	rec := httptest.NewRecorder()
	app.server.HTTPHandler().ServeHTTP(rec, req)
	var body struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusServiceUnavailable || body.Code != "memory_read_only" {
		t.Fatalf("lexical write response: status=%d body=%s", rec.Code, rec.Body.String())
	}
}
