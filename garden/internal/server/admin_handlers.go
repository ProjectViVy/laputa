package server

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ProjectViVy/laputa/garden/agentapi"
	"github.com/ProjectViVy/laputa/garden/internal/recall"
	"github.com/ProjectViVy/laputa/mentle/facade"
)

func (s *Server) indexHealth(ctx context.Context, principal Principal) (facade.IndexHealth, error) {
	if s.AgentAPI == nil || strings.TrimSpace(s.ProfileID) == "" {
		return facade.IndexHealth{Status: "unavailable", Reasons: []string{"canonical_probe_failed"}}, facade.ErrIndexHealthUnavailable
	}
	return s.AgentAPI.IndexHealth(ctx, agentapi.Principal(principal), agentapi.Binding{
		ProfileID: s.ProfileID, AgentID: "garden-http", Platform: "http",
	})
}

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
	indexHealth, healthErr := s.indexHealth(r.Context(), PrincipalRead)
	resp["index_health"] = indexHealth
	if healthErr != nil || indexHealth.Status != "ok" {
		resp["status"] = "degraded"
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

func (s *Server) handleAdminIndexHealth(w http.ResponseWriter, r *http.Request) {
	principal, ok := s.requireReadPrincipal(w, r, PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalAutodream, PrincipalOperator)
	if !ok {
		return
	}
	health, err := s.indexHealth(r.Context(), principal)
	if err != nil {
		writeErrorWithDetails(w, http.StatusServiceUnavailable, "index_health_unavailable", errors.New("live index probes unavailable"), map[string]any{"reasons": health.Reasons})
		return
	}
	writeJSON(w, http.StatusOK, health)
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
	indexHealth, healthErr := s.indexHealth(r.Context(), PrincipalRead)
	if healthErr != nil {
		merged["mentle"] = "unavailable"
	} else {
		merged["mentle"] = indexHealth.Status
	}

	entries := make([]componentEntry, 0, len(merged))
	for name, status := range merged {
		entries = append(entries, componentEntry{Name: name, Status: status, Source: "live"})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"components":   entries,
		"index_health": indexHealth,
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
