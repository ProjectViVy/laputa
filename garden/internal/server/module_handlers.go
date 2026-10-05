package server

import (
	"errors"
	"net/http"

	"github.com/ProjectViVy/laputa/garden/internal/report"
)

func (s *Server) handleModulesList(w http.ResponseWriter, r *http.Request) {
	if s.Reports == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("report service unavailable"))
		return
	}
	kind := r.URL.Query().Get("kind")
	status := r.URL.Query().Get("status")
	if status == "" {
		status = report.ModuleStatusActive
	}
	items, err := s.Reports.ListModules(r.Context(), kind, status)
	if err != nil {
		writeModuleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"kind": kind, "status": status, "items": items, "count": len(items)})
}

func (s *Server) handleModuleCreate(w http.ResponseWriter, r *http.Request) {
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
		Kind    string `json:"kind"`
		Content string `json:"content"`
	}
	if err := decodeJSON(w, r, 1<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	m, err := s.Reports.CreateModule(r.Context(), body.Kind, body.Content)
	if err != nil {
		writeModuleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) handleModuleUpdate(w http.ResponseWriter, r *http.Request) {
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
		Content string `json:"content"`
		Status  string `json:"status"`
	}
	if err := decodeJSON(w, r, 1<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	m, err := s.Reports.UpdateModule(r.Context(), r.PathValue("id"), body.Content, body.Status)
	if err != nil {
		writeModuleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func writeModuleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, report.ErrInvalidModuleKind),
		errors.Is(err, report.ErrInvalidModuleStatus),
		errors.Is(err, report.ErrEmptyModuleContent),
		errors.Is(err, report.ErrModuleContentTooLong),
		errors.Is(err, report.ErrNoModuleChange):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeHandlerError(w, err)
	}
}
