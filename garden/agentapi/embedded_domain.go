package agentapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	gardenevol "github.com/ProjectViVy/laputa/garden/evolution"
	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/laputa/persona"
	"github.com/ProjectViVy/laputa/mentle/facade"
)

// Embedded-domain capabilities derive from the one Garden owner opened by
// Client. A handle stamps a private per-handle principal and admitted scope;
// request payloads can never supply identity, scope, destination, or
// operation identity.

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
	maxInputBytes    = 64 << 10 // 64 KiB per single input field
)

// BackendMentle selects the existing Mentle canonical adapter. No other id
// mints a writer: a selected-but-unavailable backend reports unavailable
// rather than falling back.
const BackendMentle = "mentle"

// PageQuery bounds one page of results; Limit defaults to 20, max 100.
// Cursor is an opaque scope/query/ledger-version checkpoint minted by a
// previous page; a foreign or stale cursor is rejected.
type PageQuery struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

func (q PageQuery) limit() int {
	if q.Limit <= 0 {
		return defaultPageLimit
	}
	if q.Limit > maxPageLimit {
		return maxPageLimit
	}
	return q.Limit
}

// ReviewQuery filters Persona reviews; Kind/State are optional closed-set
// filters carried inside the cursor so a later page cannot change shape.
type ReviewQuery struct {
	Kind  *persona.Kind         `json:"kind,omitempty"`
	State *persona.RequestState `json:"state,omitempty"`
	PageQuery
}

// ReviewPage is one bounded page of change requests; there is no total —
// hidden-scope counts never reach the caller.
type ReviewPage struct {
	Items      []persona.ChangeRequest `json:"items"`
	NextCursor string                  `json:"next_cursor,omitempty"`
}

// ReviewDecision is the closed human review vocabulary.
type ReviewDecision string

const (
	ReviewAccept ReviewDecision = "accept"
	ReviewReject ReviewDecision = "reject"
)

// ResultPage aliases the domain-owned bounded projection of the effect and
// reflection-note ledgers.
type ResultPage = gardenevol.ResultPage

// EvolutionPorts is the run-bound cognition port set constructed inside the
// same owner: a fixed scope/destination Domain, the ingest high watermark and
// the authority Mission revision.
type EvolutionPorts struct {
	Domain          evolution.Domain
	SourceID        string
	HighWatermark   func(context.Context) (uint64, error)
	MissionRevision func(context.Context) (uint64, error)
}

// handle is private per-capability state: the trusted binding, the reduced
// principal and the admitted read union. None of it can arrive over a request.
type handle struct {
	client    *Client
	binding   Binding
	principal Principal
	scope     evolution.Scope
	admitted  []evolution.Scope
}

func (h *handle) runtime() (*Service, func(), error) {
	if h == nil || h.client == nil {
		return nil, nil, failure("unavailable", "Garden runtime unavailable")
	}
	h.client.mu.RLock()
	if h.client.runtime == nil {
		h.client.mu.RUnlock()
		return nil, nil, failure("unavailable", "Garden runtime unavailable")
	}
	return NewService(h.client.runtime), h.client.mu.RUnlock, nil
}

// selectedBackend resolves the scope-bound writer: an injected backend bound
// to exactly this scope wins; otherwise the configured BackendID names the
// writer — "mentle" mints the existing adapter and anything else is
// unavailable rather than rerouted.
func (h *handle) selectedBackend() (memory.Backend, error) {
	if h == nil || h.client == nil {
		return nil, failure("unavailable", "Garden runtime unavailable")
	}
	h.client.mu.RLock()
	defer h.client.mu.RUnlock()
	g := h.client.runtime
	if g == nil {
		return nil, failure("unavailable", "Garden runtime unavailable")
	}
	if g.Backends != nil {
		if backend, ok := g.Backends[memory.EncodeScope(h.scope)]; ok {
			return backend, nil
		}
	}
	switch h.client.backendID {
	case "", BackendMentle:
		backend, err := g.BackendFor(h.scope)
		if err != nil {
			return nil, failure("unavailable", "selected memory backend unavailable: "+err.Error())
		}
		return backend, nil
	default:
		return nil, failure("unavailable",
			fmt.Sprintf("selected memory backend %q has no binding for this scope", h.client.backendID))
	}
}

