package registry

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ProjectViVy/laputa/laputa/evolution"
)

func accessFixture(t *testing.T) (*Store, map[string]HostBinding, map[string]string) {
	t.Helper()
	s, _ := profileStore(t)
	ctx := context.Background()
	bindings := map[string]HostBinding{}
	tokens := map[string]string{}
	for _, id := range []string{"A", "B", "C"} {
		if _, err := s.RegisterProfile(ctx, ProfileRegistration{ProfileID: id, Create: true}); err != nil {
			t.Fatal(err)
		}
		b, token, err := s.CreateHostBinding(ctx, HostBinding{ProfileID: id, AgentID: "same-agent", Platform: "test", WorkspaceID: "project", Controls: HostControls{Recall: true, Capture: true}, Operations: []string{"bootstrap", "search", "expand", "remember", "capture"}})
		if err != nil {
			t.Fatal(err)
		}
		if err = s.SetHostControls(ctx, b.BindingID, HostControls{Recall: true, Capture: true}); err != nil {
			t.Fatal(err)
		}
		bindings[id] = b
		tokens[id] = token
	}
	return s, bindings, tokens
}

func TestBindingCredentialAndRevocation(t *testing.T) {
	s, bindings, tokens := accessFixture(t)
	ctx := context.Background()
	for _, id := range []string{"A", "B"} {
		b, err := s.Authenticate(ctx, tokens[id])
		if err != nil || b.ProfileID != id {
			t.Fatal(b, err)
		}
		raw, _ := json.Marshal(b)
		if strings.Contains(string(raw), tokens[id]) {
			t.Fatal("secret returned in metadata")
		}
	}
	if _, err := s.Authenticate(ctx, "agent-role-token"); err == nil {
		t.Fatal("role token authenticated")
	}
	if err := s.RevokeHostBinding(ctx, bindings["B"].BindingID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, tokens["B"]); err == nil {
		t.Fatal("revoked credential reusable")
	}
	if _, err := s.Resolve(ctx, AccessRequest{BindingID: bindings["B"].BindingID, SessionID: "chat", Operation: "search"}); err == nil {
		t.Fatal("old handle still resolves")
	}
}

func TestSharedRightsAndRemoval(t *testing.T) {
	s, bindings, _ := accessFixture(t)
	ctx := context.Background()
	space, err := s.CreateSharedSpace(ctx, "项目 S")
	if err != nil {
		t.Fatal(err)
	}
	for id, access := range map[string]string{"A": "read_write", "B": "read"} {
		if err = s.SetSharedMember(ctx, space.ID, id, access); err != nil {
			t.Fatal(err)
		}
	}
	req := AccessRequest{BindingID: bindings["B"].BindingID, SessionID: "chat", Operation: "expand", SharedSpaceID: space.ID}
	before, err := s.Resolve(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	shared := evolution.Scope{SubjectID: "shared:" + space.ID, Kind: evolution.ScopePersonal}
	if !hasScope(before.ReadScopes, shared) {
		t.Fatal("shared resource missing", before)
	}
	req.Operation = "remember"
	if _, err = s.Resolve(ctx, req); err == nil {
		t.Fatal("read-only member wrote")
	}
	req.BindingID = bindings["C"].BindingID
	req.Operation = "expand"
	if _, err = s.Resolve(ctx, req); err == nil {
		t.Fatal("C read ungranted S")
	}
	if err = s.RemoveSharedMember(ctx, space.ID, "B"); err != nil {
		t.Fatal(err)
	}
	req.BindingID = bindings["B"].BindingID
	if _, err = s.Resolve(ctx, req); err == nil {
		t.Fatal("removed membership admitted")
	}
	req.SharedSpaceID = ""
	after, err := s.Resolve(ctx, req)
	if err != nil || hasScope(after.ReadScopes, shared) || after.MembershipKey == before.MembershipKey {
		t.Fatal("stale read view", after, err)
	}
}

func TestNoCrossPrivateGrant(t *testing.T) {
	s, b, _ := accessFixture(t)
	ctx := context.Background()
	a, err := s.Resolve(ctx, AccessRequest{BindingID: b["A"].BindingID, SessionID: "chat", Operation: "search"})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.ReadScopes) != 2 {
		t.Fatal(a)
	}
	for _, scope := range a.ReadScopes {
		if scope.SubjectID != "A" {
			t.Fatal("workspace granted foreign private scope", scope)
		}
	}
	space, _ := s.CreateSharedSpace(ctx, "S")
	if err = s.SetSharedMember(ctx, space.ID, "B", "admin"); err == nil {
		t.Fatal("arbitrary permission accepted")
	}
	if err = s.SetSharedMember(ctx, "private:A", "B", "read"); err == nil {
		t.Fatal("private scope grant accepted")
	}
}

func TestHostControlsIndependent(t *testing.T) {
	s, bindings, _ := accessFixture(t)
	ctx := context.Background()
	id := bindings["A"].BindingID
	for _, c := range []HostControls{{}, {PersonaSync: true}, {Recall: true}, {Capture: true}, {Recall: true, Capture: true}} {
		if err := s.SetHostControls(ctx, id, c); err != nil {
			t.Fatal(err)
		}
		for op, allowed := range map[string]bool{"search": c.Recall, "capture": c.Capture} {
			_, err := s.Resolve(ctx, AccessRequest{BindingID: id, SessionID: "chat", Operation: op})
			if (err == nil) != allowed {
				t.Fatalf("controls %+v op %s: %v", c, op, err)
			}
		}
	}
}

func TestWorkerHasNoAuthority(t *testing.T) {
	s, _, _ := accessFixture(t)
	ctx := context.Background()
	b, _, err := s.CreateHostBinding(ctx, HostBinding{ProfileID: "A", AgentID: "worker", Platform: "test", Worker: true, Controls: HostControls{Recall: true}, Operations: []string{"bootstrap", "search", "actmem_read", "persona_get", "expand"}, SelectedCards: []string{"selected"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetHostControls(ctx, b.BindingID, HostControls{Recall: true}); err != nil {
		t.Fatal(err)
	}
	for _, op := range []string{"bootstrap", "search", "actmem_read", "persona_get"} {
		if _, err = s.Resolve(ctx, AccessRequest{BindingID: b.BindingID, SessionID: "job", Operation: op}); err == nil {
			t.Fatalf("worker inherited %s", op)
		}
	}
	a, err := s.Resolve(ctx, AccessRequest{BindingID: b.BindingID, SessionID: "job", Operation: "expand"})
	if err != nil || len(a.SelectedCards) != 1 || a.SelectedCards[0] != "selected" {
		t.Fatal(a, err)
	}
}

func hasScope(scopes []evolution.Scope, want evolution.Scope) bool {
	for _, s := range scopes {
		if s.Equal(want) {
			return true
		}
	}
	return false
}

func TestHostBindingStartsInactive(t *testing.T) {
	s, _ := profileStore(t)
	ctx := context.Background()
	_, _ = s.RegisterProfile(ctx, ProfileRegistration{ProfileID: "A", Create: true})
	b, _, err := s.CreateHostBinding(ctx, HostBinding{ProfileID: "A", AgentID: "host", Platform: "test", Controls: HostControls{PersonaSync: true, Recall: true, Capture: true}})
	if err != nil || b.Controls != (HostControls{}) {
		t.Fatal("created host activated without separate approval", b, err)
	}
	if _, err = s.Resolve(ctx, AccessRequest{BindingID: b.BindingID, SessionID: "chat", Operation: "search"}); err == nil {
		t.Fatal("inactive host can recall")
	}
}
