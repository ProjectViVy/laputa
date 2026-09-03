// Package persona implements the Laputa Markdown clean-break Persona
// authority: seven exact uppercase files, revision-checked writes, immutable
// history snapshots, and the review workflow for agent-proposed changes.
//
// Storage layout (verified against the DIVA reference implementation):
//
//	persona/IDENTITY.MD ... persona/WORLD.MD
//	persona/history/<FILE_NAME>/<rev>.md, <rev>.diff, log.jsonl
//	persona/requests/<uuid>.json
package persona

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Kind is one of the seven Persona authority files.
type Kind int

const (
	KindIdentity Kind = iota
	KindRelationship
	KindRedline
	KindUser
	KindWorld
	KindDream
	KindDark
)

// AllKinds lists the seven kinds in contract order.
var AllKinds = []Kind{KindIdentity, KindRelationship, KindRedline, KindUser, KindDream, KindDark, KindWorld}

// RequiredKinds are the five files whose presence defines initialization.
var RequiredKinds = []Kind{KindIdentity, KindRelationship, KindRedline, KindUser, KindWorld}

// FrozenKinds are the six files allowed in the Frozen Core projection.
var FrozenKinds = []Kind{KindIdentity, KindRelationship, KindRedline, KindUser, KindDream, KindDark}

// String returns the lowercase wire name, e.g. "identity".
func (k Kind) String() string {
	switch k {
	case KindIdentity:
		return "identity"
	case KindRelationship:
		return "relationship"
	case KindRedline:
		return "redline"
	case KindUser:
		return "user"
	case KindWorld:
		return "world"
	case KindDream:
		return "dream"
	case KindDark:
		return "dark"
	}
	return "unknown"
}

// FileName returns the exact authority file name, e.g. IDENTITY.MD.
func (k Kind) FileName() string {
	switch k {
	case KindIdentity:
		return "IDENTITY.MD"
	case KindRelationship:
		return "RELATIONSHIP.MD"
	case KindRedline:
		return "REDLINE.MD"
	case KindUser:
		return "USER.MD"
	case KindWorld:
		return "WORLD.MD"
	case KindDream:
		return "DREAM.MD"
	case KindDark:
		return "DARK.MD"
	}
	return "UNKNOWN.MD"
}

// ContentLimit is the visible-grapheme cap for the file body.
func (k Kind) ContentLimit() int {
	switch k {
	case KindIdentity:
		return 800
	case KindRelationship:
		return 600
	case KindRedline:
		return 400
	case KindUser:
		return 800
	case KindWorld:
		return 1000
	case KindDream:
		return 40
	case KindDark:
		return 300
	}
	return 0
}

// FrozenLimit is the visible-grapheme cap for the Frozen Core projection;
// WORLD has none because it is not part of the Frozen Core.
func (k Kind) FrozenLimit() (int, bool) {
	switch k {
	case KindIdentity:
		return 200, true
	case KindRelationship:
		return 120, true
	case KindRedline:
		return 200, true
	case KindUser:
		return 160, true
	case KindDream:
		return 10, true
	case KindDark:
		return 60, true
	}
	return 0, false
}

// RequiredForReady reports whether the file is one of the five initialization files.
func (k Kind) RequiredForReady() bool {
	for _, required := range RequiredKinds {
		if k == required {
			return true
		}
	}
	return false
}

// MarshalJSON encodes the kind as its lowercase wire name.
func (k Kind) MarshalJSON() ([]byte, error) { return json.Marshal(k.String()) }

// UnmarshalJSON decodes the kind from its lowercase wire name.
func (k *Kind) UnmarshalJSON(b []byte) error {
	var value string
	if err := json.Unmarshal(b, &value); err != nil {
		return err
	}
	parsed, err := ParseKind(value)
	if err != nil {
		return err
	}
	*k = parsed
	return nil
}