func admittedUnion(scope evolution.Scope) []evolution.Scope {
	personal := evolution.Scope{SubjectID: scope.SubjectID, Kind: evolution.ScopePersonal}
	if scope.Equal(personal) {
		return []evolution.Scope{personal}
	}
	return []evolution.Scope{personal, scope}
}

// BindAgentSession derives a reduced capability from the owner: the handle
// stamps PrincipalAgent per call instead of reusing the owner's principal,
// and the admitted read union is fixed at bind time. Wire payloads cannot
// widen it.
func (c *Client) BindAgentSession(sessionID, workspaceID string) (*BoundClient, error) {
	if c == nil {
		return nil, failure("unavailable", "Garden runtime unavailable")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.runtime == nil {
		return nil, failure("unavailable", "Garden runtime unavailable")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, failure("invalid_binding", "session_id is required")
	}
	binding := c.identity
	binding.SessionID = sessionID
	binding.WorkspaceID = workspaceID
	return &BoundClient{client: c, binding: binding, principal: PrincipalAgent}, nil
}

// HumanClient is the trusted-owner capability: persona authority writes and
// review decisions, owner ACTMEM save, and the selected-backend memory
// surface. It exists only on a Client opened with PrincipalUser; an agent
// principal can never mint it.
type HumanClient struct {
	handle
	destination string
	domain      *gardenevol.Domain
}

// BindHumanSession mints the human capability for a trusted user-owned
// client only. The workspace argument is host-issued admission, never a
// request payload.
func (c *Client) BindHumanSession(sessionID, workspaceID string) (*HumanClient, error) {
	if c == nil {
		return nil, failure("unavailable", "Garden runtime unavailable")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.runtime == nil {
		return nil, failure("unavailable", "Garden runtime unavailable")
	}
	if c.principal != PrincipalUser {
		return nil, policyError("principal_forbidden", "human capability requires a trusted user owner")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, failure("invalid_binding", "session_id is required")
	}
	binding := c.identity
	binding.SessionID = sessionID
	binding.WorkspaceID = workspaceID
	scope, err := binding.TrustedScope()
	if err != nil {
		return nil, failure("invalid_binding", err.Error())
	}
	h := &HumanClient{
		handle:      handle{client: c, binding: binding, principal: PrincipalUser, scope: scope, admitted: admittedUnion(scope)},
		destination: c.destinationID,
	}
	if h.destination == "" && c.backendID == BackendMentle {
		h.destination = BackendMentle
	}
	return h, nil
}

// resultsDomain lazily wraps garden/evolution.NewDomain inside the owner so
// bounded results reads use the domain-owned ledger format.
func (h *HumanClient) resultsDomain() (*gardenevol.Domain, error) {
	if h.domain != nil {
		return h.domain, nil
	}
	domain, err := gardenevol.NewDomain(gardenevol.Deps{
		Scope: h.scope, DestinationID: h.destination, Dir: h.client.evolutionDir,
	})
	if err != nil {
		return nil, failure("unavailable", "evolution ledger unavailable: "+err.Error())
	}
	h.domain = domain
	return domain, nil
}

// guardedDomain fails every derived call once the owner is closed; it never
// closes shared authority itself.
type guardedDomain struct {
	client *Client
	domain evolution.Domain
}

func (g *guardedDomain) live() error {
	g.client.mu.RLock()
	defer g.client.mu.RUnlock()
	if g.client.runtime == nil {
		return failure("unavailable", "Garden runtime unavailable")
	}
	return nil
}

func (g *guardedDomain) Collect(ctx context.Context, w evolution.Window) (evolution.EvidenceBatch, error) {
	if err := g.live(); err != nil {
		return evolution.EvidenceBatch{}, err
	}
	return g.domain.Collect(ctx, w)
}

func (g *guardedDomain) Apply(ctx context.Context, e evolution.Effect) (evolution.EffectReceipt, error) {
	if err := g.live(); err != nil {
		return evolution.EffectReceipt{}, err
	}
	return g.domain.Apply(ctx, e)
}

func (g *guardedDomain) Lookup(ctx context.Context, operationID string) (evolution.EffectReceipt, error) {
	if err := g.live(); err != nil {
		return evolution.EffectReceipt{}, err
	}
	return g.domain.Lookup(ctx, operationID)
}

// BindEvolution wraps garden/evolution.NewDomain inside the existing owner:
// the domain's scope/destination must match configured ownership, reads and
// effects use the selected scope-bound backend, and no VIVY caller reaches a
// garden/internal package.
func (c *Client) BindEvolution(scope evolution.Scope, destinationID string) (EvolutionPorts, error) {
	if c == nil {
		return EvolutionPorts{}, failure("unavailable", "Garden runtime unavailable")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.runtime == nil {
		return EvolutionPorts{}, failure("unavailable", "Garden runtime unavailable")
	}
	if err := scope.Validate(); err != nil {
		return EvolutionPorts{}, failure("invalid_binding", err.Error())
	}
	if scope.SubjectID != c.runtime.ProfileID {
		return EvolutionPorts{}, failure("invalid_binding", "scope subject is not this profile")
	}
	if c.destinationID != "" && destinationID != c.destinationID {
		return EvolutionPorts{}, failure("invalid_binding", "destination does not match the configured writer")
	}
	h := &handle{client: c, scope: scope, admitted: admittedUnion(scope)}
	backend, err := h.selectedBackend()
	if err != nil {
		return EvolutionPorts{}, err
	}
	domain, err := gardenevol.NewDomain(gardenevol.Deps{
		Scope: scope, DestinationID: destinationID,
		Actmem: c.runtime.Actmem, Persona: c.runtime.Persona,
		Memory: backend, Activity: c.runtime.Ingest,
		Dir: c.evolutionDir,
	})
	if err != nil {
		return EvolutionPorts{}, failure("unavailable", "evolution domain unavailable: "+err.Error())
	}
	return EvolutionPorts{
		Domain:   &guardedDomain{client: c, domain: domain},
		SourceID: fmt.Sprintf("%s/%s/%s", c.runtime.ProfileID, memory.EncodeScope(scope), destinationID),
		HighWatermark: func(ctx context.Context) (uint64, error) {
			c.mu.RLock()
			defer c.mu.RUnlock()
			if c.runtime == nil || c.runtime.Ingest == nil {
				return 0, failure("unavailable", "Garden runtime unavailable")
			}
			return c.runtime.Ingest.HighWatermark(ctx, scope.WorkspaceID)
		},
		MissionRevision: func(ctx context.Context) (uint64, error) {
			if err := (&guardedDomain{client: c}).live(); err != nil {
				return 0, err
			}
			return domain.MissionRevision(ctx)
		},
	}, nil
}

// SourcePorts is the bound committed-activity input set for a fixed
// scope/destination: the source identity stamped on every window, the
// durable ingest high watermark, and the authority Mission revision. It
// resolves no memory backend, so arming capture/source cannot fail on an
// unavailable selected writer; the bound Domain keeps BindEvolution's eager
// writer check.
type SourcePorts struct {
	SourceID        string
	HighWatermark   func(context.Context) (uint64, error)
	MissionRevision func(context.Context) (uint64, error)
}

// BindEvolutionSource binds the input ports for one scope/destination
// without resolving the selected memory backend. A Mission revision read on
// an uninitialized authority reports 0 (unassigned), not an error.
func (c *Client) BindEvolutionSource(scope evolution.Scope, destinationID string) (SourcePorts, error) {
	if c == nil {
		return SourcePorts{}, failure("unavailable", "Garden runtime unavailable")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.runtime == nil {
		return SourcePorts{}, failure("unavailable", "Garden runtime unavailable")
	}
	if err := scope.Validate(); err != nil {
		return SourcePorts{}, failure("invalid_binding", err.Error())
	}
	if scope.SubjectID != c.runtime.ProfileID {
		return SourcePorts{}, failure("invalid_binding", "scope subject is not this profile")
	}
	if c.destinationID != "" && destinationID != c.destinationID {
		return SourcePorts{}, failure("invalid_binding", "destination does not match the configured writer")
	}
	return SourcePorts{
		SourceID: fmt.Sprintf("%s/%s/%s", c.runtime.ProfileID, memory.EncodeScope(scope), destinationID),
		HighWatermark: func(ctx context.Context) (uint64, error) {
			c.mu.RLock()
			defer c.mu.RUnlock()
			if c.runtime == nil || c.runtime.Ingest == nil {
				return 0, failure("unavailable", "Garden runtime unavailable")
			}
			return c.runtime.Ingest.HighWatermark(ctx, scope.WorkspaceID)
		},
		MissionRevision: func(ctx context.Context) (uint64, error) {
			c.mu.RLock()
			defer c.mu.RUnlock()
			if c.runtime == nil || c.runtime.Persona == nil {
				return 0, failure("unavailable", "Garden runtime unavailable")
			}
			doc, err := c.runtime.Persona.GetDocument(persona.KindMission)
			if errors.Is(err, persona.ErrUninitialized) {
				return 0, nil
			}
			if err != nil {
				return 0, err
			}
			return doc.Revision, nil
		},
	}, nil
}

// --- human reads -----------------------------------------------------------

// ReadPersona retrieves one named authority document through the human
// principal; the closed eight-kind roster is enforced by the authority.
func (h *HumanClient) ReadPersona(ctx context.Context, kind string) (PersonaDocument, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return PersonaDocument{}, err
	}
	defer unlock()
	return s.ReadPersona(ctx, h.principal, h.binding, kind)
}

// PersonaStatus returns the authority's own per-kind file states — the
// revisions the control surface reports verbatim.
func (h *HumanClient) PersonaStatus(ctx context.Context) (*persona.StatusView, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err := s.check(h.binding, h.principal, OpPersonaGet); err != nil {
		return nil, err
	}
	if s.runtime.Persona == nil {
		return nil, failure("unavailable", "persona unavailable")
	}
	return s.runtime.Persona.Status()
}

// ReadActivity returns the scoped ACTMEM projection (personal union);
// sections and max_chars stay inside the domain's own caps.
func (h *HumanClient) ReadActivity(ctx context.Context, req ReadRequest) (ActivityResult, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return ActivityResult{}, err
	}
	defer unlock()
	return s.ReadActivity(ctx, h.principal, h.binding, req)
}

// ApplyWorkPatch applies scoped Work edits against base_revision; hidden
// entries and stale bases are rejected by the store, never refetched.
func (h *HumanClient) ApplyWorkPatch(ctx context.Context, patch WorkPatch) (ActivityResult, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return ActivityResult{}, err
	}
	defer unlock()
	return s.ApplyWorkPatch(ctx, h.principal, h.binding, patch)
}

// ReadOwnerACTMEM returns the whole authority document — the human owner
// view, never the agent's scoped projection.
func (h *HumanClient) ReadOwnerACTMEM(ctx context.Context) (ActmemDocument, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return ActmemDocument{}, err
	}
	defer unlock()
	return s.ReadACTMEM(ctx, h.principal, h.binding)
}

