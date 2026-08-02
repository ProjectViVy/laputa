package facade

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/dashimaki/mentle/internal/hybrid"
	"github.com/dashimaki/mentle/internal/search"
	"github.com/dashimaki/mentle/storage/govector"
)

type fakeStore struct {
	points map[string]govector.SearchResult
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
	deleted, err := svc.DeleteMemory(ctx, created.ID, "user_request", "req_test")
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
	if _, err := svc.DeleteMemory(ctx, created.ID, "user_request", "req_test2"); err != nil {
		t.Fatalf("idempotent delete=%v", err)
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
	if f.points == nil {
		f.points = map[string]govector.SearchResult{}
	}
	f.points[id] = govector.SearchResult{ID: id, Payload: payload}
	return nil
}
func (f *fakeStore) AddBatch(points []govector.Point) error {
	for _, point := range points {
		_ = f.Add(point.ID, point.Vector, point.Payload)
	}
	return nil
}
func (f *fakeStore) Delete(id string) error { delete(f.points, id); return nil }
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

	if _, err := svc.Write(ctx, "memory:test", "hello", nil); err == nil {
		t.Fatal("write should fail")
	}
	if _, err := svc.Read(ctx, "memory:test"); err == nil {
		t.Fatal("read should fail")
	}
	if _, err := svc.List(ctx, "memory:", 10); err == nil {
		t.Fatal("list should fail")
	}
	if _, err := svc.Forget(ctx, "memory:test"); err == nil {
		t.Fatal("forget should fail")
	}
	if _, err := svc.Retrieve(ctx, RetrievalQuery{Text: "hello"}); err == nil {
		t.Fatal("retrieve should fail")
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
	svc := &Service{Searcher: vector, Hybrid: hybridSearcher}
	ctx := context.Background()
	id, err := svc.Write(ctx, "memory:test", "pipeline governance", map[string]any{"wing": "technical", "room": "architecture"})
	if err != nil {
		t.Fatal(err)
	}
	if id != "memory:test" {
		t.Fatalf("id=%q", id)
	}
	record, err := svc.Read(ctx, "memory:test")
	if err != nil {
		t.Fatal(err)
	}
	if record["value"] != "pipeline governance" {
		t.Fatalf("record=%v", record)
	}
	records, err := svc.List(ctx, "memory:", 10)
	if err != nil || len(records) != 1 {
		t.Fatalf("records=%v err=%v", records, err)
	}
	hits, err := svc.Retrieve(ctx, RetrievalQuery{Text: "governance", Limit: 5})
	if err != nil || len(hits) != 1 {
		t.Fatalf("hits=%v err=%v", hits, err)
	}
	if len(hits[0].Channels) != 2 {
		t.Fatalf("channels=%v", hits[0].Channels)
	}
	ok, err := svc.Forget(ctx, "memory:test")
	if err != nil || !ok {
		t.Fatalf("forget=%v err=%v", ok, err)
	}
}
