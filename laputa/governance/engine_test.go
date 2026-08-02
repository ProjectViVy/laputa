package governance

import (
	"context"
	"testing"
)

func TestFileStore(t *testing.T) {
	ctx := context.Background()
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	// Test Write + Read
	data := map[string]any{
		"role":         "assistant",
		"capabilities": []string{"coding", "analysis"},
	}
	if err := store.Write(ctx, SectionIdentity, data); err != nil {
		t.Fatalf("write: %v", err)
	}

	read, err := store.Read(ctx, SectionIdentity)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read["role"] != "assistant" {
		t.Errorf("role mismatch: got %v, want assistant", read["role"])
	}

	// Test List
	sections, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sections) != 1 || sections[0] != SectionIdentity {
		t.Errorf("list mismatch: got %v", sections)
	}

	// Test Exists
	exists, err := store.Exists(ctx, SectionIdentity)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !exists {
		t.Error("expected section to exist")
	}

	// Test Patch
	if err := store.Patch(ctx, SectionIdentity, "constraints.no_harm", true); err != nil {
		t.Fatalf("patch: %v", err)
	}
	read, _ = store.Read(ctx, SectionIdentity)
	constraints, ok := read["constraints"].(map[string]any)
	if !ok || constraints["no_harm"] != true {
		t.Errorf("patch failed: got %v", read["constraints"])
	}

	// Test Delete
	if err := store.Delete(ctx, SectionIdentity, "constraints.no_harm"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	read, _ = store.Read(ctx, SectionIdentity)
	constraints, ok = read["constraints"].(map[string]any)
	if ok && constraints["no_harm"] != nil {
		t.Errorf("delete failed: got %v", constraints)
	}
}

func TestEngine(t *testing.T) {
	ctx := context.Background()
	store, _ := NewFileStore(t.TempDir())
	engine := NewEngine(store)

	// Initialize
	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	// Check all sections created
	sections, _ := engine.ListSections(ctx)
	if len(sections) != len(AllSections) {
		t.Errorf("expected %d sections, got %d", len(AllSections), len(sections))
	}

	// GetSection
	identity, err := engine.GetSection(ctx, SectionIdentity)
	if err != nil {
		t.Fatalf("get section: %v", err)
	}
	if identity["role"] == nil {
		t.Error("expected role field in identity section")
	}

	// SetSection
	if err := engine.SetSection(ctx, SectionIdentity, map[string]any{
		"role":         "coder",
		"capabilities": []string{"go", "rust"},
	}); err != nil {
		t.Fatalf("set section: %v", err)
	}
	identity, _ = engine.GetSection(ctx, SectionIdentity)
	if identity["role"] != "coder" {
		t.Errorf("role mismatch: got %v", identity["role"])
	}

	// UpdateSection
	if err := engine.UpdateSection(ctx, SectionIdentity, "role", "architect"); err != nil {
		t.Fatalf("update section: %v", err)
	}
	identity, _ = engine.GetSection(ctx, SectionIdentity)
	if identity["role"] != "architect" {
		t.Errorf("role mismatch after update: got %v", identity["role"])
	}

	// Snapshot
	snapshot, err := engine.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snapshot["schema_version"] != "1.0.0" {
		t.Errorf("schema_version mismatch: got %v", snapshot["schema_version"])
	}
	sectionsMap, ok := snapshot["sections"].(map[string]any)
	if !ok || len(sectionsMap) != len(AllSections) {
		t.Errorf("snapshot sections mismatch: got %d", len(sectionsMap))
	}
}

func TestDefaultSectionData(t *testing.T) {
	for _, section := range AllSections {
		data := defaultSectionData(section)
		if len(data) == 0 {
			t.Errorf("section %s has empty default data", section)
		}
	}
}

func TestSectionRegistry(t *testing.T) {
	for _, section := range AllSections {
		info, ok := SectionRegistry[section]
		if !ok {
			t.Errorf("section %s missing from registry", section)
			continue
		}
		if info.Name != section {
			t.Errorf("registry name mismatch for %s", section)
		}
		if info.Status != "stable" && info.Status != "tbd" {
			t.Errorf("section %s has invalid status: %s", section, info.Status)
		}
	}
}
