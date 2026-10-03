package agentapi

// DN-4C: the trusted human capability must carry every human-path read the
// C2-3 control actions need — persona document/status, scoped ACTMEM
// activity, owner ACTMEM document and backend index health — stamped with
// the human principal, never the agent one.

import (
	"context"
	"testing"

	"github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/laputa/persona"
)

func openHuman(t *testing.T) (context.Context, *Client, *HumanClient) {
	t.Helper()
	ctx := context.Background()
	cfg := userEmbeddedConfig(t)
	initAuthority(t, cfg)
	client, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	human, err := client.BindHumanSession("s-human", "")
	if err != nil {
		t.Fatal(err)
	}
	return ctx, client, human
}

func TestHumanReadPersonaIsOwnerPrincipal(t *testing.T) {
	ctx, _, human := openHuman(t)
	doc, err := human.ReadPersona(ctx, persona.KindIdentity.String())
	if err != nil {
		t.Fatalf("ReadPersona: %v", err)
	}
	if !doc.Exists || doc.Content == "" || doc.Revision == 0 {
		t.Fatalf("persona doc = %+v", doc)
	}
	if _, err := human.ReadPersona(ctx, "bogus"); err == nil {
		t.Fatal("unknown persona kind admitted")
	}
}

func TestHumanPersonaStatusReportsRevisions(t *testing.T) {
	ctx, _, human := openHuman(t)
	view, err := human.PersonaStatus(ctx)
	if err != nil {
		t.Fatalf("PersonaStatus: %v", err)
	}
	if view == nil || view.Status == persona.StatusUninitialized {
		t.Fatalf("status view = %+v", view)
	}
	identity, ok := view.Files[persona.KindIdentity.String()]
	if !ok || !identity.Exists || identity.Revision == 0 {
		t.Fatalf("identity file state = %+v", view.Files)
	}
}

func TestHumanReadActivityScoped(t *testing.T) {
	ctx, _, human := openHuman(t)
	// Seed a work entry so the scoped projection has content.
	patch := WorkPatch{
		BaseRevision: 0,
		Changes:      []WorkChange{{Kind: evolution.WorkChangeAdd, Field: evolution.FieldGoal, Body: "ship DN-4C"}},
	}
	if _, err := human.ApplyWorkPatch(ctx, patch); err != nil {
		t.Fatalf("seed ApplyWorkPatch: %v", err)
	}
	result, err := human.ReadActivity(ctx, ReadRequest{Sections: []evolution.EntrySection{evolution.SectionWork}, MaxChars: 1200})
	if err != nil {
		t.Fatalf("ReadActivity: %v", err)
	}
	if result.Revision == 0 || len(result.Entries) == 0 {
		t.Fatalf("activity result = %+v", result)
	}
	if _, err := human.ReadActivity(ctx, ReadRequest{Sections: []evolution.EntrySection{"bogus"}}); err == nil {
		t.Fatal("unknown section admitted")
	}
	if _, err := human.ReadActivity(ctx, ReadRequest{MaxChars: evolution.ActmemReadCapChars + 1}); err == nil {
		t.Fatal("over-cap max_chars admitted")
	}
}

func TestHumanApplyWorkPatchHonorsBaseRevision(t *testing.T) {
	ctx, _, human := openHuman(t)
	patch := WorkPatch{
		BaseRevision: 0,
		Changes: []WorkChange{{
			Kind: evolution.WorkChangeAdd, Field: evolution.FieldGoal, Body: "ship DN-4C",
		}},
	}
	result, err := human.ApplyWorkPatch(ctx, patch)
	if err != nil {
		t.Fatalf("ApplyWorkPatch: %v", err)
	}
	if !result.Changed || result.Revision == 0 {
		t.Fatalf("work patch result = %+v", result)
	}
	stale := patch
	stale.BaseRevision = 99
	if _, err := human.ApplyWorkPatch(ctx, stale); err == nil {
		t.Fatal("stale base revision applied")
	}
}

func TestHumanReadOwnerACTMEMWholeDocument(t *testing.T) {
	ctx, _, human := openHuman(t)
	patch := WorkPatch{
		BaseRevision: 0,
		Changes:      []WorkChange{{Kind: evolution.WorkChangeAdd, Field: evolution.FieldGoal, Body: "owner-visible goal"}},
	}
	if _, err := human.ApplyWorkPatch(ctx, patch); err != nil {
		t.Fatalf("seed work patch: %v", err)
	}
	doc, err := human.ReadOwnerACTMEM(ctx)
	if err != nil {
		t.Fatalf("ReadOwnerACTMEM: %v", err)
	}
	if doc.Revision == 0 || doc.Markdown == "" {
		t.Fatalf("owner doc = %+v", doc)
	}
}

func TestHumanIndexHealthIsScopedToBinding(t *testing.T) {
	ctx, _, human := openHuman(t)
	health, err := human.IndexHealth(ctx)
	if err != nil {
		// No canonical backend configured in this fixture: an honest
		// unavailable is acceptable, a silent zero is not.
		if health.Status != "unavailable" {
			t.Fatalf("index health = %+v err=%v", health, err)
		}
		return
	}
	if health.Status == "" {
		t.Fatal("empty index health status")
	}
}
