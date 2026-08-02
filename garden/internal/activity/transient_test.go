package activity

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSpoolAppendIdempotent(t *testing.T) {
	spool, err := OpenSpool(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer spool.Close()
	ctx := context.Background()
	entry := TransientEntry{EventID: "ev_1", SessionID: "sess_1", ContentHash: "sha256:abc", Content: "hello world"}
	if err := spool.Append(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if err := spool.Append(ctx, entry); err != nil {
		t.Fatal(err)
	}
	pending, err := spool.Pending(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}
}

func TestSpoolRead(t *testing.T) {
	spool, err := OpenSpool(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer spool.Close()
	ctx := context.Background()
	for i, content := range []string{"first", "second", "third"} {
		entry := TransientEntry{EventID: "ev_" + string(rune('a'+i)), SessionID: "sess_1", ContentHash: "hash_" + string(rune('a'+i)), Content: content, CreatedAt: "2026-08-01T00:00:0" + string(rune('1'+i)) + "Z"}
		if err := spool.Append(ctx, entry); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := spool.Read(ctx, TransientReadRequest{SessionID: "sess_1", LimitEvents: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2, got %d", len(entries))
	}
	if entries[0].Content != "first" {
		t.Fatalf("order wrong: %v", entries[0])
	}
}

func TestSpoolMarkDrained(t *testing.T) {
	spool, err := OpenSpool(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer spool.Close()
	ctx := context.Background()
	entry := TransientEntry{EventID: "ev_1", SessionID: "sess_1", ContentHash: "hash_1", Content: "data"}
	if err := spool.Append(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if err := spool.MarkDrained(ctx, "ev_1", "hash_1"); err != nil {
		t.Fatal(err)
	}
	pending, err := spool.Pending(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending after drain, got %d", len(pending))
	}
}

func TestSpoolPendingOrdered(t *testing.T) {
	spool, err := OpenSpool(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer spool.Close()
	ctx := context.Background()
	entries := []TransientEntry{
		{EventID: "ev_2", SessionID: "s", ContentHash: "h2", Content: "second", CreatedAt: "2026-08-01T00:00:02Z"},
		{EventID: "ev_1", SessionID: "s", ContentHash: "h1", Content: "first", CreatedAt: "2026-08-01T00:00:01Z"},
		{EventID: "ev_3", SessionID: "s", ContentHash: "h3", Content: "third", CreatedAt: "2026-08-01T00:00:03Z"},
	}
	for _, e := range entries {
		if err := spool.Append(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	pending, err := spool.Pending(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 3 || pending[0].EventID != "ev_1" || pending[2].EventID != "ev_3" {
		t.Fatalf("order wrong: %+v", pending)
	}
}

func TestSpoolBudgetChars(t *testing.T) {
	spool, err := OpenSpool(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer spool.Close()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		entry := TransientEntry{EventID: "ev_" + string(rune('a'+i)), SessionID: "s", ContentHash: "h" + string(rune('a'+i)), Content: "1234567890", CreatedAt: "2026-08-01T00:00:0" + string(rune('1'+i)) + "Z"}
		if err := spool.Append(ctx, entry); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := spool.Read(ctx, TransientReadRequest{LimitEvents: 10, BudgetChars: 25})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) > 3 {
		t.Fatalf("budget should limit results, got %d", len(entries))
	}
}
