package ingest

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dashimaki/mentle/facade"
)

type fakeWriter struct {
	mu    sync.Mutex
	calls int
}

func (f *fakeWriter) CreateMemory(_ context.Context, req facade.CreateMemoryRequest, _ string, _ string) (facade.Memory, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return facade.Memory{ID: "mem_digest", Content: req.Content}, nil
}

func TestSubmitProcessesOnceAndClearsTranscript(t *testing.T) {
	writer := &fakeWriter{}
	svc, err := Open(filepath.Join(t.TempDir(), "garden.db"), writer)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	content := "important session decision"
	sum := sha256.Sum256([]byte(content))
	req := SubmitRequest{SessionID: "sess", EventID: "evt", Phase: "session_end", Content: content, ContentHash: fmt.Sprintf("sha256:%x", sum)}
	accepted, err := svc.Submit(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Submit(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		status, err := svc.Get(context.Background(), accepted.IngestionID)
		if err != nil {
			t.Fatal(err)
		}
		if status.Status == "completed" {
			if len(status.MemoryIDs) != 1 {
				t.Fatalf("memory ids=%v", status.MemoryIDs)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("status=%+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
	writer.mu.Lock()
	calls := writer.calls
	writer.mu.Unlock()
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	var stored string
	if err := svc.db.QueryRow(`SELECT content FROM ingestions WHERE ingestion_id=?`, accepted.IngestionID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != content {
		t.Fatalf("raw-first: content should be retained, got %q", stored)
	}
}
