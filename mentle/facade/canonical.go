package facade

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dashimaki/mentle/internal/palace"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrMemoryNotFound      = errors.New("memory not found")
	ErrVersionConflict     = errors.New("version conflict")
	ErrIdempotencyConflict = errors.New("idempotency conflict")
)

type MemorySource struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
	EventID   string `json:"event_id,omitempty"`
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
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
	Status  string `json:"status"`
}

type Catalog struct{ db *sql.DB }

func OpenCatalog(path string) (*Catalog, error) {
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	schema := `
CREATE TABLE IF NOT EXISTS memories (
 id TEXT PRIMARY KEY, legacy_key TEXT UNIQUE, kind TEXT NOT NULL, content TEXT NOT NULL,
 status TEXT NOT NULL, version INTEGER NOT NULL, scope TEXT NOT NULL,
 tags_json TEXT NOT NULL, source_json TEXT NOT NULL, valid_from TEXT NOT NULL,
 valid_to TEXT, supersedes_json TEXT NOT NULL, superseded_by TEXT,
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL, metadata_json TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS memories_order ON memories(updated_at DESC, id DESC);
CREATE TABLE IF NOT EXISTS idempotency (
 key TEXT PRIMARY KEY, body_hash TEXT NOT NULL, memory_id TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS index_jobs (
 memory_id TEXT PRIMARY KEY, operation TEXT NOT NULL, content TEXT NOT NULL, metadata_json TEXT NOT NULL,
 created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS audit_log (
 sequence INTEGER PRIMARY KEY AUTOINCREMENT, memory_id TEXT NOT NULL, action TEXT NOT NULL,
 actor TEXT NOT NULL, request_id TEXT NOT NULL, reason TEXT NOT NULL, created_at TEXT NOT NULL
);`
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Catalog{db: db}, nil
}

func (c *Catalog) Close() error { return c.db.Close() }

func canonicalID() string { return "mem_" + strings.ReplaceAll(uuid.NewString(), "-", "") }

