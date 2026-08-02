package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dashimaki/garden/internal/activity"
)

func (s *Server) handleActivityEvents(w http.ResponseWriter, r *http.Request) {
	if s.Activity == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("activity service unavailable"))
		return
	}
	var body struct {
		SessionID string         `json:"session_id"`
		EventID   string         `json:"event_id"`
		Type      string         `json:"type"`
		Data      map[string]any `json:"data"`
	}
	if err := decodeJSON(w, r, 1<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	if body.SessionID == "" || body.EventID == "" || body.Type == "" {
		writeError(w, http.StatusBadRequest, errors.New("session_id, event_id, and type are required"))
		return
	}
	ev := activity.Event{ID: body.EventID, SessionID: body.SessionID, Type: body.Type, Data: body.Data}
	if err := s.Activity.Append(r.Context(), ev); err != nil {
		writeHandlerError(w, err)
		return
	}
	if body.Type == "session_end" && s.Checkpointer != nil {
		_ = s.Checkpointer.Save(r.Context(), "")
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "accepted", "event_id": body.EventID})
}

func (s *Server) handleSessionActivity(w http.ResponseWriter, r *http.Request) {
	if s.Activity == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("activity service unavailable"))
		return
	}
	sessionID := r.PathValue("session_id")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, errors.New("session_id is required"))
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
			if limit > 200 {
				limit = 200
			}
		}
	}
	events, err := s.Activity.SessionEvents(r.Context(), sessionID, limit)
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	if events == nil {
		events = []activity.Event{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"session_id": sessionID, "events": events})
}
