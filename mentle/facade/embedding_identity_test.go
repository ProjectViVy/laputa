package facade

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ProjectViVy/laputa/mentle/internal/embedder"
	"github.com/ProjectViVy/laputa/mentle/internal/hybrid"
	"github.com/ProjectViVy/laputa/mentle/storage/govector"
)

func TestEmbedderIdentityFields(t *testing.T) {
	emb := &embedder.Embedder{}

	id := emb.Identity()
	if id.Dimension != 384 {
		t.Errorf("expected dimension 384, got %d", id.Dimension)
	}
	if id.Metric != "cosine" {
		t.Errorf("expected metric cosine, got %s", id.Metric)
	}
	if !id.Normalize {
		t.Errorf("expected normalize true, got false")
	}
	if id.Provider == "" {
		t.Errorf("expected non-empty provider")
	}
	if id.Version == "" {
		t.Errorf("expected non-empty version")
	}
	if id.CapturedAt.IsZero() {
		t.Errorf("expected non-zero captured_at")
	}
	if emb.Dimension() != 384 {
		t.Errorf("expected Dimension() to return 384, got %d", emb.Dimension())
	}
}

func TestCatalogSaveAndGetIdentity(t *testing.T) {
	cat, err := OpenCatalog(":memory:")
	if err != nil {
		t.Fatalf("failed to open catalog: %v", err)
	}
	defer cat.Close()

	id := embedder.Identity{
		Model:      "test-model",
		Dimension:  384,
		Metric:     "cosine",
		Normalize:  true,
		Provider:   "go-onnx",
		Version:    "1",
		CapturedAt: time.Now().UTC(),
	}

	if err := cat.SaveEmbeddingIdentity(id); err != nil {
		t.Fatalf("failed to save identity: %v", err)
	}

	saved, err := cat.GetEmbeddingIdentity()
	if err != nil {
		t.Fatalf("failed to get identity: %v", err)
	}
	if saved == nil {
		t.Fatalf("expected identity, got nil")
	}

	if saved.Model != id.Model {
		t.Errorf("expected model %q, got %q", id.Model, saved.Model)
	}
	if saved.Dimension != id.Dimension {
		t.Errorf("expected dimension %d, got %d", id.Dimension, saved.Dimension)
	}
	if saved.Metric != id.Metric {
		t.Errorf("expected metric %q, got %q", id.Metric, saved.Metric)
	}
	if saved.Normalize != id.Normalize {
		t.Errorf("expected normalize %v, got %v", id.Normalize, saved.Normalize)
	}
	if saved.Provider != id.Provider {
		t.Errorf("expected provider %q, got %q", id.Provider, saved.Provider)
	}
	if saved.Version != id.Version {
		t.Errorf("expected version %q, got %q", id.Version, saved.Version)
	}
}

func TestCatalogIdentityNotFound(t *testing.T) {
	cat, err := OpenCatalog(":memory:")
	if err != nil {
		t.Fatalf("failed to open catalog: %v", err)
	}
	defer cat.Close()

	saved, err := cat.GetEmbeddingIdentity()
	if err != nil {
		t.Fatalf("failed to get identity: %v", err)
	}
	if saved != nil {
		t.Fatalf("expected nil identity, got %v", saved)
	}
}

func TestEnqueueReindexJob(t *testing.T) {
	cat, err := OpenCatalog(":memory:")
	if err != nil {
		t.Fatalf("failed to open catalog: %v", err)
	}
	defer cat.Close()

	if err := cat.EnqueueReindexJob("test", 384, 768, "old-model", "new-model"); err != nil {
		t.Fatalf("failed to enqueue reindex job: %v", err)
	}

	count, err := cat.PendingReindexJobs()
	if err != nil {
		t.Fatalf("failed to get pending reindex jobs: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 pending job, got %d", count)
	}
}

func TestDimensionMismatchRejection(t *testing.T) {
	cat, err := OpenCatalog(":memory:")
	if err != nil {
		t.Fatalf("failed to open catalog: %v", err)
	}
	defer cat.Close()

	id := embedder.Identity{
		Model:      "test-model-large",
		Dimension:  768, // old dimension
		Metric:     "cosine",
		Normalize:  true,
		Provider:   "go-onnx",
		Version:    "1",
		CapturedAt: time.Now().UTC(),
	}
	if err := cat.SaveEmbeddingIdentity(id); err != nil {
		t.Fatalf("failed to save identity: %v", err)
	}

	// Service with an empty embedder (dimension 384)
	emb := &embedder.Embedder{}

	store := &fakeStore{}
	svc := &Service{
		Catalog:  cat,
		Embedder: emb,                                            // dimension 384
		Hybrid:   hybrid.NewSearcher(store, fakeEmbedder{}, 0.7), // avoid nil pointer error
	}

	_, err = svc.CreateMemory(context.Background(), CreateMemoryRequest{
		Content: "test content",
	}, "", "")

	if err != ErrEmbeddingDimensionMismatch {
		t.Fatalf("expected ErrEmbeddingDimensionMismatch, got: %v", err)
	}

	count, err := cat.PendingReindexJobs()
	if err != nil || count != 1 {
		t.Fatalf("expected 1 pending reindex job, got count=%d, err=%v", count, err)
	}
}

func TestEveryEmbeddingIdentityFieldMismatchIsExplicit(t *testing.T) {
	base := embedder.Identity{Model: "model-a", Dimension: 384, Metric: "cosine", Normalize: true, Provider: "provider-a", Version: "v1", CapturedAt: time.Now().UTC()}
	cases := []struct {
		name   string
		mutate func(*embedder.Identity)
		want   error
	}{
		{name: "dimension", mutate: func(id *embedder.Identity) { id.Dimension = 768 }, want: ErrEmbeddingDimensionMismatch},
		{name: "metric", mutate: func(id *embedder.Identity) { id.Metric = "dot" }, want: ErrEmbeddingMetricMismatch},
		{name: "normalize", mutate: func(id *embedder.Identity) { id.Normalize = false }, want: ErrEmbeddingIdentityMismatch},
		{name: "model", mutate: func(id *embedder.Identity) { id.Model = "model-b" }, want: ErrEmbeddingIdentityMismatch},
		{name: "provider", mutate: func(id *embedder.Identity) { id.Provider = "provider-b" }, want: ErrEmbeddingIdentityMismatch},
		{name: "version", mutate: func(id *embedder.Identity) { id.Version = "v2" }, want: ErrEmbeddingIdentityMismatch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			catalog, err := OpenCatalog(filepath.Join(t.TempDir(), "canonical.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer catalog.Close()
			if err := catalog.SaveEmbeddingIdentity(base); err != nil {
				t.Fatal(err)
			}
			current := base
			tc.mutate(&current)
			store := &fakeStore{points: map[string]govector.SearchResult{}}
			svc := &Service{Catalog: catalog, Hybrid: hybrid.NewSearcher(store, fakeEmbedder{}, .7), EmbeddingIdentity: func() embedder.Identity { return current }}
			_, err = svc.CreateMemory(context.Background(), CreateMemoryRequest{Content: "identity guarded"}, "", "")
			if !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want=%v", err, tc.want)
			}
		})
	}
}
