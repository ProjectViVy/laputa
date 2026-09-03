package facade

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dashimaki/mentle/internal/embedder"
	"github.com/dashimaki/mentle/internal/palace"
	"github.com/dashimaki/mentle/internal/search"
	"github.com/google/uuid"
)

const (
	IndexJobPending = "pending"
	IndexJobLeased  = "leased"
	IndexJobRetry   = "retry"
	IndexJobFailed  = "failed"

	indexJobMaxAttempts = 5
	indexJobLease       = 30 * time.Second
)

var (
	ErrIndexHealthUnavailable = errors.New("index health unavailable")
	ErrIndexRebuildFailed     = errors.New("derived index rebuild failed")
)

// IndexJob is the durable derived-index outbox record. The canonical memory
// row is authoritative; this record only describes work that may be replayed.
type IndexJob struct {
	ID               string     `json:"id"`
	MemoryID         string     `json:"memory_id"`
	CanonicalVersion int        `json:"canonical_version"`
	Operation        string     `json:"operation"`
	Content          string     `json:"content,omitempty"`
	Metadata         string     `json:"metadata,omitempty"`
	State            string     `json:"state"`
	Attempts         int        `json:"attempts"`
	LastError        string     `json:"last_error,omitempty"`
	NextAttemptAt    *time.Time `json:"next_attempt_at"`
	LeaseOwner       string     `json:"lease_owner,omitempty"`
	LeaseUntil       *time.Time `json:"lease_until"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// MemoryMutationResult makes the canonical commit and derived-index state
// explicit to adapters. A pending index never turns a committed memory into
// an HTTP error.
type MemoryMutationResult struct {
	Memory     Memory  `json:"memory"`
	IndexState string  `json:"index_state"`
	IndexJobID *string `json:"index_job_id"`
}

type RebuildSummary struct {
	Status                 string     `json:"status"`
	StartedAt              *time.Time `json:"started_at"`
	CompletedAt            *time.Time `json:"completed_at"`
	CanonicalSnapshotCount *int       `json:"canonical_snapshot_count"`
	ErrorCode              *string    `json:"error_code"`
}

// IndexHealth is a live comparison between canonical SQLite and its
// disposable derived indexes.
type IndexHealth struct {
	Status                    string             `json:"status"`
	ObservedAt                time.Time          `json:"observed_at"`
	Reasons                   []string           `json:"reasons"`
	CanonicalActiveCount      int                `json:"canonical_active_count"`
	VectorActiveCount         int                `json:"vector_active_count"`
	VectorPhysicalCount       int                `json:"vector_physical_count"`
	BM25Count                 int                `json:"bm25_count"`
	TombstoneCount            int                `json:"tombstone_count"`
	TombstoneRatio            float64            `json:"tombstone_ratio"`
	EmbeddingIdentity         *embedder.Identity `json:"embedding_identity"`
	ExpectedEmbeddingIdentity *embedder.Identity `json:"expected_embedding_identity"`
	PendingJobs               int                `json:"pending_jobs"`
	FailedJobs                int                `json:"failed_jobs"`
	OldestPendingAgeMS        *int64             `json:"oldest_pending_age_ms"`
	LastRebuild               RebuildSummary     `json:"last_rebuild"`
}

func indexJobID() string {
	return "idx_" + strings.ReplaceAll(uuid.NewString(), "-", "")
}

func migrateIndexJobs(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(index_jobs)")
	if err != nil {
		return err
	}
	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			rows.Close()
			return err
		}
		columns[name] = true
	}
	if err := rows.Close(); err != nil {
		return err
	}
	additions := []struct {
		name string
		def  string
	}{
		{"job_id", "TEXT"},
		{"canonical_version", "INTEGER NOT NULL DEFAULT 1"},
		{"state", "TEXT NOT NULL DEFAULT 'pending'"},
		{"attempts", "INTEGER NOT NULL DEFAULT 0"},
		{"last_error", "TEXT NOT NULL DEFAULT ''"},
		{"next_attempt_at", "TEXT NOT NULL DEFAULT ''"},
		{"lease_owner", "TEXT NOT NULL DEFAULT ''"},
		{"lease_until", "TEXT NOT NULL DEFAULT ''"},
		{"updated_at", "TEXT NOT NULL DEFAULT ''"},
	}
	for _, addition := range additions {
		if columns[addition.name] {
			continue
		}
		if _, err := db.Exec("ALTER TABLE index_jobs ADD COLUMN " + addition.name + " " + addition.def); err != nil {
			return fmt.Errorf("add index_jobs.%s: %w", addition.name, err)
		}
	}
	if _, err := db.Exec("UPDATE index_jobs SET job_id='idx_legacy_' || memory_id WHERE job_id IS NULL OR job_id=''"); err != nil {
		return err
	}
	if _, err := db.Exec("UPDATE index_jobs SET updated_at=created_at WHERE updated_at IS NULL OR updated_at=''"); err != nil {
		return err
	}
	if _, err := db.Exec("UPDATE index_jobs SET canonical_version=COALESCE((SELECT version FROM memories WHERE memories.id=index_jobs.memory_id), 1) WHERE canonical_version IS NULL OR canonical_version=0"); err != nil {
		return err
	}
	_, err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS index_jobs_job_id ON index_jobs(job_id)")
	return err
}

func enqueueIndexJobTx(ctx context.Context, tx *sql.Tx, memoryID string, version int, operation, content, metadata string, now time.Time) (string, error) {
	jobID := indexJobID()
	_, err := tx.ExecContext(ctx, `
		INSERT INTO index_jobs(job_id,memory_id,canonical_version,operation,content,metadata_json,state,attempts,last_error,next_attempt_at,lease_owner,lease_until,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,0,'','','','',?,?)
		ON CONFLICT(memory_id) DO UPDATE SET
			job_id=excluded.job_id,
			canonical_version=excluded.canonical_version,
			operation=excluded.operation,
			content=excluded.content,
			metadata_json=excluded.metadata_json,
			state='pending',
			attempts=0,
			last_error='',
			next_attempt_at='',
			lease_owner='',
			lease_until='',
			updated_at=excluded.updated_at
	`, jobID, memoryID, version, operation, content, metadata, IndexJobPending, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	return jobID, err
}

func scanIndexJob(row interface{ Scan(...any) error }) (IndexJob, error) {
	var job IndexJob
	var nextAttempt, leaseUntil, created, updated string
	if err := row.Scan(&job.ID, &job.MemoryID, &job.CanonicalVersion, &job.Operation, &job.Content, &job.Metadata, &job.State, &job.Attempts, &job.LastError, &nextAttempt, &job.LeaseOwner, &leaseUntil, &created, &updated); err != nil {
		return job, err
	}
	job.NextAttemptAt = parseOptionalTime(nextAttempt)
	job.LeaseUntil = parseOptionalTime(leaseUntil)
	job.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	job.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return job, nil
}

const indexJobSelect = `SELECT job_id,memory_id,canonical_version,operation,content,metadata_json,state,attempts,last_error,next_attempt_at,lease_owner,lease_until,created_at,updated_at FROM index_jobs`

func parseOptionalTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil
	}
	return &t
}

func (c *Catalog) GetIndexJob(ctx context.Context, memoryID string) (*IndexJob, error) {
	if c == nil || c.db == nil {
		return nil, ErrUnavailable
	}
	job, err := scanIndexJob(c.db.QueryRowContext(ctx, indexJobSelect+` WHERE memory_id=?`, memoryID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (c *Catalog) ListIndexJobs(ctx context.Context) ([]IndexJob, error) {
	if c == nil || c.db == nil {
		return nil, ErrUnavailable
	}
	rows, err := c.db.QueryContext(ctx, indexJobSelect+` ORDER BY created_at,memory_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []IndexJob{}
	for rows.Next() {
		job, err := scanIndexJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	return out, rows.Err()
}

func (c *Catalog) claimIndexJob(ctx context.Context, memoryID string) (IndexJob, bool, error) {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return IndexJob{}, false, err
	}
	defer tx.Rollback()
	job, err := scanIndexJob(tx.QueryRowContext(ctx, indexJobSelect+` WHERE memory_id=?`, memoryID))
	if errors.Is(err, sql.ErrNoRows) {
		return IndexJob{}, false, nil
	}
	if err != nil {
		return IndexJob{}, false, err
	}
	now := time.Now().UTC()
	if job.State == IndexJobFailed {
		return IndexJob{}, false, nil
	}
	if job.State == IndexJobLeased && job.LeaseUntil != nil && job.LeaseUntil.After(now) {
		return IndexJob{}, false, nil
	}
	if job.State == IndexJobRetry && job.NextAttemptAt != nil && job.NextAttemptAt.After(now) {
		return IndexJob{}, false, nil
	}
	owner := "mentle-" + strings.ReplaceAll(uuid.NewString(), "-", "")
	leaseUntil := now.Add(indexJobLease)
	result, err := tx.ExecContext(ctx, `
		UPDATE index_jobs
		SET state=?, attempts=attempts+1, lease_owner=?, lease_until=?, updated_at=?
		WHERE memory_id=? AND job_id=? AND (
			state IN ('pending','retry') OR
			(state='leased' AND (lease_until='' OR lease_until IS NULL OR lease_until<=?))
		)
	`, IndexJobLeased, owner, leaseUntil.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), memoryID, job.ID, now.Format(time.RFC3339Nano))
	if err != nil {
		return IndexJob{}, false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return IndexJob{}, false, err
	}
	if affected != 1 {
		return IndexJob{}, false, nil
	}
	if err := tx.Commit(); err != nil {
		return IndexJob{}, false, err
	}
	job.State = IndexJobLeased
	job.Attempts++
	job.LeaseOwner = owner
	job.LeaseUntil = &leaseUntil
	job.UpdatedAt = now
	return job, true, nil
}