// IndexHealth reports live canonical/derived-index health for the bound
// scope; a degraded index is a valid report, never a silent zero.
func (h *HumanClient) IndexHealth(ctx context.Context) (facade.IndexHealth, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return facade.IndexHealth{}, err
	}
	defer unlock()
	return s.IndexHealth(ctx, h.principal, h.binding)
}

// --- human writes ----------------------------------------------------------

// InitializePersona runs the owning authority's first-run initialization;
// actor/source are stamped from trusted composition, never parameters.
func (h *HumanClient) InitializePersona(ctx context.Context, in persona.Initialization, reason string) (*persona.WriteOutcome, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err := s.check(h.binding, h.principal, OpPersonaReview); err != nil {
		return nil, err
	}
	if s.runtime.Persona == nil {
		return nil, failure("unavailable", "persona unavailable")
	}
	return s.runtime.Persona.Initialize(in, "user", persona.SourceInit, reason)
}

// SavePersona is the owner direct write; stale base revisions are rejected by
// the authority's own CAS, never forced.
func (h *HumanClient) SavePersona(ctx context.Context, kind persona.Kind, content string, baseRevision uint64, reason string) (*persona.WriteOutcome, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err := s.check(h.binding, h.principal, OpPersonaReview); err != nil {
		return nil, err
	}
	if len(content) > maxInputBytes {
		return nil, failure("invalid_request", "persona content exceeds 64 KiB")
	}
	if s.runtime.Persona == nil {
		return nil, failure("unavailable", "persona unavailable")
	}
	out, err := s.runtime.Persona.SaveUserDocument(kind, content, baseRevision, "user", persona.SourceUserDirect, reason)
	if err != nil {
		return nil, failure(persona.CodeOf(err), "persona write failed")
	}
	return out, nil
}

