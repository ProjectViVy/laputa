// Package mentle adapts the Mentle canonical facade to the Garden memory
// backend contract (contracts.md section 6). One adapter is bound to one
// subject scope and one write destination; reads are admitted to the
// configured read union, writes must match the bound writer exactly.
package mentle

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"

	"github.com/dashimaki/garden/memory"
	"github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/mentle/facade"
)

// Adapter is a scope-bound Backend over the Mentle canonical facade. The
// facade remains the sole memory authority; this type only encodes scopes,
// filters disclosures and maps receipts.
type Adapter struct {
	svc         *facade.Service
	scope       evolution.Scope // bound write scope
	destination string
	admitted    []evolution.Scope // read union: personal ∪ bound scope
}

// New binds the adapter. admitted is the read union the caller is allowed
// to see (subject personal scope plus the bound workspace scope); every
// entry must share the bound subject.
func New(svc *facade.Service, bound evolution.Scope, destination string, admitted []evolution.Scope) (*Adapter, error) {
	if svc == nil {
		return nil, errors.New("mentle adapter requires a facade service")
	}
	if err := bound.Validate(); err != nil {
		return nil, err
	}
	if destination == "" {
		return nil, errors.New("mentle adapter requires a destination id")
	}
	for _, scope := range admitted {
		if err := scope.Validate(); err != nil {
			return nil, err
		}
		if scope.SubjectID != bound.SubjectID {
			return nil, errors.New("admitted read scope has a foreign subject")
		}
	}
	return &Adapter{svc: svc, scope: bound, destination: destination, admitted: admitted}, nil
}

// BoundScope is the writer's bound scope.
func (a *Adapter) BoundScope() evolution.Scope { return a.scope }

// BoundDestination is the writer's bound destination id.
func (a *Adapter) BoundDestination() string { return a.destination }

// CreateMemory satisfies ingest.MemoryWriter with a create on the bound
// scope; the idempotency key maps to operation_id, the body hash to the
// payload digest.
func (a *Adapter) CreateMemory(ctx context.Context, req facade.CreateMemoryRequest, operationID, payloadDigest string) (facade.Memory, error) {
	receipt, err := a.mutateCreate(ctx, operationID, payloadDigest, req)
	if err != nil {
		return facade.Memory{}, err
	}
	return a.svc.GetMemory(ctx, receipt.RecordID)
}

// mutateCreate mints the canonical record id (mem_…); the caller supplies
// only the payload, digest and idempotency key.
func (a *Adapter) mutateCreate(ctx context.Context, operationID, payloadDigest string, req facade.CreateMemoryRequest) (facade.MutationReceipt, error) {
	sources := []facade.MemorySource{req.Source}
	return a.svc.Mutate(ctx, facade.MutationRequest{
		Scope:          memory.EncodeScope(a.scope),
		DestinationID:  a.destination,
		OperationID:    operationID,
		PayloadDigest:  payloadDigest,
		Operation:      "create",
		ExpectedAbsent: true,
		Body:           req.Content,
		Kind:           req.Kind,
		Sources:        sources,
		Metadata:       req.Metadata,
		Actor:          req.Actor,
		RequestID:      req.RequestID,
	})
}

func (a *Adapter) Capabilities() memory.Capabilities {
	return memory.Capabilities{Search: true, Expand: true, Mutate: !a.svc.IsReadOnly(), MutationLookup: true, Vector: !a.svc.IsReadOnly()}
}

