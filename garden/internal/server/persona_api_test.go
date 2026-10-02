package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dashimaki/laputa/persona"
)

func personaCapabilityServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := persona.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return &Server{
		Addr:    ":0",
		Persona: store,
		Capabilities: CapabilityConfig{
			UserToken:      "user-secret",
			AgentToken:     "agent-secret",
			AutodreamToken: "dream-secret",
			OperatorToken:  "operator-secret",
		},
	}, dir
}

func personaRequest(method, path, token string, body any) *http.Request {
	var data bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&data).Encode(body)
	}
	req := httptest.NewRequest(method, path, &data)
	req.RemoteAddr = "127.0.0.1:4000"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func initializePersonaOverHTTP(t *testing.T, srv *Server) {
	t.Helper()
	body := persona.Initialization{Identity: "identity", Relationship: "relationship", Redline: "redline", User: "user", World: "world"}
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, personaRequest(http.MethodPost, "/v2/persona/initialize", "user-secret", body))
	if rec.Code != http.StatusCreated {
		t.Fatalf("initialize status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPersonaCleanBreakMetadataAndExplicitRead(t *testing.T) {
	srv, _ := personaCapabilityServer(t)
	initializePersonaOverHTTP(t, srv)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, personaRequest(http.MethodGet, "/v2/persona/documents", "", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("documents status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Status    persona.Status      `json:"status"`
		Documents []persona.FileState `json:"documents"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != persona.StatusReady || len(payload.Documents) != 8 {
		t.Fatalf("payload=%+v", payload)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte(`"content"`)) {
		t.Fatal("metadata response must not contain document content")
	}
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, personaRequest(http.MethodGet, "/v2/persona/documents/identity", "", nil))
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(`"content":"identity"`)) {
		t.Fatalf("document status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPersonaCapabilityAndActorHeaderAreIndependent(t *testing.T) {
	srv, _ := personaCapabilityServer(t)
	initializePersonaOverHTTP(t, srv)
	rec := httptest.NewRecorder()
	req := personaRequest(http.MethodPut, "/v2/persona/documents/identity", "agent-secret", map[string]any{"base_revision": 1, "content": "agent overwrite", "reason": "spoof"})
	req.Header.Set("X-Garden-Actor", "user_request")
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("agent protected write status=%d body=%s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	// A forged actor label cannot make an anonymous write a user write.
	req = personaRequest(http.MethodPut, "/v2/persona/documents/identity", "", map[string]any{"base_revision": 1, "content": "anonymous overwrite", "reason": "spoof"})
	req.Header.Set("X-Garden-Actor", "user")
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous write status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPersonaReviewLifecycleUsesVerifiedAgentActor(t *testing.T) {
	srv, _ := personaCapabilityServer(t)
	initializePersonaOverHTTP(t, srv)
	rec := httptest.NewRecorder()
	body := map[string]any{"kind": "identity", "base_revision": 1, "base_hash": persona.ContentHash("identity"), "proposed_markdown": "reviewed identity", "reason": "agent proposal"}
	req := personaRequest(http.MethodPost, "/v2/persona/reviews", "agent-secret", body)
	req.Header.Set("X-Garden-Actor", "autodream")
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("review create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var review persona.ChangeRequest
	if err := json.NewDecoder(rec.Body).Decode(&review); err != nil {
		t.Fatal(err)
	}
	if review.Actor != persona.ActorAgent || review.State != persona.RequestPending {
		t.Fatalf("review=%+v", review)
	}
	rec = httptest.NewRecorder()
	decision := personaRequest(http.MethodPost, "/v2/persona/reviews/"+review.ID+"/approve", "agent-secret", map[string]any{})
	srv.HTTPHandler().ServeHTTP(rec, decision)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("agent approve status=%d body=%s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	decision = personaRequest(http.MethodPost, "/v2/persona/reviews/"+review.ID+"/approve", "user-secret", map[string]any{})
	srv.HTTPHandler().ServeHTTP(rec, decision)
	if rec.Code != http.StatusOK {
		t.Fatalf("user approve status=%d body=%s", rec.Code, rec.Body.String())
	}
	entries, err := srv.Persona.ListHistory(persona.KindIdentity)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Actor != "agent" || entries[0].Source != string(persona.SourceAgentP5Accepted) || entries[0].Reason != "agent proposal" {
		t.Fatalf("history=%+v", entries)
	}
}
