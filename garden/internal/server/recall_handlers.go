package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dashimaki/garden/agentapi"
	"github.com/dashimaki/garden/internal/recall"
)

func (s *Server) handleFastRecall(w http.ResponseWriter, r *http.Request) {
	principal, ok := s.requireReadPrincipal(w, r, PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalAutodream, PrincipalOperator)
	if !ok {
		return
	}
	// The ordinary recall route accepts recall inputs, not caller-supplied identity.
	var body struct {
		Query       string `json:"query"`
		Scope       string `json:"scope"`
		BudgetChars int    `json:"budget_chars"`
		SessionID   string `json:"session_id"`
	}
	if err := decodeJSON(w, r, 1<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	if s.AgentAPI == nil || strings.TrimSpace(s.ProfileID) == "" {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "unavailable", errors.New("fast recall unavailable"))
		return
	}
	view, err := s.AgentAPI.FastRecall(r.Context(), agentapi.Principal(principal), agentapi.FastRecallRequest{
		Binding: agentapi.Binding{ProfileID: s.ProfileID, AgentID: "garden-http", Platform: "http", SessionID: body.SessionID},
		Query:   body.Query, Scope: body.Scope, BudgetChars: body.BudgetChars,
	})
	if err != nil {
		var domainErr *agentapi.Error
		if errors.As(err, &domainErr) {
			status := http.StatusBadRequest
			if domainErr.Code == "unavailable" {
				status = http.StatusServiceUnavailable
			}
			writeErrorWithCode(w, status, domainErr.Code, err)
		} else {
			writeErrorWithCode(w, http.StatusInternalServerError, "internal_error", errors.New("fast recall unavailable"))
		}
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleDeepRecall(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireReadPrincipal(w, r, PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalAutodream, PrincipalOperator); !ok {
		return
	}
	if s.DeepRecall == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("deep recall service unavailable"))
		return
	}
	var body recall.DeepRequest
	if err := decodeJSON(w, r, 1<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	resp, err := s.DeepRecall.Recall(r.Context(), body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleRecallTrace(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireReadPrincipal(w, r, PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalAutodream, PrincipalOperator); !ok {
		return
	}
	if s.TraceStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("trace store unavailable"))
		return
	}
	traceID := r.PathValue("trace_id")
	trace, err := s.TraceStore.Get(r.Context(), traceID)
	if errors.Is(err, recall.ErrTraceNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, trace)
}
