// Package diva implements the diva/v1 evolution strategy over the shared
// laputa/evolution contracts: bounded evidence collection, model-inferred
// Work reconciliation and reflection, and ordered effect routing through
// the host-bound Domain port. The package owns the strategy's prompts and
// validation policy; it imports no host runtime and holds no scheduler.
package diva

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ProjectViVy/laputa/laputa/evolution"
)

// Stage IDs are the fixed node identities of the six-stage graph.
const (
	StageCollectID   = "collect"
	StagePrepareID   = "prepare"
	StageReconcileID = "reconcile"
	StageReflectID   = "reflect"
	StageEffectsID   = "effects"
	StageFinishID    = "finish"
)

// OutcomeStatus is the run's durable terminal classification.
type OutcomeStatus string

const (
	OutcomeApplied          OutcomeStatus = "applied"
	OutcomeSubmitted        OutcomeStatus = "submitted"
	OutcomeNoChange         OutcomeStatus = "no_change"
	OutcomePartial          OutcomeStatus = "partial"
	OutcomeRecoveryRequired OutcomeStatus = "recovery_required"
)

// Outcome reports the frozen window and every committed effect receipt.
type Outcome struct {
	Status   OutcomeStatus             `json:"status"`
	Window   evolution.Window          `json:"window"`
	Receipts []evolution.EffectReceipt `json:"receipts,omitempty"`
	Reason   string                    `json:"reason,omitempty"`
}

// chainDoc is the internal pipeline envelope carried between stages. It is
// strategy wire format, not a shared contract: only the node IDs and
// semantics are frozen, and every decode is strict.
type chainDoc struct {
	Input                evolution.Input               `json:"input"`
	Batch                *evolution.EvidenceBatch      `json:"batch"`
	AvailableSkills      []evolution.LocalSkillSummary `json:"available_skills,omitempty"`
	SelectedSkill        *evolution.LocalSkillDocument `json:"selected_skill,omitempty"`
	SkillSourceAvailable bool                          `json:"skill_source_available,omitempty"`
	ReconcileReceipt     *evolution.EffectReceipt      `json:"reconcile_receipt,omitempty"`
	Candidates           []evolution.Candidate         `json:"candidates,omitempty"`
	NoChangeReason       string                        `json:"no_change_reason,omitempty"`
	Receipts             []evolution.EffectReceipt     `json:"receipts,omitempty"`
	Stopped              bool                          `json:"stopped,omitempty"`
	StopReason           string                        `json:"stop_reason,omitempty"`
}

func marshal(v any) (json.RawMessage, error) { return json.Marshal(v) }

func decodeStrict[T any](raw json.RawMessage, dst *T) error {
	return evolution.DecodeStrictJSON(raw, dst)
}

// bindingScope maps the trusted run binding to its effective write scope:
// personal when no workspace is bound, workspace otherwise.
func bindingScope(b evolution.RunBinding) evolution.Scope {
	if b.WorkspaceID == "" {
		return evolution.Scope{SubjectID: b.SubjectID, Kind: evolution.ScopePersonal}
	}
	return evolution.Scope{SubjectID: b.SubjectID, Kind: evolution.ScopeWorkspace, WorkspaceID: b.WorkspaceID}
}

// Strategy is the diva/v1 strategy bound to a Domain and a Model. The
// constructor is the only place authority enters; stage calls receive
// JSON, never principals.
type Strategy struct {
	domain evolution.Domain
	model  evolution.Model
}

// New binds the strategy. Both ports may be nil only in tests that never
// reach their stage.
func New(domain evolution.Domain, model evolution.Model) *Strategy {
	return &Strategy{domain: domain, model: model}
}

// emptyBatch reports whether the batch carries no collectable input; the
// chain must produce no_change with zero model calls.
func emptyBatch(b *evolution.EvidenceBatch) bool {
	return b == nil || (len(b.Entries) == 0 && len(b.Sources) == 0 && len(b.Persona) == 0)
}

// Run executes the full six-stage chain in order. operationKey anchors
// effect operation IDs; the host supplies its committed node key.
func (s *Strategy) Run(ctx context.Context, input evolution.Input, operationKey string) (Outcome, error) {
	raw, err := marshal(input)
	if err != nil {
		return Outcome{}, err
	}
	for _, stage := range []string{StageCollectID, StagePrepareID, StageReconcileID, StageReflectID, StageEffectsID, StageFinishID} {
		raw, err = s.RunStage(ctx, stage, raw, operationKey)
		if err != nil {
			return Outcome{}, err
		}
	}
	var out Outcome
	if err := decodeStrict(raw, &out); err != nil {
		return Outcome{}, err
	}
	return out, nil
}

