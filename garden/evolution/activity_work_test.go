package evolution

import (
	"context"
	"strings"
	"testing"

	"github.com/dashimaki/laputa/actmem"
	laputaevolution "github.com/dashimaki/laputa/evolution"
)

func TestCollectReadsExistingScopedWork(t *testing.T) {
	store := actmem.New(t.TempDir())
	first, err := store.ApplyWorkPatch(testScope(), laputaevolution.WorkPatch{BaseRevision: 0, Changes: []laputaevolution.WorkChange{{Kind: laputaevolution.WorkChangeAdd, Field: laputaevolution.FieldGoal, Body: "existing scoped work must survive reconciliation"}}})
	if err != nil {
		t.Fatal(err)
	}
	foreign := laputaevolution.Scope{SubjectID: "foreign", Kind: laputaevolution.ScopePersonal}
	second, err := store.ApplyWorkPatch(foreign, laputaevolution.WorkPatch{BaseRevision: first.Revision, Changes: []laputaevolution.WorkChange{{Kind: laputaevolution.WorkChangeAdd, Field: laputaevolution.FieldGoal, Body: "foreign-work"}}})
	if err != nil {
		t.Fatal(err)
	}
	d := newDomain(t, Deps{Scope: testScope(), DestinationID: "dest_a", Actmem: store})
	batch, err := d.Collect(context.Background(), laputaevolution.Window{SourceID: "activity", Through: 0})
	if err != nil || batch.ActivityRevision != second.Revision || len(batch.Entries) != 1 || batch.Entries[0].ID != first.Entries[0].ID || batch.Entries[0].Body != first.Entries[0].Body || batch.Entries[0].Section != laputaevolution.SectionWork {
		t.Fatalf("MEM-S05-02: actual scoped Work missing from reconciliation input: %+v %v", batch, err)
	}
}

func TestCollectRejectsIncompleteWorkRead(t *testing.T) {
	store := actmem.New(t.TempDir())
	if _, err := store.ApplyWorkPatch(testScope(), laputaevolution.WorkPatch{BaseRevision: 0, Changes: []laputaevolution.WorkChange{{Kind: laputaevolution.WorkChangeAdd, Field: laputaevolution.FieldGoal, Body: strings.Repeat("x", 1201)}}}); err != nil {
		t.Fatal(err)
	}
	d := newDomain(t, Deps{Scope: testScope(), DestinationID: "dest_a", Actmem: store})
	batch, err := d.Collect(context.Background(), laputaevolution.Window{SourceID: "activity", Through: 0})
	if laputaevolution.CodeOf(err) != laputaevolution.ErrActmemCapExceeded || len(batch.Entries) != 0 {
		t.Fatalf("incomplete old Work reached inference: %+v %v", batch, err)
	}
}