func (c *Catalog) completeIndexJob(ctx context.Context, job IndexJob) error {
	_, err := c.db.ExecContext(ctx, `DELETE FROM index_jobs WHERE memory_id=? AND job_id=? AND state=? AND lease_owner=?`, job.MemoryID, job.ID, IndexJobLeased, job.LeaseOwner)
	return err
}

func (c *Catalog) failIndexJob(ctx context.Context, job IndexJob, cause error) error {
	now := time.Now().UTC()
	state := IndexJobRetry
	next := now.Add(indexBackoff(job.Attempts))
	if job.Attempts >= indexJobMaxAttempts {
		state = IndexJobFailed
		next = now
	}
	_, err := c.db.ExecContext(ctx, `
		UPDATE index_jobs
		SET state=?, last_error=?, next_attempt_at=?, lease_owner='', lease_until='', updated_at=?
		WHERE memory_id=? AND job_id=? AND state=? AND lease_owner=?
	`, state, cause.Error(), next.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), job.MemoryID, job.ID, IndexJobLeased, job.LeaseOwner)
	return err
}

func indexBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 6 {
		attempt = 6
	}
	return time.Duration(1<<uint(attempt-1)) * time.Second
}

func (s *Service) indexStatus(ctx context.Context, memoryID string) (string, *string) {
	if s == nil || s.Catalog == nil {
		return "index_pending", nil
	}
	job, err := s.Catalog.GetIndexJob(ctx, memoryID)
	if err != nil || job == nil {
		return "applied", nil
	}
	id := job.ID
	switch job.State {
	case IndexJobFailed:
		return "index_failed", &id
	case IndexJobLeased:
		return "index_processing", &id
	default:
		return "index_pending", &id
	}
}

