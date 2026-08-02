package activity

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var ErrSpoolFull = errors.New("transient spool is full")

const maxPendingEntries = 1000

type TransientEntry struct {
	EventID     string `json:"event_id"`
	SessionID   string `json:"session_id"`
	ContentHash string `json:"content_hash"`
	Scope       string `json:"scope"`
	Content     string `json:"content"`
	Kind        string `json:"kind"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	DrainedAt   string `json:"drained_at,omitempty"`
}

type TransientReadRequest struct {
	SessionID    string
	Scope        string
	AfterEventID string
	LimitEvents  int
	BudgetChars  int
}

type TransientSpool struct {
	db *sql.DB
}

func OpenSpool(path string) (*TransientSpool, error) {
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	schema := `CREATE TABLE IF NOT EXISTS transient_spool(
 event_id TEXT NOT NULL, content_hash TEXT NOT NULL, session_id TEXT NOT NULL,
 scope TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, kind TEXT NOT NULL DEFAULT 'raw',
 status TEXT NOT NULL DEFAULT 'pending_mentle',
 created_at TEXT NOT NULL, drained_at TEXT,
 PRIMARY KEY(event_id, content_hash));
CREATE INDEX IF NOT EXISTS spool_status ON transient_spool(status, created_at);`
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &TransientSpool{db: db}, nil
}

func (s *TransientSpool) Append(ctx context.Context, entry TransientEntry) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM transient_spool WHERE status='pending_mentle'`).Scan(&count); err != nil {
		return err
	}
	if count >= maxPendingEntries {
		return ErrSpoolFull
	}
	if entry.CreatedAt == "" {
		entry.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if entry.Kind == "" {
		entry.Kind = "raw"
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO transient_spool(event_id, content_hash, session_id, scope, content, kind, status, created_at) VALUES(?,?,?,?,?,?,?,?)`,
		entry.EventID, entry.ContentHash, entry.SessionID, entry.Scope, entry.Content, entry.Kind, "pending_mentle", entry.CreatedAt)
	return err
}

func (s *TransientSpool) Read(ctx context.Context, req TransientReadRequest) ([]TransientEntry, error) {
	if req.LimitEvents <= 0 {
		req.LimitEvents = 50
	}
	query := `SELECT event_id, content_hash, session_id, scope, content, kind, status, created_at, COALESCE(drained_at,'') FROM transient_spool WHERE 1=1`
	args := []any{}
	if req.SessionID != "" {
		query += ` AND session_id=?`
		args = append(args, req.SessionID)
	}
	if req.Scope != "" {
		query += ` AND scope=?`
		args = append(args, req.Scope)
	}
	if req.AfterEventID != "" {
		query += ` AND created_at > (SELECT created_at FROM transient_spool WHERE event_id=? LIMIT 1)`
		args = append(args, req.AfterEventID)
	}
	query += ` ORDER BY created_at ASC LIMIT ?`
	args = append(args, req.LimitEvents)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows, req.BudgetChars)
}

func (s *TransientSpool) Pending(ctx context.Context) ([]TransientEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT event_id, content_hash, session_id, scope, content, kind, status, created_at, COALESCE(drained_at,'') FROM transient_spool WHERE status='pending_mentle' ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows, 0)
}

func (s *TransientSpool) PendingCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM transient_spool WHERE status='pending_mentle'`).Scan(&count)
	return count, err
}

func (s *TransientSpool) MarkDrained(ctx context.Context, eventID, contentHash string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE transient_spool SET status='drained', drained_at=? WHERE event_id=? AND content_hash=?`,
		time.Now().UTC().Format(time.RFC3339Nano), eventID, contentHash)
	return err
}

func (s *TransientSpool) Close() error {
	return s.db.Close()
}

func scanEntries(rows *sql.Rows, budgetChars int) ([]TransientEntry, error) {
	var entries []TransientEntry
	total := 0
	for rows.Next() {
		var e TransientEntry
		if err := rows.Scan(&e.EventID, &e.ContentHash, &e.SessionID, &e.Scope, &e.Content, &e.Kind, &e.Status, &e.CreatedAt, &e.DrainedAt); err != nil {
			return nil, err
		}
		if budgetChars > 0 {
			total += len(e.Content)
			if total > budgetChars {
				break
			}
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
