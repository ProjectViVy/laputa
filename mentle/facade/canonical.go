package facade

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ProjectViVy/laputa/mentle/internal/embedder"
	"github.com/ProjectViVy/laputa/mentle/storage/sqlite"
	"github.com/google/uuid"
)

var (
	ErrMemoryNotFound             = errors.New("memory not found")
	ErrVersionConflict            = errors.New("version conflict")
	ErrIdempotencyConflict        = errors.New("idempotency conflict")
	ErrUnavailable                = errors.New("mentle unavailable")
	ErrReadOnly                   = errors.New("mentle lexical-only service is read-only")
	ErrEmbeddingDimensionMismatch = errors.New("embedding dimension mismatch")
	ErrEmbeddingMetricMismatch    = errors.New("embedding metric mismatch")
	ErrEmbeddingIdentityMismatch  = errors.New("embedding identity mismatch")
)

type MemorySource struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
	EventID   string `json:"event_id,omitempty"`
	URI       string `json:"uri,omitempty"`
	Revision  string `json:"revision,omitempty"`
}

type Memory struct {
	ID           string         `json:"id"`
	Kind         string         `json:"kind"`
	Content      string         `json:"content"`
	Status       string         `json:"status"`
	Version      int            `json:"version"`
	Scope        string         `json:"scope,omitempty"`
	Lifecycle    string         `json:"lifecycle,omitempty"`
	Collection   string         `json:"collection,omitempty"`
	Tags         []string       `json:"tags"`
	Source       MemorySource   `json:"source"`
	ValidFrom    time.Time      `json:"valid_from"`
	ValidTo      *time.Time     `json:"valid_to"`
	Supersedes   []string       `json:"supersedes"`
	SupersededBy *string        `json:"superseded_by"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	Metadata     map[string]any `json:"metadata"`
}

type CreateMemoryRequest struct {
	Content    string         `json:"content"`
	Kind       string         `json:"kind,omitempty"`
	Scope      string         `json:"scope,omitempty"`
	Tags       []string       `json:"tags,omitempty"`
	Source     MemorySource   `json:"source,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Supersedes []string       `json:"supersedes,omitempty"`
	Actor      string         `json:"-"`
	RequestID  string         `json:"-"`
}

type UpdateMemoryRequest struct {
	Content         *string   `json:"content,omitempty"`
	Reason          string    `json:"reason,omitempty"`
	ExpectedVersion *int      `json:"expected_version,omitempty"`
	Tags            *[]string `json:"tags,omitempty"`
	Actor           string    `json:"-"`
	RequestID       string    `json:"-"`
}

// DeleteMemoryRequest is the explicit optimistic-concurrency contract for a
// logical delete. A deleted row is never treated as an idempotent success.
type DeleteMemoryRequest struct {
	ExpectedVersion int    `json:"expected_version"`
	Reason          string `json:"reason"`
	Actor           string `json:"-"`
	RequestID       string `json:"-"`
}

type ListMemoryOptions struct {
	Limit  int
	Cursor string
	Status string
	Kind   string
}

type MemoryPage struct {
	Items      []Memory `json:"items"`
	NextCursor *string  `json:"next_cursor"`
}

type DeleteResult struct {
	ID         string  `json:"id"`
	Deleted    bool    `json:"deleted"`
	Status     string  `json:"status"`
	Version    int     `json:"version"`
	IndexState string  `json:"index_state"`
	IndexJobID *string `json:"index_job_id"`
}

type Catalog struct{ db *sql.DB }

