package agentapi

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dashimaki/laputa/persona"
)

func clientConfig(t *testing.T) Config {
	t.Helper()
	root := t.TempDir()
	return Config{PersonaDir: filepath.Join(root, "persona"), PalacePath: filepath.Join(root, "palace"), ModelsDir: filepath.Join(root, "models"), StateDB: filepath.Join(root, "state.db"), ProfileID: "default", AgentID: "agent-1", Platform: "vivy", Principal: PrincipalAgent}
}

func TestOpenTrustedHostBootstrapAndCapture(t *testing.T) {
	cfg := clientConfig(t)
	client, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	// Initializing Persona is an operator setup step, deliberately not exposed by Client.
	authority, err := persona.Open(cfg.PersonaDir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = authority.Initialize(persona.Initialization{Identity: "safe identity", Relationship: "relation", Redline: "redline", User: "user", World: "secret world"}, "user", persona.SourceInit, "init")
	if err != nil {
		t.Fatal(err)
	}
	bound, err := client.BindSession("session-1")
	if err != nil {
		t.Fatal(err)
	}
	view, err := bound.Bootstrap(context.Background(), BootstrapRequest{Intent: "hello", BudgetChars: 512})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(view.Context, "safe identity") || strings.Contains(view.Context, "secret world") || view.FrozenCore.SessionID != "session-1" {
		t.Fatalf("unexpected context: %+v", view)
	}
	content := "terminal result"
	digest := sha256.Sum256([]byte(content))
	req := CaptureRequest{Phase: CaptureCompleted, Content: content, ContentHash: fmt.Sprintf("sha256:%x", digest), Provenance: CaptureProvenance{RunID: "run-1", EventSeq: 1}}
	first, err := bound.Capture(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	again, err := bound.Capture(context.Background(), req)
	if err != nil || first.IngestionID != again.IngestionID {
		t.Fatalf("replay: %+v %+v %v", first, again, err)
	}
	status, err := bound.CaptureStatus(context.Background(), first.IngestionID, first.EventID)
	if err != nil || status.IngestionID != first.IngestionID {
		t.Fatalf("status: %+v %v", status, err)
	}
}

func TestClientBindingCannotSelfGrantOrSwitchProfile(t *testing.T) {
	cfg := clientConfig(t)
	cfg.Principal = PrincipalRead
	c, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	bound, err := c.BindSession("session-1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = bound.Capture(context.Background(), CaptureRequest{Phase: CaptureCompleted})
	assertCode(t, err, "principal_forbidden")
	other := binding()
	other.ProfileID = "another"
	_, err = bound.Bootstrap(context.Background(), BootstrapRequest{Binding: other})
	assertCode(t, err, "invalid_binding")
	_, err = bound.Bootstrap(context.Background(), BootstrapRequest{SessionID: "another"})
	assertCode(t, err, "invalid_binding")
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = bound.Bootstrap(context.Background(), BootstrapRequest{})
	assertCode(t, err, "unavailable")
	_, err = c.BindSession("session-2")
	assertCode(t, err, "unavailable")
}

func TestClientBindSessionUsesConfiguredIdentity(t *testing.T) {
	cfg := clientConfig(t)
	c, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	bound, err := c.BindSession("session-2")
	if err != nil {
		t.Fatal(err)
	}
	want := Binding{ProfileID: cfg.ProfileID, AgentID: cfg.AgentID, Platform: cfg.Platform, SessionID: "session-2"}
	if bound.binding != want {
		t.Fatalf("trusted identity not fixed: got %+v, want %+v", bound.binding, want)
	}
	for _, supplied := range []Binding{
		{ProfileID: "other", AgentID: cfg.AgentID, Platform: cfg.Platform, SessionID: "session-2"},
		{ProfileID: cfg.ProfileID, AgentID: "other", Platform: cfg.Platform, SessionID: "session-2"},
		{ProfileID: cfg.ProfileID, AgentID: cfg.AgentID, Platform: "other", SessionID: "session-2"},
		{ProfileID: cfg.ProfileID, AgentID: cfg.AgentID, Platform: cfg.Platform, SessionID: "other"},
	} {
		_, err := bound.Bootstrap(context.Background(), BootstrapRequest{Binding: supplied})
		assertCode(t, err, "invalid_binding")
	}
}

func TestClientBindSessionRejectsMissingSession(t *testing.T) {
	cfg := clientConfig(t)
	c, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	for _, session := range []string{"", " \t"} {
		bound, err := c.BindSession(session)
		if bound != nil {
			t.Fatalf("bound empty session %q", session)
		}
		assertCode(t, err, "invalid_binding")
	}
}

func TestClientOpenRejectsMissingTrustedIdentity(t *testing.T) {
	for _, field := range []string{"agent", "platform"} {
		t.Run(field, func(t *testing.T) {
			cfg := clientConfig(t)
			if field == "agent" {
				cfg.AgentID = " "
			} else {
				cfg.Platform = " "
			}
			client, err := Open(context.Background(), cfg)
			if client != nil {
				client.Close()
				t.Fatal("opened with missing trusted identity")
			}
			assertCode(t, err, "invalid_binding")
		})
	}
}

func TestClientOpenRejectsUntrustedOrImplicitConfiguration(t *testing.T) {
	cfg := clientConfig(t)
	cfg.Principal = ""
	c, err := Open(context.Background(), cfg)
	if c != nil {
		t.Fatal("unauthenticated client opened")
	}
	assertCode(t, err, "authentication_required")
	cfg = clientConfig(t)
	cfg.StateDB = "relative.db"
	c, err = Open(context.Background(), cfg)
	if c != nil || err == nil {
		t.Fatalf("implicit path opened: %v", err)
	}
}
