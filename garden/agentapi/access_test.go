package agentapi

import (
	"context"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/laputa/persona"
	"testing"
)

func TestBindingSpoofDenied(t *testing.T) {
	ctx := context.Background()
	i, err := OpenInstallation(ctx, installationConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	defer i.Close()
	installPersona(t, i, "A")
	installPersona(t, i, "B")
	b, _, err := i.CreateHostBinding(ctx, HostBinding{ProfileID: "A", AgentID: "host", Platform: "test", Controls: HostControls{Recall: true}, Operations: []Operation{OpBootstrap, OpSearch}})
	if err != nil {
		t.Fatal(err)
	}
	if err = i.SetHostControls(ctx, b.BindingID, HostControls{Recall: true}); err != nil {
		t.Fatal(err)
	}
	h, err := i.BindHostSession(b.BindingID, "chat-1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.Bootstrap(ctx, BootstrapRequest{Binding: Binding{ProfileID: "B", AgentID: "host", Platform: "test", SessionID: "chat-1"}})
	assertCode(t, err, "invalid_binding")
	if _, err = h.Bootstrap(ctx, BootstrapRequest{}); err != nil {
		t.Fatal(err)
	}
	if err = i.RevokeHostBinding(ctx, b.BindingID); err != nil {
		t.Fatal(err)
	}
	if _, err = h.Bootstrap(ctx, BootstrapRequest{}); err == nil {
		t.Fatal("old bound handle ignored revocation")
	}
}

func TestHostSessionCollision(t *testing.T) {
	ctx := context.Background()
	i, err := OpenInstallation(ctx, installationConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	defer i.Close()
	p := installPersona(t, i, "A")
	makeHost := func() *BoundClient {
		b, _, err := i.CreateHostBinding(ctx, HostBinding{ProfileID: "A", AgentID: "same", Platform: "test", Controls: HostControls{Recall: true}, Operations: []Operation{OpBootstrap}})
		if err != nil {
			t.Fatal(err)
		}
		if err = i.SetHostControls(ctx, b.BindingID, HostControls{Recall: true}); err != nil {
			t.Fatal(err)
		}
		h, err := i.BindHostSession(b.BindingID, "same-chat")
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	one, two := makeHost(), makeHost()
	if _, err = one.Bootstrap(ctx, BootstrapRequest{}); err != nil {
		t.Fatal(err)
	}
	svc, err := persona.Open(p.AuthorityDir)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := svc.GetDocument(persona.KindIdentity)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.SaveUserDocument(persona.KindIdentity, "new identity", doc.Revision, "owner", persona.SourceUserDirect, "update"); err != nil {
		t.Fatal(err)
	}
	v, err := two.Bootstrap(ctx, BootstrapRequest{})
	if err != nil || v.FrozenCore.Content(evolution.FrozenKindIdentity) != "new identity" {
		t.Fatalf("other host session inherited prior admission: %+v %v", v, err)
	}
	v, err = one.Bootstrap(ctx, BootstrapRequest{})
	if err != nil || v.FrozenCore.Content(evolution.FrozenKindIdentity) != "A identity" {
		t.Fatal("first host frozen authority changed", v, err)
	}
}
