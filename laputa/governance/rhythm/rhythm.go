package rhythm

import (
	"context"
	"fmt"
	"time"

	laputa "github.com/dashimaki/laputa/governance"
)

// Engine runs rhythm reports against Laputa.
type Engine struct {
	laputa    *laputa.Engine
	generator Generator
	config    Config
}

// NewEngine creates a rhythm engine.
func NewEngine(laputaEngine *laputa.Engine, generator Generator, config Config) *Engine {
	return &Engine{
		laputa:    laputaEngine,
		generator: generator,
		config:    config,
	}
}

// Run executes one rhythm cycle and writes the report to Laputa.
func (e *Engine) Run(ctx context.Context, kind RhythmKind) error {
	snapshot, err := e.laputa.Snapshot(ctx)
	if err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}

	prompt := BuildPrompt(kind, snapshot)
	report, err := e.generator.Generate(ctx, kind, prompt)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	var target laputa.SectionName
	switch kind {
	case RhythmDaily:
		target = laputa.SectionDaily
	case RhythmWeekly:
		target = laputa.SectionWeekly
	case RhythmMonthly:
		target = laputa.SectionMonthly
	default:
		return fmt.Errorf("unknown rhythm kind: %s", kind)
	}

	section, err := e.laputa.GetSection(ctx, target)
	if err != nil {
		return fmt.Errorf("read target section: %w", err)
	}

	reports, _ := section["reports"].([]any)
	entry := map[string]any{
		"title":          report.Title,
		"summary":        report.Summary,
		"highlights":     report.Highlights,
		"open_questions": report.OpenQuestions,
		"generated_at":   report.GeneratedAt.Format(time.RFC3339),
	}
	section["reports"] = append(reports, entry)

	if err := e.laputa.SetSection(ctx, target, section); err != nil {
		return fmt.Errorf("write target section: %w", err)
	}
	return nil
}
