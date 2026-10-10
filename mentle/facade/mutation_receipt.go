package facade

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// Scoped mutation receipts (contracts.md section 6): every canonical write
// records its receipt inside the same transaction, so a crash between
// commit and reply is answered by the stored receipt on retry — never by a
// blind second write. Operation IDs are bound to scope and destination; a
// lookup under another binding returns not-found rather than disclosure.
const mutationReceiptSchema = `
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
);`

// MutationRequest is the closed create/update/tombstone contract. Scope and
// destination are pre-validated encoded strings; the caller (Garden)
// guarantees they equal the bound writer before dispatch.
type MutationRequest struct {
	Scope            string
	DestinationID    string
	OperationID      string
	PayloadDigest    string
	Operation        string // create | update | tombstone
	RecordID         string
	ExpectedRevision int
	ExpectedAbsent   bool
	Body             string
	Kind             string
	Sources          []MemorySource
	Metadata         map[string]any
	Actor            string
	RequestID        string
}

// MutationReceipt is the persisted outcome of one mutation.
type MutationReceipt struct {
	OperationID     string `json:"operation_id"`
	PayloadDigest   string `json:"payload_digest"`
	RecordID        string `json:"record_id"`
	Status          string `json:"status"` // applied | rejected
	Revision        int    `json:"revision"`
	ErrorCode       string `json:"error_code"`
	CanonicalStatus string `json:"canonical_status"` // completed | failed
	IndexStatus     string `json:"index_status"`     // pending | ready | failed
}

// ErrMutationNotFound hides both absence and cross-binding receipts.
var ErrMutationNotFound = errors.New("mutation receipt not found")

// Mutate applies one scoped mutation inside a single canonical transaction
// and stores the receipt in the same commit.
func (s *Service) Mutate(ctx context.Context, req MutationRequest) (MutationReceipt, error) {
	if s != nil && s.lexicalOnly {
		return MutationReceipt{}, ErrReadOnly
	}
	if s.Catalog == nil || s.Catalog.db == nil {
		return MutationReceipt{}, ErrUnavailable
	}
	if err := validateMutationRequest(req); err != nil {
		return MutationReceipt{}, err
	}
	if existing, found, err := s.lookupReceipt(ctx, req.OperationID); err != nil {
		return MutationReceipt{}, err
	} else if found {
		// Create mints its own record id, so the caller's record_id hint is
		// not part of the replay identity; updates/tombstones bind it.
		recordMismatch := req.Operation != "create" && existing.RecordID != req.RecordID
		if existing.PayloadDigest != req.PayloadDigest ||
			existing.scope != req.Scope ||
			existing.destination != req.DestinationID ||
			recordMismatch {
			return MutationReceipt{}, ErrIdempotencyConflict
		}
		// Atomic receipt replay: the original commit already answered.
		_ = s.applyIndexJob(ctx, existing.RecordID)
		return existing.receipt(), nil
	}
	switch req.Operation {
	case "create":
		return s.mutateCreate(ctx, req)
	case "update":
		return s.mutateUpdate(ctx, req)
	case "tombstone":
		return s.mutateTombstone(ctx, req)
	default:
		return MutationReceipt{}, errors.New("unknown mutation operation")
	}
}

func validateMutationRequest(req MutationRequest) error {
	if req.OperationID == "" || req.PayloadDigest == "" || req.Scope == "" || req.DestinationID == "" {
		return errors.New("mutation operation_id, payload_digest, scope and destination are required")
	}
	switch req.Operation {
	case "create":
		if !req.ExpectedAbsent || req.ExpectedRevision != 0 {
			return errors.New("create requires expected_absent and zero expected_revision")
		}
	case "update", "tombstone":
		if req.RecordID == "" {
			return errors.New("update/tombstone require a record_id")
		}
		if req.ExpectedAbsent || req.ExpectedRevision <= 0 {
			return errors.New("update/tombstone require positive expected_revision")
		}
	default:
		return errors.New("unknown mutation operation")
	}
	return nil
}

type storedReceipt struct {
	OperationID   string
	PayloadDigest string
	RecordID      string
	Status        string
	Revision      int
	ErrorCode     string
	scope         string
	destination   string
}

func (r storedReceipt) receipt() MutationReceipt {
	return MutationReceipt{
		OperationID:     r.OperationID,
		PayloadDigest:   r.PayloadDigest,
		RecordID:        r.RecordID,
		Status:          r.Status,
		Revision:        r.Revision,
		ErrorCode:       r.ErrorCode,
		CanonicalStatus: "completed",
		IndexStatus:     "pending",
	}
}

func (s *Service) lookupReceipt(ctx context.Context, operationID string) (storedReceipt, bool, error) {
	var r storedReceipt
	err := s.Catalog.db.QueryRowContext(ctx,
		`SELECT operation_id,payload_digest,record_id,status,revision,error_code,scope,destination
		 FROM mutation_receipts WHERE operation_id=?`, operationID).
		Scan(&r.OperationID, &r.PayloadDigest, &r.RecordID, &r.Status, &r.Revision, &r.ErrorCode, &r.scope, &r.destination)
	if errors.Is(err, sql.ErrNoRows) {
		return storedReceipt{}, false, nil
	}
	if err != nil {
		return storedReceipt{}, false, err
	}
	return r, true, nil
}

