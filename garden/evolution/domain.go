// Package evolution binds the laputa/evolution trusted Domain port to the
// real Garden authorities: the ACTMEM activity store, the Persona authority,
// the scoped memory backend, the ingest ledger, and the EvoMap candidate
// intake. The binding is fixed at construction; the strategy cannot reach
// other subjects, scopes, or destinations.
package evolution

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ProjectViVy/laputa/garden/internal/ingest"
	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/actmem"
	laputaevolution "github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/laputa/persona"
)

// ActivityReader supplies the committed-activity window the strategy's
// collect stage reads. ingest.Service implements it.
type ActivityReader interface {
	Window(ctx context.Context, workspace string, after, through uint64) ([]ingest.WindowRow, error)
}

// Proposer submits a capability proposal to the EvoMap candidate intake.
// Submission is not installation: artifact lifecycle stays with EvoMap.
type Proposer interface {
	SubmitCapabilityProposal(ctx context.Context, in ProposedCapability) (string, error)
}

// ProposedCapability is the bounded proposal handed to EvoMap.
type ProposedCapability struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Artifact    string   `json:"artifact"`
	Sources     []string `json:"sources,omitempty"`
}

// Deps wires a Domain to the bound authorities. Memory, Proposals and
// Activity are optional: absent ports reject the corresponding effects
// (status rejected, code <kind>_unavailable) instead of pretending success.
type Deps struct {
	Scope         laputaevolution.Scope
	DestinationID string
	Actmem        *actmem.Store
	Persona       *persona.Service
	Memory        memory.Backend
	Proposals     Proposer
	Activity      ActivityReader
	// Dir is the domain-owned durable ledger directory (effect receipts and
	// reflection notes). Required.
	Dir string
}

// Domain implements laputaevolution.Domain against bound Garden services.
type Domain struct {
	deps   Deps
	ledger *effectLedger
}

// NewDomain constructs the bound port. Dir holds the durable ledger.
func NewDomain(deps Deps) (*Domain, error) {
	if err := deps.Scope.Validate(); err != nil {
		return nil, fmt.Errorf("evolution: bound scope: %w", err)
	}
	if deps.DestinationID == "" {
		return nil, errors.New("evolution: destination_id required")
	}
	if deps.Dir == "" {
		return nil, errors.New("evolution: ledger dir required")
	}
	ledger, err := openEffectLedger(deps.Dir)
	if err != nil {
		return nil, err
	}
	return &Domain{deps: deps, ledger: ledger}, nil
}

// Collect gathers the committed window's evidence rows plus the read-only
// ACTMEM revision and bounded Persona authority views for the bound scope.
// Reflection notes never re-enter: the ledger is not an evidence source.
func (d *Domain) Collect(ctx context.Context, w laputaevolution.Window) (laputaevolution.EvidenceBatch, error) {
	if w.SourceID == "" || w.After > w.Through {
		return laputaevolution.EvidenceBatch{}, &laputaevolution.ContractError{Code: laputaevolution.ErrInvalidSchema, Message: "window requires source_id and after <= through"}
	}
	batch := laputaevolution.EvidenceBatch{Window: w}
	if d.deps.Activity != nil && w.After < w.Through {
		rows, err := d.deps.Activity.Window(ctx, d.deps.Scope.WorkspaceID, w.After, w.Through)
		if err != nil {
			return laputaevolution.EvidenceBatch{}, err
		}
		for _, row := range rows {
			ref := laputaevolution.SourceRef{
				SourceID: w.SourceID, RecordID: fmt.Sprint(row.Seq), Scope: d.deps.Scope,
			}
			batch.Entries = append(batch.Entries, laputaevolution.Entry{
				ID:         "ing_" + fmt.Sprint(row.Seq),
				Section:    laputaevolution.SectionRecap,
				Scope:      d.deps.Scope,
				SessionID:  row.SessionID,
				EventID:    row.EventID,
				OccurredAt: row.OccurredAt.UTC().Format(time.RFC3339Nano),
				Body:       row.Content,
				Sources:    []laputaevolution.SourceRef{ref},
			})
			batch.Sources = append(batch.Sources, ref)
		}
	}
	if d.deps.Actmem != nil {
		activity, err := d.deps.Actmem.ReadScoped(d.deps.Scope, laputaevolution.ReadRequest{
			Sections: []laputaevolution.EntrySection{laputaevolution.SectionWork},
			MaxChars: laputaevolution.ActmemReadCapChars,
		})
		if err != nil {
			if actmem.CodeOf(err) == string(laputaevolution.ErrActmemCapExceeded) {
				return laputaevolution.EvidenceBatch{}, &laputaevolution.ContractError{Code: laputaevolution.ErrActmemCapExceeded, Message: "existing scoped Work exceeds reconciliation read budget"}
			}
			return laputaevolution.EvidenceBatch{}, err
		}
		batch.ActivityRevision = activity.Revision
		// Existing Work is explicit reconciliation context, not a new
		// activity source or an automatically injected foreground prompt.
		batch.Entries = append(batch.Entries, activity.Entries...)
	}
	if d.deps.Persona != nil {
		views, err := d.authorityViews()
		if err != nil {
			return laputaevolution.EvidenceBatch{}, err
		}
		batch.Persona = views
	}
	return batch, nil
}