// ParseKind parses the lowercase wire name, case-insensitive.
func ParseKind(value string) (Kind, error) {
	for _, kind := range AllKinds {
		if strings.EqualFold(kind.String(), value) {
			return kind, nil
		}
	}
	return 0, fmt.Errorf("%w: %s", ErrKindForbidden, value)
}

// Status is the profile-level lifecycle state.
type Status int

const (
	StatusUninitialized Status = iota
	StatusReady
	StatusIncomplete
)

func (s Status) String() string {
	switch s {
	case StatusReady:
		return "ready"
	case StatusIncomplete:
		return "incomplete"
	}
	return "uninitialized"
}

// MarshalJSON encodes the status as its lowercase wire name.
func (s Status) MarshalJSON() ([]byte, error) { return json.Marshal(s.String()) }

// UnmarshalJSON decodes the status from its lowercase wire name.
func (s *Status) UnmarshalJSON(b []byte) error {
	var value string
	if err := json.Unmarshal(b, &value); err != nil {
		return err
	}
	switch value {
	case "uninitialized":
		*s = StatusUninitialized
	case "ready":
		*s = StatusReady
	case "incomplete":
		*s = StatusIncomplete
	default:
		return fmt.Errorf("unknown persona status %q", value)
	}
	return nil
}

// FileState summarizes one authority file for the setup view.
type FileState struct {
	Kind         Kind       `json:"kind"`
	FileName     string     `json:"file_name"`
	Exists       bool       `json:"exists"`
	Valid        bool       `json:"valid"`
	Reason       *string    `json:"reason"`
	Revision     uint64     `json:"revision"`
	UpdatedAt    *time.Time `json:"updated_at"`
	PendingCount int        `json:"pending_count"`
	ContentLimit int        `json:"content_limit"`
	FrozenLimit  *int       `json:"frozen_limit"`
	Required     bool       `json:"required"`
	ToolOnly     bool       `json:"tool_only"`
}

// StatusView is the profile-level setup response.
type StatusView struct {
	Status Status               `json:"status"`
	Files  map[string]FileState `json:"files"`
}

// Document is a full read of one authority file.
type Document struct {
	Kind         Kind       `json:"kind"`
	FileName     string     `json:"file_name"`
	Exists       bool       `json:"exists"`
	Valid        bool       `json:"valid"`
	Content      string     `json:"content"`
	Revision     uint64     `json:"revision"`
	ContentHash  string     `json:"content_hash"`
	UpdatedAt    *time.Time `json:"updated_at"`
	PendingCount int        `json:"pending_count"`
}

// HistoryEntry is one revision entry from log.jsonl.
type HistoryEntry struct {
	Revision     uint64    `json:"revision"`
	ContentHash  string    `json:"content_hash"`
	Snapshot     string    `json:"snapshot"`
	Diff         string    `json:"diff"`
	Actor        string    `json:"actor"`
	Source       string    `json:"source"`
	Reason       string    `json:"reason"`
	BaseRevision uint64    `json:"base_revision"`
	CreatedAt    time.Time `json:"created_at"`
}

// HistoryRevision is an entry plus its snapshot content and unified diff.
type HistoryRevision struct {
	HistoryEntry
	Content     string `json:"content"`
	UnifiedDiff string `json:"unified_diff"`
}

// WriteOutcome is the result of a persona write.
type WriteOutcome struct {
	Document Document `json:"document"`
	Changed  bool     `json:"changed"`
}

// Initialization carries the five required documents' initial content.
type Initialization struct {
	Identity     string `json:"identity"`
	Relationship string `json:"relationship"`
	Redline      string `json:"redline"`
	User         string `json:"user"`
	World        string `json:"world"`
}

// RepairInput contains owner-selected required documents to reconstruct when
// a profile is incomplete. Optional Persona files cannot be repaired through
// this operation.
type RepairInput struct {
	Documents map[Kind]string `json:"documents"`
	Reason    string          `json:"reason"`
}

// PersonaRepair is the public contract spelling for RepairInput.
type PersonaRepair = RepairInput

