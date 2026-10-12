package server

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/ProjectViVy/laputa/garden/agentapi"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHostProjectionControlAndAuthenticatedLoadIngress(t *testing.T) {
	s, code := installationServer(t)
	ctx := context.Background()
	cookie := pairOwner(t, s, code)
	binding, token, err := s.Installation.CreateHostBinding(ctx, agentapi.HostBinding{ProfileID: "A", AgentID: "file-host", Platform: "test"})
	if err != nil {
		t.Fatal(err)
	}
	other, otherToken, err := s.Installation.CreateHostBinding(ctx, agentapi.HostBinding{ProfileID: "B", AgentID: "file-host", Platform: "test"})
	if err != nil {
		t.Fatal(err)
	}
	_ = other
	if err = s.Installation.SetHostControls(ctx, binding.BindingID, agentapi.HostControls{PersonaSync: true}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "CLAUDE.md")
	request := func(method, route string, body any, credential string, owner bool) *httptest.ResponseRecorder {
		var raw bytes.Buffer
		if body != nil {
			if err := json.NewEncoder(&raw).Encode(body); err != nil {
				t.Fatal(err)
			}
		}
		r := httptest.NewRequest(method, "http://localhost"+route, &raw)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "http://localhost")
		r.Header.Set("X-Garden-Session", "actual-host-session")
		if credential != "" {
			r.Header.Set("Authorization", "Bearer "+credential)
		}
		if owner {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		s.HTTPHandler().ServeHTTP(w, r)
		return w
	}
	route := "/v2/host-bindings/" + binding.BindingID + "/projection"
	preview := request("POST", route+"/preview", map[string]any{"path": path}, "", true)
	if preview.Code != 200 {
		t.Fatal(preview.Code, preview.Body.String())
	}
	var plan agentapi.HostFilePlan
	if err = json.Unmarshal(preview.Body.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		token string
		owner bool
		want  int
	}{{token, false, 403}, {"", false, 401}} {
		w := request("PUT", route, plan, row.token, row.owner)
		if w.Code != row.want {
			t.Fatal("host file write authority", w.Code, w.Body.String())
		}
	}
	applied := request("PUT", route, plan, "", true)
	if applied.Code != 200 {
		t.Fatal("apply", applied.Code, applied.Body.String())
	}
	var status agentapi.HostStatus
	if err = json.Unmarshal(applied.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.State != "pending_reload" || status.LoadedHash != "" || status.WrittenHash == "" {
		t.Fatal(status)
	}
	for _, row := range []struct {
		credential string
		owner      bool
		want       int
	}{{"", true, 401}, {otherToken, false, 403}, {token, false, 503}} {
		w := request("POST", route+"/loaded", map[string]string{"hash": status.WrittenHash}, row.credential, row.owner)
		if w.Code != row.want {
			t.Fatal("load proof ingress", w.Code, w.Body.String())
		}
	}
	got := request("GET", route, nil, "", true)
	if got.Code != 200 {
		t.Fatal("status", got.Code, got.Body.String())
	}
	if err = json.Unmarshal(got.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.LoadedHash != "" || status.State != "unknown" {
		t.Fatal("unqualified proof loaded", status)
	}
	conflict := request("PUT", route, plan, "", true)
	if conflict.Code != 409 {
		t.Fatal("stable conflict envelope", conflict.Code, conflict.Body.String())
	}
	revoked := request("DELETE", "/v2/host-bindings/"+binding.BindingID, nil, "", true)
	if revoked.Code != 409 {
		t.Fatal("manual revoke", revoked.Code, revoked.Body.String())
	}
	var envelope struct {
		Details struct {
			HostStatus agentapi.HostStatus `json:"host_status"`
		} `json:"details"`
	}
	if err = json.Unmarshal(revoked.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Details.HostStatus.ManualAction != "remove_projection" || envelope.Details.HostStatus.ManualPath != path {
		t.Fatal("manual guidance absent", revoked.Body.String())
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != plan.ProposedContent {
		t.Fatal("revoke modified host file", err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	ack := request("DELETE", route, nil, "", true)
	if ack.Code != 200 {
		t.Fatal("cleanup acknowledgment", ack.Code, ack.Body.String())
	}
	if _, err = s.Installation.AuthenticateHost(ctx, token); err == nil {
		t.Fatal("revoked credential accepted")
	}
}