// MissionRevision reports the current authority Mission revision (0 when
// unassigned). Hosts pin it into RunBinding and re-verify before admission.
func (d *Domain) MissionRevision(ctx context.Context) (uint64, error) {
	if d.deps.Persona == nil {
		return 0, nil
	}
	doc, err := d.deps.Persona.GetDocument(persona.KindMission)
	if err != nil {
		return 0, err
	}
	return doc.Revision, nil
}

// authorityViews projects every present authority file into the batch.
func (d *Domain) authorityViews() ([]laputaevolution.AuthorityView, error) {
	views := make([]laputaevolution.AuthorityView, 0, len(persona.AllKinds))
	for _, kind := range persona.AllKinds {
		doc, err := d.deps.Persona.GetDocument(kind)
		if errors.Is(err, persona.ErrUninitialized) {
			return nil, err
		}
		if err != nil {
			return nil, err
		}
		if !doc.Exists {
			continue
		}
		views = append(views, laputaevolution.AuthorityView{
			Kind:     laputaevolution.AuthorityKind(kind.String()),
			Revision: doc.Revision,
			Content:  doc.Content,
		})
	}
	return views, nil
}

// Apply routes one typed effect to its owning authority and records the
// durable receipt. A record lands in the ledger for every outcome except a
// hard authority error, which propagates instead.
func (d *Domain) Apply(ctx context.Context, e laputaevolution.Effect) (laputaevolution.EffectReceipt, error) {
	if err := e.Validate(); err != nil {
		return laputaevolution.EffectReceipt{}, err
	}
	var receipt laputaevolution.EffectReceipt
	var err error
	switch e.Kind {
	case laputaevolution.KindWorkPatch:
		receipt, err = d.applyWorkPatch(ctx, e)
	case laputaevolution.KindMemoryMutation:
		receipt, err = d.applyMemoryMutation(ctx, e)
	case laputaevolution.KindPersonaRequest:
		receipt, err = d.applyPersonaRequest(ctx, e)
	case laputaevolution.KindCapabilityProposal:
		receipt, err = d.applyCapabilityProposal(ctx, e)
	case laputaevolution.KindReflectionNote:
		receipt, err = d.applyReflectionNote(ctx, e)
	default:
		return laputaevolution.EffectReceipt{}, &laputaevolution.ContractError{Code: laputaevolution.ErrInvalidSchema, Message: "unknown effect kind " + string(e.Kind)}
	}
	if err != nil {
		return laputaevolution.EffectReceipt{}, err
	}
	if err := d.ledger.record(effectRecord{EffectReceipt: receipt, Kind: string(e.Kind)}); err != nil {
		return laputaevolution.EffectReceipt{}, err
	}
	return receipt, nil
}

// Lookup returns the committed receipt for an operation id. The local ledger
// is authoritative for every kind; memory mutations additionally fall back
// to the backend's own operation-id receipt so a crash between the backend
// commit and the ledger write still reconciles.
func (d *Domain) Lookup(ctx context.Context, operationID string) (laputaevolution.EffectReceipt, error) {
	rec, ok, err := d.ledger.lookup(operationID)
	if err != nil {
		return laputaevolution.EffectReceipt{}, err
	}
	if ok {
		return rec.EffectReceipt, nil
	}
	if d.deps.Memory != nil {
		mr, merr := d.deps.Memory.MutationStatus(ctx, operationID)
		if merr == nil && mr.OperationID != "" {
			return mr.EffectReceipt, nil
		}
		if merr != nil && !isNotFound(merr) {
			return laputaevolution.EffectReceipt{}, merr
		}
	}
	return laputaevolution.EffectReceipt{}, &laputaevolution.ContractError{Code: laputaevolution.ErrEffectNotFound, Message: "no receipt for " + operationID}
}