// ListPersonaReviews pages pending/decided reviews. The page carries no
// hidden-scope totals; a cursor minted before the review set changed is
// rejected as stale.
func (h *HumanClient) ListPersonaReviews(ctx context.Context, q ReviewQuery) (ReviewPage, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return ReviewPage{}, err
	}
	defer unlock()
	if err := s.check(h.binding, h.principal, OpPersonaReview); err != nil {
		return ReviewPage{}, err
	}
	if s.runtime.Persona == nil {
		return ReviewPage{}, failure("unavailable", "persona unavailable")
	}
	items, err := s.runtime.Persona.ListReviews(q.Kind, q.State, 0)
	if err != nil {
		return ReviewPage{}, failure(persona.CodeOf(err), "review list failed")
	}
	cur, err := decodePageCursor(q.Cursor, "reviews")
	if err != nil {
		return ReviewPage{}, err
	}
	if q.Cursor != "" && cur.SetTotal != len(items) {
		return ReviewPage{}, failure("invalid_request", "review cursor is stale")
	}
	if cur.Kind != nil && (q.Kind == nil || *q.Kind != *cur.Kind) {
		return ReviewPage{}, failure("invalid_request", "cursor was minted for a different kind")
	}
	if cur.State != nil && (q.State == nil || *q.State != *cur.State) {
		return ReviewPage{}, failure("invalid_request", "cursor was minted for a different state")
	}
	limit := q.limit()
	end := cur.Offset + limit
	if end > len(items) {
		end = len(items)
	}
	page := ReviewPage{Items: append([]persona.ChangeRequest{}, items[cur.Offset:end]...)}
	if end < len(items) {
		page.NextCursor = encodePageCursor(pageCursor{Family: "reviews", Offset: end, SetTotal: len(items), Kind: q.Kind, State: q.State})
	}
	return page, nil
}

