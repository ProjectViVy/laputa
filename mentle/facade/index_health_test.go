package facade

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ProjectViVy/laputa/mentle/internal/embedder"
	"github.com/ProjectViVy/laputa/mentle/internal/hybrid"
	"github.com/ProjectViVy/laputa/mentle/internal/search"
	"github.com/ProjectViVy/laputa/mentle/storage/govector"
)

func healthTestIdentity() embedder.Identity {
	return embedder.Identity{
		Model:      "health-test-model",
		Dimension:  2,
		Metric:     "cosine",
		Normalize:  true,
		Provider:   "health-test",
		Version:    "v1",
		CapturedAt: time.Unix(0, 0).UTC(),
	}
}

func healthTestService(t *testing.T, store *fakeStore, identity func() embedder.Identity) *Service {
	t.Helper()
	catalog, err := OpenCatalog(filepath.Join(t.TempDir(), "canonical.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = catalog.Close() })
	emb := fakeEmbedder{}
	return &Service{
		Searcher:          search.NewSearcher(store, emb),
		Hybrid:            hybrid.NewSearcher(store, emb, .7),
		Catalog:           catalog,
		EmbeddingIdentity: identity,
	}
}

func hasHealthReason(health IndexHealth, want string) bool {
	for _, reason := range health.Reasons {
		if reason == want {
			return true
		}
	}
	return false
}

func TestIndexHealthReportsMissingEmbeddingIdentityWithoutHidingCanonicalState(t *testing.T) {
	svc := healthTestService(t, &fakeStore{points: map[string]govector.SearchResult{}}, nil)
	health, err := svc.IndexHealth(context.Background())
	if err != nil {
		t.Fatalf("health error=%v", err)
	}
	if health.Status != "degraded" || !hasHealthReason(health, "embedding_identity_missing") {
		t.Fatalf("health=%+v", health)
	}
	if health.CanonicalActiveCount != 0 || health.VectorPhysicalCount != 0 || health.BM25Count != 0 {
		t.Fatalf("unexpected counts=%+v", health)
	}
}

func TestIndexHealthReportsDerivedDivergenceAndTombstonePressure(t *testing.T) {
	store := &fakeStore{points: map[string]govector.SearchResult{}}
	svc := healthTestService(t, store, healthTestIdentity)
	created, err := svc.CreateMemory(context.Background(), CreateMemoryRequest{Content: "canonical health record", Kind: "fact"}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	currentID := created.ID + "@v1"
	if err := store.Delete(currentID); err != nil {
		t.Fatal(err)
	}
	orphanID := "orphan@v1"
	if err := store.Add(orphanID, []float32{1, 0}, map[string]any{"content": "orphan derived point"}); err != nil {
		t.Fatal(err)
	}
	svc.Hybrid.IndexBM25(orphanID, "orphan derived point", map[string]any{"content": "orphan derived point"})

	health, err := svc.IndexHealth(context.Background())
	if err != nil {
		t.Fatalf("health error=%v", err)
	}
	for _, reason := range []string{"vector_count_diverged", "bm25_count_diverged", "tombstone_pressure"} {
		if !hasHealthReason(health, reason) {
			t.Fatalf("health=%+v missing reason %q", health, reason)
		}
	}
	if health.Status != "degraded" || health.CanonicalActiveCount != 1 || health.VectorActiveCount != 0 || health.VectorPhysicalCount != 1 || health.BM25Count != 2 {
		t.Fatalf("health=%+v", health)
	}
}

func TestIndexHealthReportsPendingAndFailedOutboxJobs(t *testing.T) {
	svc := healthTestService(t, &fakeStore{points: map[string]govector.SearchResult{}}, healthTestIdentity)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := svc.Catalog.db.Exec(`
		INSERT INTO index_jobs(job_id,memory_id,canonical_version,operation,content,metadata_json,state,attempts,last_error,next_attempt_at,lease_owner,lease_until,created_at,updated_at)
		VALUES('job-pending','missing-pending',1,'upsert','pending','{}','pending',0,'','', '', '', ?, ?),
		      ('job-failed','missing-failed',1,'upsert','failed','{}','failed',5,'poison','', '', '', ?, ?)`, now, now, now, now)
	if err != nil {
		t.Fatal(err)
	}

	health, err := svc.IndexHealth(context.Background())
	if err != nil {
		t.Fatalf("health error=%v", err)
	}
	if health.PendingJobs != 1 || health.FailedJobs != 1 || !hasHealthReason(health, "index_jobs_pending") || !hasHealthReason(health, "index_jobs_failed") {
		t.Fatalf("health=%+v", health)
	}
	if strings.Join(health.Reasons, ",") != "embedding_identity_missing,index_jobs_failed,index_jobs_pending" {
		// The exact reason order is part of the DTO contract; this guard also
		// catches accidental nondeterministic map iteration.
		t.Fatalf("unstable or unexpected reason order: %v", health.Reasons)
	}
}

func TestIndexHealthUnavailableOnlyWhenCanonicalProbeFails(t *testing.T) {
	svc := &Service{}
	health, err := svc.IndexHealth(context.Background())
	if !errors.Is(err, ErrIndexHealthUnavailable) || health.Status != "unavailable" || !hasHealthReason(health, "canonical_probe_failed") {
		t.Fatalf("health=%+v err=%v", health, err)
	}
}
