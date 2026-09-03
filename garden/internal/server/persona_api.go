package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dashimaki/laputa/persona"
)

// personaStatusPayload is deliberately metadata-only. A document body is
// returned only by the explicit /documents/{kind} route.
func personaStatusPayload(view *persona.StatusView) map[string]any {
	documents := make([]persona.FileState, 0, len(persona.AllKinds))
	for _, kind := range persona.AllKinds {
		documents = append(documents, view.Files[kind.String()])
	}
	return map[string]any{"status": view.Status, "documents": documents}
}

func (s *Server) personaReadPrincipal(w http.ResponseWriter, r *http.Request) (Principal, bool) {
	return s.requireReadPrincipal(w, r, PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalAutodream, PrincipalOperator)
}

func personaContentTypeOK(r *http.Request) bool {
	value := strings.TrimSpace(strings.ToLower(r.Header.Get("Content-Type")))
	return strings.HasPrefix(value, "application/json")
}

// handlePersonaDocuments implements the metadata-only document collection.
func (s *Server) handlePersonaDocuments(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.personaReadPrincipal(w, r); !ok {
		return
	}
	if len(r.URL.Query()) > 0 {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("persona document metadata does not accept query parameters"))
		return
	}
	if s.Persona == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("persona service unavailable"))
		return
	}
	view, err := s.Persona.Status()
	if err != nil {
		writePersonaAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, personaStatusPayload(view))
}

func (s *Server) handlePersonaDocumentV2(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.personaReadPrincipal(w, r); !ok {
		return
	}
	if len(r.URL.Query()) > 0 {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("persona document read does not accept query parameters"))
		return
	}
	if s.Persona == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("persona service unavailable"))
		return
	}
	kind, err := persona.ParseKind(r.PathValue("kind"))
	if err != nil {
		writeErrorWithCode(w, http.StatusBadRequest, "persona_kind_forbidden", err)
		return
	}
	doc, err := s.Persona.GetDocument(kind)
	if err != nil {
		writePersonaAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

type personaHistoryItem struct {
	Revision     uint64 `json:"revision"`
	ContentHash  string `json:"content_hash"`
	Actor        string `json:"actor"`
	Source       string `json:"source"`
	Reason       string `json:"reason"`
	BaseRevision uint64 `json:"base_revision"`
	CreatedAt    any    `json:"created_at"`
}

func historyItem(entry persona.HistoryEntry) personaHistoryItem {
	return personaHistoryItem{
		Revision:     entry.Revision,
		ContentHash:  entry.ContentHash,
		Actor:        entry.Actor,
		Source:       entry.Source,
		Reason:       entry.Reason,
		BaseRevision: entry.BaseRevision,
		CreatedAt:    entry.CreatedAt,
	}
}

func (s *Server) handlePersonaHistoryV2(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.personaReadPrincipal(w, r); !ok {
		return
	}
	if s.Persona == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("persona service unavailable"))
		return
	}
	for key := range r.URL.Query() {
		if key != "limit" && key != "cursor" {
			writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("unknown Persona history query parameter: "+key))
			return
		}
	}
	kind, err := persona.ParseKind(r.PathValue("kind"))
	if err != nil {
		writeErrorWithCode(w, http.StatusBadRequest, "persona_kind_forbidden", err)
		return
	}
	limit, err := parsePersonaLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	entries, err := s.Persona.ListHistory(kind)
	if err != nil {
		writePersonaAPIError(w, err)
		return
	}
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		cursor, parseErr := strconv.ParseUint(raw, 10, 64)
		if parseErr != nil || cursor == 0 {
			writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("cursor must be a positive history revision"))
			return
		}
		filtered := entries[:0]
		for _, entry := range entries {
			if entry.Revision < cursor {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}
	items := make([]personaHistoryItem, 0, len(entries))
	for _, entry := range entries {
		items = append(items, historyItem(entry))
	}
	var nextCursor any
	if limit > 0 && len(items) > limit {
		nextCursor = strconv.FormatUint(items[limit-1].Revision, 10)
		items = items[:limit]
	}
	writeJSON(w, http.StatusOK, map[string]any{"kind": kind.String(), "items": items, "next_cursor": nextCursor})
}

func (s *Server) handlePersonaHistoryRevisionV2(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.personaReadPrincipal(w, r); !ok {
		return
	}
	if s.Persona == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("persona service unavailable"))
		return
	}
	if len(r.URL.Query()) > 0 {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("Persona history revision does not accept query parameters"))
		return
	}
	kind, err := persona.ParseKind(r.PathValue("kind"))
	if err != nil {
		writeErrorWithCode(w, http.StatusBadRequest, "persona_kind_forbidden", err)
		return
	}
	revision, err := strconv.ParseUint(r.PathValue("revision"), 10, 64)
	if err != nil || revision == 0 {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("revision must be greater than zero"))
		return
	}
	history, err := s.Persona.ReadHistory(kind, revision)
	if err != nil {
		writePersonaAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"kind": kind.String(), "entry": historyItem(history.HistoryEntry),
		"content": history.Content, "unified_diff": history.UnifiedDiff,
	})
}

