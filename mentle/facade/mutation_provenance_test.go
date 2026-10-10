package facade

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestMutateUpdateCommitsProvenanceWithCanonicalAndOutbox(t *testing.T) {
	ctx := context.Background()
	svc := mutationService(t)
	create := createReq("provenance-create", receiptScopePersonal, "create-digest")
	create.Sources = []MemorySource{{Type: "session", URI: "source://old", Revision: "4"}}
	create.Metadata = map[string]any{"title": "retained title", "inference": "observed"}
	created, err := svc.Mutate(ctx, create)
	if err != nil {
		t.Fatal(err)
	}
	newSource := MemorySource{Type: "session", URI: "source://new", Revision: "7"}
	// Disable only the disposable index projection so its transactionally
	// committed outbox remains observable. Canonical SQLite stays real.
	hybrid := svc.Hybrid
	svc.Hybrid = nil
	defer func() { svc.Hybrid = hybrid }()
	update := MutationRequest{
		OperationID: "provenance-update", PayloadDigest: "update-digest", Operation: "update",
		Scope: receiptScopePersonal, DestinationID: receiptDest, RecordID: created.RecordID,
		ExpectedRevision: 1, Body: "new body", Sources: []MemorySource{newSource},
		Metadata: map[string]any{"inference": "inferred"},
	}
	updated, err := svc.Mutate(ctx, update)
	if err != nil || updated.IndexStatus != "pending" {
		t.Fatalf("canonical update/outbox status: %+v %v", updated, err)
	}
	record, err := svc.GetMemory(ctx, created.RecordID)
	if err != nil || record.Version != 2 || record.Source != newSource || record.Metadata["inference"] != "inferred" || record.Metadata["title"] != "retained title" {
		t.Fatalf("new body retained stale provenance: %+v %v", record, err)
	}
	var raw string
	if err := svc.Catalog.db.QueryRowContext(ctx, `SELECT metadata_json FROM index_jobs WHERE memory_id=? AND canonical_version=2`, created.RecordID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var indexed map[string]any
	if err := json.Unmarshal([]byte(raw), &indexed); err != nil || !reflect.DeepEqual(indexed, record.Metadata) {
		t.Fatalf("outbox has different metadata: %s canonical=%+v err=%v", raw, record.Metadata, err)
	}
	update.OperationID, update.PayloadDigest = "provenance-stale", "stale-digest"
	update.Sources[0].URI = "source://stale"
	if _, err := svc.Mutate(ctx, update); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale version accepted: %v", err)
	}
	record, err = svc.GetMemory(ctx, updated.RecordID)
	if err != nil || record.Version != 2 || record.Source != newSource || record.Metadata["inference"] != "inferred" {
		t.Fatalf("stale update changed authority: %+v %v", record, err)
	}
}
