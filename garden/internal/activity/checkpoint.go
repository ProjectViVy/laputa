package activity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ProjectViVy/laputa/garden/internal/sqliteconn"
)

// CheckpointStore keeps runtime WorkingSet state in Garden's SQLite database.
// It is intentionally separate from Laputa authority and ACTMEM.
type CheckpointStore struct {
	db *sql.DB
}

func OpenCheckpointStore(path string) (*CheckpointStore, error) {
	db, err := sqliteconn.Open(path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS garden_checkpoints(
 scope TEXT PRIMARY KEY,
 snapshot_json TEXT NOT NULL,
 checkpoint_at TEXT NOT NULL
);`)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &CheckpointStore{db: db}, nil
}

func (s *CheckpointStore) Save(ctx context.Context, scope string, snapshot ScopeSnapshot) error {
	if s == nil || s.db == nil {
		return errors.New("checkpoint store unavailable")
	}
	if scope == "" {
		scope = "_default"
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO garden_checkpoints(scope,snapshot_json,checkpoint_at)
VALUES(?,?,CURRENT_TIMESTAMP)
ON CONFLICT(scope) DO UPDATE SET snapshot_json=excluded.snapshot_json, checkpoint_at=excluded.checkpoint_at`, scope, string(raw))
	return err
}

func (s *CheckpointStore) Load(ctx context.Context, scope string) (ScopeSnapshot, bool, error) {
	if s == nil || s.db == nil {
		return ScopeSnapshot{}, false, errors.New("checkpoint store unavailable")
	}
	if scope == "" {
		scope = "_default"
	}
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT snapshot_json FROM garden_checkpoints WHERE scope=?`, scope).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return ScopeSnapshot{}, false, nil
	}
	if err != nil {
		return ScopeSnapshot{}, false, err
	}
	var snapshot ScopeSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return ScopeSnapshot{}, false, fmt.Errorf("checkpoint snapshot is malformed: %w", err)
	}
	return snapshot, true, nil
}

func (s *CheckpointStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

type Checkpointer struct {
	Store *CheckpointStore
	WS    *WorkingSet
}

func (c *Checkpointer) Save(ctx context.Context, scope string) error {
	if c == nil || c.Store == nil {
		return errors.New("checkpoint store unavailable")
	}
	if c.WS == nil {
		return errors.New("working set unavailable")
	}
	if err := c.Store.Save(ctx, scope, c.WS.Snapshot(scope)); err != nil {
		return err
	}
	c.WS.MarkCheckpoint(scope)
	return nil
}

func (c *Checkpointer) Load(ctx context.Context, scope string) error {
	if c == nil || c.Store == nil {
		return errors.New("checkpoint store unavailable")
	}
	if c.WS == nil {
		return errors.New("working set unavailable")
	}
	snapshot, found, err := c.Store.Load(ctx, scope)
	if err != nil {
		return err
	}
	if found {
		c.WS.Restore(scope, snapshot)
	}
	return nil
}
