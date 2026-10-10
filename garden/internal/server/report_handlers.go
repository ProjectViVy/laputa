package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ProjectViVy/laputa/garden/internal/report"
	"github.com/ProjectViVy/laputa/mentle/facade"
)

func (s *Server) handleReportsList(w http.ResponseWriter, r *http.Request) {
	if s.Reports == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("report service unavailable"))
		return
	}
	cadence := r.URL.Query().Get("cadence")
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			writeError(w, http.StatusBadRequest, errors.New("invalid limit"))
			return
		}
		limit = n
	}
	items, err := s.Reports.List(r.Context(), cadence, limit)
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cadence": cadence, "items": items, "count": len(items)})
}

func (s *Server) handleReportGenerate(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalOperator); !ok {
		return
	}
	if s.Reports == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("report service unavailable"))
		return
	}
	if !personaContentTypeOK(r) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
		return
	}
	var body struct {
		Cadence string `json:"cadence"`
	}
	if err := decodeJSON(w, r, 1<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	now := time.Now
	if s.now != nil {
		now = s.now
	}
	rep, err := s.Reports.Generate(r.Context(), body.Cadence, now().UTC())
	if errors.Is(err, report.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"generated": false, "reason": "no source memories in window"})
		return
	}
	if errors.Is(err, facade.ErrUnavailable) {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"generated": true, "report": rep})
}

func (s *Server) handleReportOrientation(w http.ResponseWriter, r *http.Request) {
	if s.Reports == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("report service unavailable"))
		return
	}
	budget := 0
	if v := r.URL.Query().Get("budget"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			writeError(w, http.StatusBadRequest, errors.New("invalid budget"))
			return
		}
		budget = n
	}
	view, err := s.Reports.Orientation(r.Context(), budget)
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}