func OpenCatalog(path string) (*Catalog, error) {
	db, err := sqlite.Open(path)
	if err != nil {
		return nil, err
	}
	schema := `
CREATE TABLE IF NOT EXISTS memories (
 id TEXT PRIMARY KEY, kind TEXT NOT NULL, content TEXT NOT NULL,
 status TEXT NOT NULL, version INTEGER NOT NULL, scope TEXT NOT NULL,
 tags_json TEXT NOT NULL, source_json TEXT NOT NULL, valid_from TEXT NOT NULL,
 valid_to TEXT, supersedes_json TEXT NOT NULL, superseded_by TEXT,
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL, metadata_json TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS memories_order ON memories(updated_at DESC, id DESC);
CREATE TABLE IF NOT EXISTS idempotency (
 key TEXT PRIMARY KEY, body_hash TEXT NOT NULL, memory_id TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS mutation_receipts (
 operation_id TEXT PRIMARY KEY,
 payload_digest TEXT NOT NULL,
 scope TEXT NOT NULL,
 destination TEXT NOT NULL,
 record_id TEXT NOT NULL,
 revision INTEGER NOT NULL,
 status TEXT NOT NULL,
 error_code TEXT NOT NULL,
 created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS index_jobs (
 job_id TEXT NOT NULL, memory_id TEXT PRIMARY KEY, canonical_version INTEGER NOT NULL DEFAULT 1,
 operation TEXT NOT NULL, content TEXT NOT NULL, metadata_json TEXT NOT NULL,
 state TEXT NOT NULL DEFAULT 'pending', attempts INTEGER NOT NULL DEFAULT 0,
 last_error TEXT NOT NULL DEFAULT '', next_attempt_at TEXT NOT NULL DEFAULT '',
 lease_owner TEXT NOT NULL DEFAULT '', lease_until TEXT NOT NULL DEFAULT '',
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS audit_log (
 sequence INTEGER PRIMARY KEY AUTOINCREMENT, memory_id TEXT NOT NULL, action TEXT NOT NULL,
 actor TEXT NOT NULL, request_id TEXT NOT NULL, reason TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS embedding_identity (
 id INTEGER PRIMARY KEY CHECK (id = 1),
 model TEXT NOT NULL,
 dimension INTEGER NOT NULL,
 metric TEXT NOT NULL DEFAULT 'cosine',
 normalize INTEGER NOT NULL DEFAULT 1,
 provider TEXT NOT NULL,
 version TEXT NOT NULL,
 captured_at TEXT NOT NULL
);
 CREATE TABLE IF NOT EXISTS reindex_jobs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  reason TEXT NOT NULL,
  old_model TEXT,
  old_dimension INTEGER,
  new_model TEXT,
  new_dimension INTEGER,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TEXT NOT NULL,
  completed_at TEXT
 );
 CREATE TABLE IF NOT EXISTS session_cursor (
  session_id TEXT PRIMARY KEY, last_timestamp TEXT, updated_at TEXT
 );
 CREATE TABLE IF NOT EXISTS session_lock (
   session_id TEXT PRIMARY KEY, owner_pid INTEGER, acquired_at TEXT, ttl_seconds INTEGER
 );
 CREATE TABLE IF NOT EXISTS index_rebuilds (
   id INTEGER PRIMARY KEY CHECK (id = 1), status TEXT NOT NULL,
   started_at TEXT NOT NULL, completed_at TEXT NOT NULL DEFAULT '',
   canonical_snapshot_count INTEGER, error_code TEXT NOT NULL DEFAULT ''
 );`
	if _, err = db.Exec(schema); err != nil {
		return nil, errors.Join(err, db.Close())
	}
	if err = migrateIndexJobs(db); err != nil {
		return nil, errors.Join(fmt.Errorf("migrate index outbox: %w", err), db.Close())
	}
	return &Catalog{db: db}, nil
}

func (c *Catalog) Close() error { return c.db.Close() }