// RunStage executes one fixed stage over its JSON input; operationKey is
// the trusted node operation key used to derive stable effect IDs.
func (s *Strategy) RunStage(ctx context.Context, stage string, input json.RawMessage, operationKey string) (json.RawMessage, error) {
	switch stage {
	case StageCollectID:
		return s.collect(ctx, input)
	case StagePrepareID:
		return s.prepare(ctx, input)
	case StageReconcileID:
		return s.reconcile(ctx, input, operationKey)
	case StageReflectID:
		return s.reflect(ctx, input)
	case StageEffectsID:
		return s.effects(ctx, input, operationKey)
	case StageFinishID:
		return s.finish(ctx, input)
	}
	return nil, &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "unknown stage " + stage}
}

// collect gathers the admitted window's evidence through Domain; the host
// input itself supplies binding and window.
func (s *Strategy) collect(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var in evolution.Input
	if err := decodeStrict(raw, &in); err != nil {
		return nil, err
	}
	batch, err := s.domain.Collect(ctx, in.Window)
	if err != nil {
		return nil, err
	}
	return marshal(chainDoc{Input: in, Batch: &batch})
}

// prepareBounds caps the total evidence characters handed to inference.
// Oversized batches are truncated deterministically — summarize is an
// option the prompts leave for later, never a new writer.
const prepareEntryBudget = 16000

const (
	maxLocalSkillSummaries = 32
	maxLocalSkillContent   = 16 << 10
)

func (s *Strategy) prepare(_ context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var doc chainDoc
	if err := decodeStrict(raw, &doc); err != nil {
		return nil, err
	}
	if doc.Batch == nil {
		return nil, &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "prepare requires batch"}
	}
	total := 0
	kept := doc.Batch.Entries[:0]
	for _, e := range doc.Batch.Entries {
		if total+len(e.Body) > prepareEntryBudget {
			break
		}
		total += len(e.Body)
		kept = append(kept, e)
	}
	doc.Batch.Entries = kept
	return marshal(doc)
}

// reconcile applies a bounded Work patch inferred from scoped ACTMEM plus
// evidence. Empty batches and empty patches do nothing. Replays resolve
// through the domain receipt before writing.
func (s *Strategy) reconcile(ctx context.Context, raw json.RawMessage, operationKey string) (json.RawMessage, error) {
	var doc chainDoc
	if err := decodeStrict(raw, &doc); err != nil {
		return nil, err
	}
	if doc.Batch == nil {
		return nil, &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "reconcile requires batch"}
	}
	if emptyBatch(doc.Batch) {
		return marshal(doc)
	}
	reply, err := s.model.Infer(ctx, evolution.ModelRequest{
		Stage:        evolution.StageReconcile,
		Prompt:       PromptReconcile,
		InputJSON:    raw,
		OutputSchema: json.RawMessage(SchemaWorkPatch),
	})
	if err != nil {
		return nil, err
	}
	var patch evolution.WorkPatch
	if err := decodeStrict(reply.OutputJSON, &patch); err != nil {
		return nil, err
	}
	if len(patch.Changes) == 0 {
		return marshal(doc)
	}
	effect, err := evolution.NewEffect(operationKey+":work", evolution.KindWorkPatch, &patch, bindingScope(doc.Input.Binding), doc.Input.Binding.DestinationID)
	if err != nil {
		return nil, err
	}
	receipt, stopped, err := s.resolveOrApply(ctx, effect)
	if err != nil {
		return nil, err
	}
	if stopped {
		doc.Stopped = true
		doc.StopReason = "apply_error"
		return marshal(doc)
	}
	doc.ReconcileReceipt = &receipt
	switch receipt.Status {
	case evolution.StatusUnknown:
		doc.Stopped = true
		doc.StopReason = "unknown_outcome"
	case evolution.StatusRejected:
		doc.Stopped = true
		doc.StopReason = "rejected"
	}
	return marshal(doc)
}

