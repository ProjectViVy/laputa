package mentle_test

import (
	"context"
	"encoding/json"
	"net/url"
	"reflect"
	"strconv"
	"testing"

	mentlebackend "github.com/ProjectViVy/laputa/garden/backends/mentle"
	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/mentle/facade"
)

func sourceMutation(scope evolution.Scope) memory.AuthorizedMutation {
	return memory.AuthorizedMutation{
		Scope: scope, DestinationID: "mentle", OperationID: "source-create", PayloadDigest: "source-create-digest",
		Operation: evolution.MutationCreate, RecordID: "requested-record", ExpectedAbsent: true,
		Body: "observed from both records", Inference: evolution.InferenceObserved,
		Sources: []evolution.SourceRef{
			{SourceID: "garden.ingest", RecordID: "record/one?#%", Revision: 7, Scope: scope},
			{SourceID: "garden.ingest", RecordID: "record-two", Revision: 9, Scope: scope},
		},
	}
}

func assertCanonicalProvenance(t *testing.T, svc *facade.Service, receipt memory.MutationReceipt, want memory.AuthorizedMutation) {
	t.Helper()
	record, err := svc.GetMemory(context.Background(), receipt.TargetRef)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(record.Metadata["evolution_sources"])
	if err != nil {
		t.Fatal(err)
	}
	var refs []evolution.SourceRef
	if err := json.Unmarshal(encoded, &refs); err != nil || !reflect.DeepEqual(refs, want.Sources) {
		t.Fatalf("canonical source references lost: got=%s want=%+v err=%v", encoded, want.Sources, err)
	}
	first := want.Sources[0]
	uri, err := url.Parse(record.Source.URI)
	if err != nil || record.Source.Type != first.SourceID || record.Source.Revision != strconv.FormatUint(first.Revision, 10) || uri.Query().Get("source_id") != first.SourceID || uri.Query().Get("record_id") != first.RecordID || uri.Query().Get("scope") != memory.EncodeScope(first.Scope) {
		t.Fatalf("canonical primary provenance not mapped: %+v err=%v", record.Source, err)
	}
	if record.Metadata["inference"] != string(want.Inference) {
		t.Fatalf("canonical inference stale: %+v", record.Metadata)
	}
}

func TestMutationSourcesSurviveCanonicalReopen(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	svc, err := facade.OpenCatalogService(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { svc.Close() })
	scope := evolution.Scope{SubjectID: "subject-a", Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-a"}
	backend, err := mentlebackend.New(svc, scope, "mentle", admitted(scope.SubjectID))
	if err != nil {
		t.Fatal(err)
	}
	mutation := sourceMutation(scope)
	receipt, err := backend.Mutate(ctx, mutation)
	if err != nil {
		t.Fatal(err)
	}
	assertCanonicalProvenance(t, svc, receipt, mutation)
	svc.Close()
	svc, err = facade.OpenCatalogService(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	assertCanonicalProvenance(t, svc, receipt, mutation)
}

func TestMutationUpdateReplacesCanonicalProvenance(t *testing.T) {
	ctx := context.Background()
	svc := mentleService(t)
	scope := evolution.Scope{SubjectID: "subject-a", Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-a"}
	backend, err := mentlebackend.New(svc, scope, "mentle", admitted(scope.SubjectID))
	if err != nil {
		t.Fatal(err)
	}
	mutation := sourceMutation(scope)
	created, err := backend.Mutate(ctx, mutation)
	if err != nil {
		t.Fatal(err)
	}
	mutation.Operation, mutation.OperationID, mutation.PayloadDigest = evolution.MutationUpdate, "source-update", "source-update-digest"
	mutation.RecordID, mutation.ExpectedRevision, mutation.ExpectedAbsent = created.TargetRef, created.Revision, false
	mutation.Body, mutation.Inference = "corrected from newer evidence", evolution.InferenceInferred
	mutation.Sources = []evolution.SourceRef{{SourceID: "garden.ingest", RecordID: "new-record", Revision: 11, Scope: scope}}
	updated, err := backend.Mutate(ctx, mutation)
	if err != nil || updated.Revision != 2 {
		t.Fatalf("update: %+v %v", updated, err)
	}
	assertCanonicalProvenance(t, svc, updated, mutation)
	mutation.OperationID, mutation.PayloadDigest = "source-stale", "source-stale-digest"
	mutation.Sources[0].RecordID = "stale-source"
	if _, err := backend.Mutate(ctx, mutation); evolution.CodeOf(err) != evolution.ErrRevisionConflict {
		t.Fatalf("stale correction accepted: %v", err)
	}
	mutation.Sources[0].RecordID = "new-record"
	assertCanonicalProvenance(t, svc, updated, mutation)
}

func TestMutationRejectsSourcesOutsideReadUnion(t *testing.T) {
	for _, foreign := range []evolution.Scope{
		{SubjectID: "subject-b", Kind: evolution.ScopePersonal},
		{SubjectID: "subject-a", Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-b"},
	} {
		t.Run(memory.EncodeScope(foreign), func(t *testing.T) {
			svc := mentleService(t)
			scope := evolution.Scope{SubjectID: "subject-a", Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-a"}
			backend, err := mentlebackend.New(svc, scope, "mentle", admitted(scope.SubjectID))
			if err != nil {
				t.Fatal(err)
			}
			mutation := sourceMutation(scope)
			mutation.Sources[0].Scope = foreign
			receipt, err := backend.Mutate(context.Background(), mutation)
			if evolution.CodeOf(err) != evolution.ErrAuthorityDenied || receipt.OperationID != "" {
				t.Fatalf("foreign source admitted: %+v %v", receipt, err)
			}
		})
	}
}

func TestMutationAllowsPersonalSourceInWorkspaceReadUnion(t *testing.T) {
	svc := mentleService(t)
	scope := evolution.Scope{SubjectID: "subject-a", Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-a"}
	backend, err := mentlebackend.New(svc, scope, "mentle", admitted(scope.SubjectID))
	if err != nil {
		t.Fatal(err)
	}
	mutation := sourceMutation(scope)
	mutation.Sources[0].Scope = evolution.Scope{SubjectID: scope.SubjectID, Kind: evolution.ScopePersonal}
	receipt, err := backend.Mutate(context.Background(), mutation)
	if err != nil {
		t.Fatal(err)
	}
	assertCanonicalProvenance(t, svc, receipt, mutation)
}
