package authority

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/dashimaki/laputa/governance"
)

type GovernanceProjection struct {
	IdentityRef       string   `json:"identity_ref"`
	Scope             string   `json:"scope"`
	AllowedSources    []string `json:"allowed_sources"`
	DeniedSources     []string `json:"denied_sources"`
	AllowedKinds      []string `json:"allowed_kinds"`
	ActiveLTMRefs     []string `json:"active_ltm_refs"`
	WorkingSetRefs    []string `json:"working_set_refs"`
	PolicyRevision    string   `json:"policy_revision"`
	ProjectionVersion string   `json:"projection_version"`
}

type GovernanceReader interface {
	GetSection(context.Context, governance.SectionName) (map[string]any, error)
}

var projectionSections = []governance.SectionName{
	governance.SectionIdentity,
	governance.SectionCommitment,
	governance.SectionPreferences,
	governance.SectionMemoryMD,
}

func BuildProjection(ctx context.Context, gov GovernanceReader) (GovernanceProjection, error) {
	if gov == nil {
		return GovernanceProjection{}, fmt.Errorf("governance unavailable")
	}
	sections := make(map[string]map[string]any, len(projectionSections))
	for _, name := range projectionSections {
		data, err := gov.GetSection(ctx, name)
		if err != nil {
			return GovernanceProjection{}, fmt.Errorf("section %s: %w", name, err)
		}
		sections[string(name)] = data
	}

	proj := GovernanceProjection{ProjectionVersion: "1"}
	proj.IdentityRef = identityRef(sections["01-identity"])
	proj.DeniedSources = deniedSources(sections["03-commitment"])
	proj.AllowedKinds = allowedKinds(sections["01-identity"])
	proj.WorkingSetRefs = refsFrom(sections["05-memory_md"])
	proj.PolicyRevision = policyRevision(sections)
	return proj, nil
}

func identityRef(identity map[string]any) string {
	if meta := nestedMap(identity, "_meta"); meta != nil {
		if v, ok := meta["version"].(string); ok && v != "" {
			return v
		}
	}
	raw, _ := json.Marshal(identity)
	h := sha256.Sum256(raw)
	return fmt.Sprintf("%x", h[:8])
}

func deniedSources(commitment map[string]any) []string {
	cfg := nestedMap(commitment, "agentic_rag")
	if cfg == nil {
		return nil
	}
	return stringsFrom(cfg["denied_sources"])
}

func allowedKinds(identity map[string]any) []string {
	cfg := nestedMap(identity, "agentic_rag")
	if cfg == nil {
		return nil
	}
	return stringsFrom(cfg["allowed_capabilities"])
}

func refsFrom(section map[string]any) []string {
	refs := []string{}
	if items := stringsFrom(section["refs"]); len(items) > 0 {
		return items
	}
	if items := stringsFrom(section["items"]); len(items) > 0 {
		return items
	}
	return refs
}

func policyRevision(sections map[string]map[string]any) string {
	var max string
	for _, data := range sections {
		meta := nestedMap(data, "_meta")
		if meta == nil {
			continue
		}
		updated, _ := meta["updated_at"].(string)
		if updated > max {
			max = updated
		}
	}
	return max
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
		for _, v := range values {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
