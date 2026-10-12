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

type EditorWrite struct {
	Kind         Kind      `json:"kind"`
	Content      string    `json:"content"`
	Scope        string    `json:"scope"`
	Reason       string    `json:"reason"`
	BaseRevision uint64    `json:"base_revision"`
	BaseHash     string    `json:"base_hash"`
	Proof        EditProof `json:"edit_proof"`
}
type ProjectionSnapshot struct {
	Documents       map[Kind]Document
	EditGenerations map[Kind]uint64
}

func (s *Service) guardWriteLocked(kind Kind, proof *EditProof) error {
	if kind.String() == "unknown" {
		return ErrKindForbidden
	}
	if proof != nil {
		_, err := s.proofLocked(kind, *proof)
		return err
	}
	r := s.editLocked(kind)
	if r.lease != nil {
		r.blockedWrites++
		return editError("persona_edit_locked", "document is being edited")
	}
	return nil
}
func (s *Service) guardSetLocked(kinds []Kind) error {
	for _, kind := range kinds {
		if err := s.guardWriteLocked(kind, nil); err != nil {
			return err
		}
	}
	return nil
}

// Only an explicitly inspected human resave may repair history/live drift;
// missing/invalid setup and drift in other required documents stay blocked.
func (s *Service) requireHumanReadyLocked(kind Kind, inspectedHash string) error {
	view, err := s.statusLocked()
	if err != nil {
		return err
	}
	if view.Status == StatusReady {
		return nil
	}
	if view.Status == StatusUninitialized {
		return ErrUninitialized
	}
	for _, k := range RequiredKinds {
		state := view.Files[k.String()]
		if !state.Valid && (k != kind || state.Reason == nil || *state.Reason != "history_mismatch") {
			return ErrIncomplete
		}
	}
	doc, err := s.readDocumentUncheckedLocked(kind)
	if err != nil {
		return err
	}
	if doc.Revision == 0 || doc.ContentHash != inspectedHash {
		return ErrIncomplete
	}
	return nil
}

func (s *Service) SaveEditorDocument(in EditorWrite) (*WriteOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.proofLocked(in.Kind, in.Proof)
	if err != nil {
		return nil, err
	}
	doc, err := s.readDocumentUncheckedLocked(in.Kind)
	if err != nil {
		return nil, err
	}
	if in.BaseRevision != r.lease.BaseRevision || in.BaseHash != r.lease.BaseHash || doc.Revision != in.BaseRevision || doc.ContentHash != in.BaseHash {
		return nil, &RevisionConflictError{Expected: in.BaseRevision, Current: doc.Revision}
	}
	if err = s.requireHumanReadyLocked(in.Kind, in.BaseHash); err != nil {
		return nil, err
	}
	proposed := in.Content
	switch in.Scope {
	case "", "document":
	case "observations":
		if in.Kind != KindUser {
			return nil, ErrKindForbidden
		}
		proposed = ReplaceMarkdownSection(doc.Content, "Observations", in.Content)
	default:
		return nil, ErrKindForbidden
	}
	source := SourceUserDirect
	if s.contentExistsInHistoryLocked(in.Kind, proposed) {
		source = SourceHistoryResave
	}
	out, err := s.writeDocumentCoreLocked(in.Kind, proposed, in.BaseRevision, "user", source, in.Reason, false, "", &in.Proof)
	if err != nil {
		return nil, err
	}
	r.lease = nil
	r.requestID = ""
	r.blockedWrites = 0
	return out, nil
}

// The callback is a bounded local admission operation. It must not reenter
// Persona or perform network/model work while this authority mutex is held.
func (s *Service) WithProjectionSnapshot(kinds []Kind, accept func(ProjectionSnapshot) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(kinds) == 0 || accept == nil {
		return ErrKindForbidden
	}
	if err := s.guardSetLocked(kinds); err != nil {
		return err
	}
	out := ProjectionSnapshot{Documents: map[Kind]Document{}, EditGenerations: map[Kind]uint64{}}
	for _, kind := range kinds {
		doc, err := s.readDocumentUncheckedLocked(kind)
		if err != nil {
			return err
		}
		entries, err := s.listHistoryLocked(kind)
		if err != nil {
			return err
		}
		if !doc.Exists || len(entries) == 0 || entries[0].ContentHash != doc.ContentHash {
			return editError("persona_authority_drift", "selected document does not match committed history")
		}
		out.Documents[kind] = *doc
		out.EditGenerations[kind] = s.editLocked(kind).generation
	}
	return accept(out)
}