// Search returns cards whose decoded scope is in the admitted read union.
// The facade never receives a caller-supplied scope string; records with
// unclassified scopes are filtered out here.
func (a *Adapter) Search(ctx context.Context, req memory.AuthorizedSearch) (memory.CardPage, error) {
	if err := req.Validate(); err != nil {
		return memory.CardPage{}, err
	}
	for _, scope := range req.Scopes {
		if !a.admit(scope) {
			return memory.CardPage{}, &evolution.ContractError{Code: evolution.ErrAuthorityDenied, Message: "search scope outside the admitted read union"}
		}
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	inner, err := a.openCursor(req.Cursor, req)
	if err != nil {
		return memory.CardPage{}, err
	}
	// Over-fetch: scope filtering happens after retrieval, so the page may
	// short-fill; callers paginate via the bound cursor.
	page, err := a.svc.SearchCards(ctx, facade.CardQuery{Text: req.Query, Collection: req.Collection, Limit: limit * 4, Cursor: inner})
	if err != nil {
		return memory.CardPage{}, err
	}
	visible := make([]facade.MemoryCard, 0, limit)
	for _, card := range page.Cards {
		if memory.ScopeVisible(card.Scope, req.Scopes) {
			visible = append(visible, card)
		}
		if len(visible) >= limit {
			break
		}
	}
	out := memory.CardPage{Items: make([]memory.MemoryCard, 0, len(visible))}
	for _, card := range visible {
		out.Items = append(out.Items, mapCard(card))
	}
	if page.NextCursor != nil {
		out.NextCursor = a.sealCursor(*page.NextCursor, req)
	}
	return out, nil
}

func scopeInUnion(scopes []evolution.Scope, s evolution.Scope) bool {
	for _, c := range scopes {
		if c.Equal(s) {
			return true
		}
	}
	return false
}

func (a *Adapter) admit(scope evolution.Scope) bool {
	for _, s := range a.admitted {
		if scope.Equal(s) {
			return true
		}
	}
	return false
}

// Expand validates the record's scope, status and expected revision against
// the same row used to render evidence — never check-then-refetch.
func (a *Adapter) Expand(ctx context.Context, req memory.AuthorizedExpansion) (memory.EvidencePage, error) {
	if err := req.Validate(); err != nil {
		return memory.EvidencePage{}, err
	}
	for _, scope := range req.Scopes {
		if !a.admit(scope) {
			return memory.EvidencePage{}, &evolution.ContractError{Code: evolution.ErrAuthorityDenied, Message: "expansion scope outside the admitted read union"}
		}
	}
	record, err := a.svc.GetMemory(ctx, req.CardID)
	if err != nil {
		return memory.EvidencePage{}, mapFacadeError(err)
	}
	decoded, err := memory.DecodeScope(record.Scope)
	if err != nil || !scopeInUnion(req.Scopes, decoded) {
		return memory.EvidencePage{}, &evolution.ContractError{Code: evolution.ErrAuthorityDenied, Message: "record outside the admitted scope union"}
	}
	if record.Status != "active" {
		return memory.EvidencePage{}, &evolution.ContractError{Code: evolution.ErrAuthorityDenied, Message: "record is not active"}
	}
	if uint64(record.Version) != req.ExpectedRevision {
		return memory.EvidencePage{}, &evolution.ContractError{Code: evolution.ErrRevisionConflict, Message: "expected_revision does not match the record"}
	}
	perItem := req.BudgetChars
	if perItem <= 0 {
		perItem = 800
	}
	fragments := []facade.EvidenceFragment{facade.RenderMemoryEvidence(record, perItem)}
	out := memory.EvidencePage{Items: make([]memory.EvidenceFragment, 0, len(fragments))}
	enc := memory.EncodeScope(decoded)
	for _, f := range fragments {
		out.Items = append(out.Items, memory.EvidenceFragment{
			CardID:       f.CardID,
			MaterialRef:  f.MaterialRef,
			SourceURI:    f.SourceURI,
			SourceRev:    f.SourceRev,
			Scope:        enc,
			Revision:     uint64(record.Version),
			Status:       record.Status,
			Excerpt:      f.Excerpt,
			StartOffset:  f.StartOffset,
			EndOffset:    f.EndOffset,
			ContentHash:  f.ContentHash,
			Validity:     f.Validity,
			EvidenceRefs: f.EvidenceRefs,
		})
	}
	return out, nil
}

// Mutate enforces the bound writer then delegates the atomic canonical
// write plus receipt to the facade.
func (a *Adapter) Mutate(ctx context.Context, req memory.AuthorizedMutation) (memory.MutationReceipt, error) {
	if err := req.Validate(); err != nil {
		return memory.MutationReceipt{}, err
	}
	if err := req.MatchesWriter(a.scope, a.destination); err != nil {
		return memory.MutationReceipt{}, err
	}
	sources := make([]facade.MemorySource, 0, len(req.Sources))
	for _, source := range req.Sources {
		if !a.admit(source.Scope) {
			return memory.MutationReceipt{}, &evolution.ContractError{Code: evolution.ErrAuthorityDenied, Message: "mutation source outside the admitted read union"}
		}
		locator := url.URL{Scheme: "garden-source", Host: "evidence", RawQuery: url.Values{
			"source_id": {source.SourceID}, "record_id": {source.RecordID},
			"scope": {memory.EncodeScope(source.Scope)},
		}.Encode()}
		sources = append(sources, facade.MemorySource{Type: source.SourceID, URI: locator.String(), Revision: strconv.FormatUint(source.Revision, 10)})
	}
	op := map[evolution.MutationOperation]string{
		evolution.MutationCreate:    "create",
		evolution.MutationUpdate:    "update",
		evolution.MutationTombstone: "tombstone",
	}[req.Operation]
	metadata := map[string]any{"inference": string(req.Inference)}
	if len(req.Sources) > 0 {
		metadata["evolution_sources"] = append([]evolution.SourceRef(nil), req.Sources...)
	}
	receipt, err := a.svc.Mutate(ctx, facade.MutationRequest{
		Scope:            memory.EncodeScope(req.Scope),
		DestinationID:    req.DestinationID,
		OperationID:      req.OperationID,
		PayloadDigest:    req.PayloadDigest,
		Operation:        op,
		RecordID:         req.RecordID,
		ExpectedRevision: int(req.ExpectedRevision),
		ExpectedAbsent:   req.ExpectedAbsent,
		Body:             req.Body,
		Sources:          sources,
		// The native Source is the primary locator; retain the complete
		// typed reference set in the same canonical row's metadata.
		Metadata:  metadata,
		Actor:     "garden",
		RequestID: req.OperationID,
	})
	if err != nil {
		return memory.MutationReceipt{}, mapFacadeError(err)
	}
	return mapReceipt(receipt), nil
}

// MutationStatus returns the bound scope's receipt only.
func (a *Adapter) MutationStatus(ctx context.Context, operationID string) (memory.MutationReceipt, error) {
	receipt, err := a.svc.MutationStatus(ctx, operationID, memory.EncodeScope(a.scope), a.destination)
	if err != nil {
		return memory.MutationReceipt{}, mapFacadeError(err)
	}
	return mapReceipt(receipt), nil
}

func (a *Adapter) Health(ctx context.Context) (memory.Health, error) {
	if a.svc == nil {
		return memory.Health{Status: memory.HealthUnavailable, ReasonCode: "backend_unavailable", DerivedIndexState: "unknown"}, nil
	}
	if a.svc.IsReadOnly() {
		return memory.Health{Status: memory.HealthDegraded, ReasonCode: "read_only", DerivedIndexState: "lexical_only"}, nil
	}
	return memory.Health{Status: memory.HealthAvailable, ReasonCode: "", DerivedIndexState: "ok"}, nil
}

// Close is a no-op: the facade lifecycle belongs to runtime composition.
func (a *Adapter) Close() error { return nil }

func mapCard(card facade.MemoryCard) memory.MemoryCard {
	return memory.MemoryCard{
		ID: card.ID, Kind: card.Kind, Collection: card.Collection, Scope: card.Scope,
		Title: card.Title, Summary: card.Summary, SourceRef: card.SourceRef,
		Revision: card.Revision, Status: card.Status, ValidFrom: card.ValidFrom,
		ValidTo: card.ValidTo, SupersededBy: card.SupersededBy, Tags: card.Tags,
		HeatScore: card.HeatScore, LastActivated: card.LastActivated, CandidateScore: card.CandidateScore,
	}
}

func mapReceipt(r facade.MutationReceipt) memory.MutationReceipt {
	status := evolution.StatusApplied
	if r.Status != "applied" {
		status = evolution.StatusRejected
	}
	canonical := memory.CanonicalCompleted
	if r.CanonicalStatus != "completed" {
		canonical = memory.CanonicalFailed
	}
	index := memory.IndexPending
	switch r.IndexStatus {
	case "ready":
		index = memory.IndexReady
	case "failed":
		index = memory.IndexFailed
	case "not_required":
		index = memory.IndexNotRequired
	}
	return memory.MutationReceipt{
		EffectReceipt: evolution.EffectReceipt{
			OperationID:   r.OperationID,
			PayloadDigest: r.PayloadDigest,
			Status:        status,
			TargetRef:     r.RecordID,
			Revision:      uint64(r.Revision),
			ErrorCode:     r.ErrorCode,
		},
		CanonicalStatus: canonical,
		IndexStatus:     index,
	}
}

func mapFacadeError(err error) error {
	switch {
	case errors.Is(err, facade.ErrIdempotencyConflict):
		return &evolution.ContractError{Code: evolution.ErrIdempotencyConflict, Message: "operation replayed with a different payload"}
	case errors.Is(err, facade.ErrVersionConflict):
		return &evolution.ContractError{Code: evolution.ErrRevisionConflict, Message: "record revision conflict"}
	case errors.Is(err, facade.ErrMutationNotFound), errors.Is(err, facade.ErrMemoryNotFound):
		return &evolution.ContractError{Code: evolution.ErrEffectNotFound, Message: "record or receipt not found"}
	case errors.Is(err, facade.ErrReadOnly):
		return &evolution.ContractError{Code: evolution.ErrCapabilityUnavailable, Message: "backend is read-only"}
	case errors.Is(err, facade.ErrUnavailable):
		return &evolution.ContractError{Code: evolution.ErrBackendUnavailable, Message: "backend unavailable"}
	default:
		return err
	}
}

type cursorBody struct {
	Seal  string `json:"seal"`
	Inner string `json:"inner"`
}

func (a *Adapter) cursorSeal(req memory.AuthorizedSearch) string {
	h := sha256.New()
	for _, scope := range req.Scopes {
		h.Write([]byte(memory.EncodeScope(scope)))
		h.Write([]byte{0})
	}
	h.Write([]byte(req.Query))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil)[:16])
}

func (a *Adapter) sealCursor(inner string, req memory.AuthorizedSearch) string {
	raw, _ := json.Marshal(cursorBody{Seal: a.cursorSeal(req), Inner: inner})
	return base64.RawURLEncoding.EncodeToString(raw)
}

func (a *Adapter) openCursor(cursor string, req memory.AuthorizedSearch) (string, error) {
	if cursor == "" {
		return "", nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return "", &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "cursor is not a bound cursor"}
	}
	var body cursorBody
	if err := json.Unmarshal(raw, &body); err != nil || body.Seal != a.cursorSeal(req) {
		return "", &evolution.ContractError{Code: evolution.ErrInvalidScope, Message: "cursor is bound to a different scope or query"}
	}
	return body.Inner, nil
}
