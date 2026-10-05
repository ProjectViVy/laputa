package server

import (
	"errors"
	"net/http"

	"github.com/ProjectViVy/laputa/garden/internal/mailbox"
)

func (s *Server) handleMailboxInbox(w http.ResponseWriter, r *http.Request) {
	s.writeMailboxList(w, r, mailbox.DirectionInbox, r.URL.Query().Get("state"))
}

func (s *Server) handleMailboxOutbox(w http.ResponseWriter, r *http.Request) {
	s.writeMailboxList(w, r, mailbox.DirectionOutbox, r.URL.Query().Get("state"))
}

func (s *Server) handleMailboxDeadLetter(w http.ResponseWriter, r *http.Request) {
	if s.Mailbox == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("mailbox unavailable"))
		return
	}
	items, err := s.Mailbox.ListDeadLetter(r.Context())
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"box": "dead_letter", "items": items, "count": len(items)})
}

func (s *Server) writeMailboxList(w http.ResponseWriter, r *http.Request, direction, state string) {
	if s.Mailbox == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("mailbox unavailable"))
		return
	}
	items, err := s.Mailbox.List(r.Context(), direction, state)
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"box": direction, "items": items, "count": len(items)})
}

func (s *Server) handleMailboxApprove(w http.ResponseWriter, r *http.Request) {
	s.writeMailboxReview(w, r, mailbox.StateApproved)
}

func (s *Server) handleMailboxReject(w http.ResponseWriter, r *http.Request) {
	s.writeMailboxReview(w, r, mailbox.StateRejected)
}

func (s *Server) writeMailboxReview(w http.ResponseWriter, r *http.Request, decision string) {
	if _, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalOperator); !ok {
		return
	}
	if s.Mailbox == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("mailbox unavailable"))
		return
	}
	if !personaContentTypeOK(r) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, 1<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	if body.Reason == "" {
		body.Reason = "explicit user review"
	}
	item, err := s.Mailbox.Review(r.Context(), r.PathValue("id"), decision, body.Reason)
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