// DecidePersonaReview resolves one pending review through the owning
// authority's accept/reject only; stale bases are never forced.
func (h *HumanClient) DecidePersonaReview(ctx context.Context, id string, decision ReviewDecision) (*persona.WriteOutcome, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err := s.check(h.binding, h.principal, OpPersonaReview); err != nil {
		return nil, err
	}
	if s.runtime.Persona == nil {
		return nil, failure("unavailable", "persona unavailable")
	}
	switch decision {
	case ReviewAccept:
		return s.runtime.Persona.AcceptRequest(id)
	case ReviewReject:
		return s.runtime.Persona.RejectRequest(id)
	default:
		return nil, failure("invalid_request", "review decision must be accept or reject")
	}
}

// SaveACTMEM is the owner CAS save through the validated parser.
func (h *HumanClient) SaveACTMEM(ctx context.Context, markdown string, baseRevision uint64) (ActmemDocument, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return ActmemDocument{}, err
	}
	defer unlock()
	if err := s.check(h.binding, h.principal, OpActmemSave); err != nil {
		return ActmemDocument{}, err
	}
	if len(markdown) > maxInputBytes {
		return ActmemDocument{}, failure("invalid_request", "ACTMEM content exceeds 64 KiB")
	}
	if s.runtime.Actmem == nil {
		return ActmemDocument{}, failure("unavailable", "ACTMEM unavailable")
	}
	result, err := s.runtime.Actmem.Save(markdown, baseRevision)
	if err != nil {
		return ActmemDocument{}, failure("actmem_rejected", err.Error())
	}
	doc, err := s.runtime.Actmem.Read()
	if err != nil {
		return ActmemDocument{}, failure("unavailable", "ACTMEM read failed: "+err.Error())
	}
	_ = result
	return ActmemDocument{Revision: doc.Revision, UpdatedAt: doc.UpdatedAt, Pulse: doc.Pulse, Recap: doc.Recap, Work: doc.Work, Markdown: doc.Markdown, Unclassified: doc.Unclassified}, nil
}

