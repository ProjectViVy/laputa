package activity

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestExistingSQLiteEventReopensAndNewWriteSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "activity.db")
	fixture, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.Exec(`CREATE TABLE activity_events(id TEXT NOT NULL, session_id TEXT NOT NULL, type TEXT NOT NULL, timestamp TEXT NOT NULL, data_json TEXT NOT NULL DEFAULT '{}', PRIMARY KEY(session_id,id)); INSERT INTO activity_events VALUES('old','s','legacy','2026-01-01T00:00:00Z','{}')`); err != nil {
		t.Fatal(err)
	}
	if err := fixture.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.Append(ctx, Event{ID: "new", SessionID: "s", Type: "new"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	events, err := store.SessionEvents(ctx, "s", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].ID != "old" || events[1].ID != "new" {
		t.Fatalf("reopened events: %+v", events)
	}
}