// ReviewState and ReviewActor are clean-break aliases for the persisted
// review state and actor enums.
type ReviewState = RequestState
type ReviewActor = RequestActor

// PersonaDocumentState is the metadata-only representation used by the
// domain adapter. It is an alias so service callers can continue to use the
// compact FileState name without introducing a second authority model.
type PersonaDocumentState = FileState

// PersonaWriteResult is the stable name used by the clean-break contract.
type PersonaWriteResult = WriteOutcome

// PersonaReview is the clean-break name for a persisted review. Its on-disk
// representation remains a small JSON request record, while HTTP exposes it
// under the /reviews domain.
type PersonaReview = ChangeRequest

// PersonaReviewCreate is the input shape for review creation. Actor is
// supplied by the verified capability claim and is ignored by HTTP adapters
// when a caller attempts to spoof it.
type PersonaReviewCreate struct {
	Kind             Kind         `json:"kind"`
	BaseRevision     uint64       `json:"base_revision"`
	BaseHash         string       `json:"base_hash"`
	ProposedMarkdown string       `json:"proposed_markdown"`
	Actor            RequestActor `json:"actor"`
	Reason           string       `json:"reason"`
}

// Map returns the initialization content keyed by kind.
func (i Initialization) Map() map[Kind]string {
	return map[Kind]string{
		KindIdentity:     i.Identity,
		KindRelationship: i.Relationship,
		KindRedline:      i.Redline,
		KindUser:         i.User,
		KindWorld:        i.World,
	}
}

// RequestActor identifies who proposed a Persona change.
type RequestActor string

const (
	ActorAgent     RequestActor = "agent"
	ActorAutodream RequestActor = "autodream"
)

// RequestState is the lifecycle of a ChangeRequest.
type RequestState string

const (
	RequestPending  RequestState = "pending"
	RequestAccepted RequestState = "accepted"
	RequestRejected RequestState = "rejected"
	RequestStale    RequestState = "stale"
)

// ChangeRequest is an agent-proposed Persona revision awaiting user review.
type ChangeRequest struct {
	ID               string       `json:"id"`
	Kind             Kind         `json:"kind"`
	BaseRevision     uint64       `json:"base_revision"`
	BaseHash         string       `json:"base_hash"`
	ProposedMarkdown string       `json:"proposed_markdown"`
	Actor            RequestActor `json:"actor"`
	Reason           string       `json:"reason"`
	CreatedAt        time.Time    `json:"created_at"`
	State            RequestState `json:"state"`
	DecidedAt        *time.Time   `json:"decided_at"`
}

// WriteSource classifies a persona write for history purposes.
type WriteSource string

const (
	SourceUserDirect      WriteSource = "user_direct"
	SourceAgentP16        WriteSource = "agent_p16"
	SourceAgentP5Accepted WriteSource = "agent_p5_accepted"
	SourceAutodreamP5     WriteSource = "autodream_p5_accepted"
	SourceInit            WriteSource = "init"
	SourceHistoryResave   WriteSource = "history_resave"
)

// ContentHash returns the canonical sha256:<hex> hash of normalized content.
func ContentHash(content string) string {
	digest := sha256.Sum256([]byte(NormalizeMarkdown(content)))
	return fmt.Sprintf("sha256:%x", digest)
}

// Error sentinels mapped by writeHandlerError in the Garden server.
var (
	ErrUninitialized      = &TypedError{Code: "persona_uninitialized", Message: "persona is not initialized"}
	ErrIncomplete         = &TypedError{Code: "persona_incomplete", Message: "persona is incomplete"}
	ErrAlreadyInitialized = &TypedError{Code: "persona_already_initialized", Message: "persona is already initialized"}
	ErrRepairNotRequired  = &TypedError{Code: "persona_repair_not_required", Message: "persona repair is not required"}
	ErrKindForbidden      = &TypedError{Code: "persona_kind_forbidden", Message: "persona kind or write scope is forbidden"}
)

