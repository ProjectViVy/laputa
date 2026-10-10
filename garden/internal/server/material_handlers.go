package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ProjectViVy/laputa/garden/agentapi"
)

func (s *Server) materialsRead(w http.ResponseWriter, r *http.Request) (agentapi.Principal, agentapi.Binding, bool) {
	principal, ok := s.requireReadPrincipal(w, r, PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalAutodream, PrincipalOperator)
	if !ok {
		return "", agentapi.Binding{}, false
	}
	if s.AgentAPI == nil || strings.TrimSpace(s.ProfileID) == "" {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "unavailable", errors.New("mentle unavailable"))
		return "", agentapi.Binding{}, false
	}
	return agentapi.Principal(principal), agentapi.Binding{ProfileID: s.ProfileID, AgentID: "garden-http", Platform: "http", SessionID: "garden-http-read"}, true
}

func writeMaterialsReadError(w http.ResponseWriter, err error) {
	var domainErr *agentapi.Error
	if errors.As(err, &domainErr) {
		status := http.StatusBadRequest
		switch domainErr.Code {
		case "unavailable":
			status = http.StatusServiceUnavailable
		case "authentication_required":
			status = http.StatusUnauthorized
		case "principal_forbidden", "profile_mismatch":
			status = http.StatusForbidden
		case "not_found":
			status = http.StatusNotFound
		case "conflict":
			status = http.StatusConflict
		}
		writeErrorWithCode(w, status, domainErr.Code, err)
		return
	}
	writeErrorWithCode(w, http.StatusInternalServerError, "internal_error", errors.New("material read failed"))
}

func (s *Server) handleMaterialsCards(w http.ResponseWriter, r *http.Request) {
	principal, binding, ok := s.materialsRead(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	query := q.Get("query")
	if query == "" {
		writeError(w, http.StatusBadRequest, errors.New("query parameter is required"))
		return
	}
	limit := 20
	if raw := q.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, errors.New("limit must be between 1 and 100"))
			return
		}
		limit = parsed
	}
	page, err := s.AgentAPI.SearchCards(r.Context(), principal, binding, agentapi.CardSearch{Query: query, Collection: q.Get("collection"), Limit: limit, Cursor: q.Get("cursor")})
	if err != nil {
		writeMaterialsReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cards": page.Cards, "next_cursor": page.NextCursor, "source": "live"})
}

func (s *Server) handleMaterialsEvidence(w http.ResponseWriter, r *http.Request) {
	principal, binding, ok := s.materialsRead(w, r)
	if !ok {
		return
	}
	cardID := r.PathValue("id")
	if cardID == "" {
		writeError(w, http.StatusBadRequest, errors.New("card id is required"))
		return
	}
	perItem := 800
	if raw := r.URL.Query().Get("per_item_budget"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 4000 {
			perItem = parsed
		}
	}
	total := 4000
	if raw := r.URL.Query().Get("total_budget"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 16000 {
			total = parsed
		}
	}
	revision, err := strconv.ParseUint(r.URL.Query().Get("expected_revision"), 10, 64)
	if err != nil || revision == 0 {
		writeError(w, http.StatusBadRequest, errors.New("expected_revision query parameter is required"))
		return
	}
	fragments, err := s.AgentAPI.ReadEvidence(r.Context(), principal, binding, agentapi.EvidenceRead{Items: []agentapi.EvidenceRef{{CardID: cardID, ExpectedRevision: revision}}, PerItemBudget: perItem, TotalBudget: total})
	if err != nil {
		writeMaterialsReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"card_id": cardID, "fragments": fragments, "source": "live"})
}

func (s *Server) handleMaterialsCollections(w http.ResponseWriter, r *http.Request) {
	principal, binding, ok := s.materialsRead(w, r)
	if !ok {
		return
	}
	collections, err := s.AgentAPI.ListCollections(r.Context(), principal, binding)
	if err != nil {
		writeMaterialsReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"collections": collections, "source": "live"})
}
