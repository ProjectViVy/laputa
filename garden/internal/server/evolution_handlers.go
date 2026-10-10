package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ProjectViVy/laputa/garden/internal/evolution"
)

func (s *Server) evolutionListLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, errors.New("limit must be between 1 and 100"))
			return 0, false
		}
		limit = parsed
	}
	return limit, true
}

func (s *Server) handleEvolutionListRuns(w http.ResponseWriter, r *http.Request) {
	if s.Evolution == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("evolution service unavailable"))
		return
	}
	limit, ok := s.evolutionListLimit(w, r)
	if !ok {
		return
	}
	runs, err := s.Evolution.ListRuns(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": runs, "count": len(runs)})
}

func (s *Server) handleEvolutionListProposals(w http.ResponseWriter, r *http.Request) {
	if s.Evolution == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("evolution service unavailable"))
		return
	}
	limit, ok := s.evolutionListLimit(w, r)
	if !ok {
		return
	}
	proposals, err := s.Evolution.ListProposals(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": proposals, "count": len(proposals)})
}

func (s *Server) handleEvolutionStartRun(w http.ResponseWriter, r *http.Request) {
	principal, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalAgent)
	if !ok {
		return
	}
	if s.Evolution == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("evolution service unavailable"))
		return
	}
	if !personaContentTypeOK(r) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
		return
	}
	var body evolution.EvolutionCandidateInput
	if err := decodeJSON(w, r, 1<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	run, err := s.Evolution.StartRunFromCandidateInput(r.Context(), body, auditLabel(r, principal))
	if errors.Is(err, evolution.ErrProviderUnavailable) {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusAccepted, run)
}

func (s *Server) handleEvolutionGetRun(w http.ResponseWriter, r *http.Request) {
	if s.Evolution == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("evolution service unavailable"))
		return
	}
	runID := r.PathValue("run_id")
	run, err := s.Evolution.GetRun(r.Context(), runID)
	if errors.Is(err, evolution.ErrRunNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) handleEvolutionGetCandidate(w http.ResponseWriter, r *http.Request) {
	if s.Evolution == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("evolution service unavailable"))
		return
	}
	candidateID := r.PathValue("candidate_id")
	candidate, err := s.Evolution.GetCandidate(r.Context(), candidateID)
	if errors.Is(err, evolution.ErrCandidateNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, candidate)
}

func (s *Server) handleEvolutionCreateProposal(w http.ResponseWriter, r *http.Request) {
	principal, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalAgent)
	if !ok {
		return
	}
	if s.Evolution == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("evolution service unavailable"))
		return
	}
	if !personaContentTypeOK(r) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
		return
	}
	var body struct {
		RunID       string `json:"run_id"`
		CandidateID string `json:"candidate_id"`
	}
	if err := decodeJSON(w, r, 1<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	proposal, err := s.Evolution.CreateProposal(r.Context(), body.RunID, body.CandidateID, auditLabel(r, principal))
	if errors.Is(err, evolution.ErrCandidateNotFound) || errors.Is(err, evolution.ErrRunNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, proposal)
}

func (s *Server) handleEvolutionGetProposal(w http.ResponseWriter, r *http.Request) {
	if s.Evolution == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("evolution service unavailable"))
		return
	}
	proposalID := r.PathValue("proposal_id")
	proposal, err := s.Evolution.GetProposal(r.Context(), proposalID)
	if errors.Is(err, evolution.ErrProposalNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, proposal)
}

func (s *Server) handleEvolutionReviewProposal(w http.ResponseWriter, r *http.Request) {
	principal, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalOperator)
	if !ok {
		return
	}
	if s.Evolution == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("evolution service unavailable"))
		return
	}
	if !personaContentTypeOK(r) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
		return
	}
	proposalID := r.PathValue("proposal_id")
	var body struct {
		Decision string `json:"decision"`
		Note     string `json:"note"`
	}
	if err := decodeJSON(w, r, 1<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	proposal, err := s.Evolution.ReviewProposal(r.Context(), proposalID, body.Decision, auditLabel(r, principal), body.Note)
	if errors.Is(err, evolution.ErrProposalNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if errors.Is(err, evolution.ErrInvalidDecision) {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, proposal)
}

func (s *Server) handleEvolutionGetEvent(w http.ResponseWriter, r *http.Request) {
	if s.Evolution == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("evolution service unavailable"))
		return
	}
	eventID := r.PathValue("event_id")
	event, err := s.Evolution.GetEvent(r.Context(), eventID)
	if errors.Is(err, evolution.ErrEventNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (s *Server) handleEvolutionHubStatus(w http.ResponseWriter, r *http.Request) {
	if s.Evolution == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("evolution service unavailable"))
		return
	}
	status, err := s.Evolution.HubStatus(r.Context())
	if errors.Is(err, evolution.ErrProviderUnavailable) {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}
