package authority

import (
	"context"
	"testing"

	"github.com/dashimaki/laputa/governance"
)

type fakeGov struct {
	sections map[governance.SectionName]map[string]any
}

func (f *fakeGov) GetSection(_ context.Context, name governance.SectionName) (map[string]any, error) {
	if data, ok := f.sections[name]; ok {
		return data, nil
	}
	return map[string]any{}, nil
}

func TestBuildProjectionFromSections(t *testing.T) {
	gov := &fakeGov{sections: map[governance.SectionName]map[string]any{
		governance.SectionIdentity: {
			"_meta":       map[string]any{"version": "id-v3"},
			"agentic_rag": map[string]any{"allowed_capabilities": []any{"fact", "decision"}},
		},
		governance.SectionCommitment: {
			"agentic_rag": map[string]any{"denied_sources": []any{"untrusted"}},
		},
		governance.SectionPreferences: {},
		governance.SectionMemoryMD: {
			"_meta": map[string]any{"updated_at": "2026-08-01T00:00:00Z"},
			"refs":  []any{"mem_abc", "mem_def"},
		},
	}}

	proj, err := BuildProjection(context.Background(), gov)
	if err != nil {
		t.Fatal(err)
	}
	if proj.IdentityRef != "id-v3" {
		t.Fatalf("IdentityRef=%q", proj.IdentityRef)
	}
	if len(proj.DeniedSources) != 1 || proj.DeniedSources[0] != "untrusted" {
		t.Fatalf("DeniedSources=%v", proj.DeniedSources)
	}
	if len(proj.AllowedKinds) != 2 {
		t.Fatalf("AllowedKinds=%v", proj.AllowedKinds)
	}
	if len(proj.WorkingSetRefs) != 2 {
		t.Fatalf("WorkingSetRefs=%v", proj.WorkingSetRefs)
	}
	if len(proj.ActiveLTMRefs) != 0 {
		t.Fatalf("ActiveLTMRefs should be empty (ADR-0002), got %v", proj.ActiveLTMRefs)
	}
	if proj.PolicyRevision != "2026-08-01T00:00:00Z" {
		t.Fatalf("PolicyRevision=%q", proj.PolicyRevision)
	}
	if proj.ProjectionVersion != "1" {
		t.Fatalf("ProjectionVersion=%q", proj.ProjectionVersion)
	}
}

func TestBuildProjectionEmptySections(t *testing.T) {
	gov := &fakeGov{sections: map[governance.SectionName]map[string]any{
		governance.SectionIdentity:    {},
		governance.SectionCommitment:  {},
		governance.SectionPreferences: {},
		governance.SectionMemoryMD:    {},
	}}

	proj, err := BuildProjection(context.Background(), gov)
	if err != nil {
		t.Fatal(err)
	}
	if proj.IdentityRef == "" {
		t.Fatal("IdentityRef should be a hash fallback")
	}
	if len(proj.DeniedSources) != 0 {
		t.Fatalf("DeniedSources=%v", proj.DeniedSources)
	}
	if proj.ProjectionVersion != "1" {
		t.Fatalf("ProjectionVersion=%q", proj.ProjectionVersion)
	}
}

func TestBuildProjectionNilGov(t *testing.T) {
	_, err := BuildProjection(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil governance")
	}
}