func contractIndexState(state string) string {
	if state == "applied" {
		return "applied"
	}
	return "index_pending"
}

func (s *Service) CreateMemoryWithIndexStatus(ctx context.Context, req CreateMemoryRequest, idempotencyKey, bodyHash string) (MemoryMutationResult, error) {
	memory, err := s.CreateMemory(ctx, req, idempotencyKey, bodyHash)
	if err != nil {
		return MemoryMutationResult{}, err
	}
	state, jobID := s.indexStatus(ctx, memory.ID)
	return MemoryMutationResult{Memory: memory, IndexState: contractIndexState(state), IndexJobID: jobID}, nil
}

func (s *Service) UpdateMemoryWithIndexStatus(ctx context.Context, id string, req UpdateMemoryRequest) (MemoryMutationResult, error) {
	memory, err := s.UpdateMemory(ctx, id, req)
	if err != nil {
		return MemoryMutationResult{}, err
	}
	state, jobID := s.indexStatus(ctx, memory.ID)
	return MemoryMutationResult{Memory: memory, IndexState: contractIndexState(state), IndexJobID: jobID}, nil
}

func (s *Service) applyIndexJob(ctx context.Context, memoryID string) error {
	if s.Catalog == nil || s.Hybrid == nil {
		return ErrUnavailable
	}
	job, claimed, err := s.Catalog.claimIndexJob(ctx, memoryID)
	if err != nil || !claimed {
		return err
	}
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()

	if job.Operation == "upsert" {
		if err := s.ensureEmbeddingIdentity(); err != nil {
			_ = s.Catalog.failIndexJob(ctx, job, err)
			return err
		}
		meta := map[string]any{}
		if err := decodeMetadata(job.Metadata, &meta); err != nil {
			_ = s.Catalog.failIndexJob(ctx, job, err)
			return err
		}
		meta["canonical_id"] = job.MemoryID
		meta["canonical_version"] = fmt.Sprint(job.CanonicalVersion)
		physicalID := fmt.Sprintf("%s@v%d", job.MemoryID, job.CanonicalVersion)
		wing, _ := meta["wing"].(string)
		room, _ := meta["room"].(string)
		source, _ := meta["source_file"].(string)
		err = s.Hybrid.Store(ctx, palace.Drawer{ID: physicalID, Content: job.Content, Wing: wing, Room: room, SourceFile: source, Metadata: stringMetadata(meta)})
		if err == nil {
			// Updates may leave older physical revisions behind. Prune them
			// after the current version is durable so stale points cannot
			// crowd out current canonical retrieval.
			err = s.Hybrid.PruneCanonicalRevisions(ctx, job.MemoryID, physicalID)
		}
	} else if job.Operation == "delete" {
		err = s.Hybrid.PruneCanonicalRevisions(ctx, job.MemoryID, "")
	} else {
		err = fmt.Errorf("unknown index operation %q", job.Operation)
	}
	if err != nil {
		_ = s.Catalog.failIndexJob(ctx, job, err)
		return fmt.Errorf("apply index job: %w", err)
	}
	return s.Catalog.completeIndexJob(ctx, job)
}

