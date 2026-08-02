package room

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRoomsFromYAML(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "mempalace.yaml")

	content := `project_name: test-project
rooms:
  - name: frontend
    keywords:
      - ui
      - components
      - views
      - widgets
  - name: backend
    keywords:
      - api
      - server
      - handlers
      - controllers
  - name: general
    keywords: []
`
	if err := os.WriteFile(yamlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write yaml: %v", err)
	}

	rooms, err := LoadRoomsFromYAML(yamlPath)
	if err != nil {
		t.Fatalf("LoadRoomsFromYAML() error = %v", err)
	}

	if len(rooms) != 3 {
		t.Fatalf("expected 3 rooms, got %d", len(rooms))
	}

	if rooms[0].Name != "frontend" {
		t.Errorf("rooms[0].Name = %q, want %q", rooms[0].Name, "frontend")
	}
	if len(rooms[0].Keywords) != 4 {
		t.Errorf("rooms[0].Keywords length = %d, want 4", len(rooms[0].Keywords))
	}
}

func TestLoadRoomsFromYAML_FileNotFound(t *testing.T) {
	_, err := LoadRoomsFromYAML("/nonexistent/mempalace.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadRoomsFromYAML_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "mempalace.yaml")

	if err := os.WriteFile(yamlPath, []byte(":\n  invalid: [yaml: content"), 0644); err != nil {
		t.Fatalf("failed to write yaml: %v", err)
	}

	_, err := LoadRoomsFromYAML(yamlPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadRoomsFromYAML_NoRoomsKey(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "mempalace.yaml")

	content := `project_name: test-project
`
	if err := os.WriteFile(yamlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write yaml: %v", err)
	}

	rooms, err := LoadRoomsFromYAML(yamlPath)
	if err != nil {
		t.Fatalf("LoadRoomsFromYAML() error = %v", err)
	}

	if len(rooms) != 0 {
		t.Fatalf("expected 0 rooms when no rooms key, got %d", len(rooms))
	}
}

func TestLoadRoomsFromYAML_EmptyKeywords(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "mempalace.yaml")

	content := `rooms:
  - name: general
`
	if err := os.WriteFile(yamlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write yaml: %v", err)
	}

	rooms, err := LoadRoomsFromYAML(yamlPath)
	if err != nil {
		t.Fatalf("LoadRoomsFromYAML() error = %v", err)
	}

	if len(rooms) != 1 {
		t.Fatalf("expected 1 room, got %d", len(rooms))
	}
	if rooms[0].Name != "general" {
		t.Errorf("rooms[0].Name = %q, want %q", rooms[0].Name, "general")
	}
	if rooms[0].Keywords != nil {
		t.Errorf("rooms[0].Keywords should be nil for empty, got %v", rooms[0].Keywords)
	}
}

func TestFindConfigFile_Priority(t *testing.T) {
	tmpDir := t.TempDir()

	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}

	path, found := FindConfigFile(emptyDir)
	if found {
		t.Error("expected no config file in empty dir")
	}
	if path != "" {
		t.Errorf("expected empty path, got %q", path)
	}

	mempalPath := filepath.Join(tmpDir, "with_mempalace")
	if err := os.MkdirAll(mempalPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mempalPath, "mempalace.yaml"), []byte("project_name: x\n"), 0644); err != nil {
		t.Fatal(err)
	}

	path, found = FindConfigFile(mempalPath)
	if !found {
		t.Error("expected to find mempalace.yaml")
	}
	if filepath.Base(path) != "mempalace.yaml" {
		t.Errorf("expected mempalace.yaml, got %q", path)
	}

	legacyPath := filepath.Join(tmpDir, "with_legacy")
	if err := os.MkdirAll(legacyPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyPath, "mempal.yaml"), []byte("project_name: y\n"), 0644); err != nil {
		t.Fatal(err)
	}

	path, found = FindConfigFile(legacyPath)
	if !found {
		t.Error("expected to find legacy mempal.yaml")
	}
	if filepath.Base(path) != "mempal.yaml" {
		t.Errorf("expected mempal.yaml, got %q", path)
	}

	bothDir := filepath.Join(tmpDir, "with_both")
	if err := os.MkdirAll(bothDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bothDir, "mempalace.yaml"), []byte("project_name: x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bothDir, "mempal.yaml"), []byte("project_name: y\n"), 0644); err != nil {
		t.Fatal(err)
	}

	path, found = FindConfigFile(bothDir)
	if !found {
		t.Error("expected to find config")
	}

	if filepath.Base(path) != "mempalace.yaml" {
		t.Errorf("mempalace.yaml should take priority, got %q", path)
	}
}

func TestSaveRoomsToYAML(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "mempalace.yaml")

	rooms := []RoomConfig{
		{Name: "frontend", Keywords: []string{"ui", "components"}},
		{Name: "backend", Keywords: []string{"api", "server"}},
		{Name: "general"},
	}

	err := SaveRoomsToYAML(yamlPath, "test-project", rooms)
	if err != nil {
		t.Fatalf("SaveRoomsToYAML() error = %v", err)
	}

	loaded, err := LoadRoomsFromYAML(yamlPath)
	if err != nil {
		t.Fatalf("LoadRoomsFromYAML() error = %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("expected 3 rooms after round-trip, got %d", len(loaded))
	}
	if loaded[0].Name != "frontend" {
		t.Errorf("loaded[0].Name = %q, want %q", loaded[0].Name, "frontend")
	}
	if len(loaded[0].Keywords) != 2 {
		t.Errorf("loaded[0].Keywords length = %d, want 2", len(loaded[0].Keywords))
	}
	if loaded[1].Name != "backend" {
		t.Errorf("loaded[1].Name = %q, want %q", loaded[1].Name, "backend")
	}
	if len(loaded[1].Keywords) != 2 {
		t.Errorf("loaded[1].Keywords length = %d, want 2", len(loaded[1].Keywords))
	}
	if loaded[2].Name != "general" {
		t.Errorf("loaded[2].Name = %q, want %q", loaded[2].Name, "general")
	}
}

func TestSaveRoomsToYAML_InvalidPath(t *testing.T) {
	err := SaveRoomsToYAML("/nonexistent/dir/mempalace.yaml", "test", nil)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}
