// Package evolution holds the versioned shared contracts for the DIVA-style
// cognitive loop (contract revision diva-cognitive/v1-review-1). It defines
// wire DTOs, strict decoders and the consumer ports used by the strategy and
// its host. The package contains no runtime, journal, or authority
// implementation.
package evolution

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
)

// Code is the closed stable error vocabulary shared by the evolution
// contracts and the Garden backend contract.
type Code string

const (
	ErrInvalidSchema          Code = "invalid_schema"
	ErrInvalidScope           Code = "invalid_scope"
	ErrAuthorityDenied        Code = "authority_denied"
	ErrRevisionConflict       Code = "revision_conflict"
	ErrIdempotencyConflict    Code = "idempotency_conflict"
	ErrEffectNotFound         Code = "effect_not_found"
	ErrOutcomeUnknown         Code = "outcome_unknown"
	ErrMissionRevisionChanged Code = "mission_revision_changed"
	ErrActmemCapExceeded      Code = "actmem_cap_exceeded"
	ErrActmemFormat           Code = "actmem_format_error"
	ErrBackendUnavailable     Code = "backend_unavailable"
	ErrCapabilityUnavailable  Code = "capability_unavailable"
	ErrRecoveryRequired       Code = "recovery_required"
)

// ContractError carries a stable Code plus a human-readable message.
type ContractError struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
}

func (e *ContractError) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

// CodeOf extracts the stable contract code from an error, or "" when err is
// not a contract error.
func CodeOf(err error) Code {
	if err == nil {
		return ""
	}
	var ce *ContractError
	if asContractError(err, &ce) {
		return ce.Code
	}
	return ""
}

func asContractError(err error, target **ContractError) bool {
	for err != nil {
		if ce, ok := err.(*ContractError); ok {
			*target = ce
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func invalidSchema(format string, args ...any) error {
	return &ContractError{Code: ErrInvalidSchema, Message: fmt.Sprintf(format, args...)}
}

// DecodeStrictJSON unmarshals one JSON value into v. Unknown fields,
// trailing JSON, and duplicate object keys are rejected as invalid_schema.
// It is the shared strict entry point for every contract decoder, including
// contracts owned by other packages.
func DecodeStrictJSON(data []byte, v any) error {
	if err := rejectDuplicateKeys(data); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return invalidSchema("decode: %v", err)
	}
	if dec.More() {
		return invalidSchema("trailing JSON value")
	}
	return nil
}

func rejectDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	return walkValue(dec)
}

func walkValue(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		if err == io.EOF {
			return invalidSchema("empty JSON input")
		}
		return invalidSchema("malformed JSON: %v", err)
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return invalidSchema("malformed JSON object: %v", err)
			}
			key, ok := keyTok.(string)
			if !ok {
				return invalidSchema("non-string object key")
			}
			if seen[key] {
				return invalidSchema("duplicate JSON key %q", key)
			}
			seen[key] = true
			if err := walkValue(dec); err != nil {
				return err
			}
		}
		if _, err := dec.Token(); err != nil { // consume '}'
			return invalidSchema("malformed JSON object: %v", err)
		}
	case '[':
		for dec.More() {
			if err := walkValue(dec); err != nil {
				return err
			}
		}
		if _, err := dec.Token(); err != nil { // consume ']'
			return invalidSchema("malformed JSON array: %v", err)
		}
	}
	return nil
}

// ScopeKind is the closed scope vocabulary.
type ScopeKind string

const (
	ScopePersonal  ScopeKind = "personal"
	ScopeWorkspace ScopeKind = "workspace"
)

// Scope is the trusted subject/workspace tuple. Comparison is exact-tuple
// equality; it is never prefix- or path-based.
type Scope struct {
	SubjectID   string    `json:"subject_id" yaml:"subject_id"`
	Kind        ScopeKind `json:"kind" yaml:"kind"`
	WorkspaceID string    `json:"workspace_id" yaml:"workspace_id"`
}

// Equal reports exact tuple equality.
func (s Scope) Equal(other Scope) bool { return s == other }

