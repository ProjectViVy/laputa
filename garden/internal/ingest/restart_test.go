package ingest

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestPendingIngestionRestartsAndRemainsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ingest.db")
	first, err := Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	content := "durable session decision"
	sum := sha256.Sum256([]byte(content))
	req := SubmitRequest{SessionID: "session-restart", EventID: "event-restart", Phase: "session_end", Content: content, ContentHash: fmt.Sprintf("sha256:%x", sum)}
	accepted, err := first.Submit(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	writer := &fakeWriter{}
	second, err := Open(path, writer)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	deadline := time.Now().Add(3 * time.Second)
	for {
		status, err := second.Get(context.Background(), accepted.IngestionID)
		if err != nil {
			t.Fatal(err)
		}
		if status.Status == "completed" {
			if len(status.MemoryIDs) != 1 || status.MemoryIDs[0] != "mem_digest" {
				t.Fatalf("recovered status=%+v", status)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("recovery timed out: %+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
	duplicate, err := second.Submit(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.IngestionID != accepted.IngestionID {
		t.Fatalf("duplicate id=%q, want %q", duplicate.IngestionID, accepted.IngestionID)
	}
	req.EventID = "new-event-same-content"
	sameContent, err := second.Submit(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if sameContent.IngestionID != accepted.IngestionID {
		t.Fatalf("content duplicate id=%q, want %q", sameContent.IngestionID, accepted.IngestionID)
	}
	req.EventID = "event-restart"
	req.Content = "changed content"
	changed := sha256.Sum256([]byte(req.Content))
	req.ContentHash = fmt.Sprintf("sha256:%x", changed)
	if _, err := second.Submit(context.Background(), req); !errors.Is(err, ErrEventConflict) {
		t.Fatalf("changed event error=%v, want ErrEventConflict", err)
	}
	writer.mu.Lock()
	calls := writer.calls
	writer.mu.Unlock()
	if calls != 1 {
		t.Fatalf("memory writes=%d, want 1", calls)
	}
}
