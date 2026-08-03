package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/dashimaki/garden/internal/authority"
	"github.com/dashimaki/laputa/governance"
)

func (s *Server) handleGovernanceProjection(w http.ResponseWriter, r *http.Request) {
	if s.Governed == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("governance service unavailable"))
		return
	}
	proj, err := authority.BuildProjection(r.Context(), s.Governed)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, proj)
}

func (s *Server) handleGovernanceMutation(w http.ResponseWriter, r *http.Request) {
	if s.Governed == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("governance service unavailable"))
		return
	}
	var body struct {
		Section string         `json:"section"`
		Action  string         `json:"action"`
		Reason  string         `json:"reason"`
		Data    map[string]any `json:"data,omitempty"`
		Path    string         `json:"path,omitempty"`
		Value   any            `json:"value,omitempty"`
	}
	if err := decodeJSON(w, r, 1<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	req := governance.MutationRequest{
		Section:   governance.SectionName(body.Section),
		Action:    body.Action,
		Actor:     mapActorRole(auditActor(r)),
		Reason:    body.Reason,
		RequestID: r.Header.Get("X-Garden-Request-ID"),
		Data:      body.Data,
		Path:      body.Path,
		Value:     body.Value,
	}
	if err := s.GovernedWriter.ApplyMutation(r.Context(), req); err != nil {
		if errors.Is(err, governance.ErrUnauthorized) {
			writeError(w, http.StatusForbidden, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "applied"})
}

func (s *Server) handleGovernanceAudit(w http.ResponseWriter, r *http.Request) {
	if s.GovernedWriter == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("governance service unavailable"))
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	entries, err := s.GovernedWriter.RecentAudit(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries, "count": len(entries)})
}

func mapActorRole(header string) governance.ActorRole {
	switch {
	case header == "" || header == "user_request":
		return governance.ActorUser
	case strings.HasPrefix(header, "agent"):
		return governance.ActorAgent
	case header == "report_system":
		return governance.ActorReportSystem
	case header == "system":
		return governance.ActorSystem
	default:
		return governance.ActorRole(header)
	}
}
