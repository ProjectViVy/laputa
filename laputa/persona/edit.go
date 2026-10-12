package persona

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

const editTTL = 120 * time.Second

type EditLease struct {
	Token        string    `json:"token"`
	Generation   uint64    `json:"generation"`
	Kind         Kind      `json:"kind"`
	BaseRevision uint64    `json:"base_revision"`
	BaseHash     string    `json:"base_hash"`
	ExpiresAt    time.Time `json:"expires_at"`
}
type EditProof struct {
	Token      string `json:"token"`
	Generation uint64 `json:"generation"`
}
type EditStatus struct {
	Held          bool       `json:"held"`
	Generation    uint64     `json:"generation"`
	ExpiresAt     *time.Time `json:"expires_at"`
	BlockedWrites uint64     `json:"blocked_writes"`
}
type editRecord struct {
	lease         *EditLease
	requestID     string
	generation    uint64
	blockedWrites uint64
}

func editError(code, message string) error { return &TypedError{Code: code, Message: message} }
func (s *Service) editLocked(kind Kind) *editRecord {
	if s.edits == nil {
		s.edits = make(map[Kind]*editRecord)
	}
	r := s.edits[kind]
	if r == nil {
		r = &editRecord{}
		s.edits[kind] = r
	}
	if r.lease != nil && !s.now().Before(r.lease.ExpiresAt) {
		r.lease = nil
		r.requestID = ""
		r.blockedWrites = 0
	}
	return r
}
func (s *Service) proofLocked(kind Kind, proof EditProof) (*editRecord, error) {
	if kind.String() == "unknown" {
		return nil, ErrKindForbidden
	}
	r := s.editLocked(kind)
	if r.lease == nil || proof.Token == "" || r.lease.Token != proof.Token || r.generation != proof.Generation {
		return nil, editError("persona_edit_expired", "edit proof is expired or no longer current")
	}
	return r, nil
}

func (s *Service) AcquireEdit(kind Kind, baseRevision uint64, baseHash, requestID string) (*EditLease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if kind.String() == "unknown" {
		return nil, ErrKindForbidden
	}
	if strings.TrimSpace(requestID) == "" || len(requestID) > 256 {
		return nil, editError("persona_invalid_content", "bounded edit request_id is required")
	}
	doc, err := s.readDocumentUncheckedLocked(kind)
	if err != nil {
		return nil, err
	}
	if doc.Revision != baseRevision || doc.ContentHash != baseHash {
		return nil, &RevisionConflictError{Expected: baseRevision, Current: doc.Revision}
	}
	r := s.editLocked(kind)
	if r.lease != nil {
		if r.requestID != requestID || r.lease.BaseRevision != baseRevision || r.lease.BaseHash != baseHash {
			return nil, editError("persona_edit_locked", "document is being edited")
		}
		out := *r.lease
		return &out, nil
	}
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return nil, storageError("cannot create edit token", err)
	}
	r.generation++
	r.requestID = requestID
	r.blockedWrites = 0
	r.lease = &EditLease{Token: hex.EncodeToString(secret), Generation: r.generation, Kind: kind, BaseRevision: baseRevision, BaseHash: baseHash, ExpiresAt: s.now().Add(editTTL)}
	out := *r.lease
	return &out, nil
}
func (s *Service) RenewEdit(kind Kind, proof EditProof) (*EditLease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.proofLocked(kind, proof)
	if err != nil {
		return nil, err
	}
	r.lease.ExpiresAt = s.now().Add(editTTL)
	out := *r.lease
	return &out, nil
}
func (s *Service) ReleaseEdit(kind Kind, proof EditProof) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.proofLocked(kind, proof)
	if err != nil {
		return err
	}
	r.lease = nil
	r.requestID = ""
	r.blockedWrites = 0
	return nil
}
func (s *Service) EditState(kind Kind) (EditStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if kind.String() == "unknown" {
		return EditStatus{}, ErrKindForbidden
	}
	r := s.editLocked(kind)
	state := EditStatus{Held: r.lease != nil, Generation: r.generation, BlockedWrites: r.blockedWrites}
	if r.lease != nil {
		expires := r.lease.ExpiresAt
		state.ExpiresAt = &expires
	}
	return state, nil
}
