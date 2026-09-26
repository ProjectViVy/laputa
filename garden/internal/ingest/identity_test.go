package ingest

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
)

func TestGetByIdentityPersistsAcrossRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "ingest.db")
	svc, err := Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	content := "durable identity lookup"
	sum := sha256.Sum256([]byte(content))
	accepted, err := svc.Submit(ctx, SubmitRequest{SessionID: "session-a", EventID: "event-a", Phase: "session_end", Content: content, ContentHash: fmt.Sprintf("sha256:%x", sum)})
	if err != nil {
		svc.Close()
		t.Fatal(err)
	}
	if err := svc.Close(); err != nil {
		t.Fatal(err)
	}
	svc, err = Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	got, err := svc.GetByIdentity(ctx, accepted.IngestionID, "session-a", "event-a")
	if err != nil {
		t.Fatal(err)
	}
	if got.IngestionID != accepted.IngestionID || got.Status == "" || got.MemoryIDs == nil || got.Warnings == nil {
		t.Fatalf("incomplete status after restart: %+v", got)
	}
}

func TestGetByIdentityRejectsMismatchedIdentity(t *testing.T) {
	ctx := context.Background()
	svc, err := Open(filepath.Join(t.TempDir(), "ingest.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	content := "private event"
	sum := sha256.Sum256([]byte(content))
	accepted, err := svc.Submit(ctx, SubmitRequest{SessionID: "session-a", EventID: "event-a", Phase: "precompact", Content: content, ContentHash: fmt.Sprintf("sha256:%x", sum)})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, id, session, event string }{
		{"other session", accepted.IngestionID, "session-b", "event-a"},
		{"other event", accepted.IngestionID, "session-a", "event-b"},
		{"missing id", "ing_missing", "session-a", "event-a"},
		{"empty session", accepted.IngestionID, "", "event-a"},
		{"empty event", accepted.IngestionID, "session-a", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := svc.GetByIdentity(ctx, tc.id, tc.session, tc.event)
			if !errors.Is(err, ErrNotFound) || got.IngestionID != "" {
				t.Fatalf("got %+v, %v; want empty status and ErrNotFound", got, err)
			}
		})
	}
}
