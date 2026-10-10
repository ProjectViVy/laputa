package ingest

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dashimaki/garden/internal/activity"
	"github.com/dashimaki/garden/internal/sqliteconn"
	"github.com/dashimaki/garden/memory"
	"github.com/dashimaki/laputa/actmem"
	"github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/mentle/facade"
	"github.com/google/uuid"
)

var (
	ErrEventConflict = errors.New("event id was already used with different content")
	ErrNotFound      = errors.New("ingestion not found")
)

type SubmitRequest struct {
	Activity    *CaptureActivity `json:"-"`
	SessionID   string           `json:"session_id"`
	EventID     string           `json:"event_id"`
	Phase       string           `json:"phase"`
	Content     string           `json:"content"`
	ContentHash string           `json:"content_hash"`
	Workspace   string           `json:"workspace,omitempty"`
	OccurredAt  time.Time        `json:"occurred_at,omitempty"`
}

type Accepted struct {
	IngestionID string `json:"ingestion_id"`
	SessionID   string `json:"session_id"`
	EventID     string `json:"event_id"`
	Status      string `json:"status"`
	// Seq is the durable ledger sequence assigned at commit; deduped
	// deliveries return the original seq, never a new one.
	Seq uint64 `json:"seq"`
}

type Status struct {
	IngestionID string   `json:"ingestion_id"`
	Status      string   `json:"status"`
	MemoryIDs   []string `json:"memory_ids"`
	TraceID     string   `json:"trace_id,omitempty"`
	Warnings    []string `json:"warnings"`
	Error       *string  `json:"error"`
}

type Service struct {
	db       *sql.DB
	memory   MemoryWriter
	Activity *activity.Store
	Actmem   *actmem.Store
	Spool    *activity.TransientSpool
	// ProfileID binds ingestions to the host profile's subject scope; set
	// by composition before Start.
	ProfileID  string
	queue      chan string
	mu         sync.Mutex
	started    bool
	closed     bool
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	activityMu sync.Mutex // serializes projection and session archive
	workerID   string
}

type MemoryWriter interface {
	CreateMemory(context.Context, facade.CreateMemoryRequest, string, string) (facade.Memory, error)
}

// SessionStateStore is an optional facade capability for durable ingest
// boundaries. Keeping it separate from MemoryWriter lets lightweight test and
// transient writers remain useful without granting them a raw DB handle.
type SessionStateStore interface {
	SaveSessionCursor(sessionID, timestamp string) error
	AcquireSessionLease(ctx context.Context, sessionID, owner string, ttl time.Duration) (bool, error)
	ReleaseSessionLease(ctx context.Context, sessionID, owner string) error
}

func Open(path string, memory MemoryWriter) (*Service, error) {
	s, err := OpenPaused(path, memory)
	if err != nil {
		return nil, err
	}
	if err := s.Start(); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

// OpenPaused prepares durable ingestion without starting the worker or replaying rows.
func OpenPaused(path string, memory MemoryWriter) (*Service, error) {
	db, err := sqliteconn.Open(path)
	if err != nil {
		return nil, err
	}
	schema := `CREATE TABLE IF NOT EXISTS ingestions(
 ingestion_id TEXT PRIMARY KEY, session_id TEXT NOT NULL, event_id TEXT NOT NULL UNIQUE,
 phase TEXT NOT NULL, content TEXT, content_hash TEXT NOT NULL, workspace TEXT,
 occurred_at TEXT NOT NULL, status TEXT NOT NULL, memory_id TEXT, trace_id TEXT,
 warnings TEXT NOT NULL DEFAULT '', error TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
 UNIQUE(session_id,content_hash));
CREATE INDEX IF NOT EXISTS ingestion_status ON ingestions(status,created_at);`
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	if err := initializeActivityColumn(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{db: db, memory: memory, queue: make(chan string, 128), ctx: ctx, cancel: cancel, workerID: "ingest-" + strings.ReplaceAll(uuid.NewString(), "-", "")}, nil
}

// Start starts the worker and replays durable pending rows exactly once per service.
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("ingestion service is closed")
	}
	if s.started {
		return nil
	}
	rows, err := s.db.Query(`SELECT ingestion_id FROM ingestions WHERE status IN ('accepted','running','spooled') OR (activity_json<>'' AND json_extract(activity_json,'$.status')<>'applied') ORDER BY rowid`)
	if err != nil {
		return err
	}
	var pending []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	s.started = true
	s.wg.Add(1)
	go s.worker(s.ctx)
	for _, id := range pending {
		select {
		case s.queue <- id:
		case <-s.ctx.Done():
			return s.ctx.Err()
		}
	}
	return nil
}

