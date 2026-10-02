package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMissionHumanOnlyMutation covers the control-plane boundary: the
// authenticated human (user capability) mutates MISSION.MD; the agent and
// autodream capabilities — including a forged X-Garden-Actor — cannot.
func TestMissionHumanOnlyMutation(t *testing.T) {
	srv, _ := personaCapabilityServer(t)
	initializePersonaOverHTTP(t, srv)

	for _, token := range []string{"agent-secret", "dream-secret"} {
		rec := httptest.NewRecorder()
		req := personaRequest(http.MethodPut, "/v2/persona/documents/mission", token, map[string]any{"base_revision": 0, "content": "agent mission", "reason": "spoofed"})
		req.Header.Set("X-Garden-Actor", "user")
		srv.HTTPHandler().ServeHTTP(rec, req)
		if rec.Code == http.StatusOK {
			t.Fatalf("token %s wrote mission: %s", token, rec.Body.String())
		}
	}

	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, personaRequest(http.MethodPut, "/v2/persona/documents/mission", "user-secret", map[string]any{"base_revision": 0, "content": "protect the archive", "reason": "set mission"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("human mission write status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, personaRequest(http.MethodGet, "/v2/persona/documents/mission", "agent-secret", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("agent mission read status=%d body=%s", rec.Code, rec.Body.String())
	}
	var doc struct {
		Content  string `json:"content"`
		Revision uint64 `json:"revision"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if doc.Content != "protect the archive" || doc.Revision != 1 {
		t.Fatalf("mission doc = %+v", doc)
	}
}

// TestMissionNotAnExecutableReview: a change request for mission is rejected
// at creation; Mission is never an executable Persona request.
func TestMissionNotAnExecutableReview(t *testing.T) {
	srv, _ := personaCapabilityServer(t)
	initializePersonaOverHTTP(t, srv)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, personaRequest(http.MethodPost, "/v2/persona/reviews", "agent-secret", map[string]any{"kind": "mission", "base_revision": 0, "base_hash": "sha256:x", "proposed_markdown": "x", "reason": "r"}))
	if rec.Code == http.StatusCreated {
		t.Fatalf("mission review created: %s", rec.Body.String())
	}
}
