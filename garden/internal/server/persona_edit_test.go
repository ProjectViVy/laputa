package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ProjectViVy/laputa/garden/agentapi"
	"github.com/ProjectViVy/laputa/laputa/persona"
)

func TestPersonaEditHTTP(t *testing.T) {
	s, code := installationServer(t)
	cookie := pairOwner(t, s, code)
	call := func(method, path, profile, origin, token string, owner bool, body any) *httptest.ResponseRecorder {
		var raw bytes.Buffer
		if body != nil {
			if err := json.NewEncoder(&raw).Encode(body); err != nil {
				t.Fatal(err)
			}
		}
		r := httptest.NewRequest(method, "http://localhost"+path, &raw)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Garden-Profile", profile)
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if owner {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		s.HTTPHandler().ServeHTTP(w, r)
		return w
	}
	root := "/v2/persona/documents/identity"
	g, err := s.Owner.Subject(context.Background(), "A")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := g.Persona.GetDocument(persona.KindIdentity)
	if err != nil {
		t.Fatal(err)
	}
	acquire := map[string]any{"base_revision": doc.Revision, "base_hash": doc.ContentHash, "request_id": "tab-a"}
	for _, origin := range []string{"", "http://foreign"} {
		w := call("POST", root+"/edit-lease", "A", origin, "", true, acquire)
		if w.Code != 401 {
			t.Fatal("owner origin bypass", w.Code, w.Body.String())
		}
	}
	host, token, err := s.Installation.CreateHostBinding(context.Background(), agentapi.HostBinding{ProfileID: "A", AgentID: "host", Platform: "test", Operations: []agentapi.Operation{agentapi.OpPersonaGet, agentapi.OpPersonaReview}})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Installation.SetHostControls(context.Background(), host.BindingID, agentapi.HostControls{PersonaSync: true}); err != nil {
		t.Fatal(err)
	}
	w := call("POST", root+"/edit-lease", "A", "http://localhost", token, false, acquire)
	if w.Code != 403 {
		t.Fatal("agent acquired owner proof", w.Code, w.Body.String())
	}
	w = call("POST", root+"/edit-lease", "A", "http://localhost", "", true, acquire)
	if w.Code != 200 {
		t.Fatal("acquire", w.Code, w.Body.String())
	}
	var lease persona.EditLease
	if err = json.Unmarshal(w.Body.Bytes(), &lease); err != nil {
		t.Fatal(err)
	}
	proof := persona.EditProof{Token: lease.Token, Generation: lease.Generation}
	get := call("GET", root, "A", "", "", true, nil)
	if get.Code != 200 || strings.Contains(get.Body.String(), lease.Token) || !strings.Contains(get.Body.String(), `"edit_status"`) {
		t.Fatal("owner status leaks/misses lease", get.Code, get.Body.String())
	}
	w = call("DELETE", root+"/edit-lease", "B", "http://localhost", "", true, proof)
	if w.Code != 409 {
		t.Fatal("A proof released B", w.Code, w.Body.String())
	}
	w = call("PATCH", root+"/edit-lease", "A", "http://localhost", "", true, proof)
	if w.Code != 200 {
		t.Fatal("renew", w.Code, w.Body.String())
	}
	body := map[string]any{"base_revision": doc.Revision, "base_hash": doc.ContentHash, "edit_proof": proof, "content": "owner changed", "scope": "document", "reason": "editor"}
	w = call("PUT", root, "A", "http://localhost", token, false, body)
	if w.Code != 403 {
		t.Fatal("agent selected editor path", w.Code, w.Body.String())
	}
	w = call("PUT", root, "A", "http://localhost", "", true, body)
	if w.Code != 200 {
		t.Fatal("save", w.Code, w.Body.String())
	}
	w = call("PUT", root, "A", "http://localhost", "", true, body)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "persona_edit_expired") {
		t.Fatal("consumed proof fell back", w.Code, w.Body.String())
	}
	latest, err := g.Persona.GetDocument(persona.KindIdentity)
	if err != nil || latest.Revision != 2 {
		t.Fatal(latest, err)
	}
	body["base_revision"] = latest.Revision
	body["base_hash"] = latest.ContentHash
	body["edit_proof"] = nil
	w = call("PUT", root, "A", "http://localhost", "", true, body)
	if w.Code != 409 {
		t.Fatal("explicit null proof fell back", w.Code, w.Body.String())
	}
}