// TypedError carries a stable code for HTTP mapping.
type TypedError struct {
	Code    string
	Message string
	Err     error
}

func (e *TypedError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("laputa/persona: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("laputa/persona: %s", e.Message)
}

func (e *TypedError) Unwrap() error { return e.Err }

// CodeOf returns the stable error code for any persona error.
func CodeOf(err error) string {
	var typed *TypedError
	if errors.As(err, &typed) {
		return typed.Code
	}
	var rev *RevisionConflictError
	if errors.As(err, &rev) {
		return "persona_revision_conflict"
	}
	var cap *CapExceededError
	if errors.As(err, &cap) {
		return "persona_cap_exceeded"
	}
	var invalid *InvalidContentError
	if errors.As(err, &invalid) {
		return "persona_invalid_content"
	}
	var exists *RequestExistsError
	if errors.As(err, &exists) {
		return "persona_review_exists"
	}
	var missing *RequestNotFoundError
	if errors.As(err, &missing) {
		return "persona_review_not_found"
	}
	var stale *RequestStaleError
	if errors.As(err, &stale) {
		return "persona_review_stale"
	}
	var history *HistoryNotFoundError
	if errors.As(err, &history) {
		return "persona_history_not_found"
	}
	var protected *WorldProtectedError
	if errors.As(err, &protected) {
		return "persona_world_protected_claim"
	}
	var entryGate *WorldEntryGateError
	if errors.As(err, &entryGate) {
		return "persona_world_entry_gate"
	}
	return "persona_storage_error"
}

// RevisionConflictError reports base-revision mismatch on write or request creation.
type RevisionConflictError struct {
	Expected uint64
	Current  uint64
}

func (e *RevisionConflictError) Error() string {
	return fmt.Sprintf("laputa/persona: revision conflict: expected %d, current %d", e.Expected, e.Current)
}

// CapExceededError reports a visible-length cap violation.
type CapExceededError struct {
	Kind  Kind
	Limit int
}

func (e *CapExceededError) Error() string {
	return fmt.Sprintf("laputa/persona: content exceeds the %d character cap for %s", e.Limit, e.Kind.FileName())
}

// InvalidContentError reports structural or encoding problems.
type InvalidContentError struct {
	Kind   Kind
	Reason string
}

func (e *InvalidContentError) Error() string {
	return fmt.Sprintf("laputa/persona: invalid content for %s: %s", e.Kind.FileName(), e.Reason)
}

// RequestExistsError reports a duplicate pending request for a kind.
type RequestExistsError struct{ Kind Kind }

func (e *RequestExistsError) Error() string {
	return fmt.Sprintf("laputa/persona: a pending request already exists for %s", e.Kind.FileName())
}

// RequestNotFoundError reports an unknown request id.
type RequestNotFoundError struct{ ID string }

func (e *RequestNotFoundError) Error() string {
	return fmt.Sprintf("laputa/persona: request not found: %s", e.ID)
}

// HistoryNotFoundError reports an unknown immutable history revision.
type HistoryNotFoundError struct {
	Kind     Kind
	Revision uint64
}

func (e *HistoryNotFoundError) Error() string {
	return fmt.Sprintf("laputa/persona: history revision %d not found for %s", e.Revision, e.Kind.FileName())
}

// WorldProtectedError reports removal or mutation of confirmed user WORLD
// material.
type WorldProtectedError struct{}

func (e *WorldProtectedError) Error() string {
	return "laputa/persona: WORLD request would overwrite protected user content"
}

// WorldEntryGateError reports an invalid R6 bounded/reviewable WORLD claim.
type WorldEntryGateError struct{}

func (e *WorldEntryGateError) Error() string {
	return "laputa/persona: WORLD request violates the R6 bounded, reviewable claim entry gate"
}

// RequestStaleError reports an accepted/rejected request being redecided.
type RequestStaleError struct{ ID string }

func (e *RequestStaleError) Error() string {
	return fmt.Sprintf("laputa/persona: request is stale: %s", e.ID)
}