func (s *Service) replayIndexJobs(ctx context.Context) error {
	if s.Catalog == nil {
		return nil
	}
	rows, err := s.Catalog.db.QueryContext(ctx, `SELECT memory_id FROM index_jobs ORDER BY created_at,memory_id`)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		// A poison job is recorded as retry/failed and must not prevent
		// startup or unrelated jobs from being replayed.
		_ = s.applyIndexJob(ctx, id)
	}
	return nil
}

// DrainIndexJobs replays currently eligible outbox records. It is safe to
// call on startup or from a bounded maintenance loop.
func (s *Service) DrainIndexJobs(ctx context.Context) error {
	return s.replayIndexJobs(ctx)
}

func decodeMetadata(raw string, target *map[string]any) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return err
	}
	if *target == nil {
		*target = map[string]any{}
	}
	return nil
}

func (c *Catalog) rebuildSummary(ctx context.Context) RebuildSummary {
	summary := RebuildSummary{Status: "never"}
	if c == nil || c.db == nil {
		return summary
	}
	var status, started, completed, errCode string
	var count sql.NullInt64
	err := c.db.QueryRowContext(ctx, `SELECT status,started_at,completed_at,canonical_snapshot_count,error_code FROM index_rebuilds WHERE id=1`).
		Scan(&status, &started, &completed, &count, &errCode)
	if errors.Is(err, sql.ErrNoRows) {
		return summary
	}
	if err != nil {
		return summary
	}
	summary.Status = status
	summary.StartedAt = parseOptionalTime(started)
	summary.CompletedAt = parseOptionalTime(completed)
	if count.Valid {
		value := int(count.Int64)
		summary.CanonicalSnapshotCount = &value
	}
	if errCode != "" {
		summary.ErrorCode = &errCode
	}
	return summary
}

