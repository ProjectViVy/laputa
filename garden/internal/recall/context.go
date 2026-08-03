package recall

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dashimaki/garden/internal/authority"
	"github.com/dashimaki/laputa/governance/cognitive"
	"github.com/dashimaki/mentle/facade"
)

func assembleContext(evidence []facade.EvidenceFragment, world []cognitive.WorldClaim, proj authority.GovernanceProjection, budget int) string {
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
		return assembleDegradedContext(world, proj, budget)
	}
	if wc := worldContext(world); wc != "" {
		sb.WriteString("\n\n")
		sb.WriteString(wc)
	}
	return truncateRunes(sb.String(), budget)
}

func assembleDegradedContext(world []cognitive.WorldClaim, proj authority.GovernanceProjection, budget int) string {
	gov := governanceContext(proj, budget)
	wc := worldContext(world)
	if wc == "" {
		return gov
	}
	return truncateRunes(gov+"\n\n"+wc, budget)
}

func worldContext(claims []cognitive.WorldClaim) string {
	if len(claims) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("world projection:")
	for _, claim := range claims {
		sb.WriteString(fmt.Sprintf("\n[%s] %s: %s", claim.Domain, claim.Title, claim.Text))
	}
	return sb.String()
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
	if len(proj.WorkingSetRefs) > 0 {
		sb.WriteString("\nworking set: ")
		sb.WriteString(strings.Join(proj.WorkingSetRefs, ", "))
	}
	if len(proj.FrozenRefs) > 0 {
		names := make([]string, 0, len(proj.FrozenRefs))
		for name := range proj.FrozenRefs {
			names = append(names, name)
		}
		sort.Strings(names)
		parts := make([]string, 0, len(names))
		for _, name := range names {
			parts = append(parts, name+"="+proj.FrozenRefs[name])
		}
		sb.WriteString("\nfrozen core: ")
		sb.WriteString(strings.Join(parts, " "))
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