// reflect produces typed candidates — never writes. Invalid model output
// (unknown kinds, mission/dream variants, malformed JSON) fails the run
// before any candidate reaches effects.
func (s *Strategy) reflect(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var doc chainDoc
	if err := decodeStrict(raw, &doc); err != nil {
		return nil, err
	}
	if doc.Batch == nil {
		return nil, &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "reflect requires batch"}
	}
	if doc.Stopped {
		return marshal(doc)
	}
	if emptyBatch(doc.Batch) {
		doc.NoChangeReason = "no new input"
		return marshal(doc)
	}
	if source, ok := s.domain.(evolution.LocalSkillSource); ok {
		doc.SkillSourceAvailable = true
		skills, err := source.ListLocalSkills(ctx)
		if err != nil {
			return nil, fmt.Errorf("list local Skills for reflection: %w", err)
		}
		if len(skills) > maxLocalSkillSummaries {
			return nil, &evolution.ContractError{Code: evolution.ErrBudgetExceeded, Message: "local Skill index exceeds reflection budget"}
		}
		doc.AvailableSkills = skills
		if len(skills) > 0 {
			selectionInput, err := marshal(struct {
				Batch  *evolution.EvidenceBatch      `json:"batch"`
				Skills []evolution.LocalSkillSummary `json:"skills"`
			}{Batch: doc.Batch, Skills: skills})
			if err != nil {
				return nil, err
			}
			selection, err := s.model.Infer(ctx, evolution.ModelRequest{
				Stage: evolution.StageSkillSelect, Prompt: PromptSkillSelect,
				InputJSON: selectionInput, OutputSchema: json.RawMessage(SchemaSkillSelect),
			})
			if err != nil {
				return nil, err
			}
			var selected struct {
				SkillName string `json:"skill_name"`
			}
			if err := decodeStrict(selection.OutputJSON, &selected); err != nil {
				return nil, err
			}
			if selected.SkillName != "" {
				var summary *evolution.LocalSkillSummary
				for i := range skills {
					if skills[i].Name == selected.SkillName {
						summary = &skills[i]
						break
					}
				}
				if summary == nil {
					return nil, &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "selected Skill is outside the authorized local index"}
				}
				document, err := source.ReadLocalSkill(ctx, selected.SkillName)
				if err != nil {
					return nil, fmt.Errorf("read selected local Skill: %w", err)
				}
				if document.Name != summary.Name || document.Hash == "" || document.Hash != summary.Hash || len(document.Content) > maxLocalSkillContent {
					return nil, &evolution.ContractError{Code: evolution.ErrRevisionConflict, Message: "selected Skill changed or exceeds reflection budget"}
				}
				doc.SelectedSkill = &document
			}
		}
	}
	reflectionInput, err := marshal(doc)
	if err != nil {
		return nil, err
	}
	reply, err := s.model.Infer(ctx, evolution.ModelRequest{
		Stage:        evolution.StageReflect,
		Prompt:       PromptReflect,
		InputJSON:    reflectionInput,
		OutputSchema: json.RawMessage(SchemaReflectionOutput),
	})
	if err != nil {
		return nil, err
	}
	out, err := evolution.DecodeReflectionOutput(reply.OutputJSON)
	if err != nil {
		return nil, err
	}
	if err := validateSkillCandidates(out.Candidates, doc); err != nil {
		return nil, err
	}
	doc.Candidates = out.Candidates
	doc.NoChangeReason = out.NoChangeReason
	// Only the model consumes the evidence batch. Effects and finish use
	// the admitted input, typed candidates and receipts; carrying the raw
	// evidence beside a full memory candidate can exceed the packet bound.
	// Candidate bodies and source references remain complete.
	doc.Batch = nil
	doc.AvailableSkills = nil
	doc.SelectedSkill = nil
	return marshal(doc)
}

func validateSkillCandidates(candidates []evolution.Candidate, doc chainDoc) error {
	available := make(map[string]evolution.LocalSkillSummary, len(doc.AvailableSkills))
	for _, skill := range doc.AvailableSkills {
		available[skill.Name] = skill
	}
	for _, candidate := range candidates {
		if candidate.Kind != evolution.KindSkillProposal {
			continue
		}
		proposal := candidate.SkillProposal
		if proposal == nil {
			return &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "local Skill proposal missing typed payload"}
		}
		if !doc.SkillSourceAvailable {
			return &evolution.ContractError{Code: evolution.ErrCapabilityUnavailable, Message: "local Skill source is not available"}
		}
		if proposal.Operation == evolution.SkillProposalUpdate {
			if doc.SelectedSkill == nil || proposal.SkillName != doc.SelectedSkill.Name || proposal.BaseHash != doc.SelectedSkill.Hash {
				return &evolution.ContractError{Code: evolution.ErrRevisionConflict, Message: "local Skill update does not match the selected revision"}
			}
		} else if _, exists := available[proposal.SkillName]; exists {
			return &evolution.ContractError{Code: evolution.ErrIdempotencyConflict, Message: "new local Skill duplicates an existing identity"}
		}
		if !referencesEvidence(proposal.Sources, doc.Batch) {
			return &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "local Skill proposal cites evidence outside the admitted batch"}
		}
	}
	return nil
}

