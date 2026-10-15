package actmem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

// RecapBinding is supplied by the trusted host, never inferred from text.
type RecapBinding struct{ SubjectID, DomainID string }

// SourceRange describes a complete contiguous range verified by the host. Digest
// is opaque: this owner has no access to the original interaction evidence.
type SourceRange struct {
	After, Through uint64
	Digest         string
}
type RecapHead struct {
	Binding                                              RecapBinding
	Revision                                             uint64
	Text                                                 string `yaml:"-"`
	CoveredThrough                                       uint64
	InvalidationEpoch                                    uint64
	OperationID, OperationDigest, PreviousDigest, Digest string
	Cause                                                string
	Sources                                              SourceRange
	RestoredFrom                                         uint64
	UpdatedAt                                            time.Time
}
type RecapCommit struct {
	Binding                  RecapBinding
	BaseRevision             uint64
	OperationID, Text, Cause string
	Sources                  SourceRange
	RestoredFrom             uint64
}

// ReadRecap returns one atomic view of committed text and coverage. An empty
// legacy directory is represented by revision zero without changing its files.
func (s *Store) ReadRecap(ctx context.Context, binding RecapBinding) (RecapHead, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	history, err := s.recapChain(ctx, binding)
	if err != nil {
		return RecapHead{}, err
	}
	if len(history) == 0 {
		return RecapHead{Binding: binding}, nil
	}
	return history[0], nil
}

func (s *Store) LookupRecapOperation(ctx context.Context, binding RecapBinding, operationID string) (RecapHead, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	history, err := s.recapChain(ctx, binding)
	if err != nil {
		return RecapHead{}, false, err
	}
	for _, h := range history {
		if h.OperationID == operationID {
			return h, true, nil
		}
	}
	return RecapHead{}, false, nil
}

// ListRecapHistory returns newest first. before is an exclusive revision cursor;
// zero starts at the current head. Only the committed digest chain is visible.
func (s *Store) ListRecapHistory(ctx context.Context, binding RecapBinding, before uint64, limit int) ([]RecapHead, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit < 1 {
		return nil, recapUnavailable("history limit must be positive", nil)
	}
	history, err := s.recapChain(ctx, binding)
	if err != nil {
		return nil, err
	}
	page := []RecapHead{}
	for _, h := range history {
		if before != 0 && h.Revision >= before {
			continue
		}
		page = append(page, h)
		if len(page) == limit {
			break
		}
	}
	return page, nil
}

func (s *Store) ReadRecapRevision(ctx context.Context, binding RecapBinding, revision uint64) (RecapHead, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	history, err := s.recapChain(ctx, binding)
	if err != nil {
		return RecapHead{}, err
	}
	for _, h := range history {
		if h.Revision == revision {
			return h, nil
		}
	}
	return RecapHead{}, recapUnavailable("Recap revision is not reachable", nil)
}

// CommitRecap performs the owner CAS, with replay lookup before base-revision
// validation. Host inference and verification of original ranges happen outside
// this lock. Preserve the entire original request when retrying an operation.
func (s *Store) CommitRecap(ctx context.Context, request RecapCommit) (RecapHead, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return RecapHead{}, err
	}
	if !validRecapBinding(request.Binding) {
		return RecapHead{}, newError("recap_scope_denied", "subject and domain are required", nil)
	}
	if request.OperationID == "" || !utf8.ValidString(request.OperationID) || !utf8.ValidString(request.Text) || !utf8.ValidString(request.Cause) || !utf8.ValidString(request.Sources.Digest) {
		return RecapHead{}, recapUnavailable("invalid UTF-8 or missing operation identity", nil)
	}
	if err := s.acquireOwner(); err != nil {
		return RecapHead{}, err
	}
	if s.recapUnknown {
		return RecapHead{}, newError("recap_commit_unknown", "reopen ACTMEM and resolve the attempted operation", nil)
	}
	history, err := s.recapChain(ctx, request.Binding)
	if err != nil {
		return RecapHead{}, err
	}
	requestDigest := recapHash(request)
	for _, h := range history {
		if h.OperationID == request.OperationID {
			if h.OperationDigest != requestDigest {
				return RecapHead{}, newError("recap_operation_conflict", "operation identity has a different request", nil)
			}
			return h, nil
		}
	}
	current := RecapHead{Binding: request.Binding}
	if len(history) > 0 {
		current = history[0]
	}
	if current.Revision != request.BaseRevision {
		return RecapHead{}, newError("recap_conflict", fmt.Sprintf("expected Recap revision %d, actual %d", request.BaseRevision, current.Revision), nil)
	}
	if current.Revision == math.MaxUint64 {
		return RecapHead{}, recapUnavailable("Recap revision exhausted", nil)
	}
	next, err := s.nextRecap(current, request, history)
	if err != nil {
		return RecapHead{}, err
	}
	next.OperationDigest = requestDigest
	next.Digest = recapHeadDigest(next)
	if err := s.persistRecap(ctx, next); err != nil {
		return RecapHead{}, err
	}
	return next, nil
}

func (s *Store) nextRecap(current RecapHead, request RecapCommit, history []RecapHead) (RecapHead, error) {
	invalid := func() (RecapHead, error) {
		return RecapHead{}, newError("recap_invalid_range", "invalid Recap cause, source range or restore payload", nil)
	}
	next := RecapHead{Binding: request.Binding, Revision: current.Revision + 1, Text: request.Text, CoveredThrough: current.CoveredThrough, InvalidationEpoch: current.InvalidationEpoch, OperationID: request.OperationID, PreviousDigest: current.Digest, Cause: request.Cause, Sources: request.Sources, RestoredFrom: request.RestoredFrom, UpdatedAt: s.now()}
	if next.UpdatedAt.IsZero() {
		return RecapHead{}, recapUnavailable("Recap timestamp is unavailable", nil)
	}
	switch request.Cause {
	case "summary":
		if request.RestoredFrom != 0 || request.Sources.After != current.CoveredThrough || request.Sources.Through <= request.Sources.After || strings.TrimSpace(request.Sources.Digest) == "" {
			return invalid()
		}
		next.CoveredThrough = request.Sources.Through
	case "edit", "clear", "restore":
		if request.Sources != (SourceRange{}) {
			return invalid()
		}
		switch request.Cause {
		case "edit":
			if request.RestoredFrom != 0 {
				return invalid()
			}
		case "clear":
			if request.Text != "" || request.RestoredFrom != 0 || next.InvalidationEpoch == math.MaxUint64 {
				return invalid()
			}
			next.InvalidationEpoch++
		case "restore":
			if request.Text != "" || request.RestoredFrom == 0 {
				return invalid()
			}
			found := false
			for _, h := range history {
				if h.Revision == request.RestoredFrom {
					next.Text = h.Text
					found = true
					break
				}
			}
			if !found {
				return invalid()
			}
		}
	default:
		return invalid()
	}
	return next, nil
}

func recapUnavailable(message string, cause error) error {
	return newError("recap_unavailable", message, cause)
}
func validRecapBinding(binding RecapBinding) bool {
	return binding.SubjectID != "" && binding.DomainID != "" && utf8.ValidString(binding.SubjectID) && utf8.ValidString(binding.DomainID)
}

// Typed JSON is used only as an unambiguous digest preimage. Authority bodies
// remain UTF-8 Markdown; neither requests nor heads are JSON authority files.
func recapHash(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func recapHeadDigest(head RecapHead) string { head.Digest = ""; return recapHash(head) }
func validRecapDigest(value string) bool {
	raw, err := hex.DecodeString(value)
	return err == nil && len(raw) == sha256.Size && strings.ToLower(value) == value
}
