package rag

import (
	"context"
	"fmt"
	"strings"
)

type RulePlanner struct{}

func (RulePlanner) Plan(_ context.Context, input PlannerInput) (RetrievalPlan, error) {
	if strings.TrimSpace(input.Intent) == "" {
		return RetrievalPlan{}, fmt.Errorf("intent is required")
	}
	return RetrievalPlan{Queries: []string{strings.TrimSpace(input.Intent)}}, nil
}
func (RulePlanner) Refine(_ context.Context, input PlannerInput) (RetrievalPlan, error) {
	return RetrievalPlan{Stop: true}, nil
}
func (RulePlanner) Summarize(_ context.Context, intent string, evidence []Evidence) (string, error) {
	return extractiveContext(intent, evidence, 4000), nil
}

type FallbackPlanner struct {
	Primary  Planner
	Fallback Planner
}

func (p FallbackPlanner) Plan(ctx context.Context, input PlannerInput) (RetrievalPlan, error) {
	if p.Primary != nil {
		if plan, err := p.Primary.Plan(ctx, input); err == nil {
			return sanitizePlan(plan, input.Intent), nil
		} else {
			fallback, fallbackErr := p.fallback().Plan(ctx, input)
			if fallbackErr != nil {
				return RetrievalPlan{}, fallbackErr
			}
			return sanitizePlan(fallback, input.Intent), fmt.Errorf("primary planner unavailable: %w", err)
		}
	}
	return p.fallback().Plan(ctx, input)
}
func (p FallbackPlanner) Refine(ctx context.Context, input PlannerInput) (RetrievalPlan, error) {
	if p.Primary != nil {
		if plan, err := p.Primary.Refine(ctx, input); err == nil {
			return sanitizePlan(plan, input.Intent), nil
		} else {
			fallback, fallbackErr := p.fallback().Refine(ctx, input)
			if fallbackErr != nil {
				return RetrievalPlan{}, fallbackErr
			}
			return fallback, fmt.Errorf("primary planner unavailable: %w", err)
		}
	}
	return p.fallback().Refine(ctx, input)
}
func (p FallbackPlanner) Summarize(ctx context.Context, intent string, evidence []Evidence) (string, error) {
	if p.Primary != nil {
		if value, err := p.Primary.Summarize(ctx, intent, evidence); err == nil && validCitations(value, evidence) {
			return value, nil
		} else {
			fallback, fallbackErr := p.fallback().Summarize(ctx, intent, evidence)
			if fallbackErr != nil {
				return "", fallbackErr
			}
			if err != nil {
				return fallback, fmt.Errorf("primary planner unavailable: %w", err)
			}
			return fallback, fmt.Errorf("primary planner returned invalid citations")
		}
	}
	return p.fallback().Summarize(ctx, intent, evidence)
}
func (p FallbackPlanner) fallback() Planner {
	if p.Fallback != nil {
		return p.Fallback
	}
	return RulePlanner{}
}

func sanitizePlan(plan RetrievalPlan, intent string) RetrievalPlan {
	seen := map[string]bool{}
	queries := []string{}
	for _, query := range plan.Queries {
		query = strings.TrimSpace(query)
		if query != "" && !seen[query] && len(queries) < 4 {
			queries = append(queries, query)
			seen[query] = true
		}
	}
	if len(queries) == 0 && !plan.Stop {
		queries = []string{intent}
	}
	plan.Queries = queries
	if len(plan.Entities) > 4 {
		plan.Entities = plan.Entities[:4]
	}
	return plan
}
