package recall

import (
	"context"
	"path/filepath"
	"testing"
)

func TestTraceStoreReopenPreservesTrace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.db")
	store, err := OpenTraceStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), RecallTrace{TraceID: "reopened", Query: "persistent trace"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenTraceStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	got, err := store.Get(context.Background(), "reopened")
	if err != nil || got.Query != "persistent trace" {
		t.Fatalf("reopened trace=%+v, err=%v", got, err)
	}
}
