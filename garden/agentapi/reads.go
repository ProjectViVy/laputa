package agentapi

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/dashimaki/garden/memory"
	"github.com/dashimaki/laputa/actmem"
	"github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/laputa/persona"
	"github.com/dashimaki/mentle/facade"
)

// PersonaDocument is an explicit full authority read; it is never automatic context.
type PersonaDocument struct {
	Kind         string     `json:"kind"`
	FileName     string     `json:"file_name"`
	Exists       bool       `json:"exists"`
	Valid        bool       `json:"valid"`
	Content      string     `json:"content"`
	Revision     uint64     `json:"revision"`
	ContentHash  string     `json:"content_hash"`
	UpdatedAt    *time.Time `json:"updated_at"`
	PendingCount int        `json:"pending_count"`
}

type ActmemDocument struct {
	Revision     uint64    `json:"revision"`
	UpdatedAt    time.Time `json:"updated_at"`
	Pulse        string    `json:"pulse"`
	Recap        string    `json:"recap"`
	Work         string    `json:"work"`
	Markdown     string    `json:"markdown"`
	Unclassified bool      `json:"unclassified"`
}
type ActmemQuery struct {
	Query    string   `json:"query"`
	Sections []string `json:"sections,omitempty"`
	MaxHits  int      `json:"max_hits,omitempty"`
	MaxChars int      `json:"max_chars,omitempty"`
}
type ActmemHit struct {
	Section     string  `json:"section"`
	WorkSection *string `json:"work_section"`
	LineIndex   int     `json:"line_index"`
	Excerpt     string  `json:"excerpt"`
}
type ActmemResult struct {
	Revision      uint64      `json:"revision"`
	Items         []ActmemHit `json:"items"`
	ReturnedChars int         `json:"returned_chars"`
	Truncated     bool        `json:"truncated"`
}
type CardSearch struct {
	Query      string `json:"query"`
	Collection string `json:"collection,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Cursor     string `json:"cursor,omitempty"`
}
type CardPage struct {
	Cards      []MemoryCard `json:"cards"`
	NextCursor *string      `json:"next_cursor,omitempty"`
}

// EvidenceRef binds one expansion to a card id and the revision the
// caller last observed; expansion under a stale revision is refused.
type EvidenceRef struct {
	CardID           string `json:"card_id"`
	ExpectedRevision uint64 `json:"expected_revision"`
}

type EvidenceRead struct {
	Items         []EvidenceRef `json:"items"`
	PerItemBudget int           `json:"per_item_budget,omitempty"`
	TotalBudget   int           `json:"total_budget,omitempty"`
}

// ReadPersona retrieves exactly one named authority document, including WORLD
// only when requested explicitly; no file paths or history are accepted.
func (s *Service) ReadPersona(_ context.Context, principal Principal, binding Binding, kind string) (PersonaDocument, error) {
	if err := s.check(binding, principal, OpPersonaGet); err != nil {
		return PersonaDocument{}, err
	}
	parsed, err := persona.ParseKind(kind)
	if err != nil || strings.TrimSpace(kind) == "" {
		return PersonaDocument{}, failure("invalid_request", "unknown persona kind")
	}
	if s.runtime.Persona == nil {
		return PersonaDocument{}, failure("unavailable", "persona unavailable")
	}
	doc, err := s.runtime.Persona.GetDocument(parsed)
	if err != nil {
		return PersonaDocument{}, failure(persona.CodeOf(err), "persona read failed")
	}
	return PersonaDocument{Kind: doc.Kind.String(), FileName: doc.FileName, Exists: doc.Exists, Valid: doc.Valid, Content: doc.Content, Revision: doc.Revision, ContentHash: doc.ContentHash, UpdatedAt: doc.UpdatedAt, PendingCount: doc.PendingCount}, nil
}

// ReadACTMEM is a tool-only read, separate from bootstrap and recall.
func (s *Service) ReadACTMEM(_ context.Context, principal Principal, binding Binding) (ActmemDocument, error) {
	if err := s.check(binding, principal, OpActmemRead); err != nil {
		return ActmemDocument{}, err
	}
	if s.runtime.Actmem == nil {
		return ActmemDocument{}, failure("unavailable", "ACTMEM unavailable")
	}
	doc, err := s.runtime.Actmem.Read()
	if err != nil {
		return ActmemDocument{}, failure(actmem.CodeOf(err), "ACTMEM read failed")
	}
	return ActmemDocument{Revision: doc.Revision, UpdatedAt: doc.UpdatedAt, Pulse: doc.Pulse, Recap: doc.Recap, Work: doc.Work, Markdown: doc.Markdown, Unclassified: doc.Unclassified}, nil
}

func (s *Service) QueryACTMEM(_ context.Context, principal Principal, binding Binding, q ActmemQuery) (ActmemResult, error) {
	if err := s.check(binding, principal, OpActmemQuery); err != nil {
		return ActmemResult{}, err
	}
	if strings.TrimSpace(q.Query) == "" {
		return ActmemResult{}, failure("invalid_request", "query is required")
	}
	if s.runtime.Actmem == nil {
		return ActmemResult{}, failure("unavailable", "ACTMEM unavailable")
	}
	scope, err := binding.TrustedScope()
	if err != nil {
		return ActmemResult{}, failure("invalid_request", err.Error())
	}
	result, err := s.runtime.Actmem.Query(scope, actmem.QueryOptions{Query: q.Query, Sections: q.Sections, MaxHits: q.MaxHits, MaxChars: q.MaxChars})
	if err != nil {
		return ActmemResult{}, failure(actmem.CodeOf(err), "ACTMEM query failed")
	}
	items := make([]ActmemHit, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, ActmemHit(item))
	}
	return ActmemResult{Revision: result.Revision, Items: items, ReturnedChars: result.ReturnedChars, Truncated: result.Truncated}, nil
}

// SearchCards returns discovery metadata, never full canonical memory content.
func (s *Service) SearchCards(ctx context.Context, principal Principal, binding Binding, q CardSearch) (CardPage, error) {
	if err := s.check(binding, principal, OpSearch); err != nil {
		return CardPage{}, err
	}
	if q.Query == "" || q.Limit < 0 || q.Limit > 100 {
		return CardPage{}, failure("invalid_request", "invalid card search query or limit")
	}
	backend, scopes, err := s.memoryBackend(binding)
	if err != nil {
		return CardPage{}, err
	}
	page, err := backend.Search(ctx, memory.AuthorizedSearch{Scopes: scopes, Query: q.Query, Collection: q.Collection, Limit: q.Limit, Cursor: q.Cursor})
	if err != nil {
		return CardPage{}, materialError(err)
	}
	out := CardPage{Cards: page.Items}
	if page.NextCursor != "" {
		out.NextCursor = &page.NextCursor
	}
	return out, nil
}

// ReadEvidence expands selected cards under explicit per-item and total budgets.
func (s *Service) ReadEvidence(ctx context.Context, principal Principal, binding Binding, q EvidenceRead) ([]EvidenceFragment, error) {
	if err := s.check(binding, principal, OpExpand); err != nil {
		return nil, err
	}
	if len(q.Items) == 0 || q.PerItemBudget < 0 || q.PerItemBudget > 4000 || q.TotalBudget < 0 || q.TotalBudget > 16000 {
		return nil, failure("invalid_request", "invalid evidence references or budgets")
	}
	for _, item := range q.Items {
		if item.CardID == "" || item.ExpectedRevision == 0 {
			return nil, failure("invalid_request", "card id and expected_revision are required")
		}
	}
	backend, scopes, err := s.memoryBackend(binding)
	if err != nil {
		return nil, err
	}
	result := make([]EvidenceFragment, 0, len(q.Items))
	for _, item := range q.Items {
		page, err := backend.Expand(ctx, memory.AuthorizedExpansion{Scopes: scopes, CardID: item.CardID, ExpectedRevision: item.ExpectedRevision, BudgetChars: q.PerItemBudget})
		if err != nil {
			return nil, materialError(err)
		}
		result = append(result, page.Items...)
	}
	return result, nil
}

// memoryBackend resolves the caller's bound scope into its admitted read
// union (subject personal plus the trusted scope) and the backend bound to
// the trusted scope. Scope is always derived from the host binding, never
// from request fields.
func (s *Service) memoryBackend(binding Binding) (memory.Backend, []evolution.Scope, error) {
	trusted, err := binding.TrustedScope()
	if err != nil {
		return nil, nil, err
	}
	backend, err := s.runtime.BackendFor(trusted)
	if err != nil {
		return nil, nil, materialError(err)
	}
	personal := evolution.Scope{SubjectID: trusted.SubjectID, Kind: evolution.ScopePersonal}
	if trusted.Equal(personal) {
		return backend, []evolution.Scope{personal}, nil
	}
	return backend, []evolution.Scope{personal, trusted}, nil
}

// CollectionInfo is the public, storage-independent collection summary.
type CollectionInfo struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ListCollections reports live Mentle collections under the same read policy as cards.
func (s *Service) ListCollections(ctx context.Context, principal Principal, binding Binding) ([]CollectionInfo, error) {
	if err := s.check(binding, principal, OpSearch); err != nil {
		return nil, err
	}
	if s.runtime.Mentle == nil {
		return nil, failure("unavailable", "Mentle unavailable")
	}
	items, err := s.runtime.Mentle.ListCollections(ctx)
	if err != nil {
		return nil, materialError(err)
	}
	result := make([]CollectionInfo, 0, len(items))
	for _, item := range items {
		result = append(result, CollectionInfo{Name: item.Name, Count: item.Count})
	}
	return result, nil
}

func materialError(err error) error {
	var normalized *Error
	if errors.Is(err, facade.ErrUnavailable) {
		normalized = failure("unavailable", "Mentle unavailable")
	} else {
		switch evolution.CodeOf(err) {
		case evolution.ErrEffectNotFound:
			normalized = failure("not_found", "record or receipt not found")
		case evolution.ErrRevisionConflict, evolution.ErrIdempotencyConflict:
			normalized = failure("conflict", "record revision or payload conflict")
		case evolution.ErrAuthorityDenied:
			normalized = failure("principal_forbidden", "record outside the admitted scope union")
		case evolution.ErrInvalidScope, evolution.ErrInvalidSchema:
			normalized = failure("invalid_request", "invalid read request")
		case evolution.ErrBackendUnavailable, evolution.ErrCapabilityUnavailable:
			normalized = failure("unavailable", "memory backend unavailable")
		}
	}
	if normalized == nil {
		normalized = failure("unavailable", "material read failed")
	}
	normalized.cause = err
	return normalized
}
