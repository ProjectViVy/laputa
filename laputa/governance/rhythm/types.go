package rhythm

import (
	"context"
	"time"
)

// RhythmKind represents the reporting cadence.
type RhythmKind string

const (
	RhythmDaily   RhythmKind = "daily"
	RhythmWeekly  RhythmKind = "weekly"
	RhythmMonthly RhythmKind = "monthly"
)

// ReportResult is what the LLM produces for a rhythm run.
type ReportResult struct {
	Title         string    `json:"title"`
	Summary       string    `json:"summary"`
	Highlights    []string  `json:"highlights"`
	OpenQuestions []string  `json:"open_questions,omitempty"`
	GeneratedAt   time.Time `json:"generated_at"`
}

// Config holds rhythm engine configuration.
type Config struct {
	BaseURL string
	APIKey  string
	Model   string
}

// ArtifactResult is the optional LLM enrichment of a report artifact
// (ADR-0005 §5): the four narrative fields a deterministic generator
// cannot derive on its own.
type ArtifactResult struct {
	Goals     []string `json:"goals"`
	Completed []string `json:"completed"`
	Decisions []string `json:"decisions"`
	OpenLoops []string `json:"open_loops"`
}

// ArtifactGenerator enriches a report prompt into artifact fields.
type ArtifactGenerator interface {
	GenerateArtifact(ctx context.Context, kind RhythmKind, prompt string) (*ArtifactResult, error)
}