func referencesEvidence(refs []evolution.SourceRef, batch *evolution.EvidenceBatch) bool {
	if batch == nil || len(refs) == 0 {
		return false
	}
	valid := make(map[evolution.SourceRef]struct{})
	for _, ref := range batch.Sources {
		valid[ref] = struct{}{}
	}
	for _, entry := range batch.Entries {
		for _, ref := range entry.Sources {
			valid[ref] = struct{}{}
		}
	}
	for _, ref := range refs {
		if _, ok := valid[ref]; ok {
			return true
		}
	}
	return false
}

// effects routes candidates in stable order. Each operation ID is the
// node operation key plus the candidate index; a committed receipt for the
// same operation replays instead of re-applying, a mismatched payload
// fails, and an unknown outcome stops the chain without rerunning.
func (s *Strategy) effects(ctx context.Context, raw json.RawMessage, operationKey string) (json.RawMessage, error) {
	var doc chainDoc
	if err := decodeStrict(raw, &doc); err != nil {
		return nil, err
	}
	if doc.Stopped {
		return marshal(doc)
	}
	scope := bindingScope(doc.Input.Binding)
	for i, candidate := range doc.Candidates {
		effect, err := evolution.NewEffect(fmt.Sprintf("%s:e%d", operationKey, i), candidate.Kind, candidate.Payload(), scope, doc.Input.Binding.DestinationID)
		if err != nil {
			return nil, err
		}
		receipt, stop, err := s.resolveOrApply(ctx, effect)
		if err != nil {
			return nil, err
		}
		if stop {
			doc.Stopped = true
			doc.StopReason = "apply_error"
			return marshal(doc)
		}
		doc.Receipts = append(doc.Receipts, receipt)
		switch receipt.Status {
		case evolution.StatusUnknown:
			doc.Stopped = true
			doc.StopReason = "unknown_outcome"
			return marshal(doc)
		case evolution.StatusRejected:
			doc.Stopped = true
			doc.StopReason = "rejected"
			return marshal(doc)
		}
	}
	return marshal(doc)
}

// resolveOrApply returns a committed receipt for the operation or applies
// the effect. A recorded digest that differs from the incoming payload is
// an idempotency conflict; non-not-found lookup errors propagate. A failed
// Apply is not a run error: it stops the chain so committed receipts stay
// visible.
func (s *Strategy) resolveOrApply(ctx context.Context, effect evolution.Effect) (evolution.EffectReceipt, bool, error) {
	recorded, err := s.domain.Lookup(ctx, effect.OperationID)
	switch {
	case err == nil:
		recordedEffect := evolution.Effect{OperationID: recorded.OperationID, PayloadDigest: recorded.PayloadDigest}
		if err := evolution.CheckReplay(recordedEffect, effect); err != nil {
			return evolution.EffectReceipt{}, false, err
		}
		return recorded, false, nil
	case evolution.CodeOf(err) == evolution.ErrEffectNotFound:
		receipt, aerr := s.domain.Apply(ctx, effect)
		if aerr != nil {
			return evolution.EffectReceipt{}, true, nil
		}
		return receipt, false, nil
	default:
		return evolution.EffectReceipt{}, false, err
	}
}

// finish classifies the run: recovery_required outranks partial, partial
// outranks submitted, submitted outranks applied, and an empty run is
// no_change. The window resolves only when nothing remains unresolved.
func (s *Strategy) finish(_ context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var doc chainDoc
	if err := decodeStrict(raw, &doc); err != nil {
		return nil, err
	}
	out := Outcome{Window: doc.Input.Window, Receipts: doc.Receipts, Reason: doc.StopReason}
	if doc.ReconcileReceipt != nil {
		out.Receipts = append([]evolution.EffectReceipt{*doc.ReconcileReceipt}, out.Receipts...)
	}
	unknown := false
	for _, r := range out.Receipts {
		if r.Status == evolution.StatusUnknown {
			unknown = true
		}
	}
	switch {
	case unknown:
		out.Status = OutcomeRecoveryRequired
	case doc.Stopped:
		out.Status = OutcomePartial
	case len(out.Receipts) == 0:
		out.Status = OutcomeNoChange
	default:
		submitted := false
		for _, r := range out.Receipts {
			if r.Status == evolution.StatusSubmitted {
				submitted = true
			}
		}
		if submitted {
			out.Status = OutcomeSubmitted
		} else {
			out.Status = OutcomeApplied
		}
	}
	return marshal(out)
}