func sortedReasons(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

type canonicalSnapshot struct {
	Memories []Memory
	Drawers  []search.Drawer
}

func (s *Service) readCanonicalSnapshot(ctx context.Context) (canonicalSnapshot, error) {
	if s == nil || s.Catalog == nil {
		return canonicalSnapshot{}, ErrUnavailable
	}
	rows, err := s.Catalog.db.QueryContext(ctx, memorySelect+` WHERE status='active' AND superseded_by IS NULL ORDER BY id`)
	if err != nil {
		return canonicalSnapshot{}, err
	}
	defer rows.Close()
	snapshot := canonicalSnapshot{Memories: []Memory{}, Drawers: []search.Drawer{}}
	for rows.Next() {
		memory, err := scanMemory(rows)
		if err != nil {
			return canonicalSnapshot{}, err
		}
		snapshot.Memories = append(snapshot.Memories, memory)
		metadata := make(map[string]string, len(memory.Metadata)+4)
		for key, value := range memory.Metadata {
			if text, ok := value.(string); ok {
				metadata[key] = text
			}
		}
		metadata["canonical_id"] = memory.ID
		metadata["canonical_version"] = fmt.Sprint(memory.Version)
		snapshot.Drawers = append(snapshot.Drawers, search.Drawer{
			ID:       fmt.Sprintf("%s@v%d", memory.ID, memory.Version),
			Content:  memory.Content,
			Metadata: metadata,
		})
	}
	if err := rows.Err(); err != nil {
		return canonicalSnapshot{}, err
	}
	return snapshot, nil
}

// IndexHealth performs live probes across canonical SQLite and both
// disposable indexes. It intentionally reports degraded health as a valid
// result; an error is reserved for a probe that prevents a trustworthy report.
func (s *Service) IndexHealth(ctx context.Context) (IndexHealth, error) {
	health := IndexHealth{Status: "ok", ObservedAt: time.Now().UTC(), Reasons: []string{}}
	reasons := map[string]struct{}{}
	if s == nil || s.Catalog == nil {
		health.Status = "unavailable"
		health.Reasons = []string{"canonical_probe_failed"}
		return health, fmt.Errorf("%w: canonical probe failed", ErrIndexHealthUnavailable)
	}
	snapshot, err := s.readCanonicalSnapshot(ctx)
	if err != nil {
		health.Status = "unavailable"
		health.Reasons = []string{"canonical_probe_failed"}
		return health, fmt.Errorf("%w: %v", ErrIndexHealthUnavailable, err)
	}
	health.CanonicalActiveCount = len(snapshot.Memories)

	savedIdentity, err := s.Catalog.GetEmbeddingIdentity()
	if err != nil {
		reasons["canonical_probe_failed"] = struct{}{}
	} else {
		health.EmbeddingIdentity = savedIdentity
	}
	if expected, ok := s.runtimeEmbeddingIdentity(); ok {
		health.ExpectedEmbeddingIdentity = expected
		if savedIdentity == nil {
			reasons["embedding_identity_missing"] = struct{}{}
		} else if !sameEmbeddingIdentity(*savedIdentity, *expected) {
			reasons["embedding_identity_mismatch"] = struct{}{}
		}
	} else if savedIdentity == nil {
		reasons["embedding_identity_missing"] = struct{}{}
	}

	if s.Hybrid == nil {
		reasons["vector_probe_failed"] = struct{}{}
		reasons["bm25_probe_failed"] = struct{}{}
	} else {
		vectorCount, vectorErr := s.Hybrid.VectorPhysicalCount(ctx)
		if vectorErr != nil {
			reasons["vector_probe_failed"] = struct{}{}
		} else {
			health.VectorPhysicalCount = vectorCount
		}
		physical, listErr := s.Hybrid.ListAll(ctx, 1_000_000)
		if listErr != nil {
			reasons["vector_probe_failed"] = struct{}{}
		} else {
			current := make(map[string]int, len(snapshot.Memories))
			for _, memory := range snapshot.Memories {
				current[memory.ID] = memory.Version
			}
			for _, drawer := range physical {
				canonicalID, version, ok := physicalCanonicalID(drawer.ID)
				if ok && current[canonicalID] == version {
					health.VectorActiveCount++
				}
			}
		}
		health.BM25Count = s.Hybrid.BM25Count()
	}
	if health.CanonicalActiveCount != health.VectorActiveCount {
		reasons["vector_count_diverged"] = struct{}{}
	}
	if health.CanonicalActiveCount != health.BM25Count {
		reasons["bm25_count_diverged"] = struct{}{}
	}
	health.TombstoneCount = health.VectorPhysicalCount - health.VectorActiveCount
	if health.TombstoneCount < 0 {
		health.TombstoneCount = 0
	}
	if health.VectorPhysicalCount > 0 {
		health.TombstoneRatio = float64(health.TombstoneCount) / float64(health.VectorPhysicalCount)
		if health.TombstoneRatio > 0.25 {
			reasons["tombstone_pressure"] = struct{}{}
		}
	}

	var pending, failed int
	var oldest sql.NullString
	err = s.Catalog.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN state IN ('pending','retry','leased') THEN 1 ELSE 0 END),0),
			COALESCE(SUM(CASE WHEN state='failed' THEN 1 ELSE 0 END),0),
			MIN(CASE WHEN state IN ('pending','retry','leased') THEN created_at END)
		FROM index_jobs`).Scan(&pending, &failed, &oldest)
	if err != nil {
		reasons["job_probe_failed"] = struct{}{}
	} else {
		health.PendingJobs = pending
		health.FailedJobs = failed
		if oldest.Valid && oldest.String != "" {
			if created := parseOptionalTime(oldest.String); created != nil {
				age := time.Since(*created)
				if age < 0 {
					age = 0
				}
				value := age.Milliseconds()
				health.OldestPendingAgeMS = &value
			}
		}
	}
	if health.PendingJobs > 0 {
		reasons["index_jobs_pending"] = struct{}{}
	}
	if health.FailedJobs > 0 {
		reasons["index_jobs_failed"] = struct{}{}
	}
	health.LastRebuild = s.Catalog.rebuildSummary(ctx)
	if health.LastRebuild.Status == "running" {
		reasons["rebuild_running"] = struct{}{}
	}
	if health.LastRebuild.Status == "failed" {
		reasons["last_rebuild_failed"] = struct{}{}
	}
	health.Reasons = sortedReasons(reasons)
	if len(health.Reasons) > 0 {
		health.Status = "degraded"
	}
	if _, failedProbe := reasons["canonical_probe_failed"]; failedProbe {
		health.Status = "unavailable"
	}
	if _, failedProbe := reasons["vector_probe_failed"]; failedProbe {
		health.Status = "unavailable"
	}
	if _, failedProbe := reasons["job_probe_failed"]; failedProbe {
		health.Status = "unavailable"
	}
	if health.Status == "unavailable" {
		return health, fmt.Errorf("%w: live probe failed", ErrIndexHealthUnavailable)
	}
	return health, nil
}

func (c *Catalog) setRebuildState(ctx context.Context, status string, started, completed time.Time, count *int, errorCode string) error {
	completedAt := ""
	if !completed.IsZero() {
		completedAt = completed.Format(time.RFC3339Nano)
	}
	_, err := c.db.ExecContext(ctx, `
		INSERT INTO index_rebuilds(id,status,started_at,completed_at,canonical_snapshot_count,error_code)
		VALUES(1,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET status=excluded.status,started_at=excluded.started_at,
		completed_at=excluded.completed_at,canonical_snapshot_count=excluded.canonical_snapshot_count,
			error_code=excluded.error_code`, status, started.Format(time.RFC3339Nano), completedAt, count, errorCode)
	return err
}

// RebuildDerivedIndexes creates a bounded canonical snapshot, prepares the
// new vector projection before touching the current one, then atomically
// replaces the in-memory BM25 projection. The staging directory is a same-
// volume operator marker; canonical.sqlite3 is never moved or overwritten.
func (s *Service) RebuildDerivedIndexes(ctx context.Context) error {
	if s == nil || s.Catalog == nil || s.Hybrid == nil {
		return ErrUnavailable
	}
	started := time.Now().UTC()
	if err := s.Catalog.setRebuildState(ctx, "running", started, time.Time{}, nil, ""); err != nil {
		return fmt.Errorf("%w: record start: %v", ErrIndexRebuildFailed, err)
	}
	// Serialize derived-index replacement with outbox application. Canonical
	// writes may still commit while this lock is held; their transactional
	// jobs remain pending and are replayed after the rebuild.
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	markFailure := func(code string, cause error) error {
		_ = s.Catalog.setRebuildState(ctx, "failed", started, time.Now().UTC(), nil, code)
		return fmt.Errorf("%w: %v", ErrIndexRebuildFailed, cause)
	}
	snapshot, err := s.readCanonicalSnapshot(ctx)
	if err != nil {
		return markFailure("canonical_snapshot_failed", err)
	}
	current, ok := s.runtimeEmbeddingIdentity()
	if !ok {
		return markFailure("embedding_identity_missing", errors.New("runtime embedding identity is unavailable"))
	}
	saved, err := s.Catalog.GetEmbeddingIdentity()
	if err != nil {
		return markFailure("embedding_identity_probe_failed", err)
	}
	if saved == nil {
		if err := s.Catalog.SaveEmbeddingIdentity(*current); err != nil {
			return markFailure("embedding_identity_save_failed", err)
		}
	} else if !sameEmbeddingIdentity(*saved, *current) {
		return markFailure("embedding_identity_mismatch", ErrEmbeddingIdentityMismatch)
	}

	stagingDir := ""
	if s.PalacePath != "" {
		stagingDir, err = os.MkdirTemp(s.PalacePath, ".index-rebuild-")
		if err != nil {
			return markFailure("staging_create_failed", err)
		}
		defer os.RemoveAll(stagingDir)
		manifest, marshalErr := json.Marshal(map[string]any{
			"contract":                 "mentle-derived-index-staging/1",
			"canonical_snapshot_count": len(snapshot.Drawers),
			"embedding_identity":       current,
			"created_at":               started.Format(time.RFC3339Nano),
		})
		if marshalErr != nil {
			return markFailure("staging_manifest_failed", marshalErr)
		}
		if writeErr := os.WriteFile(filepath.Join(stagingDir, "manifest.json"), manifest, 0o600); writeErr != nil {
			return markFailure("staging_manifest_failed", writeErr)
		}
	}
	if err := s.Hybrid.RebuildVectorIndex(ctx, snapshot.Drawers); err != nil {
		return markFailure("vector_build_failed", err)
	}
	if len(snapshot.Drawers) > 0 {
		probe := snapshot.Drawers[0].Content
		if len([]rune(probe)) > 32 {
			probe = string([]rune(probe)[:32])
		}
		if _, err := s.Hybrid.Search(ctx, probe, "", "", 1); err != nil {
			return markFailure("search_smoke_failed", err)
		}
	}
	// BM25 is memory-only and cannot fail during replacement. Do this only
	// after vector replacement and smoke verification so a failed rebuild
	// leaves the prior lexical projection intact as well.
	s.Hybrid.RebuildBM25FromDrawers(snapshot.Drawers)
	count := len(snapshot.Drawers)
	if err := s.Catalog.setRebuildState(ctx, "succeeded", started, time.Now().UTC(), &count, ""); err != nil {
		return fmt.Errorf("%w: record success: %v", ErrIndexRebuildFailed, err)
	}
	return nil
}
