package rhythm

import "time"

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
