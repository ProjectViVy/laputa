package activity

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ProjectViVy/laputa/garden/internal/sqliteconn"
)

var ErrSpoolFull = errors.New("transient spool is full")
var ErrSpoolConflict = errors.New("event id was already spooled with different exact input")

const maxPendingEntries = 1000

type TransientEntry struct {
	EventID   string `json:"event_id"`
	SessionID string `json:"session_id"`
	Scope     string `json:"scope"`
	Content   string `json:"content"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	DrainedAt string `json:"drained_at,omitempty"`
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
	db, err := sqliteconn.Open(path)
	if err != nil {
		return nil, err
	}
	schema := `CREATE TABLE IF NOT EXISTS transient_spool(
	 event_id TEXT NOT NULL, session_id TEXT NOT NULL,
	 scope TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, kind TEXT NOT NULL DEFAULT 'raw',
 status TEXT NOT NULL DEFAULT 'pending_mentle',
 created_at TEXT NOT NULL, drained_at TEXT,
	 PRIMARY KEY(event_id, content));`
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrateSpool(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS spool_status ON transient_spool(status, created_at)`); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &TransientSpool{db: db}, nil
}

func migrateSpool(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(transient_spool)`)
	if err != nil {
		return err
	}
	legacy := false
	for rows.Next() {
		var cid, notNull, primary int
		var name, kind string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &primary); err != nil {
			rows.Close()
			return err
		}
		legacy = legacy || name == "content_hash"
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if !legacy {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE TABLE transient_spool_new(
 event_id TEXT NOT NULL, session_id TEXT NOT NULL, scope TEXT NOT NULL DEFAULT '',
 content TEXT NOT NULL, kind TEXT NOT NULL DEFAULT 'raw',
 status TEXT NOT NULL DEFAULT 'pending_mentle', created_at TEXT NOT NULL, drained_at TEXT,
 PRIMARY KEY(event_id,content));
INSERT INTO transient_spool_new(rowid,event_id,session_id,scope,content,kind,status,created_at,drained_at)
 SELECT rowid,event_id,session_id,scope,content,kind,status,created_at,drained_at FROM transient_spool;
DROP TABLE transient_spool;
ALTER TABLE transient_spool_new RENAME TO transient_spool;`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *TransientSpool) Append(ctx context.Context, entry TransientEntry) error {
	if entry.Kind == "" {
		entry.Kind = "raw"
	}
	rows, err := s.db.QueryContext(ctx, `SELECT session_id,scope,content,kind FROM transient_spool WHERE event_id=?`, entry.EventID)
	if err != nil {
		return err
	}
	found := false
	for rows.Next() {
		var session, scope, content, kind string
		if err := rows.Scan(&session, &scope, &content, &kind); err != nil {
			rows.Close()
			return err
		}
		found = true
		if session != entry.SessionID || scope != entry.Scope || content != entry.Content || kind != entry.Kind {
			rows.Close()
			return ErrSpoolConflict
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if found {
		return nil
	}
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
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO transient_spool(event_id, session_id, scope, content, kind, status, created_at) VALUES(?,?,?,?,?,?,?)`,
		entry.EventID, entry.SessionID, entry.Scope, entry.Content, entry.Kind, "pending_mentle", entry.CreatedAt)
	return err
}

func (s *TransientSpool) Read(ctx context.Context, req TransientReadRequest) ([]TransientEntry, error) {
	if req.LimitEvents <= 0 {
		req.LimitEvents = 50
	}
	query := `SELECT event_id, session_id, scope, content, kind, status, created_at, COALESCE(drained_at,'') FROM transient_spool WHERE 1=1`
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
		`SELECT event_id, session_id, scope, content, kind, status, created_at, COALESCE(drained_at,'') FROM transient_spool WHERE status='pending_mentle' ORDER BY created_at ASC`)
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

func (s *TransientSpool) MarkDrained(ctx context.Context, eventID, content string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE transient_spool SET status='drained', drained_at=? WHERE event_id=? AND content=?`,
		time.Now().UTC().Format(time.RFC3339Nano), eventID, content)
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
		if err := rows.Scan(&e.EventID, &e.SessionID, &e.Scope, &e.Content, &e.Kind, &e.Status, &e.CreatedAt, &e.DrainedAt); err != nil {
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