// --- memory surface on the selected backend ---------------------------------

// SearchMemory admits only the host-stamped read union; wire-supplied scopes
// must equal it exactly or the request is rejected.
func (h *HumanClient) SearchMemory(ctx context.Context, q memory.AuthorizedSearch) (memory.CardPage, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return memory.CardPage{}, err
	}
	defer unlock()
	if err := s.check(h.binding, h.principal, OpSearch); err != nil {
		return memory.CardPage{}, err
	}
	if len(q.Query) > maxInputBytes {
		return memory.CardPage{}, failure("invalid_request", "query exceeds 64 KiB")
	}
	q.Scopes = h.admitted
	backend, err := h.selectedBackend()
	if err != nil {
		return memory.CardPage{}, err
	}
	page, err := backend.Search(ctx, q)
	if err != nil {
		return memory.CardPage{}, failure(memoryCodeOf(err), "memory search failed")
	}
	return page, nil
}

// ExpandMemory is the bounded evidence read; a stale expected_revision fails
// in the backend, never refetches different content.
func (h *HumanClient) ExpandMemory(ctx context.Context, q memory.AuthorizedExpansion) (memory.EvidencePage, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return memory.EvidencePage{}, err
	}
	defer unlock()
	if err := s.check(h.binding, h.principal, OpExpand); err != nil {
		return memory.EvidencePage{}, err
	}
	q.Scopes = h.admitted
	backend, err := h.selectedBackend()
	if err != nil {
		return memory.EvidencePage{}, err
	}
	page, err := backend.Expand(ctx, q)
	if err != nil {
		return memory.EvidencePage{}, failure(memoryCodeOf(err), "memory expansion failed")
	}
	return page, nil
}

// MutateMemory stamps operation identity and payload digest on the trusted
// side, validates scope/destination against the bound writer, then performs
// exactly one canonical mutation through the selected backend.
func (h *HumanClient) MutateMemory(ctx context.Context, m memory.AuthorizedMutation) (memory.MutationReceipt, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return memory.MutationReceipt{}, err
	}
	defer unlock()
	if err := s.check(h.binding, h.principal, OpRemember); err != nil {
		return memory.MutationReceipt{}, err
	}
	if !m.Scope.Equal(h.scope) {
		return memory.MutationReceipt{}, failure("invalid_request", "mutation scope does not match the admitted writer")
	}
	if h.destination != "" && m.DestinationID != h.destination {
		return memory.MutationReceipt{}, failure("invalid_request", "mutation destination does not match the selected writer")
	}
	if len(m.Body) > maxInputBytes {
		return memory.MutationReceipt{}, failure("invalid_request", "mutation body exceeds 64 KiB")
	}
	// The adapter issues operation identity; caller-supplied values are ignored.
	m.OperationID = "op_" + randomHex16()
	m.PayloadDigest = mutationDigest(m)
	if err := m.Validate(); err != nil {
		return memory.MutationReceipt{}, failure("invalid_request", err.Error())
	}
	backend, err := h.selectedBackend()
	if err != nil {
		return memory.MutationReceipt{}, err
	}
	receipt, err := backend.Mutate(ctx, m)
	if err != nil {
		return memory.MutationReceipt{}, failure(memoryCodeOf(err), "memory mutation failed")
	}
	return receipt, nil
}

