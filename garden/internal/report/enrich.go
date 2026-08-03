package report

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dashimaki/laputa/governance/rhythm"
)

const defaultEnrichTimeout = 30 * time.Second

// Enricher upgrades a deterministic report artifact before persistence.
type Enricher interface {
	Enrich(ctx context.Context, r Report) (Report, error)
}

// LLMEnricher fills goals/completed/decisions/open_loops via an LLM.
// Any failure leaves the deterministic artifact untouched (ADR-0005 §5).
type LLMEnricher struct {
	Gen     rhythm.ArtifactGenerator
	Timeout time.Duration
}

func (e *LLMEnricher) Enrich(ctx context.Context, r Report) (Report, error) {
	timeout := e.Timeout
	if timeout <= 0 {
		timeout = defaultEnrichTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var kind rhythm.RhythmKind
	switch r.Cadence {
	case "daily":
		kind = rhythm.RhythmDaily
	case "weekly":
		kind = rhythm.RhythmWeekly
	case "monthly":
		kind = rhythm.RhythmMonthly
	default:
		return r, fmt.Errorf("report: unknown cadence %q", r.Cadence)
	}
	res, err := e.Gen.GenerateArtifact(ctx, kind, buildArtifactPrompt(r))
	if err != nil {
		return r, err
	}
	out := r
	out.Goals = boundList(res.Goals)
	out.Completed = firstNonEmpty(boundList(res.Completed), r.Completed)
	out.Decisions = firstNonEmpty(boundList(res.Decisions), r.Decisions)
	out.OpenLoops = boundList(res.OpenLoops)
	out.Generator = GeneratorLLM
	return out, nil
}

func buildArtifactPrompt(r Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Summarize this %s report window into JSON with exactly four string-array keys: goals, completed, decisions, open_loops.\n", r.Cadence)
	b.WriteString("Each array holds at most 10 short items (<=240 characters). Respond with raw JSON only.\n\n")
	fmt.Fprintf(&b, "Window: %s to %s\n", r.WindowStart.Format(time.RFC3339), r.WindowEnd.Format(time.RFC3339))
	fmt.Fprintf(&b, "Source memories: %d\n\n", len(r.SourceIDs))
	if r.Summary != "" {
		b.WriteString("Deterministic summary:\n")
		b.WriteString(r.Summary)
		b.WriteString("\n")
	}
	if len(r.Decisions) > 0 {
		b.WriteString("Known decisions:\n")
		for _, d := range r.Decisions {
			b.WriteString("- " + d + "\n")
		}
	}
	return b.String()
}

func boundList(items []string) []string {
	out := []string{}
	for _, item := range items {
		if len(out) >= 10 {
			break
		}
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, truncate(item, 240))
	}
	return out
}

func firstNonEmpty(primary, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}
