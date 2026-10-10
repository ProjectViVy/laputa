package agentapi

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ProjectViVy/laputa/laputa/persona"
)

func TestClientDiscardAndListFrozenSessions(t *testing.T) {
	root := t.TempDir()
	cfg := Config{PersonaDir: filepath.Join(root, "persona"), PalacePath: filepath.Join(root, "palace"), ModelsDir: filepath.Join(root, "models"), StateDB: filepath.Join(root, "state.db"), ProfileID: "default", AgentID: "agent-1", Platform: "vivy", Principal: PrincipalUser}
	client, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	authority, err := persona.Open(cfg.PersonaDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authority.Initialize(persona.Initialization{Identity: "i", Relationship: "r", Redline: "red", User: "u", World: "w"}, "user", persona.SourceInit, "init"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	bound, err := client.BindSession("session-orphan")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bound.Bootstrap(ctx, BootstrapRequest{Intent: "hello", BudgetChars: 512}); err != nil {
		t.Fatal(err)
	}
	ids, err := client.ListFrozenSessions(ctx, time.Now().Add(time.Hour))
	if err != nil || len(ids) != 1 || ids[0] != "session-orphan" {
		t.Fatalf("listed = %v %v", ids, err)
	}
	if err := client.DiscardFrozenSession(ctx, "session-orphan"); err != nil {
		t.Fatal(err)
	}
	if err := client.DiscardFrozenSession(ctx, "session-orphan"); err != nil {
		t.Fatalf("idempotent discard = %v", err)
	}
	ids, err = client.ListFrozenSessions(ctx, time.Now().Add(time.Hour))
	if err != nil || len(ids) != 0 {
		t.Fatalf("after discard = %v %v", ids, err)
	}
	if err := client.DiscardFrozenSession(ctx, ""); err == nil {
		t.Fatal("empty session_id accepted")
	}
}
