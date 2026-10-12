package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/ProjectViVy/laputa/garden/agentapi"
	"github.com/ProjectViVy/laputa/laputa/persona"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeConfigNormalizesLegacyRelativePaths(t *testing.T) {
	cfg := runtimeConfig("persona", "palace", "models", "state.db", "A")
	for _, p := range []string{cfg.PersonaDir, cfg.PalacePath, cfg.ModelsDir, cfg.StateDB} {
		if !filepath.IsAbs(p) {
			t.Fatal(p)
		}
	}
}
func TestOpenAppSharesOneOwnerAcrossAdapters(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	cfg := runtimeConfig("", filepath.Join(root, "memory"), filepath.Join(root, "models"), filepath.Join(root, "garden.db"), "A")
	cfg.CatalogOnly = true
	app, err := openApp(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	for _, id := range []string{"A", "B"} {
		if _, err = app.installation.RegisterProfile(ctx, agentapi.ProfileRegistration{ProfileID: id, Create: true}); err != nil {
			t.Fatal(err)
		}
		g, _ := app.owner.Subject(ctx, id)
		if _, err = g.Persona.Initialize(persona.Initialization{Identity: id, Relationship: "r", Redline: "safe", User: "u", World: "w"}, "user", persona.SourceInit, "init"); err != nil {
			t.Fatal(err)
		}
	}
	a, _ := app.owner.Subject(ctx, "A")
	b, _ := app.owner.Subject(ctx, "B")
	if a.Mentle != b.Mentle || a.Ingest != b.Ingest || app.server.Installation != app.installation {
		t.Fatal("duplicated owner")
	}
	info, err := os.Stat(filepath.Join(root, "pairing-code"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("pairing code not private", info, err)
	}
	raw, _ := os.ReadFile(filepath.Join(root, "pairing-code"))
	body, _ := json.Marshal(map[string]string{"code": strings.TrimSpace(string(raw))})
	req := httptest.NewRequest("POST", "http://localhost/v2/pairing", bytes.NewReader(body))
	req.Header.Set("Origin", "http://localhost")
	w := httptest.NewRecorder()
	app.server.HTTPHandler().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookie := w.Result().Cookies()[0]
	for _, id := range []string{"A", "B"} {
		req = httptest.NewRequest("GET", "http://localhost/v2/persona/documents/identity", nil)
		req.AddCookie(cookie)
		req.Header.Set("X-Garden-Profile", id)
		w = httptest.NewRecorder()
		app.server.HTTPHandler().ServeHTTP(w, req)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"content":"`+id+`"`) {
			t.Fatal(id, w.Code, w.Body.String())
		}
	}
	req = httptest.NewRequest("GET", "http://localhost/v2/memories", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w = httptest.NewRecorder()
	app.server.HTTPHandler().ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatal("loopback granted", w.Code)
	}
}
func TestOpenAppDoesNotDiscoverAuthority(t *testing.T) {
	root := t.TempDir()
	cfg := runtimeConfig("", filepath.Join(root, "memory"), filepath.Join(root, "models"), filepath.Join(root, "garden.db"), "default")
	cfg.CatalogOnly = true
	app, err := openApp(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	profiles, err := app.installation.ListProfiles(context.Background())
	if err != nil || len(profiles) != 0 {
		t.Fatal("implicit profile", profiles, err)
	}
}
