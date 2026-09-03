package facade

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionCursorIsMonotonicAndLeaseIsCrashReclaimable(t *testing.T) {
	catalog, err := OpenCatalog(filepath.Join(t.TempDir(), "canonical.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()

	if err := catalog.SaveSessionCursor("session-1", "2026-09-03T10:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveSessionCursor("session-1", "2026-09-03T09:00:00Z"); err != nil {
		t.Fatal(err)
	}
	value, err := catalog.GetSessionCursor("session-1")
	if err != nil {
		t.Fatal(err)
	}
	if value != "2026-09-03T10:00:00Z" {
		t.Fatalf("cursor moved backwards: %q", value)
	}

	ctx := context.Background()
	acquired, err := catalog.AcquireSessionLease(ctx, "session-1", "owner-a", time.Minute)
	if err != nil || !acquired {
		t.Fatalf("first lease acquired=%v err=%v", acquired, err)
	}
	acquired, err = catalog.AcquireSessionLease(ctx, "session-1", "owner-b", time.Minute)
	if err != nil || acquired {
		t.Fatalf("live lease stolen: acquired=%v err=%v", acquired, err)
	}
	if err := catalog.ReleaseSessionLease(ctx, "session-1", "owner-a"); err != nil {
		t.Fatal(err)
	}
	acquired, err = catalog.AcquireSessionLease(ctx, "session-1", "owner-b", time.Minute)
	if err != nil || !acquired {
		t.Fatalf("released lease not reusable: acquired=%v err=%v", acquired, err)
	}
}
