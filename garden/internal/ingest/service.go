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
	"github.com/dashimaki/mentle/facade"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrEventConflict = errors.New("event id was already used with different content")
	ErrNotFound      = errors.New("ingestion not found")
)

type SubmitRequest struct {
	SessionID   string    `json:"session_id"`
	EventID     string    `json:"event_id"`
	Phase       string    `json:"phase"`
	Content     string    `json:"content"`
	ContentHash string    `json:"content_hash"`
	Workspace   string    `json:"workspace,omitempty"`
	OccurredAt  time.Time `json:"occurred_at,omitempty"`
}

type Accepted struct {
	IngestionID string `json:"ingestion_id"`
	SessionID   string `json:"session_id"`
	EventID     string `json:"event_id"`
	Status      string `json:"status"`
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
	Spool    *activity.TransientSpool
	queue    chan string
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

type MemoryWriter interface {
	CreateMemory(context.Context, facade.CreateMemoryRequest, string, string) (facade.Memory, error)
}

func Open(path string, memory MemoryWriter) (*Service, error) {
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_journal_mode=WAL")
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
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{db: db, memory: memory, queue: make(chan string, 128), cancel: cancel}
	s.wg.Add(1)
	go s.worker(ctx)
	rows, _ := db.Query(`SELECT ingestion_id FROM ingestions WHERE status IN ('accepted','running','spooled') ORDER BY created_at`)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				s.queue <- id
			}
		}
	}
	return s, nil
}

func (s *Service) Close() error { s.cancel(); s.wg.Wait(); return s.db.Close() }

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
	err := s.db.QueryRowContext(ctx, `SELECT ingestion_id,session_id,event_id,status,content_hash FROM ingestions WHERE event_id=?`, req.EventID).Scan(&existing.IngestionID, &existing.SessionID, &existing.EventID, &existing.Status, &savedHash)
	if err == nil {
		if savedHash != req.ContentHash {
			return Accepted{}, ErrEventConflict
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Accepted{}, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT ingestion_id,session_id,event_id,status FROM ingestions WHERE session_id=? AND content_hash=?`, req.SessionID, req.ContentHash).Scan(&existing.IngestionID, &existing.SessionID, &existing.EventID, &existing.Status)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Accepted{}, err
	}
	now := time.Now().UTC()
	if req.OccurredAt.IsZero() {
		req.OccurredAt = now
	}
	id := "ing_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = s.db.ExecContext(ctx, `INSERT INTO ingestions(ingestion_id,session_id,event_id,phase,content,content_hash,workspace,occurred_at,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, id, req.SessionID, req.EventID, req.Phase, req.Content, req.ContentHash, req.Workspace, req.OccurredAt.UTC().Format(time.RFC3339Nano), "accepted", now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return Accepted{}, err
	}
	select {
	case s.queue <- id:
	case <-ctx.Done():
		return Accepted{}, ctx.Err()
	}
	if s.Activity != nil {
		_ = s.Activity.Append(ctx, activity.Event{ID: req.EventID, SessionID: req.SessionID, Type: "ingest", Timestamp: req.OccurredAt, Data: map[string]any{"phase": req.Phase, "ingestion_id": id}})
	}
	return Accepted{IngestionID: id, SessionID: req.SessionID, EventID: req.EventID, Status: "accepted"}, nil
}

func (s *Service) Get(ctx context.Context, id string) (Status, error) {
	var st Status
	var memoryID, traceID, warnings, errText sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT ingestion_id,status,memory_id,trace_id,warnings,error FROM ingestions WHERE ingestion_id=?`, id).Scan(&st.IngestionID, &st.Status, &memoryID, &traceID, &warnings, &errText)
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
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-s.queue:
			s.process(ctx, id)
		}
	}
}
func (s *Service) process(ctx context.Context, id string) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, _ = s.db.ExecContext(ctx, `UPDATE ingestions SET status='running',updated_at=? WHERE ingestion_id=?`, now, id)
	var session, event, content, hash string
	if err := s.db.QueryRowContext(ctx, `SELECT session_id,event_id,content,content_hash FROM ingestions WHERE ingestion_id=?`, id).Scan(&session, &event, &content, &hash); err != nil {
		s.fail(ctx, id, err)
		return
	}
	if s.memory == nil {
		s.spool(ctx, id, session, event, content, hash)
		return
	}
	m, err := s.memory.CreateMemory(ctx, facade.CreateMemoryRequest{Content: content, Kind: "source_artifact", Source: facade.MemorySource{Type: "session", SessionID: session, EventID: event}, Metadata: map[string]any{"content_hash": hash, "lifecycle": "stm", "collection": "working"}}, "session:"+event, hash)
	if err != nil {
		s.spool(ctx, id, session, event, content, hash)
		return
	}
	now = time.Now().UTC().Format(time.RFC3339Nano)
	_, _ = s.db.ExecContext(ctx, `UPDATE ingestions SET status='completed',memory_id=?,trace_id=?,error=NULL,updated_at=? WHERE ingestion_id=?`, m.ID, "run_"+id, now, id)
}

func (s *Service) spool(ctx context.Context, id, session, event, content, hash string) {
	if s.Spool != nil {
		_ = s.Spool.Append(ctx, activity.TransientEntry{EventID: event, SessionID: session, ContentHash: hash, Content: content, Kind: "source_artifact"})
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE ingestions SET status='spooled',error=?,updated_at=? WHERE ingestion_id=?`, "mentle unavailable; spooled", time.Now().UTC().Format(time.RFC3339Nano), id)
}

func (s *Service) fail(ctx context.Context, id string, err error) {
	_, _ = s.db.ExecContext(ctx, `UPDATE ingestions SET status='failed',error=?,updated_at=? WHERE ingestion_id=?`, fmt.Sprint(err), time.Now().UTC().Format(time.RFC3339Nano), id)
}
