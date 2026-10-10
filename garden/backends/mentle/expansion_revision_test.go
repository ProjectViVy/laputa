package mentle_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	mentlebackend "github.com/dashimaki/garden/backends/mentle"
	"github.com/dashimaki/garden/memory"
	"github.com/dashimaki/laputa/evolution"
)

func TestConcurrentExpansionKeepsOneCanonicalRevision(t *testing.T) {
	ctx := context.Background()
	svc := mentleService(t)
	scope := evolution.Scope{SubjectID: "subject-a", Kind: evolution.ScopePersonal}
	backend, err := mentlebackend.New(svc, scope, "dest-1", admitted("subject-a"))
	if err != nil {
		t.Fatal(err)
	}
	created, err := backend.Mutate(ctx, memory.AuthorizedMutation{Scope: scope, DestinationID: "dest-1", OperationID: "create", PayloadDigest: "create", Operation: evolution.MutationCreate, RecordID: "requested", ExpectedAbsent: true, Body: "body-version-1", Inference: evolution.InferenceObserved})
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		for version := 2; version <= 200; version++ {
			_, err := backend.Mutate(ctx, memory.AuthorizedMutation{Scope: scope, DestinationID: "dest-1", OperationID: fmt.Sprintf("update-%d", version), PayloadDigest: fmt.Sprintf("digest-%d", version), Operation: evolution.MutationUpdate, RecordID: created.TargetRef, ExpectedRevision: uint64(version - 1), Body: fmt.Sprintf("body-version-%d", version), Inference: evolution.InferenceObserved})
			if err != nil {
				t.Error(err)
				return
			}
		}
	}()
	for repeat := 0; repeat < 1500; repeat++ {
		record, err := svc.GetMemory(ctx, created.TargetRef)
		if err != nil {
			t.Fatal(err)
		}
		page, err := backend.Expand(ctx, memory.AuthorizedExpansion{Scopes: []evolution.Scope{scope}, CardID: created.TargetRef, ExpectedRevision: uint64(record.Version), BudgetChars: 1200})
		if evolution.CodeOf(err) == evolution.ErrRevisionConflict {
			continue
		}
		if err != nil {
			t.Error(err)
			break
		}
		for _, fragment := range page.Items {
			if fragment.Revision != uint64(record.Version) || fragment.Excerpt != record.Content || fragment.MaterialRef != fmt.Sprintf("mem://%s@v%d", record.ID, record.Version) {
				t.Errorf("canonical revision mixed: checked=%d/%q, evidence=%+v", record.Version, record.Content, fragment)
				workers.Wait()
				return
			}
		}
	}
	workers.Wait()
}
