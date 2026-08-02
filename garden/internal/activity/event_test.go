package activity

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendIdempotent(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	ev := Event{ID: "ev_1", SessionID: "sess_1", Type: "code_edit", Timestamp: time.Now().UTC()}
	if err := store.Append(ctx, ev); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(ctx, ev); err != nil {
		t.Fatal(err)
	}
	events, err := store.SessionEvents(ctx, "sess_1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestSessionEvents(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	base := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	for i, typ := range []string{"session_start", "code_edit", "session_end"} {
		ev := Event{ID: "ev_" + string(rune('a'+i)), SessionID: "sess_1", Type: typ, Timestamp: base.Add(time.Duration(i) * time.Second)}
		if err := store.Append(ctx, ev); err != nil {
			t.Fatal(err)
		}
	}
	events, err := store.SessionEvents(ctx, "sess_1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Type != "session_start" || events[2].Type != "session_end" {
		t.Fatalf("order wrong: %v", events)
	}
}

func TestSessionEventsLimit(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	base := time.Now().UTC()
	for i := 0; i < 10; i++ {
		ev := Event{ID: "ev_" + string(rune('a'+i)), SessionID: "sess_1", Type: "code_edit", Timestamp: base.Add(time.Duration(i) * time.Second)}
		if err := store.Append(ctx, ev); err != nil {
			t.Fatal(err)
		}
	}
	events, err := store.SessionEvents(ctx, "sess_1", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

func TestSessionEventsWithData(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	ev := Event{ID: "ev_1", SessionID: "sess_1", Type: "recall", Data: map[string]any{"query": "test", "limit": 5.0}}
	if err := store.Append(ctx, ev); err != nil {
		t.Fatal(err)
	}
	events, err := store.SessionEvents(ctx, "sess_1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if events[0].Data["query"] != "test" {
		t.Fatalf("data=%v", events[0].Data)
	}
}
