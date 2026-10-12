package mentle_test

import (
	"context"
	mentlebackend "github.com/ProjectViVy/laputa/garden/backends/mentle"
	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/mentle/facade"
	"testing"
)

func TestScopedManagementMetadataAndCapabilities(t *testing.T) {
	svc := mentleService(t)
	ctx := context.Background()
	a := evolution.Scope{SubjectID: "A", Kind: evolution.ScopePersonal}
	b := evolution.Scope{SubjectID: "B", Kind: evolution.ScopePersonal}
	for _, scope := range []evolution.Scope{a, b} {
		_, err := svc.CreateMemory(ctx, facade.CreateMemoryRequest{Content: "body", Kind: "note", Scope: memory.EncodeScope(scope), Metadata: map[string]any{"collection": scope.SubjectID + "-collection"}}, "op-"+scope.SubjectID, "d-"+scope.SubjectID)
		if err != nil {
			t.Fatal(err)
		}
	}
	adapter, err := mentlebackend.New(svc, a, "mentle", []evolution.Scope{a})
	if err != nil {
		t.Fatal(err)
	}
	if adapter.Capabilities().Vector {
		t.Fatal("catalog-only service advertised vector")
	}
	page, err := adapter.List(ctx, memory.ListRequest{Scopes: []evolution.Scope{a}, Limit: 20})
	if err != nil || len(page.Items) != 1 || !page.Items[0].Scope.Equal(a) {
		t.Fatal(page, err)
	}
	collections, err := adapter.Collections(ctx, []evolution.Scope{a})
	if err != nil || len(collections) != 1 || collections[0].ID != "A-collection" {
		t.Fatal(collections, err)
	}
	if _, err = adapter.List(ctx, memory.ListRequest{Scopes: []evolution.Scope{b}, Limit: 20}); err == nil {
		t.Fatal("foreign list admitted")
	}
}