// Validate enforces the scope shape: subject_id nonempty, kind known,
// workspace_id present iff kind is workspace.
func (s Scope) Validate() error {
	if s.SubjectID == "" {
		return invalidSchema("scope subject_id required")
	}
	switch s.Kind {
	case ScopePersonal:
		if s.WorkspaceID != "" {
			return invalidSchema("personal scope must have empty workspace_id")
		}
	case ScopeWorkspace:
		if s.WorkspaceID == "" {
			return invalidSchema("workspace scope requires workspace_id")
		}
	default:
		return invalidSchema("unknown scope kind %q", s.Kind)
	}
	return nil
}

// DecodeScope strict-decodes and validates a scope record. An absent or
// malformed scope is invalid input.
func DecodeScope(data []byte) (Scope, error) {
	var s Scope
	if err := DecodeStrictJSON(data, &s); err != nil {
		return Scope{}, err
	}
	if err := s.Validate(); err != nil {
		return Scope{}, err
	}
	return s, nil
}

// RunBinding is host-populated trusted binding for one run. It is never a
// model field. mission_revision=0 means Mission is unassigned and the run may
// not admit mission-driven autonomous effects.
type RunBinding struct {
	SubjectID       string `json:"subject_id"`
	WorkspaceID     string `json:"workspace_id"`
	DestinationID   string `json:"destination_id"`
	PolicyRevision  string `json:"policy_revision"`
	StrategyDigest  string `json:"strategy_digest"`
	MissionRevision uint64 `json:"mission_revision"`
}

// MissionAssigned reports whether a Mission is bound to this run.
func (b RunBinding) MissionAssigned() bool { return b.MissionRevision > 0 }

// CheckMissionRevision re-verifies the Mission pin immediately before
// mission-driven autonomous effects. A human edit after admission changes the
// revision, so the pinned run is refused with mission_revision_changed; the
// run must open a new session rather than execute against the stale Mission.
func (b RunBinding) CheckMissionRevision(current uint64) error {
	if b.MissionRevision != current {
		return &ContractError{Code: ErrMissionRevisionChanged, Message: "Mission revision changed since admission; open a new session"}
	}
	return nil
}

// Scope derives the run's scope from the binding (empty workspace_id is the
// implicit personal workspace).
func (b RunBinding) Scope() Scope {
	kind := ScopePersonal
	if b.WorkspaceID != "" {
		kind = ScopeWorkspace
	}
	return Scope{SubjectID: b.SubjectID, Kind: kind, WorkspaceID: b.WorkspaceID}
}

// Validate requires the trusted identity fields to be present and nonempty.
func (b RunBinding) Validate() error {
	for name, value := range map[string]string{
		"subject_id":      b.SubjectID,
		"destination_id":  b.DestinationID,
		"policy_revision": b.PolicyRevision,
		"strategy_digest": b.StrategyDigest,
	} {
		if value == "" {
			return invalidSchema("run binding %s required", name)
		}
	}
	return nil
}

// DecodeRunBinding strict-decodes and validates a binding.
func DecodeRunBinding(data []byte) (RunBinding, error) {
	var b RunBinding
	if err := DecodeStrictJSON(data, &b); err != nil {
		return RunBinding{}, err
	}
	if err := b.Validate(); err != nil {
		return RunBinding{}, err
	}
	return b, nil
}

// PrincipalClass is the closed provenance vocabulary for evidence: human,
// conversation, or workflow.
type PrincipalClass string

const (
	PrincipalHuman        PrincipalClass = "human"
	PrincipalConversation PrincipalClass = "conversation"
	PrincipalWorkflow     PrincipalClass = "workflow"
)

// StrategyDIVA is the strategy id of the DIVA-inspired INOFY workflow.
const StrategyDIVA = "diva/v1"

// Fixed node identities for the six-stage sequence. Catalog implementation
// ids pin the implementation revision separately.
const (
	NodeCollect   = "laputa.evolution.collect@1"
	NodePrepare   = "laputa.evolution.prepare@1"
	NodeReconcile = "laputa.evolution.reconcile@1"
	NodeReflect   = "laputa.evolution.reflect@1"
	NodeEffects   = "laputa.evolution.effects@1"
	NodeFinish    = "laputa.evolution.finish@1"
)