func (c *Catalog) GetEmbeddingIdentity() (*embedder.Identity, error) {
	var id embedder.Identity
	var normalize int
	var capturedAt string
	err := c.db.QueryRow(`SELECT model, dimension, metric, normalize, provider, version, captured_at FROM embedding_identity WHERE id = 1`).
		Scan(&id.Model, &id.Dimension, &id.Metric, &normalize, &id.Provider, &id.Version, &capturedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	id.Normalize = normalize == 1
	id.CapturedAt, _ = time.Parse(time.RFC3339Nano, capturedAt)
	return &id, nil
}

func (c *Catalog) SaveEmbeddingIdentity(id embedder.Identity) error {
	norm := 0
	if id.Normalize {
		norm = 1
	}
	_, err := c.db.Exec(`
		INSERT INTO embedding_identity (id, model, dimension, metric, normalize, provider, version, captured_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			model=excluded.model,
			dimension=excluded.dimension,
			metric=excluded.metric,
			normalize=excluded.normalize,
			provider=excluded.provider,
			version=excluded.version,
			captured_at=excluded.captured_at
	`, id.Model, id.Dimension, id.Metric, norm, id.Provider, id.Version, id.CapturedAt.Format(time.RFC3339Nano))
	return err
}

func (c *Catalog) EnqueueReindexJob(reason string, oldDim, newDim int, oldModel, newModel string) error {
	_, err := c.db.Exec(`
		INSERT INTO reindex_jobs (reason, old_model, old_dimension, new_model, new_dimension, status, created_at)
		VALUES (?, ?, ?, ?, ?, 'pending', ?)
	`, reason, oldModel, oldDim, newModel, newDim, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (c *Catalog) PendingReindexJobs() (int, error) {
	var count int
	err := c.db.QueryRow(`SELECT COUNT(*) FROM reindex_jobs WHERE status = 'pending'`).Scan(&count)
	return count, err
}

func (c *Catalog) SaveSessionCursor(sessionID, timestamp string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("session_id is required")
	}
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(timestamp))
	if err != nil {
		return errors.New("timestamp must be RFC3339")
	}
	timestamp = parsed.UTC().Format(time.RFC3339Nano)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = c.db.Exec(`
		INSERT INTO session_cursor (session_id, last_timestamp, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			last_timestamp=excluded.last_timestamp,
			updated_at=excluded.updated_at
		WHERE session_cursor.last_timestamp IS NULL OR session_cursor.last_timestamp <= excluded.last_timestamp
	`, sessionID, timestamp, now)
	return err
}

func (c *Catalog) GetSessionCursor(sessionID string) (string, error) {
	var timestamp string
	err := c.db.QueryRow(`SELECT last_timestamp FROM session_cursor WHERE session_id = ?`, sessionID).Scan(&timestamp)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return timestamp, err
}

// SaveSessionCursor persists the monotonic source boundary through the
// facade, so ingest adapters never need a raw canonical database handle.
func (s *Service) SaveSessionCursor(sessionID, timestamp string) error {
	if s != nil && s.lexicalOnly {
		return ErrReadOnly
	}
	if s == nil || s.Catalog == nil {
		return ErrUnavailable
	}
	return s.Catalog.SaveSessionCursor(sessionID, timestamp)
}

// GetSessionCursor returns the last durable source boundary for a session.
func (s *Service) GetSessionCursor(sessionID string) (string, error) {
	if s == nil || s.Catalog == nil {
		return "", ErrUnavailable
	}
	return s.Catalog.GetSessionCursor(sessionID)
}

// AcquireSessionLease obtains a crash-reclaimable lease through the facade.
func (s *Service) AcquireSessionLease(ctx context.Context, sessionID, owner string, ttl time.Duration) (bool, error) {
	if s != nil && s.lexicalOnly {
		return false, ErrReadOnly
	}
	if s == nil || s.Catalog == nil {
		return false, ErrUnavailable
	}
	return s.Catalog.AcquireSessionLease(ctx, sessionID, owner, ttl)
}

// ReleaseSessionLease releases a lease previously acquired by owner.
func (s *Service) ReleaseSessionLease(ctx context.Context, sessionID, owner string) error {
	if s != nil && s.lexicalOnly {
		return ErrReadOnly
	}
	if s == nil || s.Catalog == nil {
		return ErrUnavailable
	}
	return s.Catalog.ReleaseSessionLease(ctx, sessionID, owner)
}

// AcquireSessionLease obtains a durable per-session lease. A live lease held
// by another owner is not stolen; an expired lease can be reclaimed after a
// crash or process restart.
func (c *Catalog) AcquireSessionLease(ctx context.Context, sessionID, owner string, ttl time.Duration) (bool, error) {
	if c == nil || c.db == nil {
		return false, ErrUnavailable
	}
	sessionID = strings.TrimSpace(sessionID)
	owner = strings.TrimSpace(owner)
	if sessionID == "" || owner == "" || ttl <= 0 {
		return false, errors.New("session_id, owner and positive ttl are required")
	}
	now := time.Now().UTC()
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var currentOwner, acquired string
	var ttlSeconds int
	err = tx.QueryRowContext(ctx, `SELECT owner_pid, acquired_at, ttl_seconds FROM session_lock WHERE session_id=?`, sessionID).Scan(&currentOwner, &acquired, &ttlSeconds)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.ExecContext(ctx, `INSERT INTO session_lock(session_id,owner_pid,acquired_at,ttl_seconds) VALUES(?,?,?,?)`, sessionID, owner, now.Format(time.RFC3339Nano), int(ttl.Seconds()))
		if err != nil {
			return false, err
		}
		return true, tx.Commit()
	}
	if err != nil {
		return false, err
	}
	acquiredAt, parseErr := time.Parse(time.RFC3339Nano, acquired)
	if parseErr == nil && currentOwner != owner && acquiredAt.Add(time.Duration(ttlSeconds)*time.Second).After(now) {
		return false, nil
	}
	_, err = tx.ExecContext(ctx, `UPDATE session_lock SET owner_pid=?,acquired_at=?,ttl_seconds=? WHERE session_id=?`, owner, now.Format(time.RFC3339Nano), int(ttl.Seconds()), sessionID)
	if err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func (c *Catalog) ReleaseSessionLease(ctx context.Context, sessionID, owner string) error {
	if c == nil || c.db == nil {
		return ErrUnavailable
	}
	_, err := c.db.ExecContext(ctx, `DELETE FROM session_lock WHERE session_id=? AND owner_pid=?`, strings.TrimSpace(sessionID), strings.TrimSpace(owner))
	return err
}

func canonicalID() string { return "mem_" + strings.ReplaceAll(uuid.NewString(), "-", "") }

func (s *Service) runtimeEmbeddingIdentity() (*embedder.Identity, bool) {
	if s == nil {
		return nil, false
	}
	if s.EmbeddingIdentity != nil {
		identity := s.EmbeddingIdentity()
		return &identity, true
	}
	if s.Embedder != nil {
		identity := s.Embedder.Identity()
		return &identity, true
	}
	return nil, false
}

func sameEmbeddingIdentity(left, right embedder.Identity) bool {
	return left.Model == right.Model &&
		left.Dimension == right.Dimension &&
		left.Metric == right.Metric &&
		left.Normalize == right.Normalize &&
		left.Provider == right.Provider &&
		left.Version == right.Version
}

// ensureEmbeddingIdentity validates the exact identity before a mutation can
// commit. A mismatch schedules an explicit reindex request, but never permits
// mixed vectors into the disposable index.
func (s *Service) ensureEmbeddingIdentity() error {
	if s != nil && s.lexicalOnly {
		return ErrReadOnly
	}
	current, ok := s.runtimeEmbeddingIdentity()
	if !ok || s.Catalog == nil {
		return nil
	}
	saved, err := s.Catalog.GetEmbeddingIdentity()
	if err != nil {
		return err
	}
	if saved == nil {
		return s.Catalog.SaveEmbeddingIdentity(*current)
	}
	if sameEmbeddingIdentity(*saved, *current) {
		return nil
	}
	reason := "embedding identity mismatch"
	if saved.Dimension != current.Dimension {
		reason = "dimension mismatch"
	} else if saved.Metric != current.Metric {
		reason = "metric mismatch"
	}
	_ = s.Catalog.EnqueueReindexJob(reason, saved.Dimension, current.Dimension, saved.Model, current.Model)
	if saved.Dimension != current.Dimension {
		return ErrEmbeddingDimensionMismatch
	}
	if saved.Metric != current.Metric {
		return ErrEmbeddingMetricMismatch
	}
	return ErrEmbeddingIdentityMismatch
}

func (s *Service) CreateMemory(ctx context.Context, req CreateMemoryRequest, idempotencyKey, bodyHash string) (Memory, error) {
	if s != nil && s.lexicalOnly {
		return Memory{}, ErrReadOnly
	}
	if s.Catalog == nil || s.Hybrid == nil {
		return Memory{}, ErrUnavailable
	}
	if err := s.ensureEmbeddingIdentity(); err != nil {
		return Memory{}, err
	}

	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		return Memory{}, errors.New("memory content is required")
	}
	if len([]byte(req.Content)) > 64<<10 {
		return Memory{}, errors.New("memory content exceeds 64 KiB")
	}
	if req.Kind == "" {
		req.Kind = "note"
	}
	if !allowed(req.Kind, "fact", "preference", "decision", "session_digest", "note", "source_artifact", "semantic_unit") {
		return Memory{}, errors.New("invalid memory kind")
	}
	if req.Source.Type == "" {
		req.Source.Type = "user"
	}
	if !allowed(req.Source.Type, "user", "agent", "session", "import", "report_projection") {
		return Memory{}, errors.New("invalid source type")
	}
	if idempotencyKey != "" {
		var savedHash, id string
		err := s.Catalog.db.QueryRowContext(ctx, `SELECT body_hash,memory_id FROM idempotency WHERE key=?`, idempotencyKey).Scan(&savedHash, &id)
		if err == nil {
			if savedHash != bodyHash {
				return Memory{}, ErrIdempotencyConflict
			}
			// The canonical row already committed on the original request. A
			// retry must return that row even when the derived-index worker is
			// currently backoff/poisoned; recovery is observable via IndexHealth.
			_ = s.applyIndexJob(ctx, id)
			return s.GetMemory(ctx, id)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return Memory{}, err
		}
	}
	now := time.Now().UTC()
	m := Memory{ID: canonicalID(), Kind: req.Kind, Content: req.Content, Status: "active", Version: 1, Scope: req.Scope, Tags: nonNil(req.Tags), Source: req.Source, ValidFrom: now, Supersedes: nonNil(req.Supersedes), CreatedAt: now, UpdatedAt: now, Metadata: nonNilMap(req.Metadata)}
	if v, ok := m.Metadata["lifecycle"].(string); ok && v != "" {
		m.Lifecycle = v
	} else {
		m.Lifecycle = "ltm"
	}
	if v, ok := m.Metadata["collection"].(string); ok && v != "" {
		m.Collection = v
	} else {
		m.Collection = "knowledge"
	}
	tx, err := s.Catalog.db.BeginTx(ctx, nil)
	if err != nil {
		return Memory{}, err
	}
	if err = insertMemory(ctx, tx, m); err == nil && idempotencyKey != "" {
		_, err = tx.ExecContext(ctx, `INSERT INTO idempotency(key,body_hash,memory_id,created_at) VALUES(?,?,?,?)`, idempotencyKey, bodyHash, m.ID, now.Format(time.RFC3339Nano))
	}
	if err == nil {
		_, err = enqueueIndexJobTx(ctx, tx, m.ID, m.Version, "upsert", m.Content, encode(m.Metadata), now)
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO audit_log(memory_id,action,actor,request_id,reason,created_at) VALUES(?,?,?,?,?,?)`, m.ID, "create", req.Actor, req.RequestID, "", now.Format(time.RFC3339Nano))
	}
	for _, supersedesID := range req.Supersedes {
		if err != nil {
			break
		}
		_, err = tx.ExecContext(ctx, `UPDATE memories SET superseded_by=?, updated_at=? WHERE id=? AND superseded_by IS NULL`, m.ID, now.Format(time.RFC3339Nano), supersedesID)
	}
	if err != nil {
		tx.Rollback()
		return Memory{}, err
	}
	if err = tx.Commit(); err != nil {
		return Memory{}, err
	}
	_ = s.applyIndexJob(ctx, m.ID)
	return m, nil
}

func (s *Service) GetMemory(ctx context.Context, id string) (Memory, error) {
	if s.Catalog == nil {
		return Memory{}, ErrUnavailable
	}
	m, err := scanMemory(s.Catalog.db.QueryRowContext(ctx, memorySelect+` WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) || (err == nil && m.Status == "deleted") {
		return Memory{}, ErrMemoryNotFound
	}
	return m, err
}

