package facade

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ProjectViVy/laputa/mentle/internal/hybrid"
	"github.com/ProjectViVy/laputa/mentle/internal/search"
	"github.com/ProjectViVy/laputa/mentle/storage/govector"
)

type fakeStore struct {
	points    map[string]govector.SearchResult
	addErr    error
	deleteErr error
}

func TestCanonicalMemoryLifecycleAndIdempotency(t *testing.T) {
	store := &fakeStore{points: map[string]govector.SearchResult{}}
	embedder := fakeEmbedder{}
	catalog, err := OpenCatalog(filepath.Join(t.TempDir(), "canonical.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	svc := &Service{Searcher: search.NewSearcher(store, embedder), Hybrid: hybrid.NewSearcher(store, embedder, .7), Catalog: catalog}
	ctx := context.Background()
	created, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "canonical decision", Kind: "decision", Tags: []string{"api"}}, "idem-1", "hash-1")
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != "active" || created.Version != 1 || created.ID == "" {
		t.Fatalf("created=%+v", created)
	}
	replayed, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "canonical decision"}, "idem-1", "hash-1")
	if err != nil || replayed.ID != created.ID {
		t.Fatalf("replayed=%+v err=%v", replayed, err)
	}
	if _, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "other"}, "idem-1", "hash-2"); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("conflict=%v", err)
	}
	content := "updated decision"
	expected := 1
	updated, err := svc.UpdateMemory(ctx, created.ID, UpdateMemoryRequest{Content: &content, ExpectedVersion: &expected})
	if err != nil || updated.Version != 2 {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	hits, err := svc.Retrieve(ctx, RetrievalQuery{Text: "updated", Limit: 5})
	if err != nil || len(hits) != 1 || hits[0].ID != created.ID || hits[0].Content != content {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
	page, err := svc.ListMemories(ctx, ListMemoryOptions{Limit: 10})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	deleted, err := svc.DeleteMemory(ctx, created.ID, updated.Version, "user_request", "req_test")
	if err != nil || !deleted.Deleted {
		t.Fatalf("deleted=%+v err=%v", deleted, err)
	}
	if _, err := svc.GetMemory(ctx, created.ID); !errors.Is(err, ErrMemoryNotFound) {
		t.Fatalf("get deleted=%v", err)
	}
	hits, err = svc.Retrieve(ctx, RetrievalQuery{Text: "updated", Limit: 5})
	if err != nil || len(hits) != 0 {
		t.Fatalf("deleted hits=%+v err=%v", hits, err)
	}
	if _, err := svc.DeleteMemory(ctx, created.ID, deleted.Version, "user_request", "req_test2"); !errors.Is(err, ErrMemoryNotFound) {
		t.Fatalf("repeated delete=%v", err)
	}
	var audits int
	if err := catalog.db.QueryRow(`SELECT COUNT(*) FROM audit_log WHERE memory_id=?`, created.ID).Scan(&audits); err != nil || audits != 3 {
		t.Fatalf("audits=%d err=%v", audits, err)
	}
}

func (f *fakeStore) Search(_ []float32, limit int, filter map[string]any) ([]govector.SearchResult, error) {
	out := []govector.SearchResult{}
	for _, point := range f.points {
		matched := true
		for key, value := range filter {
			if point.Payload[key] != value {
				matched = false
			}
		}
		if matched {
			point.Score = .9
			out = append(out, point)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
func (f *fakeStore) Add(id string, _ []float32, payload map[string]any) error {
	if f.addErr != nil {
		return f.addErr
	}
	if f.points == nil {
		f.points = map[string]govector.SearchResult{}
	}
	f.points[id] = govector.SearchResult{ID: id, Payload: payload}
	return nil
}
func (f *fakeStore) AddBatch(points []govector.Point) error {
	for _, point := range points {
		if err := f.Add(point.ID, point.Vector, point.Payload); err != nil {
			return err
		}
	}
	return nil
}
func (f *fakeStore) Delete(id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.points, id)
	return nil
}
func (f *fakeStore) ListAll(limit int) ([]govector.SearchResult, error) {
	out := []govector.SearchResult{}
	for _, point := range f.points {
		out = append(out, point)
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
func (f *fakeStore) Close() error { return nil }

type fakeEmbedder struct{}

func (fakeEmbedder) CreateEmbedding(context.Context, string) ([]float32, error) {
	return []float32{1, 0}, nil
}
func (fakeEmbedder) CreateEmbeddings(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = []float32{1, 0}
	}
	return out, nil
}

func TestUninitializedServiceReportsUnavailable(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	if _, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "hello"}, "", ""); err == nil {
		t.Fatal("create should fail")
	}
	if _, err := svc.GetMemory(ctx, "mem_1"); err == nil {
		t.Fatal("get should fail")
	}
	if _, err := svc.ListMemories(ctx, ListMemoryOptions{Limit: 10}); err == nil {
		t.Fatal("list should fail")
	}
	if _, err := svc.Retrieve(ctx, RetrievalQuery{Text: "hello"}); err == nil {
		t.Fatal("retrieve should fail")
	}
}

func TestConcurrentUpdateUsesSingleVersionWinner(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	created, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "before", Kind: "fact"}, "cas-create", "cas-create-hash")
	if err != nil {
		t.Fatal(err)
	}
	expected := created.Version
	contents := []string{"winner-a", "winner-b"}
	errs := make([]error, len(contents))
	var wg sync.WaitGroup
	for i := range contents {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			content := contents[i]
			_, errs[i] = svc.UpdateMemory(ctx, created.ID, UpdateMemoryRequest{Content: &content, ExpectedVersion: &expected})
		}(i)
	}
	wg.Wait()
	winners := 0
	conflicts := 0
	for _, err := range errs {
		if err == nil {
			winners++
		} else if errors.Is(err, ErrVersionConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected concurrent update error: %v", err)
		}
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("winners=%d conflicts=%d errors=%v", winners, conflicts, errs)
	}
	current, err := svc.GetMemory(ctx, created.ID)
	if err != nil || current.Version != 2 {
		t.Fatalf("current=%+v err=%v", current, err)
	}
}

