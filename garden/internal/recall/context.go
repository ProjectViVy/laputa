package recall

import (
	"strings"

	"github.com/dashimaki/garden/internal/personactx"
	"github.com/dashimaki/mentle/facade"
)

func assembleContext(core personactx.FrozenCore, evidence []facade.EvidenceFragment, budget int) string {
	var parts []string
	if frozen := personactx.Render(core, budget); frozen != "" {
		parts = append(parts, frozen)
	}
	for _, item := range evidence {
		if text := strings.TrimSpace(item.Excerpt); text != "" {
			parts = append(parts, text)
		}
	}
	return truncateRunes(strings.Join(parts, "\n\n"), budget)
}

func assembleDegradedContext(core personactx.FrozenCore, budget int) string {
	return assembleContext(core, nil, budget)
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max-1]) + "…"
}