func (s *Service) Close() error {
	s.mu.Lock()
	s.closed = true
	s.cancel()
	s.mu.Unlock()
	s.wg.Wait()
	return s.db.Close()
}

func (s *Service) Submit(ctx context.Context, req SubmitRequest) (Accepted, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.EventID = strings.TrimSpace(req.EventID)
	req.ContentHash = strings.TrimSpace(req.ContentHash)
	if req.SessionID == "" || req.EventID == "" || strings.TrimSpace(req.Content) == "" || req.ContentHash == "" {
		return Accepted{}, errors.New("session_id, event_id, content, and content_hash are required")
	}
	if req.Phase != "precompact" && req.Phase != "session_end" {
		return Accepted{}, errors.New("phase must be precompact or session_end")
	}
	sum := sha256.Sum256([]byte(req.Content))
	expected := "sha256:" + hex.EncodeToString(sum[:])
	if req.ContentHash != expected {
		return Accepted{}, errors.New("content_hash does not match content")
	}
	var existing Accepted
	var savedHash string
	err := s.db.QueryRowContext(ctx, `SELECT rowid,ingestion_id,session_id,event_id,status,content_hash FROM ingestions WHERE event_id=?`, req.EventID).Scan(&existing.Seq, &existing.IngestionID, &existing.SessionID, &existing.EventID, &existing.Status, &savedHash)
	if err == nil {
		if savedHash != req.ContentHash {
			return Accepted{}, ErrEventConflict
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Accepted{}, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT rowid,ingestion_id,session_id,event_id,status FROM ingestions WHERE session_id=? AND content_hash=?`, req.SessionID, req.ContentHash).Scan(&existing.Seq, &existing.IngestionID, &existing.SessionID, &existing.EventID, &existing.Status)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Accepted{}, err
	}
	activityJSON, err := initialActivity(req.Activity)
	if err != nil {
		return Accepted{}, err
	}
	if req.Activity != nil && (s.Actmem == nil || strings.TrimSpace(s.ProfileID) == "") {
		return Accepted{}, errors.New("native ACTMEM capture binding unavailable")
	}
	now := time.Now().UTC()
	if req.OccurredAt.IsZero() {
		req.OccurredAt = now
	}
	id := "ing_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err = s.db.ExecContext(ctx, `INSERT INTO ingestions(ingestion_id,session_id,event_id,phase,content,content_hash,workspace,occurred_at,status,created_at,updated_at,activity_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, id, req.SessionID, req.EventID, req.Phase, req.Content, req.ContentHash, req.Workspace, req.OccurredAt.UTC().Format(time.RFC3339Nano), "accepted", now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), activityJSON)
	if err != nil {
		return Accepted{}, err
	}
	var seq uint64
	if err = s.db.QueryRowContext(ctx, `SELECT rowid FROM ingestions WHERE ingestion_id=?`, id).Scan(&seq); err != nil {
		return Accepted{}, err
	}
	if s.started {
		select {
		case s.queue <- id:
		case <-ctx.Done():
			return Accepted{}, ctx.Err()
		}
	}
	if s.Activity != nil {
		_ = s.Activity.Append(ctx, activity.Event{ID: req.EventID, SessionID: req.SessionID, Type: "ingest", Timestamp: req.OccurredAt, Data: map[string]any{"phase": req.Phase, "ingestion_id": id}})
	}
	return Accepted{IngestionID: id, SessionID: req.SessionID, EventID: req.EventID, Status: "accepted", Seq: seq}, nil
}

// WindowRow is one committed ingestion inside an (after, through] window of
// the durable ledger sequence.
type WindowRow struct {
	Seq         uint64    `json:"seq"`
	IngestionID string    `json:"ingestion_id"`
	SessionID   string    `json:"session_id"`
	EventID     string    `json:"event_id"`
	Phase       string    `json:"phase"`
	Content     string    `json:"content"`
	ContentHash string    `json:"content_hash"`
	Workspace   string    `json:"workspace"`
	Status      string    `json:"status"`
	OccurredAt  time.Time `json:"occurred_at"`
}

// Window returns committed ingestions in (after, through] for one workspace
// in ledger order. The workspace string is the binding's raw workspace id
// (empty selects personal-scope captures), never a decoded scope.
func (s *Service) Window(ctx context.Context, workspace string, after, through uint64) ([]WindowRow, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT rowid,ingestion_id,session_id,event_id,phase,content,content_hash,workspace,status,occurred_at FROM ingestions WHERE rowid>? AND rowid<=? AND workspace=? ORDER BY rowid ASC`, int64(after), int64(through), workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WindowRow
	for rows.Next() {
		var row WindowRow
		var occurred string
		if err := rows.Scan(&row.Seq, &row.IngestionID, &row.SessionID, &row.EventID, &row.Phase, &row.Content, &row.ContentHash, &row.Workspace, &row.Status, &occurred); err != nil {
			return nil, err
		}
		row.OccurredAt, _ = time.Parse(time.RFC3339Nano, occurred)
		out = append(out, row)
	}
	return out, rows.Err()
}

// HighWatermark returns the highest committed ledger sequence for the
// workspace; zero means nothing captured yet.
func (s *Service) HighWatermark(ctx context.Context, workspace string) (uint64, error) {
	var seq sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `SELECT MAX(rowid) FROM ingestions WHERE workspace=? AND rowid<COALESCE((SELECT MIN(rowid) FROM ingestions WHERE workspace=? AND activity_json<>'' AND COALESCE(json_extract(activity_json,'$.status'),'unknown')<>'applied'),9223372036854775807)`, workspace, workspace).Scan(&seq); err != nil {
		return 0, err
	}
	if !seq.Valid {
		return 0, nil
	}
	return uint64(seq.Int64), nil
}

func (s *Service) Get(ctx context.Context, id string) (Status, error) {
	return s.get(ctx, `SELECT ingestion_id,status,memory_id,trace_id,warnings,error FROM ingestions WHERE ingestion_id=?`, id)
}

// GetByIdentity reads status only when all three durable ingestion identifiers match.
// An identity mismatch is indistinguishable from a missing ingestion to callers.
func (s *Service) GetByIdentity(ctx context.Context, id, sessionID, eventID string) (Status, error) {
	return s.get(ctx, `SELECT ingestion_id,status,memory_id,trace_id,warnings,error FROM ingestions WHERE ingestion_id=? AND session_id=? AND event_id=?`, id, sessionID, eventID)
}

func (s *Service) get(ctx context.Context, query string, args ...any) (Status, error) {
	var st Status
	var memoryID, traceID, warnings, errText sql.NullString
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&st.IngestionID, &st.Status, &memoryID, &traceID, &warnings, &errText)
	if errors.Is(err, sql.ErrNoRows) {
		return st, ErrNotFound
	}
	if err != nil {
		return st, err
	}
	st.MemoryIDs = []string{}
	st.Warnings = []string{}
	if memoryID.Valid {
		st.MemoryIDs = []string{memoryID.String}
	}
	if traceID.Valid {
		st.TraceID = traceID.String
	}
	if warnings.Valid && warnings.String != "" {
		st.Warnings = strings.Split(warnings.String, "\n")
	}
	if errText.Valid {
		st.Error = &errText.String
	}
	return st, nil
}

func (s *Service) worker(ctx context.Context) {
	defer s.wg.Done()
	// The timer exists only while a durable projection needs recovery. It is
	// owned by this service and stops on Close; no new delivery is required.
	var timer *time.Timer
	var retry <-chan time.Time
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	drain := func() {
		if err := s.drainActivity(ctx); err != nil && ctx.Err() == nil {
			if timer == nil {
				timer = time.NewTimer(time.Second)
			} else {
				timer.Reset(time.Second)
			}
			retry = timer.C
		} else {
			if timer != nil {
				timer.Stop()
			}
			retry = nil
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-retry:
			drain()
		case id := <-s.queue:
			status, err := s.Get(ctx, id)
			if err == nil && status.Status != "completed" && status.Status != "failed" {
				s.process(ctx, id)
			}
			drain()
		}
	}
}
func (s *Service) process(ctx context.Context, id string) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, _ = s.db.ExecContext(ctx, `UPDATE ingestions SET status='running',updated_at=? WHERE ingestion_id=?`, now, id)
	var session, event, content, hash, occurredAt, workspace string
	if err := s.db.QueryRowContext(ctx, `SELECT session_id,event_id,content,content_hash,occurred_at,workspace FROM ingestions WHERE ingestion_id=?`, id).Scan(&session, &event, &content, &hash, &occurredAt, &workspace); err != nil {
		s.fail(ctx, id, err)
		return
	}
	if s.memory == nil {
		s.spool(ctx, id, session, event, content, hash)
		return
	}
	var state SessionStateStore
	if candidate, ok := s.memory.(SessionStateStore); ok {
		state = candidate
		acquired, err := state.AcquireSessionLease(ctx, session, s.workerID, time.Minute)
		if err != nil {
			s.spool(ctx, id, session, event, content, hash)
			return
		}
		if !acquired {
			// Another process owns this session. Leave the row accepted so the
			// owner can finish it; its durable lease is the retry boundary.
			_, _ = s.db.ExecContext(ctx, `UPDATE ingestions SET status='accepted',updated_at=? WHERE ingestion_id=?`, time.Now().UTC().Format(time.RFC3339Nano), id)
			return
		}
		defer func() { _ = state.ReleaseSessionLease(context.Background(), session, s.workerID) }()
	}
	m, err := s.memory.CreateMemory(ctx, facade.CreateMemoryRequest{Content: content, Kind: "source_artifact", Scope: s.scopeFor(workspace), Source: facade.MemorySource{Type: "session", SessionID: session, EventID: event}, Metadata: map[string]any{"content_hash": hash, "lifecycle": "stm", "collection": "working"}}, "session:"+event, hash)
	if err != nil {
		s.spool(ctx, id, session, event, content, hash)
		return
	}
	if state != nil {
		if err := state.SaveSessionCursor(session, occurredAt); err != nil {
			warning := "session cursor save failed: " + err.Error()
			_, _ = s.db.ExecContext(ctx, `UPDATE ingestions SET warnings=?,updated_at=? WHERE ingestion_id=?`, warning, time.Now().UTC().Format(time.RFC3339Nano), id)
		}
	}
	now = time.Now().UTC().Format(time.RFC3339Nano)
	_, _ = s.db.ExecContext(ctx, `UPDATE ingestions SET status='completed',memory_id=?,trace_id=?,error=NULL,updated_at=? WHERE ingestion_id=?`, m.ID, "run_"+id, now, id)
}

// scopeFor encodes the profile's personal or workspace scope for one
// ingestion row. The value is derived server-side from the stored
// workspace binding — the request body never supplies a scope string.
func (s *Service) scopeFor(workspace string) string {
	scope := evolution.Scope{SubjectID: s.ProfileID, Kind: evolution.ScopePersonal}
	if workspace != "" {
		scope.Kind = evolution.ScopeWorkspace
		scope.WorkspaceID = workspace
	}
	return memory.EncodeScope(scope)
}

func (s *Service) spool(ctx context.Context, id, session, event, content, hash string) {
	var workspace string
	_ = s.db.QueryRowContext(ctx, `SELECT workspace FROM ingestions WHERE ingestion_id=?`, id).Scan(&workspace)
	if s.Spool != nil {
		_ = s.Spool.Append(ctx, activity.TransientEntry{EventID: event, SessionID: session, ContentHash: hash, Content: content, Kind: "source_artifact", Scope: s.scopeFor(workspace)})
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE ingestions SET status='spooled',error=?,updated_at=? WHERE ingestion_id=?`, "mentle unavailable; spooled", time.Now().UTC().Format(time.RFC3339Nano), id)
}

func (s *Service) fail(ctx context.Context, id string, err error) {
	_, _ = s.db.ExecContext(ctx, `UPDATE ingestions SET status='failed',error=?,updated_at=? WHERE ingestion_id=?`, fmt.Sprint(err), time.Now().UTC().Format(time.RFC3339Nano), id)
}

// AcceptedByEvent rejoins an already committed acceptance without resubmitting
// changed content. Callers must supply their trusted session/event identity.
func (s *Service) AcceptedByEvent(ctx context.Context, sessionID, eventID string) (Accepted, error) {
	var accepted Accepted
	err := s.db.QueryRowContext(ctx, `SELECT rowid,ingestion_id,session_id,event_id,status FROM ingestions WHERE session_id=? AND event_id=?`, sessionID, eventID).Scan(&accepted.Seq, &accepted.IngestionID, &accepted.SessionID, &accepted.EventID, &accepted.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return Accepted{}, ErrNotFound
	}
	return accepted, err
}
