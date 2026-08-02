package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/dashimaki/mentle/facade"
)

type MaterialsProvider interface {
	SearchCards(context.Context, facade.CardQuery) (facade.CardPage, error)
	ReadEvidence(context.Context, facade.EvidenceQuery) ([]facade.EvidenceFragment, error)
	ListCollections(context.Context) ([]facade.CollectionInfo, error)
}

func (s *Server) handleMaterialsCards(w http.ResponseWriter, r *http.Request) {
	if s.Materials == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("mentle unavailable"))
		return
	}
	q := r.URL.Query()
	query := q.Get("query")
	if query == "" {
		writeError(w, http.StatusBadRequest, errors.New("query parameter is required"))
		return
	}
	limit := 20
	if raw := q.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, errors.New("limit must be between 1 and 100"))
			return
		}
		limit = parsed
	}
	page, err := s.Materials.SearchCards(r.Context(), facade.CardQuery{
		Text:       query,
		Collection: q.Get("collection"),
		Scope:      q.Get("scope"),
		Limit:      limit,
		Cursor:     q.Get("cursor"),
	})
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"cards":       page.Cards,
		"next_cursor": page.NextCursor,
		"source":      "live",
	})
}

func (s *Server) handleMaterialsEvidence(w http.ResponseWriter, r *http.Request) {
	if s.Materials == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("mentle unavailable"))
		return
	}
	cardID := r.PathValue("id")
	if cardID == "" {
		writeError(w, http.StatusBadRequest, errors.New("card id is required"))
		return
	}
	perItem := 800
	if raw := r.URL.Query().Get("per_item_budget"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 4000 {
			perItem = parsed
		}
	}
	total := 4000
	if raw := r.URL.Query().Get("total_budget"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 16000 {
			total = parsed
		}
	}
	fragments, err := s.Materials.ReadEvidence(r.Context(), facade.EvidenceQuery{
		CardIDs:       []string{cardID},
		PerItemBudget: perItem,
		TotalBudget:   total,
	})
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"card_id":   cardID,
		"fragments": fragments,
		"source":    "live",
	})
}

func (s *Server) handleMaterialsCollections(w http.ResponseWriter, r *http.Request) {
	if s.Materials == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("mentle unavailable"))
		return
	}
	collections, err := s.Materials.ListCollections(r.Context())
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"collections": collections,
		"source":      "live",
	})
}
