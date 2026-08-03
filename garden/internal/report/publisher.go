package report

import (
	"context"
	"fmt"

	"github.com/dashimaki/laputa/governance"
)

// maxSectionReports caps the canonical section reports array; oldest entries drop first.
const maxSectionReports = 200

// Publisher writes a freshly generated report to its canonical store.
type Publisher interface {
	Publish(ctx context.Context, r Report) error
}

// GovernedGateway is the governed-section surface GovernedPublisher needs.
type GovernedGateway interface {
	GetSection(ctx context.Context, section governance.SectionName) (map[string]any, error)
	Mutate(ctx context.Context, req governance.MutationRequest) error
}

// GovernedPublisher publishes reports into sections 07/08/09 as report_system.
type GovernedPublisher struct {
	Gov GovernedGateway
}

func sectionForCadence(cadence string) (governance.SectionName, error) {
	switch cadence {
	case "daily":
		return governance.SectionDaily, nil
	case "weekly":
		return governance.SectionWeekly, nil
	case "monthly":
		return governance.SectionMonthly, nil
	default:
		return "", fmt.Errorf("report: unknown cadence %q", cadence)
	}
}

func (p *GovernedPublisher) Publish(ctx context.Context, r Report) error {
	section, err := sectionForCadence(r.Cadence)
	if err != nil {
		return err
	}
	data, err := p.Gov.GetSection(ctx, section)
	if err != nil {
		return fmt.Errorf("read section %s: %w", section, err)
	}
	reports, _ := data["reports"].([]any)
	for _, item := range reports {
		if entry, ok := item.(map[string]any); ok && entry["source_hash"] == r.SourceHash {
			return nil
		}
	}
	entry := map[string]any{
		"title":          r.Title,
		"summary":        r.Summary,
		"highlights":     toAnyList(r.Highlights),
		"open_questions": toAnyList(r.OpenQuestions),
		"scope":          r.Scope,
		"goals":          toAnyList(r.Goals),
		"completed":      toAnyList(r.Completed),
		"decisions":      toAnyList(r.Decisions),
		"open_loops":     toAnyList(r.OpenLoops),
		"source_refs":    toAnyList(r.SourceRefs),
		"source_hash":    r.SourceHash,
		"revision":       r.Revision,
		"generator":      r.Generator,
		"window_start":   r.WindowStart.Format("2006-01-02T15:04:05Z07:00"),
		"window_end":     r.WindowEnd.Format("2006-01-02T15:04:05Z07:00"),
		"generated_at":   r.GeneratedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	reports = append(reports, entry)
	if len(reports) > maxSectionReports {
		reports = reports[len(reports)-maxSectionReports:]
	}
	data["reports"] = reports
	return p.Gov.Mutate(ctx, governance.MutationRequest{
		Section: section,
		Action:  "write",
		Actor:   governance.ActorReportSystem,
		Reason:  "report publication",
		Data:    data,
	})
}

func toAnyList(items []string) []any {
	out := make([]any, len(items))
	for i, v := range items {
		out[i] = v
	}
	return out
}
