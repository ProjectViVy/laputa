package mentle_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
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

func TestSearchCursorPagesAndBindsToAdapterHost(t *testing.T) {
	svc := mentleService(t)
	ctx := context.Background()
	scope := evolution.Scope{SubjectID: "subject-a", Kind: evolution.ScopePersonal}
	for i, color := range []string{"red", "green", "blue"} {
		_, err := svc.CreateMemory(ctx, facade.CreateMemoryRequest{
			Content: "hostboundcursorprobe " + color,
			Kind:    "note",
			Scope:   memory.EncodeScope(scope),
		}, fmt.Sprintf("host-cursor-key-%d", i), fmt.Sprintf("host-cursor-hash-%d", i))
		if err != nil {
			t.Fatal(err)
		}
	}
	backendA, err := mentlebackend.New(svc, scope, "dest-1", []evolution.Scope{scope})
	if err != nil {
		t.Fatal(err)
	}
	query := memory.AuthorizedSearch{Scopes: []evolution.Scope{scope}, Query: "hostboundcursorprobe", Limit: 1}
	first, err := backendA.Search(ctx, query)
	if err != nil || len(first.Items) != 1 || first.NextCursor == "" {
		t.Fatalf("first page: %+v %v", first, err)
	}
	query.Cursor = first.NextCursor
	second, err := backendA.Search(ctx, query)
	if err != nil || len(second.Items) != 1 || second.Items[0].ID == first.Items[0].ID {
		t.Fatalf("second page: %+v %v", second, err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(first.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"offset"`) || strings.Contains(string(raw), `"inner"`) {
		t.Fatalf("cursor exposes its encrypted payload: %q", raw)
	}
	tampered := append([]byte(nil), raw...)
	tampered[len(tampered)-1] ^= 1
	query.Cursor = base64.RawURLEncoding.EncodeToString(tampered)
	if _, err := backendA.Search(ctx, query); evolution.CodeOf(err) != evolution.ErrInvalidScope {
		t.Fatalf("tampered encrypted continuation was accepted: %v", err)
	}
	query.Query = "different query"
	query.Cursor = first.NextCursor
	if _, err := backendA.Search(ctx, query); evolution.CodeOf(err) != evolution.ErrInvalidScope {
		t.Fatalf("cursor was accepted for a different query: %v", err)
	}
	query.Query = "hostboundcursorprobe"
	backendB, err := mentlebackend.New(svc, scope, "dest-1", []evolution.Scope{scope})
	if err != nil {
		t.Fatal(err)
	}
	query.Cursor = first.NextCursor
	if _, err := backendB.Search(ctx, query); evolution.CodeOf(err) != evolution.ErrInvalidScope {
		t.Fatalf("cursor minted by adapter A was accepted by adapter B: %v", err)
	}
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
