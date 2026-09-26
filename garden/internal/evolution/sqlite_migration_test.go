package evolution

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestExistingEvolutionEventsSurviveReopenAndRetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evolution.db")
	fixture, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.Exec(`CREATE TABLE evolution_events(event_id TEXT PRIMARY KEY, run_id TEXT, proposal_id TEXT, type TEXT NOT NULL, actor TEXT NOT NULL, detail TEXT NOT NULL DEFAULT '', timestamp TEXT NOT NULL); INSERT INTO evolution_events VALUES('old','r','','started','user','','2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := fixture.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := OpenEventStore(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ev := EvolutionEvent{EventID: "retry", RunID: "r", Type: "reviewed", Actor: "user"}
	if err := store.Append(ctx, ev); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(ctx, ev); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenEventStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	events, err := store.ByRun(ctx, "r")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].EventID != "old" || events[1].EventID != "retry" {
		t.Fatalf("reopened events: %+v", events)
	}
}
