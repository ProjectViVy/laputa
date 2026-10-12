package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ProjectViVy/laputa/garden/agentapi"
)

func installationServer(t *testing.T) (*Server, string) {
	t.Helper()
	home := t.TempDir()
	i, err := agentapi.OpenInstallation(context.Background(), agentapi.InstallationConfig{Home: home, StateDB: filepath.Join(home, "garden.db"), PalacePath: filepath.Join(home, "memory"), ModelsDir: filepath.Join(home, "models")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = i.Close() })
	for _, id := range []string{"A", "B"} {
		if _, err = i.RegisterProfile(context.Background(), agentapi.ProfileRegistration{ProfileID: id, Create: true}); err != nil {
			t.Fatal(err)
		}
	}
	p, code, err := NewPairing()
	if err != nil {
		t.Fatal(err)
	}
	return &Server{Installation: i, Pairing: p, Capabilities: CapabilityConfig{AgentToken: "old-role-token"}}, code
}

func pairOwner(t *testing.T, s *Server, code string) *http.Cookie {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"code": code})
	req := httptest.NewRequest("POST", "http://localhost/v2/pairing", bytes.NewReader(body))
	req.Header.Set("Origin", "http://localhost")
	w := httptest.NewRecorder()
	s.HTTPHandler().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal("pairing", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatal("unprotected owner cookie", cookies)
	}
	return cookies[0]
}

func TestLoopbackAndRoleTokenDenied(t *testing.T) {
	s, _ := installationServer(t)
	for _, token := range []string{"", "old-role-token", "A"} {
		req := httptest.NewRequest("GET", "http://localhost/v2/profiles", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		s.HTTPHandler().ServeHTTP(w, req)
		if w.Code != 401 {
			t.Fatalf("%q granted profiles: %d %s", token, w.Code, w.Body.String())
		}
	}
}

func TestPairingOneUseAndOrigin(t *testing.T) {
	s, code := installationServer(t)
	cookie := pairOwner(t, s, code)
	get := httptest.NewRequest("GET", "http://localhost/v2/profiles", nil)
	get.AddCookie(cookie)
	w := httptest.NewRecorder()
	s.HTTPHandler().ServeHTTP(w, get)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, origin := range []string{"http://foreign", ""} {
		r := httptest.NewRequest("POST", "http://localhost/v2/shared-spaces", bytes.NewBufferString(`{"name":"S"}`))
		r.AddCookie(cookie)
		r.Header.Set("Origin", origin)
		w = httptest.NewRecorder()
		s.HTTPHandler().ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal("cross-origin owner mutation", w.Code)
		}
	}
	r := httptest.NewRequest("POST", "http://localhost/v2/pairing", bytes.NewBufferString(`{"code":"`+code+`"}`))
	r.Header.Set("Origin", "http://localhost")
	w = httptest.NewRecorder()
	s.HTTPHandler().ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("pair code reused", w.Code)
	}
}

func TestHostCannotManageParticipants(t *testing.T) {
	s, _ := installationServer(t)
	_, token, err := s.Installation.CreateHostBinding(context.Background(), agentapi.HostBinding{ProfileID: "A", AgentID: "host", Platform: "test", Controls: agentapi.HostControls{Recall: true}, Operations: []agentapi.Operation{agentapi.OpSearch}})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "http://localhost/v2/shared-spaces", bytes.NewBufferString(`{"name":"S"}`))
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.HTTPHandler().ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("host got human management", w.Code, w.Body.String())
	}
}
