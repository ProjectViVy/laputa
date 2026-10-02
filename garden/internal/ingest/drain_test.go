package ingest

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dashimaki/garden/internal/activity"
	"github.com/dashimaki/garden/memory"
	"github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/mentle/facade"
)

type drainWriterFunc func(context.Context, facade.CreateMemoryRequest, string, string) (facade.Memory, error)

func (f drainWriterFunc) CreateMemory(ctx context.Context, req facade.CreateMemoryRequest, key, hash string) (facade.Memory, error) {
	return f(ctx, req, key, hash)
}

func newDrainFixture(t *testing.T, writer MemoryWriter) (*Service, activity.TransientEntry) {
	t.Helper()
	ctx := context.Background()
	svc, err := Open(filepath.Join(t.TempDir(), "ingest.db"), writer)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	spool, err := activity.OpenSpool(filepath.Join(t.TempDir(), "spool.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = spool.Close() })
	svc.Spool = spool
	entry := activity.TransientEntry{EventID: "event-drain", SessionID: "session-drain", ContentHash: "hash-drain", Content: "recover me", Kind: "source_artifact", Scope: memory.EncodeScope(evolution.Scope{SubjectID: "profile_1", Kind: evolution.ScopePersonal})}
	if err := spool.Append(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `INSERT INTO ingestions(ingestion_id,session_id,event_id,phase,content,content_hash,occurred_at,status,error,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, "ing_drain", entry.SessionID, entry.EventID, "session_end", entry.Content, entry.ContentHash, "now", "spooled", "mentle unavailable; spooled", "now", "now"); err != nil {
		t.Fatal(err)
	}
	return svc, entry
}

func TestDrainSpoolReturnsIngestionUpdateFailure(t *testing.T) {
	ctx := context.Background()
	svc, _ := newDrainFixture(t, drainWriterFunc(func(_ context.Context, req facade.CreateMemoryRequest, _, _ string) (facade.Memory, error) {
		return facade.Memory{ID: "mem_drain", Content: req.Content}, nil
	}))
	if _, err := svc.db.Exec(`CREATE TRIGGER reject_drain_update BEFORE UPDATE ON ingestions WHEN NEW.status='completed' BEGIN SELECT RAISE(ABORT, 'ingestion update failed'); END`); err != nil {
		t.Fatal(err)
	}
	drained, err := svc.DrainSpool(ctx)
	if drained != 0 || err == nil || !strings.Contains(err.Error(), "ingestion update failed") {
		t.Fatalf("DrainSpool() = (%d, %v), want (0, update failure)", drained, err)
	}
	status, err := svc.Get(ctx, "ing_drain")
	if err != nil || status.Status != "spooled" {
		t.Fatalf("ingestion status = (%+v, %v), want spooled", status, err)
	}
	pending, err := svc.Spool.PendingCount(ctx)
	if err != nil || pending != 1 {
		t.Fatalf("pending = (%d, %v), want retryable entry", pending, err)
	}
}

func TestDrainSpoolCompletesIngestionWithRecoveredMemoryID(t *testing.T) {
	ctx := context.Background()
	svc, entry := newDrainFixture(t, drainWriterFunc(func(_ context.Context, req facade.CreateMemoryRequest, key, hash string) (facade.Memory, error) {
		if req.Content != "recover me" || key != "session:event-drain" || hash != "hash-drain" {
			t.Fatalf("canonical request = (%+v, %q, %q)", req, key, hash)
		}
		return facade.Memory{ID: "mem_drain", Content: req.Content}, nil
	}))
	drained, err := svc.DrainSpool(ctx)
	if err != nil || drained != 1 {
		t.Fatalf("DrainSpool() = (%d, %v), want (1, nil)", drained, err)
	}
	pending, err := svc.Spool.PendingCount(ctx)
	if err != nil || pending != 0 {
		t.Fatalf("pending = (%d, %v), want 0", pending, err)
	}
	status, err := svc.Get(ctx, "ing_drain")
	if err != nil || status.Status != "completed" || status.Error != nil || len(status.MemoryIDs) != 1 || status.MemoryIDs[0] != "mem_drain" {
		t.Fatalf("recovered ingestion for %q = (%+v, %v)", entry.EventID, status, err)
	}
}

func TestDrainSpoolReturnsMarkDrainedFailure(t *testing.T) {
	ctx := context.Background()
	var svc *Service
	var err error
	svc, _ = newDrainFixture(t, drainWriterFunc(func(_ context.Context, req facade.CreateMemoryRequest, _, _ string) (facade.Memory, error) {
		if err = svc.Spool.Close(); err != nil {
			t.Fatal(err)
		}
		return facade.Memory{ID: "mem_drain", Content: req.Content}, nil
	}))
	drained, err := svc.DrainSpool(ctx)
	if drained != 0 || err == nil {
		t.Fatalf("DrainSpool() = (%d, %v), want (0, spool failure)", drained, err)
	}
	status, err := svc.Get(ctx, "ing_drain")
	if err != nil || status.Status != "completed" || len(status.MemoryIDs) != 1 || status.MemoryIDs[0] != "mem_drain" {
		t.Fatalf("canonical recovery status = (%+v, %v), want completed with memory id", status, err)
	}
}

func TestDrainSpoolReturnsCanonicalWriteFailureWithoutDraining(t *testing.T) {
	ctx := context.Background()
	writeErr := errors.New("canonical memory unavailable")
	svc, _ := newDrainFixture(t, drainWriterFunc(func(_ context.Context, req facade.CreateMemoryRequest, _, _ string) (facade.Memory, error) {
		if req.Content != "recover me" {
			t.Fatalf("content = %q, want recover me", req.Content)
		}
		return facade.Memory{}, writeErr
	}))
	drained, err := svc.DrainSpool(ctx)
	if drained != 0 || !errors.Is(err, writeErr) {
		t.Fatalf("DrainSpool() = (%d, %v), want (0, write error)", drained, err)
	}
	pending, err := svc.Spool.PendingCount(ctx)
	if err != nil || pending != 1 {
		t.Fatalf("pending = (%d, %v), want 1", pending, err)
	}
	status, err := svc.Get(ctx, "ing_drain")
	if err != nil || status.Status != "spooled" {
		t.Fatalf("ingestion status = (%+v, %v), want spooled", status, err)
	}
}
