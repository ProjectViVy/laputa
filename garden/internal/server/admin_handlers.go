package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dashimaki/garden/internal/recall"
	"github.com/dashimaki/laputa/governance"
)

func (s *Server) handleAdminOverview(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	components := map[string]string{"garden": "ok", "laputa": "ok"}
	for name, value := range s.Components {
		components[name] = value
		if value != "ok" {
			status = "degraded"
		}
	}

	resp := map[string]any{
		"status":     status,
		"components": components,
		"source":     "live",
	}

	if s.Ingestions != nil {
		if stats, err := s.Ingestions.Stats(r.Context()); err == nil {
			resp["ingestion"] = stats
		}
		if s.Ingestions.Spool != nil {
			if count, err := s.Ingestions.Spool.PendingCount(r.Context()); err == nil {
				resp["spool_pending"] = count
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminComponents(w http.ResponseWriter, r *http.Request) {
	type componentEntry struct {
		Name   string `json:"name"`
		Status string `json:"status"`
		Source string `json:"source"`
	}

	merged := map[string]string{"garden": "ok", "laputa": "ok"}
	for name, value := range s.Components {
		merged[name] = value
	}

	entries := make([]componentEntry, 0, len(merged))
	for name, status := range merged {
		entries = append(entries, componentEntry{Name: name, Status: status, Source: "live"})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"components":   entries,
		"api_contract": "garden-hermes/1",
	})
}

func (s *Server) handleAdminContextManifest(w http.ResponseWriter, r *http.Request) {
	if s.TraceStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("trace store unavailable"))
		return
	}
	trace, err := s.TraceStore.Get(r.Context(), r.PathValue("trace_id"))
	if errors.Is(err, recall.ErrTraceNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"trace": trace, "source": "live"})
}

func (s *Server) handleAdminSpool(w http.ResponseWriter, r *http.Request) {
	if s.Ingestions == nil || s.Ingestions.Spool == nil {
		writeJSON(w, http.StatusOK, map[string]any{"pending_count": 0, "entries": []any{}, "source": "live"})
		return
	}
	entries, err := s.Ingestions.Spool.Pending(r.Context())
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	type spoolEntry struct {
		EventID   string `json:"event_id"`
		SessionID string `json:"session_id"`
		Scope     string `json:"scope"`
		Kind      string `json:"kind"`
		CreatedAt string `json:"created_at"`
	}

	const maxEntries = 50
	out := make([]spoolEntry, 0, min(len(entries), maxEntries))
	for i, e := range entries {
		if i >= maxEntries {
			break
		}
		out = append(out, spoolEntry{EventID: e.EventID, SessionID: e.SessionID, Scope: e.Scope, Kind: e.Kind, CreatedAt: e.CreatedAt})
	}

	writeJSON(w, http.StatusOK, map[string]any{"pending_count": len(entries), "entries": out, "source": "live"})
}

func (s *Server) handleAdminAudit(w http.ResponseWriter, r *http.Request) {
	if s.Governed == nil {
		writeJSON(w, http.StatusOK, map[string]any{"entries": []any{}, "count": 0, "source": "live"})
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 200 {
		limit = 200
	}
	entries, err := s.Governed.RecentAudit(r.Context(), limit)
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	if entries == nil {
		entries = []governance.AuditEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries, "count": len(entries), "source": "live"})
}
