// Package memory owns the Garden-side backend contract (contracts.md
// section 6): the authorized DTOs and the Backend port that a primary
// memory backend implements. Backend instances are bound to one subject and
// one write destination; nothing here grants model-supplied scope.
package memory

import (
	"context"
	"encoding/json"

	"github.com/ProjectViVy/laputa/laputa/evolution"
)

// Capabilities advertises what a backend can serve. The first four are
// required of the selected primary writer; the rest are read optimizations.
type Capabilities struct {
	Search         bool `json:"search"`
	Expand         bool `json:"expand"`
	Mutate         bool `json:"mutate"`
	MutationLookup bool `json:"mutation_lookup"`
	Vector         bool `json:"vector"`
	Timeline       bool `json:"timeline"`
	KnowledgeGraph bool `json:"knowledge_graph"`
}

// ServesPrimaryWriter reports whether the backend satisfies the required
// primary-writer capability set.
func (c Capabilities) ServesPrimaryWriter() bool {
	return c.Search && c.Expand && c.Mutate && c.MutationLookup
}

// AuthorizedSearch is a scoped bounded card search.
type AuthorizedSearch struct {
	Scopes      []evolution.Scope `json:"scopes"`
	Query       string            `json:"query"`
	Collection  string            `json:"collection,omitempty"`
	Cursor      string            `json:"cursor"`
	Limit       int               `json:"limit"`
	BudgetChars int               `json:"budget_chars"`
}

// Validate enforces nonempty valid scopes and positive bounds.
func (s AuthorizedSearch) Validate() error {
	if len(s.Scopes) == 0 {
		return &evolution.ContractError{Code: evolution.ErrInvalidScope, Message: "search requires at least one scope"}
	}
	for _, scope := range s.Scopes {
		if err := scope.Validate(); err != nil {
			return err
		}
	}
	if s.Limit < 0 || s.BudgetChars < 0 {
		return &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "negative limit or budget"}
	}
	return nil
}

// DecodeAuthorizedSearch strict-decodes a search request.
func DecodeAuthorizedSearch(data []byte) (AuthorizedSearch, error) {
	var s AuthorizedSearch
	if err := evolution.DecodeStrictJSON(data, &s); err != nil {
		return AuthorizedSearch{}, err
	}
	if err := s.Validate(); err != nil {
		return AuthorizedSearch{}, err
	}
	return s, nil
}

// AuthorizedExpansion is a scoped bounded evidence read. A missing or
// changed expected_revision fails rather than refetching different content.
type AuthorizedExpansion struct {
	Scopes           []evolution.Scope `json:"scopes"`
	CardID           string            `json:"card_id"`
	ExpectedRevision uint64            `json:"expected_revision"`
	BudgetChars      int               `json:"budget_chars"`
}

// Validate requires at least one valid admitted scope, a card id and a
// positive revision. The backend checks the record's own scope against
// this admitted union — the caller never declares which scope it is in.
func (e AuthorizedExpansion) Validate() error {
	if len(e.Scopes) == 0 {
		return &evolution.ContractError{Code: evolution.ErrInvalidScope, Message: "expansion requires at least one scope"}
	}
	for _, scope := range e.Scopes {
		if err := scope.Validate(); err != nil {
			return err
		}
	}
	if e.CardID == "" {
		return &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "card_id required"}
	}
	if e.ExpectedRevision == 0 {
		return &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "expected_revision required"}
	}
	return nil
}

// AuthorizedMutation is one scoped canonical write. Scope and destination
// must equal the bound writer; create/update/tombstone carry the contract's
// revision and absent-precondition shapes.
type AuthorizedMutation struct {
	Scope            evolution.Scope             `json:"scope"`
	DestinationID    string                      `json:"destination_id"`
	OperationID      string                      `json:"operation_id"`
	PayloadDigest    string                      `json:"payload_digest"`
	Operation        evolution.MutationOperation `json:"operation"`
	RecordID         string                      `json:"record_id"`
	ExpectedRevision uint64                      `json:"expected_revision"`
	ExpectedAbsent   bool                        `json:"expected_absent"`
	Body             string                      `json:"body"`
	Sources          []evolution.SourceRef       `json:"sources"`
	Inference        evolution.InferenceClass    `json:"inference"`
}

