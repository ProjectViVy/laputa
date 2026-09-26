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

func TestOpenPausedDoesNotProcessUntilStart(t *testing.T) {
	writer := &fakeWriter{}
	svc, err := OpenPaused(filepath.Join(t.TempDir(), "garden.db"), writer)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	content := "paused session decision"
	sum := sha256.Sum256([]byte(content))
	accepted, err := svc.Submit(context.Background(), SubmitRequest{
		SessionID: "paused", EventID: "paused-event", Phase: "session_end",
		Content: content, ContentHash: fmt.Sprintf("sha256:%x", sum),
	})
	if err != nil {
		t.Fatal(err)
	}
	status, err := svc.Get(context.Background(), accepted.IngestionID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "accepted" {
		t.Fatalf("paused ingestion status=%s, want accepted", status.Status)
	}
	writer.mu.Lock()
	calls := writer.calls
	writer.mu.Unlock()
	if calls != 0 {
		t.Fatalf("writer called before Start: %d", calls)
	}
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		status, err = svc.Get(context.Background(), accepted.IngestionID)
		if err != nil {
			t.Fatal(err)
		}
		if status.Status == "completed" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("status after Start=%+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestOpenPausedReplaysDurableRowsOnlyAfterStart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "garden.db")
	first, err := OpenPaused(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	content := "durable session decision"
	sum := sha256.Sum256([]byte(content))
	accepted, err := first.Submit(context.Background(), SubmitRequest{
		SessionID: "durable", EventID: "durable-event", Phase: "session_end",
		Content: content, ContentHash: fmt.Sprintf("sha256:%x", sum),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	writer := &fakeWriter{}
	svc, err := OpenPaused(path, writer)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	// A paused open must not replay even when there is already durable work.
	time.Sleep(30 * time.Millisecond)
	status, err := svc.Get(context.Background(), accepted.IngestionID)
	if err != nil || status.Status != "accepted" {
		t.Fatalf("before Start: status=%+v err=%v", status, err)
	}
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		status, err = svc.Get(context.Background(), accepted.IngestionID)
		if err != nil {
			t.Fatal(err)
		}
		if status.Status == "completed" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("status after Start=%+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	writer.mu.Lock()
	calls := writer.calls
	writer.mu.Unlock()
	if calls != 1 {
		t.Fatalf("Start must not replay processed row; calls=%d", calls)
	}
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
