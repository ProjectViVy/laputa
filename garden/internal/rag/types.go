package rag

import "context"

type RetrievalPlan struct {
	Queries  []string `json:"queries"`
	Entities []string `json:"entities,omitempty"`
	Temporal bool     `json:"temporal,omitempty"`
	Stop     bool     `json:"stop,omitempty"`
}

type PlannerInput struct {
	Intent     string
	Governance map[string]map[string]any
	Candidates []Candidate
	Prior      RetrievalPlan
}

type Planner interface {
	Plan(context.Context, PlannerInput) (RetrievalPlan, error)
	Refine(context.Context, PlannerInput) (RetrievalPlan, error)
	Summarize(context.Context, string, []Evidence) (string, error)
}

type Candidate struct {
	Source   string
	Locator  string
	Content  string
	Score    float64
	Channels []string
}

type Evidence struct {
	ID      string  `json:"id"`
	Source  string  `json:"source"`
	Locator string  `json:"locator"`
	Excerpt string  `json:"excerpt"`
	Score   float64 `json:"score"`
}
