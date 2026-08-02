package room

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigBasedRoomDetector_DefaultRooms(t *testing.T) {
	tmpDir := t.TempDir()

	d, err := NewConfigBasedRoomDetector(tmpDir)
	if err != nil {
		t.Fatalf("NewConfigBasedRoomDetector() error = %v", err)
	}

	rooms := d.Rooms()
	if len(rooms) != 1 || rooms[0].Name != "general" {
		t.Errorf("expected default [general] room, got %v", rooms)
	}
}

func TestConfigBasedRoomDetector_FromYAML(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "mempalace.yaml")

	content := `rooms:
  - name: frontend
    keywords: [ui, components, views]
  - name: backend
    keywords: [api, server]
  - name: general
`
	if err := os.WriteFile(yamlPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	d, err := NewConfigBasedRoomDetector(tmpDir)
	if err != nil {
		t.Fatalf("NewConfigBasedRoomDetector() error = %v", err)
	}

	rooms := d.Rooms()
	if len(rooms) != 3 {
		t.Fatalf("expected 3 rooms, got %d", len(rooms))
	}
	if rooms[0].Name != "frontend" {
		t.Errorf("rooms[0].Name = %q, want %q", rooms[0].Name, "frontend")
	}
	if len(rooms[0].Keywords) != 3 {
		t.Errorf("rooms[0].Keywords length = %d, want 3", len(rooms[0].Keywords))
	}
	if rooms[1].Name != "backend" {
		t.Errorf("rooms[1].Name = %q, want %q", rooms[1].Name, "backend")
	}
	if len(rooms[1].Keywords) != 2 {
		t.Errorf("rooms[1].Keywords length = %d, want 2", len(rooms[1].Keywords))
	}
	if rooms[2].Name != "general" {
		t.Errorf("rooms[2].Name = %q, want %q", rooms[2].Name, "general")
	}
	if len(rooms[2].Keywords) != 0 {
		t.Errorf("rooms[2].Keywords should be nil for empty, got %v", rooms[2].Keywords)
	}
}

func TestConfigBasedRoomDetector_Priority1_FolderPath(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "mempalace.yaml")

	content := `rooms:
  - name: frontend
    keywords: [ui, components]
  - name: general
`
	if err := os.WriteFile(yamlPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	d, err := NewConfigBasedRoomDetector(tmpDir)
	if err != nil {
		t.Fatalf("NewConfigBasedRoomDetector() error = %v", err)
	}

	result := d.DetectRoom(
		filepath.Join(tmpDir, "frontend", "button.tsx"),
		"export const Button = () => {}",
		tmpDir,
	)
	if result != "frontend" {
		t.Errorf("expected 'frontend', got %q", result)
	}
}

func TestConfigBasedRoomDetector_Priority2_Filename(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "mempalace.yaml")

	content := `rooms:
  - name: frontend
    keywords: [ui]
  - name: general
`
	if err := os.WriteFile(yamlPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	d, err := NewConfigBasedRoomDetector(tmpDir)
	if err != nil {
		t.Fatalf("NewConfigBasedRoomDetector() error = %v", err)
	}

	result := d.DetectRoom(
		filepath.Join(tmpDir, "src", "frontend.ts"),
		"const x = 1",
		tmpDir,
	)
	if result != "frontend" {
		t.Errorf("expected 'frontend' from filename match, got %q", result)
	}
}

func TestConfigBasedRoomDetector_Priority3_KeywordScoring(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "mempalace.yaml")

	content := `rooms:
  - name: backend
    keywords: [api, server, handlers, controllers]
  - name: general
`
	if err := os.WriteFile(yamlPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	d, err := NewConfigBasedRoomDetector(tmpDir)
	if err != nil {
		t.Fatalf("NewConfigBasedRoomDetector() error = %v", err)
	}

	result := d.DetectRoom(
		filepath.Join(tmpDir, "src", "utils.ts"),
		"This module provides API handlers for backend server and controllers",
		tmpDir,
	)
	if result != "backend" {
		t.Errorf("expected 'backend' from keyword scoring, got %q", result)
	}
}

func TestConfigBasedRoomDetector_Priority4_Fallback(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "mempalace.yaml")

	content := `rooms:
  - name: frontend
    keywords: [ui]
  - name: general
`
	if err := os.WriteFile(yamlPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	d, err := NewConfigBasedRoomDetector(tmpDir)
	if err != nil {
		t.Fatalf("NewConfigBasedRoomDetector() error = %v", err)
	}

	result := d.DetectRoom(
		filepath.Join(tmpDir, "misc", "notes.txt"),
		"random notes about nothing in particular",
		tmpDir,
	)
	if result != "general" {
		t.Errorf("expected 'general' fallback, got %q", result)
	}
}

func TestConfigBasedRoomDetector_EmptyRooms(t *testing.T) {
	d := &ConfigBasedRoomDetector{rooms: nil}

	result := d.DetectRoom("/some/path.txt", "content", "/some")
	if result != "general" {
		t.Errorf("expected 'general' for empty rooms, got %q", result)
	}
}

func TestConfigBasedRoomDetector_ShortContent(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "mempalace.yaml")

	content := `rooms:
  - name: frontend
    keywords: [ui]
  - name: general
`
	if err := os.WriteFile(yamlPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	d, err := NewConfigBasedRoomDetector(tmpDir)
	if err != nil {
		t.Fatalf("NewConfigBasedRoomDetector() error = %v", err)
	}

	result := d.DetectRoom(
		filepath.Join(tmpDir, "misc", "file.txt"),
		"hi",
		tmpDir,
	)
	if result != "general" {
		t.Errorf("expected 'general' for short content with no matches, got %q", result)
	}
}

func TestConfigBasedRoomDetector_KeywordInFolderPath(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "mempalace.yaml")

	content := `rooms:
  - name: frontend
    keywords: [ui, components]
  - name: general
`
	if err := os.WriteFile(yamlPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	d, err := NewConfigBasedRoomDetector(tmpDir)
	if err != nil {
		t.Fatalf("NewConfigBasedRoomDetector() error = %v", err)
	}

	result := d.DetectRoom(
		filepath.Join(tmpDir, "components", "button.tsx"),
		"export const Button = () => {}",
		tmpDir,
	)
	if result != "frontend" {
		t.Errorf("expected 'frontend' from keyword in folder path, got %q", result)
	}
}