func (d *Domain) applyWorkPatch(ctx context.Context, e laputaevolution.Effect) (laputaevolution.EffectReceipt, error) {
	if d.deps.Actmem == nil {
		return rejected(e, "actmem_unavailable", "ACTMEM store is not bound"), nil
	}
	patch, _ := e.Payload().(*laputaevolution.WorkPatch)
	if patch == nil {
		return laputaevolution.EffectReceipt{}, &laputaevolution.ContractError{Code: laputaevolution.ErrInvalidSchema, Message: "work_patch payload missing"}
	}
	result, err := d.deps.Actmem.ApplyWorkPatch(d.deps.Scope, *patch)
	if err != nil {
		return rejected(e, "actmem_rejected", err.Error()), nil
	}
	status := laputaevolution.StatusApplied
	if !result.Changed {
		status = laputaevolution.StatusNoChange
	}
	return laputaevolution.EffectReceipt{
		OperationID:   e.OperationID,
		PayloadDigest: e.PayloadDigest,
		Status:        status,
		TargetRef:     "actmem",
		Revision:      result.Revision,
	}, nil
}

func (d *Domain) applyMemoryMutation(ctx context.Context, e laputaevolution.Effect) (laputaevolution.EffectReceipt, error) {
	if d.deps.Memory == nil {
		return rejected(e, "memory_unavailable", "memory backend is not bound"), nil
	}
	payload, _ := e.Payload().(*laputaevolution.MemoryMutationPayload)
	if payload == nil {
		return laputaevolution.EffectReceipt{}, &laputaevolution.ContractError{Code: laputaevolution.ErrInvalidSchema, Message: "memory_mutation payload missing"}
	}
	mutation := memory.AuthorizedMutation{
		Scope:            d.deps.Scope,
		DestinationID:    d.deps.DestinationID,
		OperationID:      e.OperationID,
		PayloadDigest:    e.PayloadDigest,
		Operation:        payload.Operation,
		RecordID:         payload.RecordID,
		ExpectedRevision: payload.ExpectedRevision,
		ExpectedAbsent:   payload.ExpectedAbsent,
		Body:             payload.Body,
		Sources:          payload.Sources,
		Inference:        payload.Inference,
	}
	if err := mutation.Validate(); err != nil {
		return laputaevolution.EffectReceipt{}, err
	}
	receipt, err := d.deps.Memory.Mutate(ctx, mutation)
	if err != nil {
		return rejected(e, "memory_rejected", err.Error()), nil
	}
	out := receipt.EffectReceipt
	out.OperationID = e.OperationID
	out.PayloadDigest = e.PayloadDigest
	return out, nil
}

func (d *Domain) applyPersonaRequest(ctx context.Context, e laputaevolution.Effect) (laputaevolution.EffectReceipt, error) {
	if d.deps.Persona == nil {
		return rejected(e, "persona_unavailable", "persona service is not bound"), nil
	}
	payload, _ := e.Payload().(*laputaevolution.PersonaRequestPayload)
	if payload == nil {
		return laputaevolution.EffectReceipt{}, &laputaevolution.ContractError{Code: laputaevolution.ErrInvalidSchema, Message: "persona_request payload missing"}
	}
	kind, actor, err := personaRequestTarget(payload.Kind)
	if err != nil {
		return laputaevolution.EffectReceipt{}, err
	}
	doc, err := d.deps.Persona.GetDocument(kind)
	if err != nil {
		return laputaevolution.EffectReceipt{}, err
	}
	if doc.Revision != payload.BaseRevision {
		return rejected(e, "persona_revision_mismatch", "base revision changed since inference"), nil
	}
	request, err := d.deps.Persona.CreateRequest(persona.ChangeRequest{
		Kind:             kind,
		BaseRevision:     payload.BaseRevision,
		BaseHash:         doc.ContentHash,
		ProposedMarkdown: payload.ProposedMarkdown,
		Actor:            actor,
		Reason:           payload.Reason,
	})
	if err != nil {
		return rejected(e, "persona_rejected", err.Error()), nil
	}
	return laputaevolution.EffectReceipt{
		OperationID:   e.OperationID,
		PayloadDigest: e.PayloadDigest,
		Status:        laputaevolution.StatusSubmitted,
		TargetRef:     "persona_request:" + request.ID,
		Revision:      request.BaseRevision,
	}, nil
}

