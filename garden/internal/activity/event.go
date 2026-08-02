package activity

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Event struct {
	ID        string         `json:"id"`
	SessionID string         `json:"session_id"`
	Type      string         `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data,omitempty"`
}

type Store struct {
	db *sql.DB
}

func OpenStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	schema := `CREATE TABLE IF NOT EXISTS activity_events(
 id TEXT NOT NULL, session_id TEXT NOT NULL, type TEXT NOT NULL,
 timestamp TEXT NOT NULL, data_json TEXT NOT NULL DEFAULT '{}',
 PRIMARY KEY(session_id, id));
CREATE INDEX IF NOT EXISTS activity_session ON activity_events(session_id, timestamp);`
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Append(ctx context.Context, ev Event) error {
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	dataJSON := "{}"
	if len(ev.Data) > 0 {
		raw, err := json.Marshal(ev.Data)
		if err != nil {
			return err
		}
		dataJSON = string(raw)
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO activity_events(id, session_id, type, timestamp, data_json) VALUES(?,?,?,?,?)`,
		ev.ID, ev.SessionID, ev.Type, ev.Timestamp.UTC().Format(time.RFC3339Nano), dataJSON)
	return err
}

func (s *Store) SessionEvents(ctx context.Context, sessionID string, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, session_id, type, timestamp, data_json FROM activity_events WHERE session_id=? ORDER BY timestamp ASC LIMIT ?`,
		sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []Event
	for rows.Next() {
		var ev Event
		var ts, dataJSON string
		if err := rows.Scan(&ev.ID, &ev.SessionID, &ev.Type, &ts, &dataJSON); err != nil {
			return nil, err
		}
		ev.Timestamp, _ = time.Parse(time.RFC3339Nano, ts)
		if dataJSON != "" && dataJSON != "{}" {
			_ = json.Unmarshal([]byte(dataJSON), &ev.Data)
		}
		events = append(events, ev)
	}
	return events, rows.Err()
}

func (s *Store) Close() error {
	return s.db.Close()
}
