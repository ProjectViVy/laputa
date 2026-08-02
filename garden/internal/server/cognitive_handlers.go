package server

import (
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleCognitiveWorld(w http.ResponseWriter, r *http.Request) {
	if s.Cognitive == nil {
		writeError(w, http.StatusServiceUnavailable, errCognitiveUnavailable)
		return
	}

	var scopes []string
	if raw := r.URL.Query().Get("scope"); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			if s := strings.TrimSpace(part); s != "" {
				scopes = append(scopes, s)
			}
		}
	}

	budget := 4000
	if raw := r.URL.Query().Get("budget"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 16000 {
			writeError(w, http.StatusBadRequest, errInvalidBudget)
			return
		}
		budget = parsed
	}

	claims := s.Cognitive.Project(scopes, budget)
	writeJSON(w, http.StatusOK, map[string]any{
		"claims":       claims,
		"total":        len(s.Cognitive.Claims),
		"projected":    len(claims),
		"budget_chars": budget,
		"source":       "live",
	})
}

var errCognitiveUnavailable = &httpError{"cognitive world store unavailable", http.StatusServiceUnavailable}
var errInvalidBudget = &httpError{"budget must be between 1 and 16000", http.StatusBadRequest}

type httpError struct {
	msg  string
	code int
}

func (e *httpError) Error() string { return e.msg }