// personaRequestTarget maps the closed proposal vocabulary to the authority
// kind and actor. Dark-persona proposals follow the autodream actor channel;
// every other proposable kind is an agent request.
func personaRequestTarget(kind laputaevolution.PersonaRequestKind) (persona.Kind, persona.RequestActor, error) {
	switch kind {
	case laputaevolution.PersonaRequestIdentity:
		return persona.KindIdentity, persona.ActorAgent, nil
	case laputaevolution.PersonaRequestRelationship:
		return persona.KindRelationship, persona.ActorAgent, nil
	case laputaevolution.PersonaRequestUserObservations:
		return persona.KindUser, persona.ActorAgent, nil
	case laputaevolution.PersonaRequestWorld:
		return persona.KindWorld, persona.ActorAgent, nil
	case laputaevolution.PersonaRequestDark:
		return persona.KindDark, persona.ActorAutodream, nil
	default:
		return 0, "", &laputaevolution.ContractError{Code: laputaevolution.ErrInvalidSchema, Message: "persona kind " + string(kind) + " is not proposable"}
	}
}

func (d *Domain) applyCapabilityProposal(ctx context.Context, e laputaevolution.Effect) (laputaevolution.EffectReceipt, error) {
	if d.deps.Proposals == nil {
		return rejected(e, "evomap_unavailable", "capability proposal intake is not bound"), nil
	}
	payload, _ := e.Payload().(*laputaevolution.CapabilityProposalPayload)
	if payload == nil {
		return laputaevolution.EffectReceipt{}, &laputaevolution.ContractError{Code: laputaevolution.ErrInvalidSchema, Message: "capability_proposal payload missing"}
	}
	sources := make([]string, 0, len(payload.Sources))
	for _, s := range payload.Sources {
		sources = append(sources, s.SourceID+"/"+s.RecordID)
	}
	ref, err := d.deps.Proposals.SubmitCapabilityProposal(ctx, ProposedCapability{
		Name: payload.Name, Description: payload.Description, Artifact: payload.ProposedArtifact, Sources: sources,
	})
	if err != nil {
		return rejected(e, "evomap_rejected", err.Error()), nil
	}
	return laputaevolution.EffectReceipt{
		OperationID:   e.OperationID,
		PayloadDigest: e.PayloadDigest,
		Status:        laputaevolution.StatusSubmitted,
		TargetRef:     ref,
	}, nil
}

func (d *Domain) applyReflectionNote(ctx context.Context, e laputaevolution.Effect) (laputaevolution.EffectReceipt, error) {
	payload, _ := e.Payload().(*laputaevolution.ReflectionNotePayload)
	if payload == nil {
		return laputaevolution.EffectReceipt{}, &laputaevolution.ContractError{Code: laputaevolution.ErrInvalidSchema, Message: "reflection_note payload missing"}
	}
	noteID, err := d.ledger.recordNote(e.OperationID, e.PayloadDigest, payload)
	if err != nil {
		return laputaevolution.EffectReceipt{}, err
	}
	return laputaevolution.EffectReceipt{
		OperationID:   e.OperationID,
		PayloadDigest: e.PayloadDigest,
		Status:        laputaevolution.StatusApplied,
		TargetRef:     "reflection_note:" + noteID,
	}, nil
}

func rejected(e laputaevolution.Effect, code, _ string) laputaevolution.EffectReceipt {
	return laputaevolution.EffectReceipt{
		OperationID:   e.OperationID,
		PayloadDigest: e.PayloadDigest,
		Status:        laputaevolution.StatusRejected,
		ErrorCode:     code,
	}
}

func isNotFound(err error) bool {
	return laputaevolution.CodeOf(err) == laputaevolution.ErrEffectNotFound || errors.Is(err, os.ErrNotExist)
}

// effectLedger is the domain-owned durable receipt and note store: an
// append-only fsync'd JSONL file. Receipts reconcile restarts; notes persist
// reports without re-entering evidence.
type effectLedger struct {
	mu   sync.Mutex
	path string
}

type effectRecord struct {
	laputaevolution.EffectReceipt
	Kind string `json:"kind"`
}

type noteRecord struct {
	OperationID   string                      `json:"operation_id"`
	PayloadDigest string                      `json:"payload_digest"`
	Body          string                      `json:"body"`
	Sources       []laputaevolution.SourceRef `json:"sources"`
	At            string                      `json:"at"`
}

func openEffectLedger(dir string) (*effectLedger, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &effectLedger{path: filepath.Join(dir, "effects.jsonl")}, nil
}

func (l *effectLedger) record(rec effectRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return appendJSONL(l.path, rec)
}

func (l *effectLedger) recordNote(operationID, digest string, payload *laputaevolution.ReflectionNotePayload) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec := noteRecord{
		OperationID: operationID, PayloadDigest: digest,
		Body: payload.Body, Sources: payload.Sources,
		At: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := appendJSONL(filepath.Join(filepath.Dir(l.path), "notes.jsonl"), rec); err != nil {
		return "", err
	}
	return operationID, nil
}

