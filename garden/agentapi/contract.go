package agentapi

import (
	"time"

	"github.com/dashimaki/laputa/evolution"
)

// ACTMEM scoped-operation DTOs alias the shared contract; the agentapi
// surface adds no second representation.
type (
	ReadRequest    = evolution.ReadRequest
	WorkPatch      = evolution.WorkPatch
	WorkChange     = evolution.WorkChange
	WorkChangeKind = evolution.WorkChangeKind
	WorkField      = evolution.WorkField
	Entry          = evolution.Entry
	EntrySection   = evolution.EntrySection
	ActivityResult = evolution.ActivityResult
	SourceRef      = evolution.SourceRef
	Scope          = evolution.Scope
)

// Binding is host identity and audit provenance, not an authorization claim.
// ProfileID is checked against server configuration on every request.
// WorkspaceID is the trusted workspace binding issued by the host at Open;
// an empty value is the implicit personal workspace. It is never a
// model-supplied grant.
type Binding struct {
	ProfileID   string `json:"profile_id"`
	AgentID     string `json:"agent_id"`
	Platform    string `json:"platform"`
	SessionID   string `json:"session_id"`
	TurnID      string `json:"turn_id,omitempty"`
	EventID     string `json:"event_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

// TrustedScope derives the contract scope from the host-bound identity:
// ProfileID is the stable subject; an empty workspace_id is the implicit
// personal workspace.
func (b Binding) TrustedScope() (evolution.Scope, error) {
	kind := evolution.ScopePersonal
	if b.WorkspaceID != "" {
		kind = evolution.ScopeWorkspace
	}
	scope := evolution.Scope{SubjectID: b.ProfileID, Kind: kind, WorkspaceID: b.WorkspaceID}
	if err := scope.Validate(); err != nil {
		return evolution.Scope{}, err
	}
	return scope, nil
}

// BootstrapRequest extends the existing recall/bootstrap body with a binding.
// Adapter authentication must supply Principal independently of this body.
type BootstrapRequest struct {
	Binding     Binding `json:"binding"`
	SessionID   string  `json:"session_id"`
	Intent      string  `json:"intent"`
	BudgetChars int     `json:"budget_chars"`
}

// BootstrapResponse is the exact six-key /v2/recall/bootstrap success shape.
type BootstrapResponse struct {
	TraceID    string             `json:"trace_id"`
	Context    string             `json:"context"`
	FrozenCore FrozenCore         `json:"frozen_core"`
	Evidence   []EvidenceFragment `json:"evidence"`
	Degraded   bool               `json:"degraded"`
	Warnings   []string           `json:"warnings"`
}

// CapturePhase describes a host Run's terminal event, not an ingest phase.
type CapturePhase string

const (
	CaptureCompleted CapturePhase = "completed"
	CaptureFailed    CapturePhase = "failed"
	CaptureCanceled  CapturePhase = "canceled"
)

// CaptureProvenance holds bounded host references, not raw authority or SQL state.
type CaptureProvenance struct {
	Source   string `json:"source,omitempty"`
	RunID    string `json:"run_id,omitempty"`
	EventSeq uint64 `json:"event_seq,omitempty"`
}

type CaptureRequest struct {
	Binding     Binding           `json:"binding"`
	Phase       CapturePhase      `json:"phase"`
	Content     string            `json:"content"`
	ContentHash string            `json:"content_hash"`
	Provenance  CaptureProvenance `json:"provenance"`
	OccurredAt  time.Time         `json:"occurred_at,omitempty"`
}

// CaptureReceipt matches the existing asynchronous ingest acceptance keys.
type CaptureReceipt struct {
	IngestionID string `json:"ingestion_id"`
	SessionID   string `json:"session_id"`
	EventID     string `json:"event_id"`
	Status      string `json:"status"`
}

// Error retains the current wire envelope while exposing a normalized code.
// LegacyError mirrors Message in existing HTTP responses; adapters set both.
type Error struct {
	Code        string         `json:"code"`
	Message     string         `json:"message"`
	LegacyError string         `json:"error"`
	Retryable   bool           `json:"retryable"`
	RequestID   string         `json:"request_id"`
	Details     map[string]any `json:"details"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// FrozenCore is the shared v2 envelope: schema_version, mission first of
// exactly seven named slots, mission_status. WORLD and ACTMEM can never be
// slots; a pre-Mission v1 admission cannot masquerade as v2.
type FrozenCore = evolution.FrozenCoreV2
type FrozenSection = evolution.FrozenSectionV2

type MemoryCard struct {
	ID             string     `json:"id"`
	Kind           string     `json:"kind"`
	Collection     string     `json:"collection"`
	Scope          string     `json:"scope"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	SourceRef      string     `json:"source_ref"`
	Revision       int        `json:"revision"`
	Status         string     `json:"status"`
	ValidFrom      time.Time  `json:"valid_from"`
	ValidTo        *time.Time `json:"valid_to,omitempty"`
	SupersededBy   *string    `json:"superseded_by,omitempty"`
	Tags           []string   `json:"tags"`
	HeatScore      float64    `json:"heat_score"`
	LastActivated  *time.Time `json:"last_activated,omitempty"`
	CandidateScore float64    `json:"candidate_score"`
}

// EvidenceFragment carries scope/revision/status in its envelope so
// Garden can verify backend adapter results (contracts.md section 6). The
// added fields are additive-only on the existing wire.
type EvidenceFragment struct {
	CardID       string   `json:"card_id"`
	MaterialRef  string   `json:"material_ref"`
	SourceURI    string   `json:"source_uri,omitempty"`
	SourceRev    string   `json:"source_rev,omitempty"`
	Scope        string   `json:"scope,omitempty"`
	Revision     uint64   `json:"revision,omitempty"`
	Status       string   `json:"status,omitempty"`
	Excerpt      string   `json:"excerpt"`
	StartOffset  int      `json:"start_offset"`
	EndOffset    int      `json:"end_offset"`
	ContentHash  string   `json:"content_hash"`
	Validity     string   `json:"validity"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

// ContextView mirrors the recall/fast JSON fields without importing internal
// packages or exposing storage handles. Automatic context has no tool-only slot.
type ContextView struct {
	TraceID       string             `json:"trace_id"`
	Scope         string             `json:"scope"`
	Mode          string             `json:"mode"`
	FrozenCore    FrozenCore         `json:"frozen_core"`
	Cards         []MemoryCard       `json:"cards"`
	Evidence      []EvidenceFragment `json:"evidence"`
	Context       string             `json:"context"`
	BudgetChars   int                `json:"budget_chars"`
	Degraded      bool               `json:"degraded"`
	Warnings      []string           `json:"warnings"`
	RecallTraceID *string            `json:"recall_trace_id,omitempty"`
}
