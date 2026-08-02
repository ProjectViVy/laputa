package evolution

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrRunNotFound       = errors.New("evolution: run not found")
	ErrProposalNotFound  = errors.New("evolution: proposal not found")
	ErrCandidateNotFound = errors.New("evolution: candidate not found")
)

type Store struct {
	db *sql.DB
}

func OpenStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	schema := `
CREATE TABLE IF NOT EXISTS evolution_runs(
 run_id TEXT PRIMARY KEY, status TEXT NOT NULL, bundle_json TEXT NOT NULL,
 provider TEXT NOT NULL, candidates_json TEXT NOT NULL DEFAULT '[]',
 error TEXT NOT NULL DEFAULT '', started_at TEXT NOT NULL, completed_at TEXT);
CREATE TABLE IF NOT EXISTS evolution_proposals(
 proposal_id TEXT PRIMARY KEY, run_id TEXT NOT NULL, candidate_id TEXT NOT NULL,
 kind TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending',
 summary TEXT NOT NULL DEFAULT '', leakage_json TEXT NOT NULL DEFAULT '{}',
 reviewer TEXT NOT NULL DEFAULT '', review_note TEXT NOT NULL DEFAULT '',
 reviewed_at TEXT, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS evolution_candidates(
 candidate_id TEXT PRIMARY KEY, run_id TEXT NOT NULL, kind TEXT NOT NULL,
 candidate_json TEXT NOT NULL, created_at TEXT NOT NULL);`
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) SaveRun(_ context.Context, run EvolutionRun) error {
	bundle := "{}"
	candidates, _ := json.Marshal(run.Candidates)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO evolution_runs(run_id, status, bundle_json, provider, candidates_json, error, started_at, completed_at) VALUES(?,?,?,?,?,?,?,?)`,
		run.RunID, run.Status, bundle, run.Provider, string(candidates), run.Error,
		run.StartedAt.UTC().Format(time.RFC3339Nano), formatTimePtr(run.CompletedAt),
	)
	return err
}

func (s *Store) GetRun(_ context.Context, runID string) (EvolutionRun, error) {
	var run EvolutionRun
	var candidatesJSON, startedAt string
	var completedAt sql.NullString
	err := s.db.QueryRow(
		`SELECT run_id, status, provider, candidates_json, error, started_at, completed_at FROM evolution_runs WHERE run_id = ?`, runID,
	).Scan(&run.RunID, &run.Status, &run.Provider, &candidatesJSON, &run.Error, &startedAt, &completedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return EvolutionRun{}, ErrRunNotFound
	}
	if err != nil {
		return EvolutionRun{}, err
	}
	_ = json.Unmarshal([]byte(candidatesJSON), &run.Candidates)
	run.StartedAt, _ = time.Parse(time.RFC3339Nano, startedAt)
	if completedAt.Valid {
		t, _ := time.Parse(time.RFC3339Nano, completedAt.String)
		run.CompletedAt = &t
	}
	return run, nil
}

func (s *Store) UpdateRunStatus(_ context.Context, runID, status, errMsg string) error {
	var completedAt *string
	if status == "completed" || status == "failed" || status == "degraded" {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		completedAt = &now
	}
	_, err := s.db.Exec(
		`UPDATE evolution_runs SET status = ?, error = ?, completed_at = COALESCE(?, completed_at) WHERE run_id = ?`,
		status, errMsg, completedAt, runID,
	)
	return err
}

func (s *Store) SaveCandidate(_ context.Context, c GeneCandidate) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT OR REPLACE INTO evolution_candidates(candidate_id, run_id, kind, candidate_json, created_at) VALUES(?,?,?,?,?)`,
		c.CandidateID, c.RunID, c.Kind, string(data), c.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) GetCandidate(_ context.Context, candidateID string) (GeneCandidate, error) {
	var raw string
	err := s.db.QueryRow(`SELECT candidate_json FROM evolution_candidates WHERE candidate_id = ?`, candidateID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return GeneCandidate{}, ErrCandidateNotFound
	}
	if err != nil {
		return GeneCandidate{}, err
	}
	var c GeneCandidate
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return GeneCandidate{}, err
	}
	return c, nil
}

func (s *Store) SaveProposal(_ context.Context, p EvolutionProposal) error {
	leakage, _ := json.Marshal(p.LeakageReport)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO evolution_proposals(proposal_id, run_id, candidate_id, kind, status, summary, leakage_json, reviewer, review_note, reviewed_at, created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		p.ProposalID, p.RunID, p.CandidateID, p.Kind, p.Status, p.Summary, string(leakage),
		p.Reviewer, p.ReviewNote, formatTimePtr(p.ReviewedAt), p.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) GetProposal(_ context.Context, proposalID string) (EvolutionProposal, error) {
	var p EvolutionProposal
	var leakageJSON, createdAt string
	var reviewedAt sql.NullString
	err := s.db.QueryRow(
		`SELECT proposal_id, run_id, candidate_id, kind, status, summary, leakage_json, reviewer, review_note, reviewed_at, created_at FROM evolution_proposals WHERE proposal_id = ?`, proposalID,
	).Scan(&p.ProposalID, &p.RunID, &p.CandidateID, &p.Kind, &p.Status, &p.Summary, &leakageJSON, &p.Reviewer, &p.ReviewNote, &reviewedAt, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return EvolutionProposal{}, ErrProposalNotFound
	}
	if err != nil {
		return EvolutionProposal{}, err
	}
	_ = json.Unmarshal([]byte(leakageJSON), &p.LeakageReport)
	p.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	if reviewedAt.Valid {
		t, _ := time.Parse(time.RFC3339Nano, reviewedAt.String)
		p.ReviewedAt = &t
	}
	return p, nil
}

func (s *Store) UpdateProposalReview(_ context.Context, proposalID, status, reviewer, note string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(
		`UPDATE evolution_proposals SET status = ?, reviewer = ?, review_note = ?, reviewed_at = ? WHERE proposal_id = ?`,
		status, reviewer, note, now, proposalID,
	)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}

func formatTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}
