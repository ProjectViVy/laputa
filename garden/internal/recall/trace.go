package recall

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var ErrTraceNotFound = errors.New("recall: trace not found")

type TraceStep struct {
	Step       string `json:"step"`
	Status     string `json:"status"`
	DurationMS int64  `json:"duration_ms"`
	ErrorCode  string `json:"error_code,omitempty"`
}

type BudgetConsumption struct {
	BudgetChars     int `json:"budget_chars"`
	UsedChars       int `json:"used_chars"`
	KGQueries       int `json:"kg_queries"`
	TimelineQueries int `json:"timeline_queries"`
	CardSearches    int `json:"card_searches"`
}

type RecallTrace struct {
	TraceID          string            `json:"trace_id"`
	Query            string            `json:"query"`
	Scope            string            `json:"scope"`
	TriggerReason    string            `json:"trigger_reason"`
	SourceSet        []string          `json:"source_set"`
	FilterConditions []string          `json:"filter_conditions"`
	CandidateIDs     []string          `json:"candidate_ids"`
	EvidenceRefs     []string          `json:"evidence_refs"`
	Budget           BudgetConsumption `json:"budget"`
	Degraded         bool              `json:"degraded"`
	FailureState     string            `json:"failure_state,omitempty"`
	Steps            []TraceStep       `json:"steps"`
	StartedAt        string            `json:"started_at"`
	DurationMS       int64             `json:"duration_ms"`
}

type TraceStore struct {
	db *sql.DB
}

func OpenTraceStore(path string) (*TraceStore, error) {
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	schema := `CREATE TABLE IF NOT EXISTS recall_traces(
 trace_id TEXT PRIMARY KEY, trace_json TEXT NOT NULL, created_at TEXT NOT NULL);`
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &TraceStore{db: db}, nil
}

func (s *TraceStore) Save(_ context.Context, trace RecallTrace) error {
	data, err := json.Marshal(trace)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		"INSERT OR REPLACE INTO recall_traces(trace_id, trace_json, created_at) VALUES(?, ?, ?)",
		trace.TraceID, string(data), time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *TraceStore) Get(_ context.Context, traceID string) (RecallTrace, error) {
	var raw string
	err := s.db.QueryRow("SELECT trace_json FROM recall_traces WHERE trace_id = ?", traceID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return RecallTrace{}, ErrTraceNotFound
	}
	if err != nil {
		return RecallTrace{}, err
	}
	var trace RecallTrace
	if err := json.Unmarshal([]byte(raw), &trace); err != nil {
		return RecallTrace{}, err
	}
	return trace, nil
}

func (s *TraceStore) Close() error {
	return s.db.Close()
}

type traceBuilder struct {
	trace RecallTrace
	start time.Time
}

func newTraceBuilder(query, scope, reason string, budget int) *traceBuilder {
	return &traceBuilder{
		start: time.Now(),
		trace: RecallTrace{
			TraceID:          fmt.Sprintf("deep_%d", time.Now().UnixNano()),
			Query:            query,
			Scope:            scope,
			TriggerReason:    reason,
			SourceSet:        []string{},
			FilterConditions: []string{},
			CandidateIDs:     []string{},
			EvidenceRefs:     []string{},
			Steps:            []TraceStep{},
			Budget:           BudgetConsumption{BudgetChars: budget},
			StartedAt:        time.Now().UTC().Format(time.RFC3339Nano),
		},
	}
}

func (b *traceBuilder) step(name, status string, elapsed time.Duration, errCode string) {
	b.trace.Steps = append(b.trace.Steps, TraceStep{
		Step:       name,
		Status:     status,
		DurationMS: elapsed.Milliseconds(),
		ErrorCode:  errCode,
	})
}

func (b *traceBuilder) finish(degraded bool, failure string) RecallTrace {
	b.trace.Degraded = degraded
	b.trace.FailureState = failure
	b.trace.DurationMS = time.Since(b.start).Milliseconds()
	return b.trace
}