// Wake describes one wake event as seen by the trigger gate.
type Wake struct {
	NowUnixMS      int64 `json:"now_unix_ms"`
	Manual         bool  `json:"manual"`
	NewActivity    bool  `json:"new_activity"`
	ForegroundBusy bool  `json:"foreground_busy"`
}

// TriggerPolicy configures the automatic trigger gate.
type TriggerPolicy struct {
	Enabled       bool  `json:"enabled"`
	MinIntervalMS int64 `json:"min_interval_ms"`
}

// TriggerState is the trigger gate's view of prior runs.
type TriggerState struct {
	LastCompletedUnixMS int64  `json:"last_completed_unix_ms"`
	ActiveRunID         string `json:"active_run_id"`
}

// EligibilityReason is the closed reason set, in evaluation order.
type EligibilityReason string

const (
	ReasonDisabled       EligibilityReason = "disabled"
	ReasonActive         EligibilityReason = "active"
	ReasonNoNewInput     EligibilityReason = "no_new_input"
	ReasonForegroundBusy EligibilityReason = "foreground_busy"
	ReasonInterval       EligibilityReason = "interval"
	ReasonNotBeforeClock EligibilityReason = "not_before_clock"
	ReasonEligible       EligibilityReason = "eligible"
)

// Eligibility is the trigger gate outcome.
type Eligibility struct {
	Run    bool              `json:"run"`
	Reason EligibilityReason `json:"reason"`
}

// Evaluate is a pure eligibility function. Order: disabled, active, no new
// input, automatic foreground busy, backward clock, elapsed interval,
// eligible. Manual wakes ignore the clock/interval/foreground gates but not
// disabled/active/no-input. min_interval_ms=0 disables the interval gate
// while still coalescing an active run.
func Evaluate(w Wake, p TriggerPolicy, s TriggerState) Eligibility {
	no := func(r EligibilityReason) Eligibility { return Eligibility{Run: false, Reason: r} }
	if !p.Enabled {
		return no(ReasonDisabled)
	}
	if s.ActiveRunID != "" {
		return no(ReasonActive)
	}
	if !w.NewActivity {
		return no(ReasonNoNewInput)
	}
	if w.Manual {
		return Eligibility{Run: true, Reason: ReasonEligible}
	}
	if w.ForegroundBusy {
		return no(ReasonForegroundBusy)
	}
	if w.NowUnixMS < s.LastCompletedUnixMS {
		return no(ReasonNotBeforeClock)
	}
	if w.NowUnixMS-s.LastCompletedUnixMS < p.MinIntervalMS {
		return no(ReasonInterval)
	}
	return Eligibility{Run: true, Reason: ReasonEligible}
}

// Window bounds the evidence collection for a run.
type Window struct {
	SourceID string `json:"source_id"`
	After    uint64 `json:"after"`
	Through  uint64 `json:"through"`
}

// Input is the trusted run input handed to the strategy by the host.
type Input struct {
	Binding RunBinding `json:"binding"`
	Window  Window     `json:"window"`
}

// SourceRef cites one evidence record.
type SourceRef struct {
	SourceID string `json:"source_id" yaml:"source_id"`
	RecordID string `json:"record_id" yaml:"record_id"`
	Revision uint64 `json:"revision" yaml:"revision"`
	Scope    Scope  `json:"scope" yaml:"scope"`
}

// Validate requires all ids nonempty and the scope valid.
func (r SourceRef) Validate() error {
	if r.SourceID == "" || r.RecordID == "" {
		return invalidSchema("source ref ids required")
	}
	return r.Scope.Validate()
}

// EntrySection names a top-level ACTMEM section.
type EntrySection string

const (
	SectionPulse EntrySection = "pulse"
	SectionRecap EntrySection = "recap"
	SectionWork  EntrySection = "work"
)

// WorkField names the Work metadata fields; empty for ring entries.
type WorkField string

const (
	FieldGoal        WorkField = "goal"
	FieldOpen        WorkField = "open"
	FieldNext        WorkField = "next"
	FieldConstraints WorkField = "constraints"
	FieldPointers    WorkField = "pointers"
)

