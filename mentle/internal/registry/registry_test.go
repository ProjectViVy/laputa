package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if r == nil {
		t.Fatal("New returned nil")
	}
	if r.Count() != 0 {
		t.Errorf("expected 0 entities, got %d", r.Count())
	}
}

func TestRegistryAddAndLookup(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	e := r.Add("Alice", "person", 0.9, "onboarding")
	if e == nil {
		t.Fatal("Add returned nil")
	}
	if e.Name != "Alice" {
		t.Errorf("expected name 'Alice', got '%s'", e.Name)
	}
	if e.Type != "person" {
		t.Errorf("expected type 'person', got '%s'", e.Type)
	}

	lookup := r.Lookup("Alice")
	if lookup == nil {
		t.Fatal("Lookup returned nil")
	}
	if lookup.Name != "Alice" {
		t.Errorf("expected name 'Alice', got '%s'", lookup.Name)
	}
}

func TestRegistryLookupCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	r.Add("Alice", "person", 0.9, "onboarding")

	// Should find with different case
	e := r.Lookup("alice")
	if e == nil {
		t.Fatal("case-insensitive lookup failed")
	}
	if e.Name != "Alice" {
		t.Errorf("expected name 'Alice', got '%s'", e.Name)
	}
}

func TestRegistryAddAlias(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	r.Add("Alice", "person", 0.9, "onboarding")

	if err := r.AddAlias("Alice", "Alicia"); err != nil {
		t.Fatalf("AddAlias failed: %v", err)
	}

	// Should find by alias
	e := r.Lookup("Alicia")
	if e == nil {
		t.Fatal("lookup by alias failed")
	}
	if e.Name != "Alice" {
		t.Errorf("expected name 'Alice', got '%s'", e.Name)
	}
}

func TestRegistryAddRelationship(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	r.Add("Alice", "person", 0.9, "onboarding")

	if err := r.AddRelationship("Alice", "Bob", "friend"); err != nil {
		t.Fatalf("AddRelationship failed: %v", err)
	}

	e := r.Lookup("Alice")
	if e == nil {
		t.Fatal("Lookup returned nil")
	}
	if rel, ok := e.Relationships["Bob"]; !ok || rel != "friend" {
		t.Errorf("expected relationship Bob=friend, got %v", e.Relationships)
	}
}

func TestRegistryList(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	r.Add("Alice", "person", 0.9, "onboarding")
	r.Add("Bob", "person", 0.8, "onboarding")
	r.Add("ProjectX", "project", 0.7, "learned")

	// List all
	all := r.List("")
	if len(all) != 3 {
		t.Errorf("expected 3 entities, got %d", len(all))
	}

	// Filter by type
	people := r.List("person")
	if len(people) != 2 {
		t.Errorf("expected 2 people, got %d", len(people))
	}

	projects := r.List("project")
	if len(projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(projects))
	}
}

func TestRegistrySaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	// Create and save
	r1, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	r1.Add("Alice", "person", 0.9, "onboarding")
	r1.AddAlias("Alice", "Alicia")

	if err := r1.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load in new instance
	r2, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	e := r2.Lookup("Alice")
	if e == nil {
		t.Fatal("Lookup after load returned nil")
	}
	if len(e.Aliases) != 1 || e.Aliases[0] != "Alicia" {
		t.Errorf("expected alias 'Alicia', got %v", e.Aliases)
	}
}

func TestRegistryAddExistingEntity(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	e1 := r.Add("Alice", "person", 0.9, "onboarding")
	e2 := r.Add("Alice", "person", 0.95, "learned")

	// Should return same entity, update confidence and frequency
	if e1 != e2 {
		t.Error("expected same entity pointer")
	}
	if e2.Frequency != 2 {
		t.Errorf("expected frequency 2, got %d", e2.Frequency)
	}
	if e2.Confidence != 0.95 {
		t.Errorf("expected confidence 0.95, got %f", e2.Confidence)
	}
}

func TestRegistryAddNonexistentEntity(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	err = r.AddAlias("Nonexistent", "alias")
	if err == nil {
		t.Error("expected error for nonexistent entity")
	}

	err = r.AddRelationship("Nonexistent", "Other", "friend")
	if err == nil {
		t.Error("expected error for nonexistent entity")
	}
}

func TestRegistryCount(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if r.Count() != 0 {
		t.Errorf("expected 0, got %d", r.Count())
	}

	r.Add("A", "person", 0.9, "onboarding")
	if r.Count() != 1 {
		t.Errorf("expected 1, got %d", r.Count())
	}

	r.Add("B", "person", 0.8, "onboarding")
	if r.Count() != 2 {
		t.Errorf("expected 2, got %d", r.Count())
	}
}

func TestRegistryLoadNonexistentFile(t *testing.T) {
	dir := t.TempDir()
	r, err := New(filepath.Join(dir, "nonexistent"))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	// Should not error for nonexistent file
	if r.Count() != 0 {
		t.Errorf("expected 0 entities, got %d", r.Count())
	}
}

func TestRegistryPersistence(t *testing.T) {
	dir := t.TempDir()

	// Create and save
	r1, _ := New(dir)
	r1.Add("Alice", "person", 0.9, "onboarding")
	r1.Add("Bob", "person", 0.8, "learned")
	r1.Save()

	// Verify file exists
	path := filepath.Join(dir, "entity_registry.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("registry file not created")
	}

	// Load and verify
	r2, _ := New(dir)
	if r2.Count() != 2 {
		t.Errorf("expected 2 entities after load, got %d", r2.Count())
	}
}
