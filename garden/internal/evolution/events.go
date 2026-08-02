package evolution

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var ErrEventNotFound = errors.New("evolution: event not found")

type EventStore struct {
	db *sql.DB
}

func OpenEventStore(path string) (*EventStore, error) {
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	schema := `CREATE TABLE IF NOT EXISTS evolution_events(
 event_id TEXT PRIMARY KEY, run_id TEXT, proposal_id TEXT,
 type TEXT NOT NULL, actor TEXT NOT NULL, detail TEXT NOT NULL DEFAULT '',
 timestamp TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS evo_events_run ON evolution_events(run_id, timestamp);
CREATE INDEX IF NOT EXISTS evo_events_proposal ON evolution_events(proposal_id, timestamp);`
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &EventStore{db: db}, nil
}

func (s *EventStore) Append(_ context.Context, ev EvolutionEvent) error {
	if ev.EventID == "" {
		ev.EventID = "evoevt_" + uuid.NewString()[:8]
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO evolution_events(event_id, run_id, proposal_id, type, actor, detail, timestamp) VALUES(?,?,?,?,?,?,?)`,
		ev.EventID, ev.RunID, ev.ProposalID, ev.Type, ev.Actor, ev.Detail,
		ev.Timestamp.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *EventStore) Get(_ context.Context, eventID string) (EvolutionEvent, error) {
	var ev EvolutionEvent
	var ts string
	err := s.db.QueryRow(
		`SELECT event_id, run_id, proposal_id, type, actor, detail, timestamp FROM evolution_events WHERE event_id = ?`, eventID,
	).Scan(&ev.EventID, &ev.RunID, &ev.ProposalID, &ev.Type, &ev.Actor, &ev.Detail, &ts)
	if errors.Is(err, sql.ErrNoRows) {
		return EvolutionEvent{}, ErrEventNotFound
	}
	ev.Timestamp, _ = time.Parse(time.RFC3339Nano, ts)
	return ev, err
}

func (s *EventStore) ByRun(_ context.Context, runID string) ([]EvolutionEvent, error) {
	rows, err := s.db.Query(
		`SELECT event_id, run_id, proposal_id, type, actor, detail, timestamp FROM evolution_events WHERE run_id = ? ORDER BY timestamp`, runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []EvolutionEvent
	for rows.Next() {
		var ev EvolutionEvent
		var ts string
		if err := rows.Scan(&ev.EventID, &ev.RunID, &ev.ProposalID, &ev.Type, &ev.Actor, &ev.Detail, &ts); err != nil {
			return nil, err
		}
		ev.Timestamp, _ = time.Parse(time.RFC3339Nano, ts)
		events = append(events, ev)
	}
	return events, rows.Err()
}

func (s *EventStore) Close() error {
	return s.db.Close()
}
