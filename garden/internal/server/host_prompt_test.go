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

func TestHostPromptBindingAndRetryHTTP(t *testing.T) {
	s, code := installationServer(t)
	cookie := pairOwner(t, s, code)
	ctx := context.Background()
	a, token, err := s.Installation.CreateHostBinding(ctx, agentapi.HostBinding{ProfileID: "A", AgentID: "hostA", Platform: "openclaw"})
	if err != nil {
		t.Fatal(err)
	}
	b, bToken, err := s.Installation.CreateHostBinding(ctx, agentapi.HostBinding{ProfileID: "B", AgentID: "hostB", Platform: "openclaw"})
	if err != nil {
		t.Fatal(err)
	}
	root := "/v2/host-bindings/" + a.BindingID + "/projection"
	call := func(method, path, profile, session, bearer, origin string, owner bool, body any) *httptest.ResponseRecorder {
		var raw bytes.Buffer
		if body != nil {
			if err := json.NewEncoder(&raw).Encode(body); err != nil {
				t.Fatal(err)
			}
		}
		r := httptest.NewRequest(method, "http://localhost"+path, &raw)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Garden-Profile", profile)
		r.Header.Set("X-Garden-Session", session)
		r.Header.Set("Origin", origin)
		if bearer != "" {
			r.Header.Set("Authorization", "Bearer "+bearer)
		}
		if owner {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		s.HTTPHandler().ServeHTTP(w, r)
		return w
	}
	cfg := agentapi.NativeBindingConfig{AdapterID: "openclaw", AdapterVersion: "garden-native-prompt/v1", HostVersion: "2026.10.2", HostInstance: "existing-test-host", ScopeFingerprint: strings.Repeat("a", 64)}
	req := agentapi.NativePromptRequest{AdapterVersion: cfg.AdapterVersion, HostVersion: cfg.HostVersion, HostInstance: cfg.HostInstance, ScopeFingerprint: cfg.ScopeFingerprint, SessionID: "actual-session"}
	if w := call("PUT", root+"/native-config", "A", "", "", "http://foreign", true, cfg); w.Code != 403 {
		t.Fatal("foreign owner origin", w.Code, w.Body.String())
	}
	if w := call("PUT", root+"/native-config", "A", "", token, "http://localhost", false, cfg); w.Code != 403 {
		t.Fatal("host configured itself", w.Code, w.Body.String())
	}
	if w := call("PUT", root+"/native-config", "A", "", "", "http://localhost", true, cfg); w.Code != 200 {
		t.Fatal("configure", w.Code, w.Body.String())
	}
	previewResponse := call("POST", root+"/retry-preview", "A", "", "", "http://localhost", true, map[string]any{})
	var preview agentapi.RetryPreview
	if previewResponse.Code != 200 || json.Unmarshal(previewResponse.Body.Bytes(), &preview) != nil {
		t.Fatal("preview while off", previewResponse.Code, previewResponse.Body.String())
	}
	if w := call("POST", root+"/prompt", "A", req.SessionID, "", "http://localhost", true, req); w.Code != 401 {
		t.Fatal("owner masquerade", w.Code, w.Body.String())
	}
	if err = s.Installation.SetHostControls(ctx, a.BindingID, agentapi.HostControls{PersonaSync: true}); err != nil {
		t.Fatal(err)
	}
	if w := call("POST", root+"/retry-confirm", "A", "", "", "http://localhost", true, map[string]string{"preview_id": preview.ID}); w.Code != 200 {
		t.Fatal("confirm", w.Code, w.Body.String())
	}
	for _, input := range []struct{ profile, session, token string }{{"A", req.SessionID, bToken}, {"B", req.SessionID, token}, {"A", "different-session", token}} {
		w := call("POST", root+"/prompt", input.profile, input.session, input.token, "", false, req)
		if w.Code != 403 && w.Code != 400 {
			t.Fatal("binding/session bypass", b.BindingID, w.Code, w.Body.String())
		}
	}
	w := call("POST", root+"/prompt", "A", req.SessionID, token, "", false, req)
	var projection agentapi.PromptProjection
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &projection) != nil || len(projection.SourceRevisions) != 4 || strings.Contains(w.Body.String(), "B_IDENTITY") || projection.BindingID != a.BindingID {
		t.Fatal("bound prompt", w.Code, w.Body.String())
	}
	var fields map[string]any
	json.Unmarshal(w.Body.Bytes(), &fields)
	if len(fields) != 4 {
		t.Fatal("unexpected prompt fields", fields)
	}
	malicious := map[string]any{"adapter_version": req.AdapterVersion, "host_version": req.HostVersion, "host_instance": req.HostInstance, "session_id": req.SessionID, "scope_fingerprint": req.ScopeFingerprint, "profile_id": "B"}
	if w = call("POST", root+"/prompt", "A", req.SessionID, token, "", false, malicious); w.Code != 400 {
		t.Fatal("body selects persona", w.Code, w.Body.String())
	}
	g, err := s.Owner.Subject(ctx, "A")
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := g.Persona.GetDocument(persona.KindIdentity)
	lease, err := g.Persona.AcquireEdit(persona.KindIdentity, doc.Revision, doc.ContentHash, "http-edit")
	if err != nil {
		t.Fatal(err)
	}
	if w = call("POST", root+"/prompt", "A", req.SessionID, token, "", false, req); w.Code != 409 || !strings.Contains(w.Body.String(), "persona_edit_locked") || strings.Contains(w.Body.String(), "markdown") {
		t.Fatal("locked contract", w.Code, w.Body.String())
	}
	if err = g.Persona.ReleaseEdit(persona.KindIdentity, persona.EditProof{Token: lease.Token, Generation: lease.Generation}); err != nil {
		t.Fatal(err)
	}
	previewResponse = call("POST", root+"/retry-preview", "A", "", "", "http://localhost", true, map[string]any{})
	json.Unmarshal(previewResponse.Body.Bytes(), &preview)
	if err = s.Installation.SetHostControls(ctx, a.BindingID, agentapi.HostControls{}); err != nil {
		t.Fatal(err)
	}
	if err = s.Installation.SetHostControls(ctx, a.BindingID, agentapi.HostControls{PersonaSync: true}); err != nil {
		t.Fatal(err)
	}
	if w = call("POST", root+"/retry-confirm", "A", "", "", "http://localhost", true, map[string]string{"preview_id": preview.ID}); w.Code != 409 || !strings.Contains(w.Body.String(), "stale_preview") {
		t.Fatal("public controls bypass invalidation", w.Code, w.Body.String())
	}
	if err = s.Installation.RevokeHostBinding(ctx, a.BindingID); err != nil {
		t.Fatal(err)
	}
	if w = call("POST", root+"/prompt", "A", req.SessionID, token, "", false, req); w.Code != 401 {
		t.Fatal("revoked prompt", w.Code, w.Body.String())
	}
}