type personaDocumentWriteV2 struct {
	BaseRevision *uint64 `json:"base_revision"`
	Content      string  `json:"content"`
	Scope        string  `json:"scope"`
	Reason       string  `json:"reason"`
}

func (s *Server) handlePersonaDocumentPutV2(w http.ResponseWriter, r *http.Request) {
	principal, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalAgent, PrincipalAutodream)
	if !ok {
		return
	}
	if s.Persona == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("persona service unavailable"))
		return
	}
	if !personaContentTypeOK(r) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
		return
	}
	kind, err := persona.ParseKind(r.PathValue("kind"))
	if err != nil {
		writeErrorWithCode(w, http.StatusBadRequest, "persona_kind_forbidden", err)
		return
	}
	var body personaDocumentWriteV2
	if err := decodeJSON(w, r, 4<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	if body.BaseRevision == nil {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("base_revision is required"))
		return
	}
	body.Scope = strings.TrimSpace(body.Scope)
	if body.Scope == "" {
		body.Scope = "document"
	}
	body.Reason = strings.TrimSpace(body.Reason)
	if body.Reason == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "persona_invalid_content", errors.New("reason is required"))
		return
	}
	var outcome *persona.WriteOutcome
	switch body.Scope {
	case "document":
		if principal == PrincipalUser {
			outcome, err = s.Persona.SaveUserDocument(kind, body.Content, *body.BaseRevision, auditLabel(r, principal), persona.SourceUserDirect, body.Reason)
		} else if kind == persona.KindDream || kind == persona.KindDark {
			outcome, err = s.Persona.SaveAgentP16WithRevision(kind, body.Content, *body.BaseRevision, body.Reason)
		} else {
			err = persona.ErrKindForbidden
		}
	case "observations":
		if kind != persona.KindUser {
			err = persona.ErrKindForbidden
		} else if principal == PrincipalUser {
			outcome, err = s.Persona.SaveUserObservations(body.Content, *body.BaseRevision, body.Reason)
		} else {
			outcome, err = s.Persona.SaveAgentP16WithRevision(kind, body.Content, *body.BaseRevision, body.Reason)
		}
	default:
		err = persona.ErrKindForbidden
	}
	if err != nil {
		writePersonaAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, outcome)
}

func (s *Server) handlePersonaInitializeV2(w http.ResponseWriter, r *http.Request) {
	principal, ok := s.requirePrincipal(w, r, PrincipalUser)
	if !ok {
		return
	}
	if s.Persona == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("persona service unavailable"))
		return
	}
	if !personaContentTypeOK(r) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
		return
	}
	var input persona.Initialization
	if err := decodeJSON(w, r, 4<<20, &input); err != nil {
		writeRequestError(w, err)
		return
	}
	if _, err := s.Persona.Initialize(input, auditLabel(r, principal), persona.SourceInit, "Persona initialization"); err != nil {
		writePersonaAPIError(w, err)
		return
	}
	view, err := s.Persona.Status()
	if err != nil {
		writePersonaAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, personaStatusPayload(view))
}

type personaRepairV2 struct {
	Documents map[string]string `json:"documents"`
	Reason    string            `json:"reason"`
}

func (s *Server) handlePersonaRepairV2(w http.ResponseWriter, r *http.Request) {
	principal, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalOperator)
	if !ok {
		return
	}
	if s.Persona == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("persona service unavailable"))
		return
	}
	if !personaContentTypeOK(r) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
		return
	}
	var body personaRepairV2
	if err := decodeJSON(w, r, 4<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	documents := make(map[persona.Kind]string, len(body.Documents))
	for rawKind, content := range body.Documents {
		kind, err := persona.ParseKind(rawKind)
		if err != nil {
			writePersonaAPIError(w, err)
			return
		}
		documents[kind] = content
	}
	if strings.TrimSpace(body.Reason) == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "persona_invalid_content", errors.New("reason is required"))
		return
	}
	_ = principal
	view, err := s.Persona.Repair(persona.RepairInput{Documents: documents, Reason: strings.TrimSpace(body.Reason)})
	if err != nil {
		writePersonaAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, personaStatusPayload(view))
}

type personaReviewCreateV2 struct {
	Kind             string  `json:"kind"`
	BaseRevision     *uint64 `json:"base_revision"`
	BaseHash         string  `json:"base_hash"`
	ProposedMarkdown string  `json:"proposed_markdown"`
	Reason           string  `json:"reason"`
}

