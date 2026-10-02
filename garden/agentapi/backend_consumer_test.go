package agentapi

import (
	"context"
	"testing"

	"github.com/dashimaki/garden/internal/runtimecore"
	"github.com/dashimaki/garden/memory"
	"github.com/dashimaki/garden/memory/memorytest"
	"github.com/dashimaki/laputa/evolution"
)

// A clean consumer composes Garden with an injected backend and no Mentle
// dependency; reads resolve through the bound scope alone (Task 3).
func TestGardenWithInjectedBackendAndNoMentle(t *testing.T) {
	store := memorytest.NewFakeStore()
	scope := evolution.Scope{SubjectID: "default", Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-1"}
	g := &runtimecore.Garden{
		ProfileID: "default",
		Backends: map[string]memory.Backend{
			memory.EncodeScope(scope): memorytest.NewFakeBackend(store, scope, "fake-dest"),
		},
	}
	svc := NewService(g)
	b := Binding{ProfileID: "default", AgentID: "agent", Platform: "test", WorkspaceID: "ws-1"}

	if _, err := svc.SearchCards(context.Background(), PrincipalAgent, b, CardSearch{Query: "needle"}); err != nil {
		t.Fatal(err)
	}
}

// With no injected backend and no Mentle, memory reads are explicitly
// unavailable while the runtime stays usable (Task 1).
func TestGardenWithoutBackendReportsMemoryUnavailable(t *testing.T) {
	g := &runtimecore.Garden{ProfileID: "default"}
	svc := NewService(g)
	b := Binding{ProfileID: "default", AgentID: "agent", Platform: "test"}
	if _, err := svc.SearchCards(context.Background(), PrincipalAgent, b, CardSearch{Query: "needle"}); err == nil {
		t.Fatal("search succeeded with no backend")
	}
}
