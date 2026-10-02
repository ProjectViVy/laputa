package evolution

import (
	"encoding/json"
)

// EffectKind is the closed set of effect variants. Mission and Dream
// mutations are intentionally absent; any other value rejects before
// mutation.
type EffectKind string

const (
	KindWorkPatch          EffectKind = "work_patch"
	KindMemoryMutation     EffectKind = "memory_mutation"
	KindPersonaRequest     EffectKind = "persona_request"
	KindCapabilityProposal EffectKind = "capability_proposal"
	KindReflectionNote     EffectKind = "reflection_note"
)

// MutationOperation is the closed memory-mutation vocabulary.
type MutationOperation string

const (
	MutationCreate    MutationOperation = "create"
	MutationUpdate    MutationOperation = "update"
	MutationTombstone MutationOperation = "tombstone"
)

// InferenceClass marks whether a mutation body was observed or inferred.
type InferenceClass string

const (
	InferenceObserved InferenceClass = "observed"
	InferenceInferred InferenceClass = "inferred"
)

// PersonaRequestKind is the closed persona-review vocabulary; mission and
// dream are not agent-proposable kinds.
type PersonaRequestKind string

const (
	PersonaRequestIdentity         PersonaRequestKind = "identity"
	PersonaRequestRelationship     PersonaRequestKind = "relationship"
	PersonaRequestDark             PersonaRequestKind = "dark"
	PersonaRequestUserObservations PersonaRequestKind = "user_observations"
	PersonaRequestWorld            PersonaRequestKind = "world"
)

// MemoryMutationPayload is the typed memory_mutation payload. Scope is the
// bound destination scope; it is not a payload field.
type MemoryMutationPayload struct {
	Operation        MutationOperation `json:"operation"`
	RecordID         string            `json:"record_id"`
	ExpectedRevision uint64            `json:"expected_revision"`
	ExpectedAbsent   bool              `json:"expected_absent"`
	Body             string            `json:"body"`
	Sources          []SourceRef       `json:"sources"`
	Inference        InferenceClass    `json:"inference"`
}

// Validate enforces the closed enums and source shapes.
func (p *MemoryMutationPayload) Validate() error {
	switch p.Operation {
	case MutationCreate, MutationUpdate, MutationTombstone:
	default:
		return invalidSchema("unknown mutation operation %q", p.Operation)
	}
	switch p.Inference {
	case InferenceObserved, InferenceInferred:
	default:
		return invalidSchema("unknown inference class %q", p.Inference)
	}
	if p.RecordID == "" {
		return invalidSchema("memory_mutation record_id required")
	}
	return validateSources(p.Sources)
}