func (l *effectLedger) lookup(operationID string) (effectRecord, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	data, err := os.ReadFile(l.path)
	if errors.Is(err, os.ErrNotExist) {
		return effectRecord{}, false, nil
	}
	if err != nil {
		return effectRecord{}, false, err
	}
	var found effectRecord
	ok := false
	for _, line := range splitLines(data) {
		var rec effectRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			return effectRecord{}, false, fmt.Errorf("evolution: corrupt effect ledger: %w", err)
		}
		if rec.OperationID == operationID {
			found, ok = rec, true
		}
	}
	return found, ok, nil
}

func appendJSONL(path string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if len(data) > start {
		lines = append(lines, data[start:])
	}
	return lines
}

// ResultItem is one bounded ledger projection: an applied/rejected effect
// receipt or a persisted reflection note. Notes carry body; receipts carry
// status/target. Reflection notes never re-enter evidence through this read.
type ResultItem struct {
	OperationID   string                      `json:"operation_id"`
	PayloadDigest string                      `json:"payload_digest,omitempty"`
	Kind          string                      `json:"kind"`
	Status        string                      `json:"status,omitempty"`
	TargetRef     string                      `json:"target_ref,omitempty"`
	Revision      uint64                      `json:"revision,omitempty"`
	ErrorCode     string                      `json:"error_code,omitempty"`
	Body          string                      `json:"body,omitempty"`
	Sources       []laputaevolution.SourceRef `json:"sources,omitempty"`
	At            string                      `json:"at,omitempty"`
}

// ResultPage is one page of ledger results; there is deliberately no total —
// the cursor already binds the observed ledger version.
type ResultPage struct {
	Items      []ResultItem `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

type resultsCursor struct {
	Offset int `json:"o"`
	Total  int `json:"n"`
}

// Results pages the domain's own durable ledgers (effect receipts, then
// reflection notes). The cursor is opaque and ledger-version-bound: a read
// after new records were appended with an older cursor is rejected as stale,
// never silently shifted.
func (d *Domain) Results(cursor string, limit int) (ResultPage, error) {
	var cur resultsCursor
	if cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil || json.Unmarshal(raw, &cur) != nil || cur.Offset < 0 {
			return ResultPage{}, errors.New("evolution: malformed results cursor")
		}
	}
	if limit <= 0 {
		limit = 20
	}
	items, err := d.ledger.readAll()
	if err != nil {
		return ResultPage{}, err
	}
	if cursor != "" && cur.Total != len(items) {
		return ResultPage{}, errors.New("evolution: results cursor is stale")
	}
	if cur.Offset > len(items) {
		return ResultPage{}, errors.New("evolution: malformed results cursor")
	}
	end := cur.Offset + limit
	if end > len(items) {
		end = len(items)
	}
	page := ResultPage{Items: append([]ResultItem{}, items[cur.Offset:end]...)}
	if end < len(items) {
		raw, err := json.Marshal(resultsCursor{Offset: end, Total: len(items)})
		if err != nil {
			return ResultPage{}, err
		}
		page.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	return page, nil
}

// readAll projects both ledgers in durable order: effect receipts first, then
// reflection notes. Malformed lines are ledger corruption, not an empty page.
func (l *effectLedger) readAll() ([]ResultItem, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var items []ResultItem
	if data, err := os.ReadFile(l.path); err == nil {
		for _, line := range splitLines(data) {
			var rec effectRecord
			if err := json.Unmarshal(line, &rec); err != nil {
				return nil, fmt.Errorf("evolution: corrupt effect ledger: %w", err)
			}
			items = append(items, ResultItem{
				OperationID: rec.OperationID, PayloadDigest: rec.PayloadDigest,
				Kind: rec.Kind, Status: string(rec.Status), TargetRef: rec.TargetRef,
				Revision: rec.Revision, ErrorCode: rec.ErrorCode,
			})
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	notesPath := filepath.Join(filepath.Dir(l.path), "notes.jsonl")
	if data, err := os.ReadFile(notesPath); err == nil {
		for _, line := range splitLines(data) {
			var rec noteRecord
			if err := json.Unmarshal(line, &rec); err != nil {
				return nil, fmt.Errorf("evolution: corrupt notes ledger: %w", err)
			}
			items = append(items, ResultItem{
				OperationID: rec.OperationID, PayloadDigest: rec.PayloadDigest,
				Kind: string(laputaevolution.KindReflectionNote),
				Body: rec.Body, Sources: rec.Sources, At: rec.At,
			})
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return items, nil
}
