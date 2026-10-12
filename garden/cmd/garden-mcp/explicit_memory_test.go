package main

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProjectViVy/laputa/garden/agentapi"
	"github.com/ProjectViVy/laputa/garden/internal/runtimecore"
	"github.com/ProjectViVy/laputa/garden/internal/server"
	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/persona"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestCLIExplicitMemoryIndependentOfPersona(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	owner, err := runtimecore.OpenOwner(ctx, runtimecore.OwnerConfig{Home: home, Config: runtimecore.Config{StateDB: filepath.Join(home, "garden.db"), PalacePath: filepath.Join(home, "memory"), ModelsDir: filepath.Join(home, "models"), CatalogOnly: true}})
	if err != nil {
		t.Fatal(err)
	}
	installation := agentapi.NewInstallation(owner)
	defer installation.Close()
	for _, id := range []string{"A", "B"} {
		if _, err := installation.RegisterProfile(ctx, agentapi.ProfileRegistration{ProfileID: id, Create: true}); err != nil {
			t.Fatal(err)
		}
		g, err := owner.Subject(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := g.Persona.Initialize(persona.Initialization{Identity: id, Relationship: "r", Redline: "safe", User: "u", World: "w"}, "owner", persona.SourceInit, "fixture"); err != nil {
			t.Fatal(err)
		}
	}
	space, err := installation.CreateSharedSpace(ctx, "selected CLI facts")
	if err != nil {
		t.Fatal(err)
	}
	for id, access := range map[string]string{"A": "read_write", "B": "read"} {
		if err := installation.SetSharedMember(ctx, space.ID, id, access); err != nil {
			t.Fatal(err)
		}
	}
	backend := httptest.NewServer((&server.Server{Owner: owner, Installation: installation}).HTTPHandler())
	defer backend.Close()
	clients := map[string]*Client{}
	bindings := map[string]string{}
	for _, id := range []string{"A", "B"} {
		binding, credential, err := installation.CreateHostBinding(ctx, agentapi.HostBinding{ProfileID: id, AgentID: "main", Platform: "codex-cli"})
		if err != nil {
			t.Fatal(err)
		}
		bindings[id] = binding.BindingID
		if err := installation.SetHostControls(ctx, binding.BindingID, agentapi.HostControls{PersonaSync: false, Recall: true, Capture: true}); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "credential")
		if err := os.WriteFile(path, []byte(credential), 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GARDEN_HOST_CREDENTIAL_FILE", path)
		t.Setenv("GARDEN_HOST_SESSION_ID", "existing-session")
		clients[id] = NewClient(backend.URL)
	}
	call := func(id, tool string, args map[string]any) *mcp.CallToolResult {
		t.Helper()
		req := mcp.CallToolRequest{}
		req.Params.Arguments = args
		result, err := newMCPServer(clients[id]).GetTool(tool).Handler(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	selected := map[string]any{"content": "needle selected fact", "event_id": "explicit-selection-1", "shared_space_id": space.ID}
	first := call("A", toolRecordSignal, selected)
	if first.IsError {
		t.Fatalf("selected write with Persona off: %+v", first)
	}
	replay := call("A", toolRecordSignal, selected)
	if replay.IsError || strings.Fields(first.Content[0].(mcp.TextContent).Text)[1] != strings.Fields(replay.Content[0].(mcp.TextContent).Text)[1] {
		t.Fatalf("selected event lost original canonical record: first=%+v replay=%+v", first, replay)
	}
	originalReceipt := first.StructuredContent.(map[string]any)["receipt"].(memory.MutationReceipt)
	replayedReceipt := replay.StructuredContent.(map[string]any)["receipt"].(memory.MutationReceipt)
	if originalReceipt.OperationID == "" || replayedReceipt.OperationID != originalReceipt.OperationID || replayedReceipt.TargetRef != originalReceipt.TargetRef || replayedReceipt.Revision != originalReceipt.Revision {
		t.Fatal("selection receipt identity/revision changed")
	}
	changed := call("A", toolRecordSignal, map[string]any{"content": "changed selected fact", "event_id": "explicit-selection-1", "shared_space_id": space.ID})
	if !changed.IsError {
		t.Fatal("same selected event accepted different content")
	}
	search := call("B", toolMemorySearch, map[string]any{"query": "needle"})
	if search.IsError || !strings.Contains(search.Content[0].(mcp.TextContent).Text, "selected fact") {
		t.Fatalf("read member lost permitted recall: %+v", search)
	}
	denied := call("B", toolRecordSignal, selected)
	if !denied.IsError {
		t.Fatal("read-only shared member wrote selected fact")
	}
	if err := installation.RevokeHostBinding(ctx, bindings["A"]); err != nil {
		t.Fatal(err)
	}
	if result := call("A", toolMemorySearch, map[string]any{"query": "needle"}); !result.IsError {
		t.Fatal("revoked explicit recall admitted")
	}
	if result := call("A", toolRecordSignal, selected); !result.IsError {
		t.Fatal("revoked selected write admitted")
	}
}
