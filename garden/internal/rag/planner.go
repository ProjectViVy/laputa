package rag

import (
	"context"
	"fmt"
	"strings"
)

func extractiveContext(intent string, evidence []Evidence, maxTokens int) string {
	var b strings.Builder
	b.WriteString("Intent: " + intent + "\n\nRelevant evidence:\n")
	limit := maxTokens * 4
	for _, item := range evidence {
		line := fmt.Sprintf("- [%s] (%s: %s) %s\n", item.ID, item.Source, item.Locator, item.Excerpt)
		if b.Len()+len(line) > limit {
			break
		}
		b.WriteString(line)
	}
	return b.String()
}

func validCitations(value string, evidence []Evidence) bool {
	if len(evidence) == 0 {
		return true
	}
	valid := map[string]bool{}
	for _, item := range evidence {
		valid[item.ID] = true
	}
	found := false
	for start := 0; start < len(value); {
		i := strings.Index(value[start:], "[ev_")
		if i < 0 {
			break
		}
		i += start
		j := strings.Index(value[i:], "]")
		if j < 0 {
			return false
		}
		id := value[i+1 : i+j]
		if !valid[id] {
			return false
		}
		found = true
		start = i + j + 1
	}
	return found
}

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