func TestCanonicalCommitSurvivesDerivedIndexFailureAndRecovery(t *testing.T) {
	store := &fakeStore{points: map[string]govector.SearchResult{}, addErr: errors.New("vector backend unavailable")}
	catalog, err := OpenCatalog(filepath.Join(t.TempDir(), "canonical.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	emb := fakeEmbedder{}
	svc := &Service{Searcher: search.NewSearcher(store, emb), Hybrid: hybrid.NewSearcher(store, emb, .7), Catalog: catalog}
	created, err := svc.CreateMemory(context.Background(), CreateMemoryRequest{Content: "durable before index"}, "", "")
	if err != nil {
		t.Fatalf("canonical mutation failed with derived failure: %v", err)
	}
	job, err := catalog.GetIndexJob(context.Background(), created.ID)
	if err != nil || job == nil || job.Attempts != 1 || job.State != IndexJobRetry {
		t.Fatalf("outbox=%+v err=%v", job, err)
	}
	if _, err := svc.GetMemory(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	store.addErr = nil
	if _, err := catalog.db.Exec(`UPDATE index_jobs SET next_attempt_at='' WHERE memory_id=?`, created.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DrainIndexJobs(context.Background()); err != nil {
		t.Fatal(err)
	}
	job, err = catalog.GetIndexJob(context.Background(), created.ID)
	if err != nil || job != nil {
		t.Fatalf("recovered job=%+v err=%v", job, err)
	}
}

func TestCloseNilService(t *testing.T) {
	svc := &Service{}
	if err := svc.Close(); err != nil {
		t.Fatalf("close nil service: %v", err)
	}
}

func TestServiceRealCRUDAndRetrieval(t *testing.T) {
	store := &fakeStore{points: map[string]govector.SearchResult{}}
	embedder := fakeEmbedder{}
	vector := search.NewSearcher(store, embedder)
	hybridSearcher := hybrid.NewSearcher(store, embedder, .7)
	catalog, err := OpenCatalog(filepath.Join(t.TempDir(), "canonical.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	svc := &Service{Searcher: vector, Hybrid: hybridSearcher, Catalog: catalog}
	ctx := context.Background()
	created, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "pipeline governance", Kind: "decision", Tags: []string{"technical"}}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatalf("created=%+v", created)
	}
	record, err := svc.GetMemory(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.Content != "pipeline governance" {
		t.Fatalf("record=%v", record)
	}
	page, err := svc.ListMemories(ctx, ListMemoryOptions{Limit: 10})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("page=%v err=%v", page, err)
	}
	hits, err := svc.Retrieve(ctx, RetrievalQuery{Text: "governance", Limit: 5})
	if err != nil || len(hits) != 1 {
		t.Fatalf("hits=%v err=%v", hits, err)
	}
	if len(hits[0].Channels) != 2 {
		t.Fatalf("channels=%v", hits[0].Channels)
	}
	deleted, err := svc.DeleteMemory(ctx, created.ID, created.Version, "user_request", "test")
	if err != nil || !deleted.Deleted {
		t.Fatalf("delete=%v err=%v", deleted, err)
	}
}