// MemoryReceipt returns the backend's own receipt for a host-issued operation id.
func (h *HumanClient) MemoryReceipt(ctx context.Context, operationID string) (memory.MutationReceipt, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return memory.MutationReceipt{}, err
	}
	defer unlock()
	if err := s.check(h.binding, h.principal, OpRemember); err != nil {
		return memory.MutationReceipt{}, err
	}
	if strings.TrimSpace(operationID) == "" {
		return memory.MutationReceipt{}, failure("invalid_request", "operation_id is required")
	}
	backend, err := h.selectedBackend()
	if err != nil {
		return memory.MutationReceipt{}, err
	}
	receipt, err := backend.MutationStatus(ctx, operationID)
	if err != nil {
		return memory.MutationReceipt{}, failure(memoryCodeOf(err), "receipt lookup failed")
	}
	return receipt, nil
}

// ReadFrozen returns the session-bound FrozenCore v2 snapshot; a session
// without a captured snapshot is an explicit not-found, never a default.
func (h *HumanClient) ReadFrozen(ctx context.Context) (FrozenCore, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return FrozenCore{}, err
	}
	defer unlock()
	if err := s.check(h.binding, h.principal, OpBootstrap); err != nil {
		return FrozenCore{}, err
	}
	if s.runtime.Frozen == nil {
		return FrozenCore{}, failure("unavailable", "frozen store unavailable")
	}
	core, err := s.runtime.Frozen.Get(ctx, h.binding.SessionID)
	if err != nil {
		return FrozenCore{}, failure("not_found", "no captured frozen core for this session")
	}
	return core, nil
}

// Results pages the owned effect/reflection-note ledger through the domain's
// own reader — bounded, cursor-bound, no hidden totals.
func (h *HumanClient) Results(ctx context.Context, q PageQuery) (ResultPage, error) {
	s, unlock, err := h.runtime()
	if err != nil {
		return ResultPage{}, err
	}
	defer unlock()
	if err := s.check(h.binding, h.principal, OpActmemRead); err != nil {
		return ResultPage{}, err
	}
	domain, err := h.resultsDomain()
	if err != nil {
		return ResultPage{}, err
	}
	page, err := domain.Results(q.Cursor, q.limit())
	if err != nil {
		return ResultPage{}, failure("invalid_request", err.Error())
	}
	return page, nil
}

// --- pagination cursors ------------------------------------------------------

type pageCursor struct {
	Family   string                `json:"f"`
	Offset   int                   `json:"o"`
	SetTotal int                   `json:"n"`
	Kind     *persona.Kind         `json:"k,omitempty"`
	State    *persona.RequestState `json:"s,omitempty"`
}

func encodePageCursor(cur pageCursor) string {
	raw, err := json.Marshal(cur)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodePageCursor(raw, family string) (pageCursor, error) {
	if raw == "" {
		return pageCursor{Family: family}, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return pageCursor{}, failure("invalid_request", "malformed page cursor")
	}
	var cur pageCursor
	if err := json.Unmarshal(data, &cur); err != nil {
		return pageCursor{}, failure("invalid_request", "malformed page cursor")
	}
	if cur.Family != family || cur.Offset < 0 {
		return pageCursor{}, failure("invalid_request", "cursor was minted for a different query")
	}
	return cur, nil
}

// --- stamping helpers --------------------------------------------------------

func randomHex16() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(buf[:])
}

// mutationDigest binds the stamped operation to the canonical payload fields
// so a replay with different content is an idempotency conflict.
func mutationDigest(m memory.AuthorizedMutation) string {
	raw, err := json.Marshal(map[string]any{
		"operation":         m.Operation,
		"record_id":         m.RecordID,
		"body":              m.Body,
		"sources":           m.Sources,
		"inference":         m.Inference,
		"expected_revision": m.ExpectedRevision,
		"expected_absent":   m.ExpectedAbsent,
	})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func memoryCodeOf(err error) string {
	if code := evolution.CodeOf(err); code != "" {
		return string(code)
	}
	return "unavailable"
}
