package rag

import (
	"context"
	"fmt"

	"github.com/dashimaki/laputa/governance"
)

type GovernanceReader interface {
	GetSection(context.Context, governance.SectionName) (map[string]any, error)
}

type PolicyResolver struct{ Governance GovernanceReader }

var governanceSections = []governance.SectionName{
	governance.SectionIdentity, governance.SectionCommitment, governance.SectionPreferences,
	governance.SectionMemoryMD, governance.SectionHistoryMD, governance.SectionDaily,
	governance.SectionWeekly, governance.SectionMonthly,
}

func (r PolicyResolver) Load(ctx context.Context) (map[string]map[string]any, error) {
	if r.Governance == nil {
		return nil, fmt.Errorf("governance unavailable")
	}
	out := map[string]map[string]any{}
	for _, section := range governanceSections {
		data, err := r.Governance.GetSection(ctx, section)
		if err != nil {
			return nil, err
		}
		out[string(section)] = data
	}
	return out, nil
}

func ResolvePolicy(sections map[string]map[string]any) Policy {
	p := Policy{AllowedCapabilities: stringBoolSet([]string{"governance", "llm", "hybrid", "kg", "timeline"}), DeniedSources: map[string]bool{}, DeniedWings: map[string]bool{}, DeniedRooms: map[string]bool{}, PreferredWings: map[string]bool{}, ExternalContext: "full"}
	identity := sections[string(governance.SectionIdentity)]
	if cfg := nestedMap(identity, "agentic_rag"); cfg != nil {
		if values := stringsFrom(cfg["allowed_capabilities"]); len(values) > 0 {
			p.AllowedCapabilities = stringBoolSet(values)
		}
	}
	commitment := sections[string(governance.SectionCommitment)]
	if cfg := nestedMap(commitment, "agentic_rag"); cfg != nil {
		p.DeniedSources = stringBoolSet(stringsFrom(cfg["denied_sources"]))
		p.DeniedWings = stringBoolSet(stringsFrom(cfg["denied_wings"]))
		p.DeniedRooms = stringBoolSet(stringsFrom(cfg["denied_rooms"]))
		if value, ok := cfg["external_context"].(string); ok && value != "" {
			p.ExternalContext = value
		}
	}
	preferences := sections[string(governance.SectionPreferences)]
	if cfg := nestedMap(preferences, "agentic_rag"); cfg != nil {
		p.PreferredWings = stringBoolSet(stringsFrom(cfg["preferred_wings"]))
	}
	return p
}

func nestedMap(root map[string]any, key string) map[string]any {
	if root == nil {
		return nil
	}
	value, _ := root[key].(map[string]any)
	return value
}
func stringsFrom(value any) []string {
	switch values := value.(type) {
	case []string:
		return values
	case []any:
		out := []string{}
		for _, value := range values {
			if s, ok := value.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
func stringBoolSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		out[value] = true
	}
	return out
}