func validateSources(sources []SourceRef) error {
	for _, s := range sources {
		if err := s.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// PersonaRequestPayload is the typed persona_request payload. The authority's
// finer review rules still apply downstream.
type PersonaRequestPayload struct {
	Kind             PersonaRequestKind `json:"kind"`
	BaseRevision     uint64             `json:"base_revision"`
	ProposedMarkdown string             `json:"proposed_markdown"`
	Reason           string             `json:"reason"`
	Sources          []SourceRef        `json:"sources"`
}

// Validate enforces the closed proposable-kind enum. Mission and Dream are
// never proposable.
func (p *PersonaRequestPayload) Validate() error {
	switch p.Kind {
	case PersonaRequestIdentity, PersonaRequestRelationship, PersonaRequestDark, PersonaRequestUserObservations, PersonaRequestWorld:
	default:
		return invalidSchema("unknown persona request kind %q", p.Kind)
	}
	if p.ProposedMarkdown == "" {
		return invalidSchema("persona_request proposed_markdown required")
	}
	return validateSources(p.Sources)
}

// CapabilityProposalPayload submits to EvoMap only; it carries no install or
// publish permission.
type CapabilityProposalPayload struct {
	Name             string      `json:"name"`
	Description      string      `json:"description"`
	ProposedArtifact string      `json:"proposed_artifact"`
	Sources          []SourceRef `json:"sources"`
}

// Validate enforces the closed enum and required fields.
func (p *CapabilityProposalPayload) Validate() error {
	if p.Name == "" || p.ProposedArtifact == "" {
		return invalidSchema("capability_proposal name and proposed_artifact required")
	}
	return validateSources(p.Sources)
}

// ReflectionNotePayload is the typed reflection_note payload.
type ReflectionNotePayload struct {
	Body    string      `json:"body"`
	Sources []SourceRef `json:"sources"`
}

// Validate enforces the required body.
func (p *ReflectionNotePayload) Validate() error {
	if p.Body == "" {
		return invalidSchema("reflection_note body required")
	}
	return validateSources(p.Sources)
}

// effectPayloads is the shared typed-union carrier for Effect and Candidate.
// Exactly one field is non-nil and must match Kind.
type effectPayloads struct {
	Kind               EffectKind                 `json:"kind"`
	WorkPatch          *WorkPatch                 `json:"work_patch,omitempty"`
	MemoryMutation     *MemoryMutationPayload     `json:"memory_mutation,omitempty"`
	PersonaRequest     *PersonaRequestPayload     `json:"persona_request,omitempty"`
	CapabilityProposal *CapabilityProposalPayload `json:"capability_proposal,omitempty"`
	ReflectionNote     *ReflectionNotePayload     `json:"reflection_note,omitempty"`
}

// Payload returns the typed payload matching Kind, or nil.
func (p *effectPayloads) Payload() any {
	switch p.Kind {
	case KindWorkPatch:
		if p.WorkPatch != nil {
			return p.WorkPatch
		}
	case KindMemoryMutation:
		if p.MemoryMutation != nil {
			return p.MemoryMutation
		}
	case KindPersonaRequest:
		if p.PersonaRequest != nil {
			return p.PersonaRequest
		}
	case KindCapabilityProposal:
		if p.CapabilityProposal != nil {
			return p.CapabilityProposal
		}
	case KindReflectionNote:
		if p.ReflectionNote != nil {
			return p.ReflectionNote
		}
	}
	return nil
}

func (p *effectPayloads) validate() error {
	present := 0
	for _, set := range []bool{p.WorkPatch != nil, p.MemoryMutation != nil, p.PersonaRequest != nil, p.CapabilityProposal != nil, p.ReflectionNote != nil} {
		if set {
			present++
		}
	}
	switch p.Kind {
	case KindWorkPatch, KindMemoryMutation, KindPersonaRequest, KindCapabilityProposal, KindReflectionNote:
	default:
		return invalidSchema("unknown effect kind %q", p.Kind)
	}
	if present != 1 || p.Payload() == nil {
		return invalidSchema("effect %s requires exactly one matching typed payload", p.Kind)
	}
	switch payload := p.Payload().(type) {
	case *WorkPatch:
		for i := range payload.Changes {
			if err := payload.Changes[i].Validate(); err != nil {
				return err
			}
		}
	case *MemoryMutationPayload:
		return payload.Validate()
	case *PersonaRequestPayload:
		return payload.Validate()
	case *CapabilityProposalPayload:
		return payload.Validate()
	case *ReflectionNotePayload:
		return payload.Validate()
	}
	return nil
}

// Effect is one typed, replayable effect. OperationID comes from the
// committed operation key plus stable candidate index, never from model
// text.
type Effect struct {
	OperationID   string `json:"operation_id"`
	PayloadDigest string `json:"payload_digest"`
	effectPayloads
}

// effectWire is the on-the-wire envelope shape.
type effectWire struct {
	OperationID        string                     `json:"operation_id"`
	PayloadDigest      string                     `json:"payload_digest"`
	Kind               EffectKind                 `json:"kind"`
	WorkPatch          *WorkPatch                 `json:"work_patch,omitempty"`
	MemoryMutation     *MemoryMutationPayload     `json:"memory_mutation,omitempty"`
	PersonaRequest     *PersonaRequestPayload     `json:"persona_request,omitempty"`
	CapabilityProposal *CapabilityProposalPayload `json:"capability_proposal,omitempty"`
	ReflectionNote     *ReflectionNotePayload     `json:"reflection_note,omitempty"`
}

// Payload returns the typed payload matching the effect kind.
func (e Effect) Payload() any { return e.effectPayloads.Payload() }

// Validate enforces the envelope invariants.
func (e Effect) Validate() error {
	if e.OperationID == "" {
		return invalidSchema("effect operation_id required")
	}
	if e.PayloadDigest == "" {
		return invalidSchema("effect payload_digest required")
	}
	return e.effectPayloads.validate()
}

// DecodeEffect strict-decodes one effect. Unknown fields, unknown or absent
// kinds, mismatched payload keys, and duplicate keys reject before mutation.
func DecodeEffect(data []byte) (Effect, error) {
	var w effectWire
	if err := DecodeStrictJSON(data, &w); err != nil {
		return Effect{}, err
	}
	e := Effect{
		OperationID:   w.OperationID,
		PayloadDigest: w.PayloadDigest,
		effectPayloads: effectPayloads{
			Kind:               w.Kind,
			WorkPatch:          w.WorkPatch,
			MemoryMutation:     w.MemoryMutation,
			PersonaRequest:     w.PersonaRequest,
			CapabilityProposal: w.CapabilityProposal,
			ReflectionNote:     w.ReflectionNote,
		},
	}
	if err := e.Validate(); err != nil {
		return Effect{}, err
	}
	return e, nil
}

// MarshalJSON emits the envelope with exactly one typed payload field.
func (e Effect) MarshalJSON() ([]byte, error) {
	return json.Marshal(effectWire{
		OperationID:        e.OperationID,
		PayloadDigest:      e.PayloadDigest,
		Kind:               e.Kind,
		WorkPatch:          e.WorkPatch,
		MemoryMutation:     e.MemoryMutation,
		PersonaRequest:     e.PersonaRequest,
		CapabilityProposal: e.CapabilityProposal,
		ReflectionNote:     e.ReflectionNote,
	})
}

// NewEffect builds an effect whose payload digest is SHA-256 of the canonical
// typed payload plus effective scope and destination.
func NewEffect(operationID string, kind EffectKind, payload any, scope Scope, destinationID string) (Effect, error) {
	e := Effect{OperationID: operationID}
	e.effectPayloads.Kind = kind
	switch p := payload.(type) {
	case *WorkPatch:
		e.WorkPatch = p
	case *MemoryMutationPayload:
		e.MemoryMutation = p
	case *PersonaRequestPayload:
		e.PersonaRequest = p
	case *CapabilityProposalPayload:
		e.CapabilityProposal = p
	case *ReflectionNotePayload:
		e.ReflectionNote = p
	default:
		return Effect{}, invalidSchema("effect %s payload type mismatch", kind)
	}
	if e.Payload() == nil {
		return Effect{}, invalidSchema("effect %s payload type mismatch", kind)
	}
	digest, err := digestOf(digestInput{Scope: scope, DestinationID: destinationID, Kind: kind, Payload: e.Payload()})
	if err != nil {
		return Effect{}, err
	}
	e.PayloadDigest = digest
	if err := e.Validate(); err != nil {
		return Effect{}, err
	}
	return e, nil
}

// CheckReplay compares a recorded effect with an incoming one bearing the
// same operation_id. Same digest replays (the caller returns the recorded
// receipt); a changed payload or scope changes the digest and conflicts.
func CheckReplay(recorded, incoming Effect) error {
	if recorded.OperationID != incoming.OperationID {
		return &ContractError{Code: ErrInvalidSchema, Message: "replay requires equal operation_id"}
	}
	if recorded.PayloadDigest != incoming.PayloadDigest {
		return &ContractError{Code: ErrIdempotencyConflict, Message: "operation payload changed under the same operation_id"}
	}
	return nil
}

// EffectStatus is the closed receipt vocabulary.
type EffectStatus string

const (
	StatusApplied   EffectStatus = "applied"
	StatusNoChange  EffectStatus = "no_change"
	StatusSubmitted EffectStatus = "submitted"
	StatusRejected  EffectStatus = "rejected"
	StatusUnknown   EffectStatus = "unknown"
)

// EffectReceipt reports one effect's durable outcome.
type EffectReceipt struct {
	OperationID   string       `json:"operation_id"`
	PayloadDigest string       `json:"payload_digest"`
	Status        EffectStatus `json:"status"`
	TargetRef     string       `json:"target_ref"`
	Revision      uint64       `json:"revision"`
	ErrorCode     string       `json:"error_code"`
}

// Candidate is a proposed effect before binding: the operation id and digest
// are assigned by the host from the committed operation key plus index.
type Candidate struct {
	effectPayloads
}

// DecodeCandidate strict-decodes one candidate payload union.
func DecodeCandidate(data []byte) (Candidate, error) {
	var c Candidate
	if err := DecodeStrictJSON(data, &c.effectPayloads); err != nil {
		return Candidate{}, err
	}
	if err := c.effectPayloads.validate(); err != nil {
		return Candidate{}, err
	}
	return c, nil
}

// ReflectionOutput is the reflect node's closed output: exactly one of
// candidates or no_change_reason is nonempty.
type ReflectionOutput struct {
	Candidates     []Candidate `json:"candidates"`
	NoChangeReason string      `json:"no_change_reason"`
}

// DecodeReflectionOutput strict-decodes and enforces the one-of invariant.
func DecodeReflectionOutput(data []byte) (ReflectionOutput, error) {
	var out ReflectionOutput
	if err := DecodeStrictJSON(data, &out); err != nil {
		return ReflectionOutput{}, err
	}
	for i := range out.Candidates {
		if err := out.Candidates[i].effectPayloads.validate(); err != nil {
			return ReflectionOutput{}, err
		}
	}
	if (len(out.Candidates) > 0) == (out.NoChangeReason != "") {
		return ReflectionOutput{}, invalidSchema("exactly one of candidates or no_change_reason required")
	}
	return out, nil
}
