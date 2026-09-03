package report

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const defaultEnrichTimeout = 30 * time.Second

type Enricher interface {
	Enrich(context.Context, Report) (Report, error)
}

// ArtifactResult and ArtifactGenerator are Garden-owned boundaries for an
// optional report enrichment provider. They carry no Persona authority data.
type ArtifactResult struct {
	Goals     []string `json:"goals"`
	Completed []string `json:"completed"`
	Decisions []string `json:"decisions"`
	OpenLoops []string `json:"open_loops"`
}

type ArtifactGenerator interface {
	GenerateArtifact(context.Context, string, string) (*ArtifactResult, error)
}

type LLMEnricher struct {
	Gen     ArtifactGenerator
	Timeout time.Duration
}

func (e *LLMEnricher) Enrich(ctx context.Context, r Report) (Report, error) {
	timeout := e.Timeout
	if timeout <= 0 {
		timeout = defaultEnrichTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if r.Cadence != "daily" && r.Cadence != "weekly" && r.Cadence != "monthly" {
		return r, fmt.Errorf("report: unknown cadence %q", r.Cadence)
	}
	result, err := e.Gen.GenerateArtifact(ctx, r.Cadence, buildArtifactPrompt(r))
	if err != nil {
		return r, err
	}
	if result == nil {
		return r, fmt.Errorf("report: enrichment returned no result")
	}
	out := r
	out.Goals = boundList(result.Goals)
	out.Completed = firstNonEmpty(boundList(result.Completed), r.Completed)
	out.Decisions = firstNonEmpty(boundList(result.Decisions), r.Decisions)
	out.OpenLoops = boundList(result.OpenLoops)
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
		for _, decision := range r.Decisions {
			b.WriteString("- " + decision + "\n")
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
		if item != "" {
			out = append(out, truncate(item, 240))
		}
	}
	return out
}

func firstNonEmpty(primary, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}
