package server

import (
	"errors"
	"net/http"

	"github.com/dashimaki/garden/internal/evolution"
)

func (s *Server) handleEvolutionStartRun(w http.ResponseWriter, r *http.Request) {
	if s.Evolution == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("evolution service unavailable"))
		return
	}
	var body evolution.EvolutionEvidenceBundle
	if err := decodeJSON(w, r, 1<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	run, err := s.Evolution.StartRun(r.Context(), body, auditActor(r))
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
	if s.Evolution == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("evolution service unavailable"))
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
	proposal, err := s.Evolution.CreateProposal(r.Context(), body.RunID, body.CandidateID, auditActor(r))
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
	if s.Evolution == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("evolution service unavailable"))
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
	proposal, err := s.Evolution.ReviewProposal(r.Context(), proposalID, body.Decision, auditActor(r), body.Note)
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