func (s *Server) handlePersonaReviewCreateV2(w http.ResponseWriter, r *http.Request) {
	principal, ok := s.requirePrincipal(w, r, PrincipalAgent, PrincipalAutodream)
	if !ok {
		return
	}
	if s.Persona == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("persona service unavailable"))
		return
	}
	if !personaContentTypeOK(r) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
		return
	}
	var body personaReviewCreateV2
	if err := decodeJSON(w, r, 4<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	if body.BaseRevision == nil || strings.TrimSpace(body.BaseHash) == "" || strings.TrimSpace(body.Reason) == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "persona_invalid_content", errors.New("base_revision, base_hash and reason are required"))
		return
	}
	kind, err := persona.ParseKind(body.Kind)
	if err != nil {
		writePersonaAPIError(w, err)
		return
	}
	actor := persona.ActorAgent
	if principal == PrincipalAutodream {
		actor = persona.ActorAutodream
	}
	review, err := s.Persona.CreateRequest(persona.ChangeRequest{
		Kind: kind, BaseRevision: *body.BaseRevision, BaseHash: strings.TrimSpace(body.BaseHash),
		ProposedMarkdown: body.ProposedMarkdown, Actor: actor, Reason: strings.TrimSpace(body.Reason),
	})
	if err != nil {
		writePersonaAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, review)
}

func parsePersonaLimit(raw string) (int, error) {
	if raw == "" {
		return 50, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > 200 {
		return 0, errors.New("limit must be between 1 and 200")
	}
	return limit, nil
}

func (s *Server) handlePersonaReviewsV2(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.personaReadPrincipal(w, r); !ok {
		return
	}
	if s.Persona == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("persona service unavailable"))
		return
	}
	query := r.URL.Query()
	for key := range query {
		if key != "kind" && key != "state" && key != "limit" && key != "cursor" {
			writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("unknown Persona review query parameter: "+key))
			return
		}
	}
	limit, err := parsePersonaLimit(query.Get("limit"))
	if err != nil {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	var kind *persona.Kind
	if raw := query.Get("kind"); raw != "" {
		parsed, err := persona.ParseKind(raw)
		if err != nil {
			writePersonaAPIError(w, err)
			return
		}
		kind = &parsed
	}
	var state *persona.RequestState
	if raw := query.Get("state"); raw != "" {
		parsed := persona.RequestState(raw)
		switch parsed {
		case persona.RequestPending, persona.RequestAccepted, persona.RequestRejected, persona.RequestStale:
			state = &parsed
		default:
			writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("unknown review state"))
			return
		}
	}
	items, err := s.Persona.ListReviews(kind, state, 0)
	if err != nil {
		writePersonaAPIError(w, err)
		return
	}
	if raw := query.Get("cursor"); raw != "" {
		parts := strings.SplitN(raw, "|", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
			writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("cursor is invalid"))
			return
		}
		cursorTime, parseErr := time.Parse(time.RFC3339Nano, parts[0])
		if parseErr != nil {
			writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("cursor is invalid"))
			return
		}
		filtered := items[:0]
		for _, item := range items {
			if item.CreatedAt.Before(cursorTime) || (item.CreatedAt.Equal(cursorTime) && item.ID < parts[1]) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	var nextCursor any
	if limit > 0 && len(items) > limit {
		last := items[limit-1]
		nextCursor = last.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + last.ID
		items = items[:limit]
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "next_cursor": nextCursor})
}

func (s *Server) handlePersonaReviewV2(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.personaReadPrincipal(w, r); !ok {
		return
	}
	if s.Persona == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("persona service unavailable"))
		return
	}
	if len(r.URL.Query()) > 0 {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("Persona review read does not accept query parameters"))
		return
	}
	review, err := s.Persona.GetReview(strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		writePersonaAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, review)
}

func (s *Server) handlePersonaReviewDecisionV2(w http.ResponseWriter, r *http.Request, accept bool) {
	if _, ok := s.requirePrincipal(w, r, PrincipalUser); !ok {
		return
	}
	if s.Persona == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("persona service unavailable"))
		return
	}
	if r.Body != nil && r.ContentLength != 0 {
		if !personaContentTypeOK(r) {
			writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
			return
		}
		var body struct {
			DecisionReason string `json:"decision_reason"`
		}
		if err := decodeJSON(w, r, 64<<10, &body); err != nil {
			writeRequestError(w, err)
			return
		}
	}
	id := strings.TrimSpace(r.PathValue("id"))
	var err error
	if accept {
		_, err = s.Persona.AcceptRequest(id)
	} else {
		_, err = s.Persona.RejectRequest(id)
	}
	if err != nil {
		writePersonaAPIError(w, err)
		return
	}
	review, err := s.Persona.GetReview(id)
	if err != nil {
		writePersonaAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, review)
}

func writePersonaAPIError(w http.ResponseWriter, err error) {
	code := persona.CodeOf(err)
	status := http.StatusInternalServerError
	switch code {
	case "persona_uninitialized", "persona_incomplete", "persona_already_initialized", "persona_repair_not_required", "persona_revision_conflict", "persona_review_exists", "persona_review_stale":
		status = http.StatusConflict
	case "persona_kind_forbidden", "persona_invalid_content", "persona_world_entry_gate":
		status = http.StatusBadRequest
	case "persona_world_protected_claim":
		status = http.StatusForbidden
	case "persona_review_not_found", "persona_history_not_found":
		status = http.StatusNotFound
	case "persona_cap_exceeded":
		status = http.StatusRequestEntityTooLarge
	case "persona_storage_error":
		status = http.StatusInternalServerError
	}
	writeErrorWithCode(w, status, code, err)
}
