package ingest

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ProjectViVy/laputa/mentle/facade"
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

// Rename the real SQLite table to make Start's SELECT fail, then restore it.
func TestStartSelectErrorCanRetry(t *testing.T) {
	writer := &fakeWriter{}
	svc, err := OpenPaused(filepath.Join(t.TempDir(), "garden.db"), writer)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	content := "select retry"
	sum := sha256.Sum256([]byte(content))
	accepted, err := svc.Submit(context.Background(), SubmitRequest{SessionID: "select", EventID: "select-event", Phase: "session_end", Content: content, ContentHash: fmt.Sprintf("sha256:%x", sum)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.Exec(`ALTER TABLE ingestions RENAME TO temporarily_unavailable`); err != nil {
		t.Fatal(err)
	}
	if err := svc.Start(); err == nil || !strings.Contains(err.Error(), "no such table: ingestions") {
		t.Fatalf("Start SELECT error = %v", err)
	}
	if svc.started {
		t.Fatal("failed Start marked service started")
	}
	if _, err := svc.db.Exec(`ALTER TABLE temporarily_unavailable RENAME TO ingestions`); err != nil {
		t.Fatal(err)
	}
	if err := svc.Start(); err != nil {
		t.Fatalf("retry Start: %v", err)
	}
	waitForCompleted(t, svc, accepted.IngestionID)
	writer.mu.Lock()
	calls := writer.calls
	writer.mu.Unlock()
	if calls != 1 {
		t.Fatalf("writer calls = %d, want 1", calls)
	}
}

// SQLite allows NULL in a TEXT PRIMARY KEY on a rowid table. Use it to fail
// database/sql Scan without replacing SQLite or Rows with a mock.
func TestStartScanErrorCanRetry(t *testing.T) {
	writer := &fakeWriter{}
	svc, err := OpenPaused(filepath.Join(t.TempDir(), "garden.db"), writer)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	content := "scan retry"
	sum := sha256.Sum256([]byte(content))
	accepted, err := svc.Submit(context.Background(), SubmitRequest{SessionID: "scan", EventID: "scan-event", Phase: "session_end", Content: content, ContentHash: fmt.Sprintf("sha256:%x", sum)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.Exec(`UPDATE ingestions SET ingestion_id=NULL WHERE event_id=?`, accepted.EventID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Start(); err == nil || !strings.Contains(err.Error(), "converting NULL to string is unsupported") {
		t.Fatalf("Start Scan error = %v", err)
	}
	if svc.started {
		t.Fatal("failed Start marked service started")
	}
	if _, err := svc.db.Exec(`UPDATE ingestions SET ingestion_id=? WHERE event_id=?`, accepted.IngestionID, accepted.EventID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Start(); err != nil {
		t.Fatalf("retry Start: %v", err)
	}
	waitForCompleted(t, svc, accepted.IngestionID)
}

func TestCloseIsIdempotentAndPreventsStart(t *testing.T) {
	for _, started := range []bool{false, true} {
		t.Run(fmt.Sprintf("started=%t", started), func(t *testing.T) {
			svc, err := OpenPaused(filepath.Join(t.TempDir(), "garden.db"), nil)
			if err != nil {
				t.Fatal(err)
			}
			if started {
				if err := svc.Start(); err != nil {
					t.Fatal(err)
				}
				if err := svc.Start(); err != nil {
					t.Fatalf("repeated Start: %v", err)
				}
			}
			if err := svc.Close(); err != nil {
				t.Fatalf("first Close: %v", err)
			}
			if err := svc.Close(); err != nil {
				t.Fatalf("repeated Close: %v", err)
			}
			if err := svc.Start(); err == nil || !strings.Contains(err.Error(), "closed") {
				t.Fatalf("Start after Close = %v", err)
			}
		})
	}
}

func waitForCompleted(t *testing.T, svc *Service, id string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		status, err := svc.Get(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if status.Status == "completed" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("status after Start = %+v", status)
		}
		time.Sleep(10 * time.Millisecond)
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
