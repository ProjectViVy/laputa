package rag

import "context"

type ResolveRequest struct {
	Intent    string `json:"intent"`
	SessionID string `json:"session_id,omitempty"`
	Mode      string `json:"mode,omitempty"`
}

type Evidence struct {
	ID      string  `json:"id"`
	Source  string  `json:"source"`
	Locator string  `json:"locator"`
	Excerpt string  `json:"excerpt"`
	Score   float64 `json:"score"`
}

type ContextPackage struct {
	TraceID    string     `json:"trace_id"`
	Mode       string     `json:"mode"`
	Context    string     `json:"context"`
	Evidence   []Evidence `json:"evidence"`
	Confidence float64    `json:"confidence"`
	Degraded   bool       `json:"degraded"`
	Warnings   []string   `json:"warnings"`
}

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

type Policy struct {
	AllowedCapabilities map[string]bool
	DeniedSources       map[string]bool
	DeniedWings         map[string]bool
	DeniedRooms         map[string]bool
	PreferredWings      map[string]bool
	ExternalContext     string
}

type Resolver interface {
	Resolve(context.Context, ResolveRequest) (ContextPackage, error)
}

// vNext card/evidence DTOs (Wave 1). Used by the recall package in Wave 2.

type MemoryCard struct {
	ID             string   `json:"id"`
	Kind           string   `json:"kind"`
	Collection     string   `json:"collection"`
	Scope          string   `json:"scope"`
	Title          string   `json:"title"`
	Summary        string   `json:"summary"`
	SourceRef      string   `json:"source_ref"`
	Revision       int      `json:"revision"`
	Status         string   `json:"status"`
	Tags           []string `json:"tags"`
	HeatScore      float64  `json:"heat_score"`
	CandidateScore float64  `json:"candidate_score"`
}

type EvidenceFragment struct {
	CardID       string   `json:"card_id"`
	MaterialRef  string   `json:"material_ref"`
	SourceURI    string   `json:"source_uri,omitempty"`
	Excerpt      string   `json:"excerpt"`
	StartOffset  int      `json:"start_offset"`
	EndOffset    int      `json:"end_offset"`
	ContentHash  string   `json:"content_hash"`
	Validity     string   `json:"validity"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}