// Entry is one ACTMEM line item as carried into evidence collection.
type Entry struct {
	ID         string       `json:"id"`
	Section    EntrySection `json:"section"`
	Field      WorkField    `json:"field"`
	Scope      Scope        `json:"scope"`
	SessionID  string       `json:"session_id"`
	EventID    string       `json:"event_id"`
	OccurredAt string       `json:"occurred_at"`
	Body       string       `json:"body"`
	Sources    []SourceRef  `json:"sources"`
}

// AuthorityKind names the closed eight-file authority roster: MISSION.MD plus
// the seven existing files. ACTMEM stays outside it.
type AuthorityKind string

const (
	AuthorityMission      AuthorityKind = "mission"
	AuthorityIdentity     AuthorityKind = "identity"
	AuthorityRelationship AuthorityKind = "relationship"
	AuthorityRedline      AuthorityKind = "redline"
	AuthorityUser         AuthorityKind = "user"
	AuthorityDream        AuthorityKind = "dream"
	AuthorityDark         AuthorityKind = "dark"
	AuthorityWorld        AuthorityKind = "world"
)

// AuthorityKinds is the exact closed roster in spec order.
var AuthorityKinds = []AuthorityKind{
	AuthorityMission, AuthorityIdentity, AuthorityRelationship,
	AuthorityRedline, AuthorityUser, AuthorityDream, AuthorityDark,
	AuthorityWorld,
}

// AuthorityView is a bounded, authorized Markdown projection of one
// authority kind.
type AuthorityView struct {
	Kind     AuthorityKind `json:"kind"`
	Revision uint64        `json:"revision"`
	Content  string        `json:"content"`
}

// Validate enforces the closed eight-kind roster.
func (v AuthorityView) Validate() error {
	for _, k := range AuthorityKinds {
		if v.Kind == k {
			return nil
		}
	}
	return invalidSchema("unknown authority kind %q", v.Kind)
}

// EvidenceBatch is the collect output for a run window.
type EvidenceBatch struct {
	Window           Window          `json:"window"`
	ActivityRevision uint64          `json:"activity_revision"`
	Entries          []Entry         `json:"entries"`
	Persona          []AuthorityView `json:"persona"`
	Sources          []SourceRef     `json:"sources"`
}

// ModelStage names the closed model call sites.
type ModelStage string

const (
	StagePrepare   ModelStage = "prepare"
	StageReconcile ModelStage = "reconcile"
	StageReflect   ModelStage = "reflect"
)

// ModelRequest is the bounded model call envelope.
type ModelRequest struct {
	Stage        ModelStage      `json:"stage"`
	Prompt       string          `json:"prompt"`
	InputJSON    json.RawMessage `json:"input_json"`
	OutputSchema json.RawMessage `json:"output_schema"`
}

// ModelReply carries the raw structured output.
type ModelReply struct {
	OutputJSON json.RawMessage `json:"output_json"`
}

// Model is the consumer-defined inference port. The host implements it;
// implementations must not be required to enable the library.
type Model interface {
	Infer(ctx context.Context, req ModelRequest) (ModelReply, error)
}

// Domain is the consumer-defined authority port bound to a trusted
// destination at construction; the strategy cannot reach other domains.
type Domain interface {
	Collect(ctx context.Context, window Window) (EvidenceBatch, error)
	Apply(ctx context.Context, effect Effect) (EffectReceipt, error)
	Lookup(ctx context.Context, operationID string) (EffectReceipt, error)
}

// digestInput is the canonical digest preimage: the normalized typed payload
// plus effective scope, destination, target and expected revision.
type digestInput struct {
	Scope         Scope      `json:"scope"`
	DestinationID string     `json:"destination_id"`
	Kind          EffectKind `json:"kind"`
	Payload       any        `json:"payload"`
}

// digestOf hashes the canonical JSON preimage.
func digestOf(in digestInput) (string, error) {
	encoded, err := json.Marshal(in)
	if err != nil {
		return "", invalidSchema("digest preimage: %v", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}
