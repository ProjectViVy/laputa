package evolution

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/ProjectViVy/laputa/garden/internal/sqliteconn"
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
	db, err := sqliteconn.Open(path)
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
 candidate_json TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS evomap_runs(
 run_id TEXT PRIMARY KEY, mode TEXT NOT NULL, status TEXT NOT NULL,
 asset_ids_json TEXT NOT NULL DEFAULT '[]', signals_json TEXT NOT NULL DEFAULT '[]',
 bundle_json TEXT NOT NULL DEFAULT '{}', error TEXT NOT NULL DEFAULT '',
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL);`
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

func (s *Store) ListRuns(_ context.Context, limit int) ([]EvolutionRun, error) {
	rows, err := s.db.Query(
		`SELECT run_id, status, provider, candidates_json, error, started_at, completed_at FROM evolution_runs ORDER BY started_at DESC, run_id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runs := []EvolutionRun{}
	for rows.Next() {
		var run EvolutionRun
		var candidatesJSON, startedAt string
		var completedAt sql.NullString
		if err := rows.Scan(&run.RunID, &run.Status, &run.Provider, &candidatesJSON, &run.Error, &startedAt, &completedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(candidatesJSON), &run.Candidates)
		run.StartedAt, _ = time.Parse(time.RFC3339Nano, startedAt)
		if completedAt.Valid {
			t, _ := time.Parse(time.RFC3339Nano, completedAt.String)
			run.CompletedAt = &t
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

// HubRun is the provider-local record of one EvoMap hub round trip
// (ADR-0010 §2.1). The hub has no run lifecycle; this is Garden's own
// bookkeeping for publish-mode and discovery-mode runs.
type HubRun struct {
	RunID     string
	Mode      string // "publish" | "discovery"
	Status    string // "running" | "completed" | "failed"
	AssetIDs  []string
	Signals   []string
	Bundle    EvolutionEvidenceBundle
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *Store) SaveHubRun(_ context.Context, r HubRun) error {
	assets, _ := json.Marshal(r.AssetIDs)
	signals, _ := json.Marshal(r.Signals)
	bundle, _ := json.Marshal(r.Bundle)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO evomap_runs(run_id, mode, status, asset_ids_json, signals_json, bundle_json, error, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		r.RunID, r.Mode, r.Status, string(assets), string(signals), string(bundle), r.Error,
		r.CreatedAt.UTC().Format(time.RFC3339Nano), r.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) GetHubRun(_ context.Context, runID string) (HubRun, error) {
	var r HubRun
	var assetsJSON, signalsJSON, bundleJSON, createdAt, updatedAt string
	err := s.db.QueryRow(
		`SELECT run_id, mode, status, asset_ids_json, signals_json, bundle_json, error, created_at, updated_at FROM evomap_runs WHERE run_id = ?`, runID,
	).Scan(&r.RunID, &r.Mode, &r.Status, &assetsJSON, &signalsJSON, &bundleJSON, &r.Error, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return HubRun{}, ErrRunNotFound
	}
	if err != nil {
		return HubRun{}, err
	}
	_ = json.Unmarshal([]byte(assetsJSON), &r.AssetIDs)
	_ = json.Unmarshal([]byte(signalsJSON), &r.Signals)
	_ = json.Unmarshal([]byte(bundleJSON), &r.Bundle)
	r.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	r.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	return r, nil
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

func (s *Store) ListProposals(_ context.Context, limit int) ([]EvolutionProposal, error) {
	rows, err := s.db.Query(
		`SELECT proposal_id, run_id, candidate_id, kind, status, summary, leakage_json, reviewer, review_note, reviewed_at, created_at FROM evolution_proposals ORDER BY created_at DESC, proposal_id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	proposals := []EvolutionProposal{}
	for rows.Next() {
		var p EvolutionProposal
		var leakageJSON, createdAt string
		var reviewedAt sql.NullString
		if err := rows.Scan(&p.ProposalID, &p.RunID, &p.CandidateID, &p.Kind, &p.Status, &p.Summary, &leakageJSON, &p.Reviewer, &p.ReviewNote, &reviewedAt, &createdAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(leakageJSON), &p.LeakageReport)
		p.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		if reviewedAt.Valid {
			t, _ := time.Parse(time.RFC3339Nano, reviewedAt.String)
			p.ReviewedAt = &t
		}
		proposals = append(proposals, p)
	}
	return proposals, rows.Err()
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
