package agentapi

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/laputa/persona"
	"github.com/ProjectViVy/laputa/mentle/facade"
)

func installationConfig(t *testing.T) InstallationConfig {
	t.Helper()
	home := t.TempDir()
	return InstallationConfig{Home: home, StateDB: filepath.Join(home, "garden.db"), PalacePath: filepath.Join(home, "memory"), ModelsDir: filepath.Join(home, "models")}
}

func installPersona(t *testing.T, i *Installation, id string) Profile {
	t.Helper()
	p, err := i.RegisterProfile(context.Background(), ProfileRegistration{ProfileID: id, Create: true})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := persona.Open(p.AuthorityDir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Initialize(persona.Initialization{Identity: id + " identity", Relationship: "relation", Redline: "redline", User: "user", World: id + " private world"}, "owner", persona.SourceInit, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func installedClient(t *testing.T, i *Installation, id string) *Client {
	t.Helper()
	c, err := i.BindProfile(id, ClientConfig{AgentID: "host", Platform: "test", Principal: PrincipalAgent})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestInstallationSharesBackend(t *testing.T) {
	ctx := context.Background()
	cfg := installationConfig(t)
	cat, err := facade.OpenCatalog(filepath.Join(cfg.PalacePath, "canonical.sqlite3"))
	// Catalog needs an existing parent, as the actual supported attachment does.
	if err != nil {
		if e := os.MkdirAll(cfg.PalacePath, 0700); e != nil {
			t.Fatal(e)
		}
		cat, err = facade.OpenCatalog(filepath.Join(cfg.PalacePath, "canonical.sqlite3"))
	}
	if err != nil {
		t.Fatal(err)
	}
	_ = cat.Close()
	i, err := OpenInstallation(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer i.Close()
	installPersona(t, i, "A")
	installPersona(t, i, "B")
	a, b := installedClient(t, i, "A"), installedClient(t, i, "B")
	if a.runtime.Mentle == nil || a.runtime.Mentle != b.runtime.Mentle || a.runtime.Ingest != b.runtime.Ingest || a.runtime.Frozen != b.runtime.Frozen {
		t.Fatal("profiles duplicated installation resources")
	}
}

func TestSubjectSessionCollision(t *testing.T) {
	ctx := context.Background()
	cfg := installationConfig(t)
	i, err := OpenInstallation(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	pa := installPersona(t, i, "A")
	installPersona(t, i, "B")
	for _, id := range []string{"A", "B"} {
		c := installedClient(t, i, id)
		h, err := c.BindSession("chat-1")
		if err != nil {
			t.Fatal(err)
		}
		v, err := h.Bootstrap(ctx, BootstrapRequest{})
		if err != nil || v.FrozenCore.Content(evolution.FrozenKindIdentity) != id+" identity" || v.FrozenCore.SessionID != "chat-1" {
			t.Fatalf("%s snapshot: %+v %v", id, v, err)
		}
	}
	if err = i.Close(); err != nil {
		t.Fatal(err)
	}
	svc, err := persona.Open(pa.AuthorityDir)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := svc.GetDocument(persona.KindIdentity)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.SaveUserDocument(persona.KindIdentity, "A changed", doc.Revision, "owner", persona.SourceUserDirect, "change")
	if err != nil {
		t.Fatal(err)
	}
	i, err = OpenInstallation(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer i.Close()
	for _, id := range []string{"A", "B"} {
		h, _ := installedClient(t, i, id).BindSession("chat-1")
		v, err := h.Bootstrap(ctx, BootstrapRequest{})
		if err != nil || v.FrozenCore.Content(evolution.FrozenKindIdentity) != id+" identity" {
			t.Fatalf("restart replaced %s frozen authority: %+v %v", id, v, err)
		}
	}
}

func TestViewCloseDoesNotCloseOwner(t *testing.T) {
	ctx := context.Background()
	i, err := OpenInstallation(ctx, installationConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	defer i.Close()
	installPersona(t, i, "A")
	installPersona(t, i, "B")
	a, b := installedClient(t, i, "A"), installedClient(t, i, "B")
	ah, _ := a.BindSession("chat")
	bh, _ := b.BindSession("chat")
	if err = a.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = ah.Bootstrap(ctx, BootstrapRequest{}); err == nil {
		t.Fatal("closed client usable")
	}
	if _, err = bh.Bootstrap(ctx, BootstrapRequest{}); err != nil {
		t.Fatal("view closed shared owner", err)
	}
	if err = i.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = bh.Bootstrap(ctx, BootstrapRequest{}); err == nil {
		t.Fatal("closed installation usable")
	}
}

func TestPartialOwnerOpenReleasesLocks(t *testing.T) {
	ctx := context.Background()
	cfg := installationConfig(t)
	if err := os.Mkdir(cfg.StateDB, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenInstallation(ctx, cfg); err == nil {
		t.Fatal("invalid state path accepted")
	}
	if err := os.Remove(cfg.StateDB); err != nil {
		t.Fatal(err)
	}
	i, err := OpenInstallation(ctx, cfg)
	if err != nil {
		t.Fatal("partial startup retained ownership", err)
	}
	defer i.Close()
}

func TestInstallationFrozenCleanupBoundToProfile(t *testing.T) {
	ctx := context.Background()
	i, err := OpenInstallation(ctx, installationConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	defer i.Close()
	installPersona(t, i, "A")
	installPersona(t, i, "B")
	a, b := installedClient(t, i, "A"), installedClient(t, i, "B")
	for _, pair := range []struct {
		c       *Client
		session string
	}{{a, "a-session"}, {b, "b-session"}} {
		h, _ := pair.c.BindSession(pair.session)
		if _, err = h.Bootstrap(ctx, BootstrapRequest{}); err != nil {
			t.Fatal(err)
		}
	}
	ids, err := a.ListFrozenSessions(ctx, time.Now().Add(time.Second))
	if err != nil || len(ids) != 1 || ids[0] != "a-session" {
		t.Fatalf("foreign session metadata: %v %v", ids, err)
	}
	if err = a.DiscardFrozenSession(ctx, "a-session"); err != nil {
		t.Fatal(err)
	}
	ids, err = a.ListFrozenSessions(ctx, time.Now().Add(time.Second))
	if err != nil || len(ids) != 0 {
		t.Fatal("did not discard admitted key", ids, err)
	}
	ids, err = b.ListFrozenSessions(ctx, time.Now().Add(time.Second))
	if err != nil || len(ids) != 1 || ids[0] != "b-session" {
		t.Fatal("discard affected B", ids, err)
	}
}
