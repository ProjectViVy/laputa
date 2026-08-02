package recall

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dashimaki/garden/internal/arbiter"
	"github.com/dashimaki/garden/internal/rag"
	"github.com/dashimaki/mentle/facade"
)

type GraphSource interface {
	QueryEntity(context.Context, string, string, string) ([]facade.GraphFact, error)
	Timeline(context.Context, string) ([]facade.TimelineEvent, error)
}

type TemporalAssertion = arbiter.Assertion

type DeepRequest struct {
	Query         string   `json:"query"`
	Scope         string   `json:"scope"`
	BudgetChars   int      `json:"budget_chars"`
	SessionID     string   `json:"session_id"`
	TriggerReason string   `json:"trigger_reason"`
	Capabilities  []string `json:"capabilities"`
	Entities      []string `json:"entities,omitempty"`
	UsePlanner    bool     `json:"use_planner,omitempty"`
	AsOf          string   `json:"as_of,omitempty"`
}

type DeepResponse struct {
	ContextView
	Assertions []TemporalAssertion        `json:"assertions,omitempty"`
	Proposals  []arbiter.ConflictProposal `json:"proposals,omitempty"`
	Trace      RecallTrace                `json:"recall_trace"`
}

type DeepService struct {
	Fast    *FastService
	Graph   GraphSource
	Planner rag.Planner
	Arbiter *arbiter.Arbiter
	Traces  *TraceStore
}

const (
	deepMaxCards    = 20
	deepCardLimit   = 20
	maxPlanQueries  = 4
	maxPlanEntities = 4
)

func (s *DeepService) Recall(ctx context.Context, req DeepRequest) (DeepResponse, error) {
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		return DeepResponse{}, errors.New("query is required")
	}
	if req.TriggerReason == "" {
		return DeepResponse{}, errors.New("trigger_reason is required for deep recall")
	}
	if req.BudgetChars == 0 {
		req.BudgetChars = defaultBudget
	}
	if req.BudgetChars < minBudget || req.BudgetChars > maxBudget {
		return DeepResponse{}, fmt.Errorf("budget_chars must be between %d and %d", minBudget, maxBudget)
	}

	caps := toCapabilitySet(req.Capabilities)
	tb := newTraceBuilder(req.Query, req.Scope, req.TriggerReason, req.BudgetChars)
	if caps["kg"] {
		tb.trace.SourceSet = append(tb.trace.SourceSet, "kg")
	}
	if caps["timeline"] {
		tb.trace.SourceSet = append(tb.trace.SourceSet, "timeline")
	}
	tb.trace.SourceSet = append(tb.trace.SourceSet, "cards", "governance")

	seedStart := time.Now()
	seed, err := s.Fast.Recall(ctx, FastRequest{
		Query:       req.Query,
		Scope:       req.Scope,
		BudgetChars: req.BudgetChars,
		SessionID:   req.SessionID,
	})
	if err != nil {
		tb.step("fast_seed", "error", time.Since(seedStart), "seed_failed")
		trace := tb.finish(true, "seed_failed: "+err.Error())
		s.saveTrace(ctx, trace)
		return DeepResponse{
			ContextView: ContextView{
				TraceID:     trace.TraceID,
				Scope:       req.Scope,
				Mode:        "deep",
				BudgetChars: req.BudgetChars,
				Degraded:    true,
				Warnings:    []string{"fast recall seed failed: " + err.Error()},
				Cards:       []facade.MemoryCard{},
				Evidence:    []facade.EvidenceFragment{},
			},
			Trace: trace,
		}, nil
	}
	tb.step("fast_seed", "ok", time.Since(seedStart), "")
	for _, card := range seed.Cards {
		tb.trace.CandidateIDs = append(tb.trace.CandidateIDs, card.ID)
	}
	tb.trace.Budget.CardSearches++

	resp := DeepResponse{ContextView: seed}
	resp.Mode = "deep"

	entities := req.Entities
	if req.UsePlanner && s.Planner != nil {
		planStart := time.Now()
		plan, planErr := s.Planner.Plan(ctx, rag.PlannerInput{Intent: req.Query})
		if planErr != nil {
			tb.step("planner", "error", time.Since(planStart), "plan_failed")
			resp.Warnings = append(resp.Warnings, "planner failed: "+planErr.Error())
		} else {
			tb.step("planner", "ok", time.Since(planStart), "")
			entities = mergeEntities(entities, plan.Entities, maxPlanEntities)
			if len(plan.Queries) > 0 && s.Fast.Searcher != nil {
				s.expandCards(ctx, &resp, plan.Queries, req.Scope, tb)
			}
		}
	}

	var assertions []TemporalAssertion

	if caps["kg"] && s.Graph != nil && len(entities) > 0 {
		kgStart := time.Now()
		kgOK := true
		for _, entity := range entities {
			facts, kgErr := s.Graph.QueryEntity(ctx, entity, req.AsOf, "outgoing")
			tb.trace.Budget.KGQueries++
			if kgErr != nil {
				kgOK = false
				resp.Warnings = append(resp.Warnings, fmt.Sprintf("kg query %q failed: %v", entity, kgErr))
				continue
			}
			assertions = append(assertions, assertionsFromFacts(entity, facts)...)
		}
		status := "ok"
		errCode := ""
		if !kgOK {
			status = "error"
			errCode = "kg_partial_failure"
		}
		tb.step("kg_expansion", status, time.Since(kgStart), errCode)
	} else if caps["kg"] && s.Graph == nil {
		tb.step("kg_expansion", "skipped", 0, "graph_unavailable")
		resp.Warnings = append(resp.Warnings, "kg requested but graph source unavailable")
	}

	if caps["timeline"] && s.Graph != nil && len(entities) > 0 {
		tlStart := time.Now()
		tlOK := true
		for _, entity := range entities {
			events, tlErr := s.Graph.Timeline(ctx, entity)
			tb.trace.Budget.TimelineQueries++
			if tlErr != nil {
				tlOK = false
				resp.Warnings = append(resp.Warnings, fmt.Sprintf("timeline %q failed: %v", entity, tlErr))
				continue
			}
			assertions = append(assertions, assertionsFromTimeline(entity, events)...)
		}
		status := "ok"
		errCode := ""
		if !tlOK {
			status = "error"
			errCode = "timeline_partial_failure"
		}
		tb.step("timeline_expansion", status, time.Since(tlStart), errCode)
	} else if caps["timeline"] && s.Graph == nil {
		tb.step("timeline_expansion", "skipped", 0, "graph_unavailable")
		resp.Warnings = append(resp.Warnings, "timeline requested but graph source unavailable")
	}

	if len(assertions) > 0 {
		resp.Assertions = assertions
		if s.Arbiter != nil {
			arbStart := time.Now()
			resp.Proposals = s.Arbiter.DetectConflicts(assertions)
			tb.step("arbiter", "ok", time.Since(arbStart), "")
		}
	}

	for _, ev := range resp.Evidence {
		tb.trace.EvidenceRefs = append(tb.trace.EvidenceRefs, ev.MaterialRef)
	}
	tb.trace.Budget.UsedChars = len([]rune(resp.Context))

	traceID := tb.trace.TraceID
	resp.RecallTraceID = &traceID

	degraded := resp.Degraded || len(resp.Warnings) > 0
	failure := ""
	if degraded && len(resp.Warnings) > 0 {
		failure = strings.Join(resp.Warnings, "; ")
	}
	trace := tb.finish(degraded, failure)
	resp.Trace = trace
	s.saveTrace(ctx, trace)

	return resp, nil
}