func (s *Service) CreateMemory(ctx context.Context, req CreateMemoryRequest, idempotencyKey, bodyHash string) (Memory, error) {
	if s.Catalog == nil || s.Hybrid == nil {
		return Memory{}, ErrUnavailable
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
	if !allowed(req.Kind, "fact", "preference", "decision", "session_digest", "note") {
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
			if err := s.applyIndexJob(ctx, id); err != nil {
				return Memory{}, err
			}
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
	if err = insertMemory(ctx, tx, m, ""); err == nil && idempotencyKey != "" {
		_, err = tx.ExecContext(ctx, `INSERT INTO idempotency(key,body_hash,memory_id,created_at) VALUES(?,?,?,?)`, idempotencyKey, bodyHash, m.ID, now.Format(time.RFC3339Nano))
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO index_jobs(memory_id,operation,content,metadata_json,created_at) VALUES(?,?,?,?,?)`, m.ID, "upsert", m.Content, encode(m.Metadata), now.Format(time.RFC3339Nano))
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
	if err = s.applyIndexJob(ctx, m.ID); err != nil {
		return Memory{}, err
	}
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
	m, err := s.GetMemory(ctx, id)
	if err != nil {
		return Memory{}, err
	}
	if req.ExpectedVersion != nil && *req.ExpectedVersion != m.Version {
		return Memory{}, ErrVersionConflict
	}
	if req.Content == nil && req.Tags == nil {
		return Memory{}, errors.New("at least one mutable field is required")
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
	m.Version++
	m.UpdatedAt = time.Now().UTC()
	tx, err := s.Catalog.db.BeginTx(ctx, nil)
	if err != nil {
		return Memory{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE memories SET content=?,tags_json=?,version=?,updated_at=? WHERE id=? AND status!='deleted'`, m.Content, encode(m.Tags), m.Version, m.UpdatedAt.Format(time.RFC3339Nano), id); err != nil {
		tx.Rollback()
		return Memory{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO index_jobs(memory_id,operation,content,metadata_json,created_at) VALUES(?,?,?,?,?) ON CONFLICT(memory_id) DO UPDATE SET operation=excluded.operation,content=excluded.content,metadata_json=excluded.metadata_json,created_at=excluded.created_at`, m.ID, "upsert", m.Content, encode(m.Metadata), m.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
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
	if err = s.applyIndexJob(ctx, m.ID); err != nil {
		return Memory{}, err
	}
	return m, nil
}

func (s *Service) DeleteMemory(ctx context.Context, id, actor, requestID string) (DeleteResult, error) {
	if s.Catalog == nil {
		return DeleteResult{}, ErrUnavailable
	}
	var status string
	err := s.Catalog.db.QueryRowContext(ctx, `SELECT status FROM memories WHERE id=?`, id).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return DeleteResult{}, ErrMemoryNotFound
	}
	if err != nil {
		return DeleteResult{}, err
	}
	if status != "deleted" {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		tx, e := s.Catalog.db.BeginTx(ctx, nil)
		if e != nil {
			return DeleteResult{}, e
		}
		if _, err = tx.ExecContext(ctx, `UPDATE memories SET status='deleted',valid_to=?,updated_at=?,version=version+1 WHERE id=?`, now, now, id); err != nil {
			tx.Rollback()
			return DeleteResult{}, err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO index_jobs(memory_id,operation,content,metadata_json,created_at) VALUES(?,?,?,?,?) ON CONFLICT(memory_id) DO UPDATE SET operation=excluded.operation,created_at=excluded.created_at`, id, "delete", "", "{}", now); err != nil {
			tx.Rollback()
			return DeleteResult{}, err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO audit_log(memory_id,action,actor,request_id,reason,created_at) VALUES(?,?,?,?,?,?)`, id, "delete", actor, requestID, "", now); err != nil {
			tx.Rollback()
			return DeleteResult{}, err
		}
		if err = tx.Commit(); err != nil {
			return DeleteResult{}, err
		}
		if err = s.applyIndexJob(ctx, id); err != nil {
			return DeleteResult{}, err
		}
	}
	return DeleteResult{ID: id, Deleted: true, Status: "deleted"}, nil
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

func (s *Service) backfillCanonical(ctx context.Context) error {
	if s.Catalog == nil || s.Searcher == nil {
		return nil
	}
	drawers, err := s.Searcher.ListAll(ctx, 50000)
	if err != nil {
		return err
	}
	for _, d := range drawers {
		if strings.HasPrefix(d.ID, "mem_") {
			continue
		}
		legacy := "memory:" + d.ID
		var exists int
		if err := s.Catalog.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM memories WHERE legacy_key=?`, legacy).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		now := time.Now().UTC()
		m := Memory{ID: canonicalID(), Kind: "note", Content: d.Content, Status: "active", Version: 1, Tags: []string{}, Source: MemorySource{Type: "import"}, ValidFrom: now, Supersedes: []string{}, CreatedAt: now, UpdatedAt: now, Metadata: map[string]any{"legacy_key": legacy}}
		if _, err := s.Catalog.db.ExecContext(ctx, `INSERT INTO memories(id,legacy_key,kind,content,status,version,scope,tags_json,source_json,valid_from,valid_to,supersedes_json,superseded_by,created_at,updated_at,metadata_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, m.ID, legacy, m.Kind, m.Content, m.Status, m.Version, m.Scope, encode(m.Tags), encode(m.Source), now.Format(time.RFC3339Nano), nil, encode(m.Supersedes), nil, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), encode(m.Metadata)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) replayIndexJobs(ctx context.Context) error {
	if s.Catalog == nil {
		return nil
	}
	rows, err := s.Catalog.db.QueryContext(ctx, `SELECT memory_id FROM index_jobs ORDER BY created_at`)
	if err != nil {
		return err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	for _, id := range ids {
		if err := s.applyIndexJob(ctx, id); err != nil {
			return err
		}
	}
	return rows.Err()
}
func (s *Service) applyIndexJob(ctx context.Context, id string) error {
	if s.Catalog == nil || s.Hybrid == nil {
		return ErrUnavailable
	}
	var op, content, metadata string
	err := s.Catalog.db.QueryRowContext(ctx, `SELECT operation,content,metadata_json FROM index_jobs WHERE memory_id=?`, id).Scan(&op, &content, &metadata)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	if op == "delete" {
		// Canonical deletes are filtered by the catalog. Keeping immutable vector
		// revisions avoids corrupting the current govector HNSW graph when its
		// last node is removed; offline compaction can reclaim tombstones.
		err = nil
	} else {
		var meta map[string]any
		_ = json.Unmarshal([]byte(metadata), &meta)
		var version int
		if err = s.Catalog.db.QueryRowContext(ctx, `SELECT version FROM memories WHERE id=?`, id).Scan(&version); err != nil {
			return err
		}
		meta["canonical_id"] = id
		meta["canonical_version"] = fmt.Sprint(version)
		physicalID := fmt.Sprintf("%s@v%d", id, version)
		err = s.Hybrid.Store(ctx, palace.Drawer{ID: physicalID, Content: content, Metadata: stringMetadata(meta)})
	}
	if err != nil {
		return fmt.Errorf("apply index job: %w", err)
	}
	_, err = s.Catalog.db.ExecContext(ctx, `DELETE FROM index_jobs WHERE memory_id=?`, id)
	return err
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
func insertMemory(ctx context.Context, tx *sql.Tx, m Memory, legacy string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO memories(id,legacy_key,kind,content,status,version,scope,tags_json,source_json,valid_from,valid_to,supersedes_json,superseded_by,created_at,updated_at,metadata_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, m.ID, nullString(legacy), m.Kind, m.Content, m.Status, m.Version, m.Scope, encode(m.Tags), encode(m.Source), m.ValidFrom.Format(time.RFC3339Nano), nil, encode(m.Supersedes), nil, m.CreatedAt.Format(time.RFC3339Nano), m.UpdatedAt.Format(time.RFC3339Nano), encode(m.Metadata))
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
func nullString(v string) any {
	if v == "" {
		return nil
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
