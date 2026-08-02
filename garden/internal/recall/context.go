package recall

import (
	"strings"

	"github.com/dashimaki/garden/internal/authority"
	"github.com/dashimaki/mentle/facade"
)

func assembleContext(evidence []facade.EvidenceFragment, proj authority.GovernanceProjection, budget int) string {
	var sb strings.Builder
	for _, ev := range evidence {
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(ev.Excerpt)
		if sb.Len() >= budget {
			break
		}
	}
	if sb.Len() == 0 {
		return governanceContext(proj, budget)
	}
	return truncateRunes(sb.String(), budget)
}

func governanceContext(proj authority.GovernanceProjection, budget int) string {
	var sb strings.Builder
	sb.WriteString("governance projection: ")
	sb.WriteString(proj.IdentityRef)
	if len(proj.AllowedKinds) > 0 {
		sb.WriteString("\nallowed: ")
		sb.WriteString(strings.Join(proj.AllowedKinds, ", "))
	}
	if len(proj.DeniedSources) > 0 {
		sb.WriteString("\ndenied: ")
		sb.WriteString(strings.Join(proj.DeniedSources, ", "))
	}
	return truncateRunes(sb.String(), budget)
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 0 {
		return ""
	}
	return string(r[:max-1]) + "…"
}