// MutationStatus returns the receipt for a committed operation, bound to
// the caller's scope+destination. A receipt stored under another binding is
// indistinguishable from absence.
func (s *Service) MutationStatus(ctx context.Context, operationID, scope, destination string) (MutationReceipt, error) {
	if s.Catalog == nil || s.Catalog.db == nil {
		return MutationReceipt{}, ErrUnavailable
	}
	r, found, err := s.lookupReceipt(ctx, operationID)
	if err != nil {
		return MutationReceipt{}, err
	}
	if !found || r.scope != scope || r.destination != destination {
		return MutationReceipt{}, ErrMutationNotFound
	}
	receipt := r.receipt()
	// Index readiness is observational, never part of canonical commit.
	var state string
	if err := s.Catalog.db.QueryRowContext(ctx, `SELECT state FROM index_jobs WHERE memory_id=?`, r.RecordID).Scan(&state); err == nil {
		switch state {
		case "done", "ready":
			receipt.IndexStatus = "ready"
		case "poisoned", "failed":
			receipt.IndexStatus = "failed"
		default:
			receipt.IndexStatus = "pending"
		}
	}
	return receipt, nil
}

func (s *Service) mutateCreate(ctx context.Context, req MutationRequest) (MutationReceipt, error) {
	if err := s.ensureEmbeddingIdentity(); err != nil {
		return MutationReceipt{}, err
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return MutationReceipt{}, errors.New("memory content is required")
	}
	if len([]byte(body)) > 64<<10 {
		return MutationReceipt{}, errors.New("memory content exceeds 64 KiB")
	}
	now := time.Now().UTC()
	source := MemorySource{Type: "agent"}
	if len(req.Sources) > 0 {
		source = req.Sources[0]
	}
	if source.Type == "" {
		source.Type = "agent"
	}
	kind := req.Kind
	if kind == "" {
		kind = "note"
	}
	m := Memory{ID: canonicalID(), Kind: kind, Content: body, Status: "active", Version: 1, Scope: req.Scope, Tags: []string{}, Source: source, ValidFrom: now, Supersedes: []string{}, CreatedAt: now, UpdatedAt: now, Metadata: nonNilMap(req.Metadata)}
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
		return MutationReceipt{}, err
	}
	err = insertMemory(ctx, tx, m)
	if err != nil {
		tx.Rollback()
		return MutationReceipt{}, err
	}
	if _, err = enqueueIndexJobTx(ctx, tx, m.ID, m.Version, "upsert", m.Content, encode(m.Metadata), now); err != nil {
		tx.Rollback()
		return MutationReceipt{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO audit_log(memory_id,action,actor,request_id,reason,created_at) VALUES(?,?,?,?,?,?)`, m.ID, "create", req.Actor, req.RequestID, "", now.Format(time.RFC3339Nano)); err != nil {
		tx.Rollback()
		return MutationReceipt{}, err
	}
	receipt := MutationReceipt{OperationID: req.OperationID, PayloadDigest: req.PayloadDigest, RecordID: m.ID, Status: "applied", Revision: 1, CanonicalStatus: "completed", IndexStatus: "pending"}
	if _, err = tx.ExecContext(ctx, `INSERT INTO mutation_receipts(operation_id,payload_digest,scope,destination,record_id,revision,status,error_code,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		req.OperationID, req.PayloadDigest, req.Scope, req.DestinationID, m.ID, 1, "applied", "", now.Format(time.RFC3339Nano)); err != nil {
		tx.Rollback()
		return MutationReceipt{}, err
	}
	if err = tx.Commit(); err != nil {
		return MutationReceipt{}, err
	}
	_ = s.applyIndexJob(ctx, m.ID)
	return receipt, nil
}

func (s *Service) mutateUpdate(ctx context.Context, req MutationRequest) (MutationReceipt, error) {
	if err := s.ensureEmbeddingIdentity(); err != nil {
		return MutationReceipt{}, err
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return MutationReceipt{}, errors.New("memory content is required")
	}
	if len([]byte(body)) > 64<<10 {
		return MutationReceipt{}, errors.New("memory content exceeds 64 KiB")
	}
	m, err := s.GetMemory(ctx, req.RecordID)
	if err != nil {
		return MutationReceipt{}, err
	}
	if m.Scope != req.Scope {
		// The record lives outside the bound writer's scope; the caller must
		// not learn whether it exists elsewhere.
		return MutationReceipt{}, ErrMutationNotFound
	}
	// A corrected body and its supplied provenance are one canonical
	// revision. Omitted fields preserve prior metadata/source; provided
	// metadata keys replace their old values without dropping other keys.
	m.Metadata = nonNilMap(m.Metadata)
	for key, value := range req.Metadata {
		m.Metadata[key] = value
	}
	if len(req.Sources) > 0 {
		m.Source = req.Sources[0]
		if m.Source.Type == "" {
			m.Source.Type = "agent"
		}
	}
	now := time.Now().UTC()
	tx, err := s.Catalog.db.BeginTx(ctx, nil)
	if err != nil {
		return MutationReceipt{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE memories SET content=?,source_json=?,metadata_json=?,version=version+1,updated_at=? WHERE id=? AND version=? AND status='active'`, body, encode(m.Source), encode(m.Metadata), now.Format(time.RFC3339Nano), req.RecordID, req.ExpectedRevision)
	if err != nil {
		tx.Rollback()
		return MutationReceipt{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return MutationReceipt{}, err
	}
	if affected != 1 {
		tx.Rollback()
		if _, getErr := s.GetMemory(ctx, req.RecordID); errors.Is(getErr, ErrMemoryNotFound) {
			return MutationReceipt{}, ErrMutationNotFound
		}
		return MutationReceipt{}, ErrVersionConflict
	}
	newVersion := req.ExpectedRevision + 1
	if _, err = enqueueIndexJobTx(ctx, tx, req.RecordID, newVersion, "upsert", body, encode(m.Metadata), now); err != nil {
		tx.Rollback()
		return MutationReceipt{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO audit_log(memory_id,action,actor,request_id,reason,created_at) VALUES(?,?,?,?,?,?)`, req.RecordID, "update", req.Actor, req.RequestID, "", now.Format(time.RFC3339Nano)); err != nil {
		tx.Rollback()
		return MutationReceipt{}, err
	}
	receipt := MutationReceipt{OperationID: req.OperationID, PayloadDigest: req.PayloadDigest, RecordID: req.RecordID, Status: "applied", Revision: newVersion, CanonicalStatus: "completed", IndexStatus: "pending"}
	if _, err = tx.ExecContext(ctx, `INSERT INTO mutation_receipts(operation_id,payload_digest,scope,destination,record_id,revision,status,error_code,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		req.OperationID, req.PayloadDigest, req.Scope, req.DestinationID, req.RecordID, newVersion, "applied", "", now.Format(time.RFC3339Nano)); err != nil {
		tx.Rollback()
		return MutationReceipt{}, err
	}
	if err = tx.Commit(); err != nil {
		return MutationReceipt{}, err
	}
	_ = s.applyIndexJob(ctx, req.RecordID)
	return receipt, nil
}

func (s *Service) mutateTombstone(ctx context.Context, req MutationRequest) (MutationReceipt, error) {
	m, err := s.GetMemory(ctx, req.RecordID)
	if err != nil {
		return MutationReceipt{}, err
	}
	if m.Scope != req.Scope {
		return MutationReceipt{}, ErrMutationNotFound
	}
	now := time.Now().UTC()
	tx, err := s.Catalog.db.BeginTx(ctx, nil)
	if err != nil {
		return MutationReceipt{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE memories SET status='deleted',valid_to=?,updated_at=?,version=version+1 WHERE id=? AND version=? AND status='active'`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), req.RecordID, req.ExpectedRevision)
	if err != nil {
		tx.Rollback()
		return MutationReceipt{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return MutationReceipt{}, err
	}
	if affected != 1 {
		tx.Rollback()
		var status string
		lookupErr := s.Catalog.db.QueryRowContext(ctx, `SELECT status FROM memories WHERE id=?`, req.RecordID).Scan(&status)
		if errors.Is(lookupErr, sql.ErrNoRows) || status == "deleted" {
			return MutationReceipt{}, ErrMutationNotFound
		}
		if lookupErr != nil {
			return MutationReceipt{}, lookupErr
		}
		return MutationReceipt{}, ErrVersionConflict
	}
	newVersion := req.ExpectedRevision + 1
	if _, err = enqueueIndexJobTx(ctx, tx, req.RecordID, newVersion, "delete", "", "{}", now); err != nil {
		tx.Rollback()
		return MutationReceipt{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO audit_log(memory_id,action,actor,request_id,reason,created_at) VALUES(?,?,?,?,?,?)`, req.RecordID, "delete", req.Actor, req.RequestID, "", now.Format(time.RFC3339Nano)); err != nil {
		tx.Rollback()
		return MutationReceipt{}, err
	}
	receipt := MutationReceipt{OperationID: req.OperationID, PayloadDigest: req.PayloadDigest, RecordID: req.RecordID, Status: "applied", Revision: newVersion, CanonicalStatus: "completed", IndexStatus: "pending"}
	if _, err = tx.ExecContext(ctx, `INSERT INTO mutation_receipts(operation_id,payload_digest,scope,destination,record_id,revision,status,error_code,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		req.OperationID, req.PayloadDigest, req.Scope, req.DestinationID, req.RecordID, newVersion, "applied", "", now.Format(time.RFC3339Nano)); err != nil {
		tx.Rollback()
		return MutationReceipt{}, err
	}
	if err = tx.Commit(); err != nil {
		return MutationReceipt{}, err
	}
	_ = s.applyIndexJob(ctx, req.RecordID)
	return receipt, nil
}
