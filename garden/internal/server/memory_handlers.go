package server

import (
	"errors"
	"github.com/ProjectViVy/laputa/garden/agentapi"
	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"net/http"
	"strconv"
)

type memoryInput struct {
	Content         *string               `json:"content"`
	Kind            string                `json:"kind,omitempty"`
	Tags            *[]string             `json:"tags,omitempty"`
	Collection      string                `json:"collection,omitempty"`
	SharedSpaceID   string                `json:"shared_space_id,omitempty"`
	ExpectedVersion uint64                `json:"expected_version,omitempty"`
	Reason          string                `json:"reason,omitempty"`
	Sources         []evolution.SourceRef `json:"sources,omitempty"`
}

func memoryWire(r memory.Record) map[string]any {
	return map[string]any{"id": r.ID, "scope": memory.EncodeScope(r.Scope), "content": r.Body, "kind": r.Kind, "tags": r.Tags, "status": r.Status, "version": r.Revision, "source": r.Sources, "valid_from": r.OccurredAt, "created_at": r.CapturedAt, "updated_at": r.UpdatedAt}
}
func (s *Server) memoryReady(w http.ResponseWriter) bool {
	if s.AgentAPI == nil {
		writeErrorWithCode(w, 503, "unavailable", errors.New("memory unavailable"))
		return false
	}
	return true
}
func (s *Server) handleListMemories(w http.ResponseWriter, r *http.Request) {
	if !s.memoryReady(w) {
		return
	}
	p, ok := s.requireReadPrincipal(w, r, PrincipalUser, PrincipalAgent, PrincipalRead)
	if !ok {
		return
	}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, e := strconv.Atoi(raw)
		if e != nil || n < 1 || n > 100 {
			writeError(w, 400, errors.New("limit must be 1..100"))
			return
		}
		limit = n
	}
	page, err := s.AgentAPI.ListMemory(r.Context(), agentapi.Principal(p), s.requestBinding("garden-http-read"), memory.ListRequest{Collection: r.URL.Query().Get("collection"), Cursor: r.URL.Query().Get("cursor"), Limit: limit})
	if err != nil {
		writeMaterialsReadError(w, err)
		return
	}
	rows := []map[string]any{}
	for _, row := range page.Items {
		rows = append(rows, memoryWire(row))
	}
	writeJSON(w, 200, map[string]any{"items": rows, "next_cursor": page.NextCursor, "truncated": page.Truncated})
}
func (s *Server) handleGetMemory(w http.ResponseWriter, r *http.Request) {
	if !s.memoryReady(w) {
		return
	}
	p, ok := s.requireReadPrincipal(w, r, PrincipalUser, PrincipalAgent, PrincipalRead)
	if !ok {
		return
	}
	row, err := s.AgentAPI.GetMemory(r.Context(), agentapi.Principal(p), s.requestBinding("garden-http-read"), r.PathValue("id"))
	if err != nil {
		writeMaterialsReadError(w, err)
		return
	}
	writeJSON(w, 200, memoryWire(row))
}
func (s *Server) mutateHTTP(w http.ResponseWriter, r *http.Request, op evolution.MutationOperation) {
	if !s.memoryReady(w) {
		return
	}
	p, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalAgent)
	if !ok {
		return
	}
	if !personaContentTypeOK(r) {
		writeError(w, 400, errors.New("content type must be application/json"))
		return
	}
	var body memoryInput
	if err := decodeJSON(w, r, 64<<10, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	m := memory.AuthorizedMutation{OperationID: r.Header.Get("Idempotency-Key"), Operation: op, RecordID: r.PathValue("id"), ExpectedRevision: body.ExpectedVersion, ExpectedAbsent: op == evolution.MutationCreate, Kind: body.Kind, Tags: body.Tags, Collection: body.Collection, Sources: body.Sources, Inference: evolution.InferenceObserved}
	if body.Content != nil {
		m.Body = *body.Content
	}
	if op == evolution.MutationUpdate {
		if body.Content == nil && body.Tags == nil {
			writeError(w, 400, errors.New("content or tags required"))
			return
		}

	}
	if op != evolution.MutationCreate && (m.ExpectedRevision == 0 || body.Reason == "") {
		writeError(w, 400, errors.New("expected_version and reason required"))
		return
	}
	receipt, err := s.AgentAPI.MutateMemory(r.Context(), agentapi.Principal(p), s.requestBinding("garden-http-write"), agentapi.MemoryTarget{SharedSpaceID: body.SharedSpaceID}, m)
	if err != nil {
		writeMaterialsReadError(w, err)
		return
	}
	status := 200
	if op == evolution.MutationCreate {
		status = 201
	}
	writeJSON(w, status, map[string]any{"memory": map[string]any{"id": receipt.TargetRef, "version": receipt.Revision}, "id": receipt.TargetRef, "version": receipt.Revision, "receipt": receipt, "canonical_status": receipt.CanonicalStatus, "index_status": receipt.IndexStatus})
}
func (s *Server) handleCreateMemory(w http.ResponseWriter, r *http.Request) {
	s.mutateHTTP(w, r, evolution.MutationCreate)
}
func (s *Server) handleUpdateMemory(w http.ResponseWriter, r *http.Request) {
	s.mutateHTTP(w, r, evolution.MutationUpdate)
}
func (s *Server) handleDeleteMemory(w http.ResponseWriter, r *http.Request) {
	s.mutateHTTP(w, r, evolution.MutationTombstone)
}
