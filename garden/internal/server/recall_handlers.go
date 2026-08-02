package server

import (
	"errors"
	"net/http"

	"github.com/dashimaki/garden/internal/recall"
)

func (s *Server) handleFastRecall(w http.ResponseWriter, r *http.Request) {
	if s.FastRecall == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("recall service unavailable"))
		return
	}
	var body recall.FastRequest
	if err := decodeJSON(w, r, 1<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	view, err := s.FastRecall.Recall(r.Context(), body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleDeepRecall(w http.ResponseWriter, r *http.Request) {
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