// Validate enforces the per-operation preconditions.
func (m AuthorizedMutation) Validate() error {
	if err := m.Scope.Validate(); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"destination_id": m.DestinationID,
		"operation_id":   m.OperationID,
		"payload_digest": m.PayloadDigest,
		"record_id":      m.RecordID,
	} {
		if value == "" {
			return &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "mutation " + name + " required"}
		}
	}
	switch m.Operation {
	case evolution.MutationCreate:
		if !m.ExpectedAbsent || m.ExpectedRevision != 0 {
			return &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "create requires expected_absent=true and expected_revision=0"}
		}
	case evolution.MutationUpdate, evolution.MutationTombstone:
		if m.ExpectedAbsent || m.ExpectedRevision == 0 {
			return &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "update/tombstone require expected_absent=false and positive expected_revision"}
		}
	default:
		return &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "unknown mutation operation"}
	}
	switch m.Inference {
	case evolution.InferenceObserved, evolution.InferenceInferred:
	default:
		return &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "unknown inference class"}
	}
	for _, s := range m.Sources {
		if err := s.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// MatchesWriter checks the mutation against the instance's bound writer.
// Scope comparison is exact-tuple, never prefix-based.
func (m AuthorizedMutation) MatchesWriter(boundScope evolution.Scope, boundDestination string) error {
	if !m.Scope.Equal(boundScope) || m.DestinationID != boundDestination {
		return &evolution.ContractError{Code: evolution.ErrInvalidScope, Message: "mutation scope/destination does not match the bound writer"}
	}
	return nil
}

// DecodeAuthorizedMutation strict-decodes and validates a mutation.
func DecodeAuthorizedMutation(data []byte) (AuthorizedMutation, error) {
	var m AuthorizedMutation
	if err := evolution.DecodeStrictJSON(data, &m); err != nil {
		return AuthorizedMutation{}, err
	}
	if err := m.Validate(); err != nil {
		return AuthorizedMutation{}, err
	}
	return m, nil
}

// CanonicalStatus reports the canonical write outcome.
type CanonicalStatus string

const (
	CanonicalAccepted  CanonicalStatus = "accepted"
	CanonicalCompleted CanonicalStatus = "completed"
	CanonicalFailed    CanonicalStatus = "failed"
)

// IndexStatus reports derived-index readiness; index readiness never
// substitutes for canonical commit.
type IndexStatus string

const (
	IndexPending     IndexStatus = "pending"
	IndexReady       IndexStatus = "ready"
	IndexFailed      IndexStatus = "failed"
	IndexNotRequired IndexStatus = "not_required"
)

// MutationReceipt is an EffectReceipt plus canonical/index status.
type MutationReceipt struct {
	evolution.EffectReceipt
	CanonicalStatus CanonicalStatus `json:"canonical_status"`
	IndexStatus     IndexStatus     `json:"index_status"`
}

// MarshalJSON flattens the embedded receipt fields.
func (r MutationReceipt) MarshalJSON() ([]byte, error) {
	type wire struct {
		OperationID     string                 `json:"operation_id"`
		PayloadDigest   string                 `json:"payload_digest"`
		Status          evolution.EffectStatus `json:"status"`
		TargetRef       string                 `json:"target_ref"`
		Revision        uint64                 `json:"revision"`
		ErrorCode       string                 `json:"error_code"`
		CanonicalStatus CanonicalStatus        `json:"canonical_status"`
		IndexStatus     IndexStatus            `json:"index_status"`
	}
	return json.Marshal(wire{
		OperationID:     r.OperationID,
		PayloadDigest:   r.PayloadDigest,
		Status:          r.Status,
		TargetRef:       r.TargetRef,
		Revision:        r.Revision,
		ErrorCode:       r.ErrorCode,
		CanonicalStatus: r.CanonicalStatus,
		IndexStatus:     r.IndexStatus,
	})
}

// UnmarshalJSON strict-decodes the flattened shape.
func (r *MutationReceipt) UnmarshalJSON(data []byte) error {
	type wire struct {
		OperationID     string                 `json:"operation_id"`
		PayloadDigest   string                 `json:"payload_digest"`
		Status          evolution.EffectStatus `json:"status"`
		TargetRef       string                 `json:"target_ref"`
		Revision        uint64                 `json:"revision"`
		ErrorCode       string                 `json:"error_code"`
		CanonicalStatus CanonicalStatus        `json:"canonical_status"`
		IndexStatus     IndexStatus            `json:"index_status"`
	}
	var w wire
	if err := evolution.DecodeStrictJSON(data, &w); err != nil {
		return err
	}
	r.EffectReceipt = evolution.EffectReceipt{
		OperationID:   w.OperationID,
		PayloadDigest: w.PayloadDigest,
		Status:        w.Status,
		TargetRef:     w.TargetRef,
		Revision:      w.Revision,
		ErrorCode:     w.ErrorCode,
	}
	r.CanonicalStatus = w.CanonicalStatus
	r.IndexStatus = w.IndexStatus
	return nil
}

// CardPage is a bounded card page; the cursor is opaque and bound to the
// scope and query.
type CardPage struct {
	Items      []MemoryCard `json:"items"`
	NextCursor string       `json:"next_cursor"`
}

// EvidencePage is a bounded evidence page. Scope/revision/status ride in
// each evidence envelope so the adapter can verify results.
type EvidencePage struct {
	Items     []EvidenceFragment `json:"items"`
	Truncated bool               `json:"truncated"`
}

// HealthStatus is the closed backend health vocabulary.
type HealthStatus string

const (
	HealthAvailable   HealthStatus = "available"
	HealthDegraded    HealthStatus = "degraded"
	HealthUnavailable HealthStatus = "unavailable"
)

// Health reports backend availability, a stable reason code and the
// derived-index state.
type Health struct {
	Status            HealthStatus `json:"status"`
	ReasonCode        string       `json:"reason_code"`
	DerivedIndexState string       `json:"derived_index_state"`
}

// Backend is the Garden-side memory backend port. Implementations are bound
// to one subject and one write destination; MutationStatus cannot reveal
// another instance's records.
type Backend interface {
	Capabilities() Capabilities
	Search(context.Context, AuthorizedSearch) (CardPage, error)
	Expand(context.Context, AuthorizedExpansion) (EvidencePage, error)
	Mutate(context.Context, AuthorizedMutation) (MutationReceipt, error)
	MutationStatus(context.Context, string) (MutationReceipt, error)
	Health(context.Context) (Health, error)
	Close() error
}