func (s *DeepService) expandCards(ctx context.Context, resp *DeepResponse, queries []string, scope string, tb *traceBuilder) {
	if len(queries) > maxPlanQueries {
		queries = queries[:maxPlanQueries]
	}
	expandStart := time.Now()
	seen := map[string]bool{}
	for _, card := range resp.Cards {
		seen[card.ID] = true
	}
	var extra []facade.MemoryCard
	for _, q := range queries {
		page, err := s.Fast.Searcher.SearchCards(ctx, facade.CardQuery{Text: q, Scope: scope, Limit: deepCardLimit})
		tb.trace.Budget.CardSearches++
		if err != nil {
			continue
		}
		for _, card := range page.Cards {
			if !seen[card.ID] {
				seen[card.ID] = true
				extra = append(extra, card)
			}
		}
	}
	if len(extra) > 0 {
		merged := append(resp.Cards, extra...)
		merged = RankCards(merged, scope)
		merged = DeduplicateCards(merged)
		if len(merged) > deepMaxCards {
			merged = merged[:deepMaxCards]
		}
		resp.Cards = merged
		for _, card := range merged {
			tb.trace.CandidateIDs = append(tb.trace.CandidateIDs, card.ID)
		}
	}
	tb.step("card_expansion", "ok", time.Since(expandStart), "")
}

func (s *DeepService) saveTrace(ctx context.Context, trace RecallTrace) {
	if s.Traces != nil {
		_ = s.Traces.Save(ctx, trace)
	}
}

func assertionsFromFacts(subject string, facts []facade.GraphFact) []TemporalAssertion {
	out := make([]TemporalAssertion, 0, len(facts))
	for _, f := range facts {
		out = append(out, TemporalAssertion{
			Subject:    subject,
			Predicate:  f.Predicate,
			Object:     f.Object,
			ValidFrom:  f.ValidFrom,
			ValidTo:    f.ValidTo,
			Confidence: f.Confidence,
			Status:     deriveStatus(f.ValidTo, f.Confidence),
		})
	}
	return out
}

func assertionsFromTimeline(subject string, events []facade.TimelineEvent) []TemporalAssertion {
	out := make([]TemporalAssertion, 0, len(events))
	for _, e := range events {
		out = append(out, TemporalAssertion{
			Subject:    subject,
			Predicate:  e.Predicate,
			Object:     e.Object,
			ValidFrom:  e.ValidFrom,
			ValidTo:    e.ValidTo,
			Confidence: 0.5,
			Status:     deriveStatus(e.ValidTo, 0.5),
		})
	}
	return out
}

func deriveStatus(validTo string, confidence float64) string {
	if confidence < 0.3 {
		return "contextual"
	}
	if validTo != "" {
		return "superseded"
	}
	return "active"
}

func toCapabilitySet(caps []string) map[string]bool {
	m := make(map[string]bool, len(caps))
	for _, c := range caps {
		m[c] = true
	}
	return m
}

func mergeEntities(existing, planned []string, max int) []string {
	seen := map[string]bool{}
	var out []string
	for _, e := range append(existing, planned...) {
		if !seen[e] && len(out) < max {
			seen[e] = true
			out = append(out, e)
		}
	}
	return out
}
