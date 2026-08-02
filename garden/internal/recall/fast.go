package recall

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dashimaki/garden/internal/activity"
	"github.com/dashimaki/garden/internal/authority"
	"github.com/dashimaki/mentle/facade"
)

type CardSearcher interface {
	SearchCards(context.Context, facade.CardQuery) (facade.CardPage, error)
	ReadEvidence(context.Context, facade.EvidenceQuery) ([]facade.EvidenceFragment, error)
}

type FastService struct {
	Gov      authority.GovernanceReader
	Searcher CardSearcher
	WS       *activity.WorkingSet
}

type FastRequest struct {
	Query       string `json:"query"`
	Scope       string `json:"scope"`
	BudgetChars int    `json:"budget_chars"`
	SessionID   string `json:"session_id"`
}

type ContextView struct {
	TraceID       string                         `json:"trace_id"`
	Scope         string                         `json:"scope"`
	Mode          string                         `json:"mode"`
	Governance    authority.GovernanceProjection `json:"governance"`
	Cards         []facade.MemoryCard            `json:"cards"`
	Evidence      []facade.EvidenceFragment      `json:"evidence"`
	Context       string                         `json:"context"`
	BudgetChars   int                            `json:"budget_chars"`
	Degraded      bool                           `json:"degraded"`
	Warnings      []string                       `json:"warnings"`
	RecallTraceID *string                        `json:"recall_trace_id,omitempty"`
}

const (
	defaultBudget = 6000
	minBudget     = 256
	maxBudget     = 64000
	maxCards      = 12
	cardLimit     = 20
)

func (s *FastService) Recall(ctx context.Context, req FastRequest) (ContextView, error) {
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		return ContextView{}, errors.New("query is required")
	}
	if req.BudgetChars == 0 {
		req.BudgetChars = defaultBudget
	}
	if req.BudgetChars < minBudget || req.BudgetChars > maxBudget {
		return ContextView{}, fmt.Errorf("budget_chars must be between %d and %d", minBudget, maxBudget)
	}

	traceID := fmt.Sprintf("recall_%d", time.Now().UnixNano())
	view := ContextView{
		TraceID:     traceID,
		Scope:       req.Scope,
		Mode:        "fast",
		BudgetChars: req.BudgetChars,
		Cards:       []facade.MemoryCard{},
		Evidence:    []facade.EvidenceFragment{},
		Warnings:    []string{},
	}

	proj, err := authority.BuildProjection(ctx, s.Gov)
	if err != nil {
		return ContextView{}, fmt.Errorf("governance projection: %w", err)
	}
	view.Governance = proj

	if s.Searcher == nil {
		view.Degraded = true
		view.Warnings = append(view.Warnings, "mentle unavailable; governance-only context")
		view.Context = governanceContext(proj, req.BudgetChars)
		return view, nil
	}

	page, err := s.Searcher.SearchCards(ctx, facade.CardQuery{Text: req.Query, Scope: req.Scope, Limit: cardLimit})
	if err != nil {
		view.Degraded = true
		view.Warnings = append(view.Warnings, "mentle search failed; governance-only context")
		view.Context = governanceContext(proj, req.BudgetChars)
		return view, nil
	}

	cards := FilterCards(page.Cards, proj)
	cards = RankCards(cards, req.Scope)
	cards = DeduplicateCards(cards)
	if len(cards) > maxCards {
		cards = cards[:maxCards]
	}
	view.Cards = cards

	if len(cards) > 0 {
		ids := make([]string, len(cards))
		for i, card := range cards {
			ids[i] = card.ID
		}
		totalBudget := req.BudgetChars
		if totalBudget > 4000 {
			totalBudget = 4000
		}
		evidence, evErr := s.Searcher.ReadEvidence(ctx, facade.EvidenceQuery{CardIDs: ids, PerItemBudget: 800, TotalBudget: totalBudget})
		if evErr != nil {
			view.Degraded = true
			view.Warnings = append(view.Warnings, "evidence read failed")
		} else {
			view.Evidence = evidence
		}
	}

	if s.WS != nil && req.SessionID != "" {
		cardIDs := make([]string, len(view.Cards))
		for i, c := range view.Cards {
			cardIDs[i] = c.ID
		}
		evRefs := make([]string, len(view.Evidence))
		for i, e := range view.Evidence {
			evRefs[i] = e.MaterialRef
		}
		s.WS.Update(req.Scope, cardIDs, evRefs)
	}

	view.Context = assembleContext(view.Evidence, proj, req.BudgetChars)
	return view, nil
}