func (s *Service) UpdateMemory(ctx context.Context, id string, req UpdateMemoryRequest) (Memory, error) {
	if s != nil && s.lexicalOnly {
		return Memory{}, ErrReadOnly
	}
	if s.Catalog == nil {
		return Memory{}, ErrUnavailable
	}
	if err := s.ensureEmbeddingIdentity(); err != nil {
		return Memory{}, err
	}
	if req.ExpectedVersion == nil || *req.ExpectedVersion <= 0 {
		return Memory{}, ErrVersionConflict
	}
	if req.Content == nil && req.Tags == nil {
		return Memory{}, errors.New("at least one mutable field is required")
	}
	m, err := s.GetMemory(ctx, id)
	if err != nil {
		return Memory{}, err
	}
	if req.Content != nil {
		content := strings.TrimSpace(*req.Content)
		if content == "" {
			return Memory{}, errors.New("memory content is required")
		}
		if len([]byte(content)) > 64<<10 {
			return Memory{}, errors.New("memory content exceeds 64 KiB")
		}
		m.Content = content
	}
	if req.Tags != nil {
		m.Tags = nonNil(*req.Tags)
	}
	newVersion := m.Version + 1
	m.UpdatedAt = time.Now().UTC()
	tx, err := s.Catalog.db.BeginTx(ctx, nil)
	if err != nil {
		return Memory{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE memories SET content=?,tags_json=?,version=version+1,updated_at=? WHERE id=? AND version=? AND status='active'`, m.Content, encode(m.Tags), m.UpdatedAt.Format(time.RFC3339Nano), id, *req.ExpectedVersion)
	if err != nil {
		tx.Rollback()
		return Memory{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return Memory{}, err
	}
	if affected != 1 {
		tx.Rollback()
		if _, getErr := s.GetMemory(ctx, id); errors.Is(getErr, ErrMemoryNotFound) {
			return Memory{}, ErrMemoryNotFound
		}
		return Memory{}, ErrVersionConflict
	}
	m.Version = newVersion
	if _, err = enqueueIndexJobTx(ctx, tx, m.ID, m.Version, "upsert", m.Content, encode(m.Metadata), m.UpdatedAt); err != nil {
		tx.Rollback()
		return Memory{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO audit_log(memory_id,action,actor,request_id,reason,created_at) VALUES(?,?,?,?,?,?)`, m.ID, "update", req.Actor, req.RequestID, req.Reason, m.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
		tx.Rollback()
		return Memory{}, err
	}
	if err = tx.Commit(); err != nil {
		return Memory{}, err
	}
	_ = s.applyIndexJob(ctx, m.ID)
	return m, nil
}

func (s *Service) DeleteMemory(ctx context.Context, id string, expectedVersion int, actor, requestID string, reasons ...string) (DeleteResult, error) {
	if s != nil && s.lexicalOnly {
		return DeleteResult{}, ErrReadOnly
	}
	if s.Catalog == nil {
		return DeleteResult{}, ErrUnavailable
	}
	if expectedVersion <= 0 {
		return DeleteResult{}, ErrVersionConflict
	}
	now := time.Now().UTC()
	tx, err := s.Catalog.db.BeginTx(ctx, nil)
	if err != nil {
		return DeleteResult{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE memories SET status='deleted',valid_to=?,updated_at=?,version=version+1 WHERE id=? AND version=? AND status='active'`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), id, expectedVersion)
	if err != nil {
		tx.Rollback()
		return DeleteResult{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return DeleteResult{}, err
	}
	if affected != 1 {
		tx.Rollback()
		var status string
		lookupErr := s.Catalog.db.QueryRowContext(ctx, `SELECT status FROM memories WHERE id=?`, id).Scan(&status)
		if errors.Is(lookupErr, sql.ErrNoRows) || status == "deleted" {
			return DeleteResult{}, ErrMemoryNotFound
		}
		if lookupErr != nil {
			return DeleteResult{}, lookupErr
		}
		return DeleteResult{}, ErrVersionConflict
	}
	if _, err = enqueueIndexJobTx(ctx, tx, id, expectedVersion, "delete", "", "{}", now); err != nil {
		tx.Rollback()
		return DeleteResult{}, err
	}
	reason := ""
	if len(reasons) > 0 {
		reason = reasons[0]
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO audit_log(memory_id,action,actor,request_id,reason,created_at) VALUES(?,?,?,?,?,?)`, id, "delete", actor, requestID, reason, now.Format(time.RFC3339Nano)); err != nil {
		tx.Rollback()
		return DeleteResult{}, err
	}
	if err = tx.Commit(); err != nil {
		return DeleteResult{}, err
	}
	_ = s.applyIndexJob(ctx, id)
	state, jobID := s.indexStatus(ctx, id)
	return DeleteResult{ID: id, Deleted: true, Status: "deleted", Version: expectedVersion + 1, IndexState: contractIndexState(state), IndexJobID: jobID}, nil
}

func (s *Service) ListMemories(ctx context.Context, opts ListMemoryOptions) (MemoryPage, error) {
	if s.Catalog == nil {
		return MemoryPage{}, ErrUnavailable
	}
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Limit > 200 {
		opts.Limit = 200
	}
	if opts.Status == "" {
		opts.Status = "active"
	}
	args := []any{opts.Status}
	where := ` WHERE status=?`
	if opts.Kind != "" {
		where += ` AND kind=?`
		args = append(args, opts.Kind)
	}
	if opts.Cursor != "" {
		where += ` AND (updated_at < ? OR (updated_at = ? AND id < ?))`
		parts := strings.SplitN(opts.Cursor, "|", 2)
		if len(parts) != 2 {
			return MemoryPage{}, errors.New("invalid cursor")
		}
		args = append(args, parts[0], parts[0], parts[1])
	}
	args = append(args, opts.Limit+1)
	rows, err := s.Catalog.db.QueryContext(ctx, memorySelect+where+` ORDER BY updated_at DESC,id DESC LIMIT ?`, args...)
	if err != nil {
		return MemoryPage{}, err
	}
	defer rows.Close()
	items := []Memory{}
	for rows.Next() {
		m, e := scanMemory(rows)
		if e != nil {
			return MemoryPage{}, e
		}
		items = append(items, m)
	}
	var next *string
	if len(items) > opts.Limit {
		items = items[:opts.Limit]
		value := items[len(items)-1].UpdatedAt.Format(time.RFC3339Nano) + "|" + items[len(items)-1].ID
		next = &value
	}
	return MemoryPage{Items: items, NextCursor: next}, rows.Err()
}

const memorySelect = `SELECT id,kind,content,status,version,scope,tags_json,source_json,valid_from,valid_to,supersedes_json,superseded_by,created_at,updated_at,metadata_json FROM memories`

type scanner interface{ Scan(...any) error }

func scanMemory(row scanner) (Memory, error) {
	var m Memory
	var tags, source, supersedes, metadata, created, updated, valid string
	var validTo, superBy sql.NullString
	err := row.Scan(&m.ID, &m.Kind, &m.Content, &m.Status, &m.Version, &m.Scope, &tags, &source, &valid, &validTo, &supersedes, &superBy, &created, &updated, &metadata)
	if err != nil {
		return m, err
	}
	_ = json.Unmarshal([]byte(tags), &m.Tags)
	_ = json.Unmarshal([]byte(source), &m.Source)
	_ = json.Unmarshal([]byte(supersedes), &m.Supersedes)
	_ = json.Unmarshal([]byte(metadata), &m.Metadata)
	m.ValidFrom, _ = time.Parse(time.RFC3339Nano, valid)
	m.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	m.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	if validTo.Valid {
		t, _ := time.Parse(time.RFC3339Nano, validTo.String)
		m.ValidTo = &t
	}
	if superBy.Valid {
		m.SupersededBy = &superBy.String
	}
	if v, ok := m.Metadata["lifecycle"].(string); ok && v != "" {
		m.Lifecycle = v
	} else {
		m.Lifecycle = "ltm"
	}
	if v, ok := m.Metadata["collection"].(string); ok && v != "" {
		m.Collection = v
	} else {
		m.Collection = "knowledge"
	}
	return m, nil
}
func insertMemory(ctx context.Context, tx *sql.Tx, m Memory) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO memories(id,kind,content,status,version,scope,tags_json,source_json,valid_from,valid_to,supersedes_json,superseded_by,created_at,updated_at,metadata_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, m.ID, m.Kind, m.Content, m.Status, m.Version, m.Scope, encode(m.Tags), encode(m.Source), m.ValidFrom.Format(time.RFC3339Nano), nil, encode(m.Supersedes), nil, m.CreatedAt.Format(time.RFC3339Nano), m.UpdatedAt.Format(time.RFC3339Nano), encode(m.Metadata))
	return err
}
func encode(v any) string { b, _ := json.Marshal(v); return string(b) }
func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
func nonNilMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
}
func stringMetadata(v map[string]any) map[string]string {
	out := map[string]string{}
	for k, x := range v {
		if s, ok := x.(string); ok {
			out[k] = s
		}
	}
	return out
}
func allowed(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
