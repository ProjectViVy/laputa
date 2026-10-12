package server

import (
	"context"
	"encoding/json"
	"github.com/ProjectViVy/laputa/garden/agentapi"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/mentle/facade"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestApprovedSliceCapsuleWorkspaceBoundary(t *testing.T) {
	s, code := installationServer(t)
	ownerCookie := pairOwner(t, s, code)
	ctx := context.Background()
	g, err := s.Owner.Subject(ctx, "A")
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.Actmem.AppendEntry(evolution.Entry{Section: evolution.SectionPulse, Scope: evolution.Scope{SubjectID: "A", Kind: evolution.ScopeWorkspace, WorkspaceID: "secret"}, SessionID: "private-session", Body: "OTHER_WORKSPACE_SECRET"})
	if err != nil {
		t.Fatal(err)
	}
	caps, err := g.Actmem.FoldSession("private-session")
	if err != nil || len(caps) == 0 {
		t.Fatal(caps, err)
	}
	b, token, err := s.Installation.CreateHostBinding(ctx, agentapi.HostBinding{ProfileID: "A", AgentID: "review", Platform: "test", WorkspaceID: "public"})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Installation.SetHostControls(ctx, b.BindingID, agentapi.HostControls{Recall: true}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/v2/actmem/capsules", "/v2/actmem/capsules/" + caps[0].Name} {
		r := httptest.NewRequest("GET", "http://localhost"+path, nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		s.HTTPHandler().ServeHTTP(w, r)
		if w.Code != 503 || strings.Contains(w.Body.String(), caps[0].Name) {
			t.Errorf("installed host archive not explicitly unavailable: %d %s", w.Code, w.Body.String())
		}
		ownerReq := httptest.NewRequest("GET", "http://localhost"+path, nil)
		ownerReq.AddCookie(ownerCookie)
		ownerReq.Header.Set("X-Garden-Profile", "A")
		w = httptest.NewRecorder()
		s.HTTPHandler().ServeHTTP(w, ownerReq)
		if w.Code != 200 || !strings.Contains(w.Body.String(), caps[0].Name) {
			t.Fatal("owner archive unavailable", w.Code, w.Body.String())
		}
	}
}

func TestApprovedSliceHTTPPreservesBoundOriginSession(t *testing.T) {
	s, _ := installationServer(t)
	ctx := context.Background()
	b, token, err := s.Installation.CreateHostBinding(ctx, agentapi.HostBinding{ProfileID: "A", AgentID: "review", Platform: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Installation.SetHostControls(ctx, b.BindingID, agentapi.HostControls{Capture: true}); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "http://localhost/v2/memories", strings.NewReader(`{"content":"authored fact"}`))
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Garden-Session", "real-session-42")
	w := httptest.NewRecorder()
	s.HTTPHandler().ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	g, err := s.Owner.Subject(ctx, "A")
	if err != nil {
		t.Fatal(err)
	}
	page, err := g.Mentle.ListMemories(ctx, facade.ListMemoryOptions{Limit: 10})
	if err != nil || len(page.Items) != 1 {
		t.Fatal(page, err)
	}
	raw, err := json.Marshal(page.Items[0].Metadata["origin"])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "real-session-42") {
		t.Fatalf("bound session replaced in canonical provenance: %s", raw)
	}
}

func TestApprovedSliceFoldCannotRemoveForeignWorkspace(t *testing.T) {
	s, _ := installationServer(t)
	ctx := context.Background()
	g, err := s.Owner.Subject(ctx, "A")
	if err != nil {
		t.Fatal(err)
	}
	scope := evolution.Scope{SubjectID: "A", Kind: evolution.ScopeWorkspace, WorkspaceID: "secret"}
	_, err = g.Actmem.AppendEntry(evolution.Entry{Section: evolution.SectionPulse, Scope: scope, SessionID: "shared-session-label", Body: "OTHER_WORKSPACE_SECRET"})
	if err != nil {
		t.Fatal(err)
	}
	b, token, err := s.Installation.CreateHostBinding(ctx, agentapi.HostBinding{ProfileID: "A", AgentID: "review", Platform: "test", WorkspaceID: "public", Operations: []agentapi.Operation{agentapi.OpActmemMaintain}})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Installation.SetHostControls(ctx, b.BindingID, agentapi.HostControls{Capture: true}); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "http://localhost/v2/actmem/maintenance", strings.NewReader(`{"operation":"fold_session","session_key":"shared-session-label"}`))
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.HTTPHandler().ServeHTTP(w, r)
	after, err := g.Actmem.ReadScoped(scope, evolution.ReadRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != 503 || len(after.Entries) != 1 {
		t.Fatalf("workspace-scoped maintain removed foreign head entries: %d %s", w.Code, w.Body.String())
	}
}
