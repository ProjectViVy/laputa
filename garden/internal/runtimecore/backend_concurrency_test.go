package runtimecore

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/mentle/facade"
)

// Real canonical/BM25 facade for cache/lifetime unit proof, not a model or
// complete App acceptance substitute. Each scope retains its own adapter.
func TestBackendForConcurrentScopeBindings(t *testing.T) {
	svc, err := facade.OpenCatalogService(context.Background(), filepath.Join(t.TempDir(), "palace"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	g := &Garden{ProfileID: "profile", Mentle: svc}
	start := make(chan struct{})
	errors := make(chan error, 32)
	var workers sync.WaitGroup
	for i := 0; i < 32; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			<-start
			scope := evolution.Scope{SubjectID: "profile", Kind: evolution.ScopeWorkspace, WorkspaceID: fmt.Sprintf("workspace-%d", i)}
			for repeat := 0; repeat < 16; repeat++ {
				backend, err := g.BackendFor(scope)
				if err != nil {
					errors <- err
					return
				}
				bound, ok := backend.(interface{ BoundScope() evolution.Scope })
				if !ok || !bound.BoundScope().Equal(scope) {
					errors <- fmt.Errorf("scope adapter crossed workspace-%d", i)
					return
				}
			}
		}(i)
	}
	close(start)
	workers.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
	if len(g.Backends) != 32 {
		t.Fatalf("scope cache=%d, want32", len(g.Backends))
	}
}
