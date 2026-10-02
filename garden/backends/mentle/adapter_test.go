package mentle_test

import (
	"context"
	"testing"

	mentlebackend "github.com/dashimaki/garden/backends/mentle"
	"github.com/dashimaki/garden/memory"
	memory_test "github.com/dashimaki/garden/memory/memorytest"
	"github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/mentle/facade"
)

func mentleService(t *testing.T) *facade.Service {
	t.Helper()
	svc, err := facade.OpenCatalogService(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { svc.Close() })
	return svc
}

func admitted(subject string) []evolution.Scope {
	return []evolution.Scope{
		{SubjectID: subject, Kind: evolution.ScopePersonal},
		{SubjectID: subject, Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-a"},
	}
}

func TestMentleBackendConformance(t *testing.T) {
	svc := mentleService(t)
	factory := func(t testing.TB, bound evolution.Scope, dest string) memory.Backend {
		b, err := mentlebackend.New(svc, bound, dest, admitted(bound.SubjectID))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	memory_test.BackendConformance(t, factory)
}

// A restarted service keeps canonical scope/digest state: writes replay
// against stored receipts instead of silently re-creating.
func TestRestartPreservesReceipts(t *testing.T) {
	dir := t.TempDir()
	svc, err := facade.OpenCatalogService(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	scope := evolution.Scope{SubjectID: "subject-a", Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-a"}
	b1, err := mentlebackend.New(svc, scope, "dest-1", admitted("subject-a"))
	if err != nil {
		t.Fatal(err)
	}
	r1, err := b1.Mutate(context.Background(), memory.AuthorizedMutation{
		Scope: scope, DestinationID: "dest-1", OperationID: "op-1", PayloadDigest: "pd-1",
		Operation: evolution.MutationCreate, RecordID: "rec-1", ExpectedAbsent: true,
		Body: "alpha", Inference: evolution.InferenceObserved,
	})
	if err != nil {
		t.Fatal(err)
	}
	svc.Close()

	svc2, err := facade.OpenCatalogService(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer svc2.Close()
	b2, err := mentlebackend.New(svc2, scope, "dest-1", admitted("subject-a"))
	if err != nil {
		t.Fatal(err)
	}
	r2, err := b2.Mutate(context.Background(), memory.AuthorizedMutation{
		Scope: scope, DestinationID: "dest-1", OperationID: "op-1", PayloadDigest: "pd-1",
		Operation: evolution.MutationCreate, RecordID: "rec-1", ExpectedAbsent: true,
		Body: "alpha", Inference: evolution.InferenceObserved,
	})
	if err != nil {
		t.Fatal(err)
	}
	if r2.Revision != r1.Revision || r2.TargetRef != r1.TargetRef {
		t.Fatalf("restarted replay = %+v want %+v", r2, r1)
	}
}
