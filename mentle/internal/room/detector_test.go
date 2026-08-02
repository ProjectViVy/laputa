package room

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDetectRoomsFromFolders(t *testing.T) {
	tmpDir := t.TempDir()

	folders := []string{"frontend", "backend", "docs", "design", "meetings"}
	for _, f := range folders {
		if err := os.MkdirAll(filepath.Join(tmpDir, f), 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", f, err)
		}
	}

	d := NewDetector()
	rooms := d.DetectRoomsFromFolders(tmpDir)

	if len(rooms) == 0 {
		t.Fatal("expected rooms, got none")
	}

	roomNames := make(map[string]bool)
	for _, r := range rooms {
		roomNames[r.Name] = true
	}

	for _, expected := range []string{"frontend", "backend", "documentation", "design", "meetings"} {
		if !roomNames[expected] {
			t.Errorf("expected room %q not found in %v", expected, roomNames)
		}
	}
}

func TestSkipDirs(t *testing.T) {
	tmpDir := t.TempDir()

	skipFolders := []string{".git", "node_modules", "__pycache__", ".venv", "venv", "env", "dist", "build", ".next", "coverage"}
	for _, f := range skipFolders {
		if err := os.MkdirAll(filepath.Join(tmpDir, f), 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", f, err)
		}
	}

	d := NewDetector()
	rooms := d.DetectRoomsFromFolders(tmpDir)

	for _, r := range rooms {
		for _, skip := range skipFolders {
			if r.Name == skip {
				t.Errorf("skip dir %q should not be a room", skip)
			}
		}
	}
}

func TestFallbackToGeneral(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(tmpDir, "ab"), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	d := NewDetector()
	rooms := d.DetectRoomsFromFolders(tmpDir)

	found := false
	for _, r := range rooms {
		if r.Name == "general" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected general room fallback when no known rooms found")
	}
}

func TestRoomMapping(t *testing.T) {
	tmpDir := t.TempDir()

	mappings := []struct {
		folder string
		room   string
	}{
		{"frontend", "frontend"},
		{"front-end", "frontend"},
		{"front_end", "frontend"},
		{"client", "frontend"},
		{"ui", "frontend"},
		{"backend", "backend"},
		{"back-end", "backend"},
		{"server", "backend"},
		{"api", "backend"},
		{"docs", "documentation"},
		{"doc", "documentation"},
		{"wiki", "documentation"},
		{"design", "design"},
		{"designs", "design"},
		{"mockups", "design"},
		{"wireframes", "design"},
		{"assets", "design"},
		{"storyboard", "design"},
		{"costs", "costs"},
		{"cost", "costs"},
		{"budget", "costs"},
		{"finance", "costs"},
		{"pricing", "costs"},
		{"invoices", "costs"},
		{"meetings", "meetings"},
		{"calls", "meetings"},
		{"standup", "meetings"},
		{"minutes", "meetings"},
		{"team", "team"},
		{"hr", "team"},
		{"hiring", "team"},
		{"employees", "team"},
		{"research", "research"},
		{"papers", "research"},
		{"planning", "planning"},
		{"roadmap", "planning"},
		{"strategy", "planning"},
		{"specs", "planning"},
		{"requirements", "planning"},
		{"tests", "testing"},
		{"test", "testing"},
		{"testing", "testing"},
		{"qa", "testing"},
		{"scripts", "scripts"},
		{"tools", "scripts"},
		{"utils", "scripts"},
		{"config", "configuration"},
		{"configs", "configuration"},
		{"settings", "configuration"},
		{"infrastructure", "configuration"},
		{"infra", "configuration"},
		{"deploy", "configuration"},
	}

	for _, m := range mappings {
		if err := os.MkdirAll(filepath.Join(tmpDir, m.folder), 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", m.folder, err)
		}
	}

	d := NewDetector()
	rooms := d.DetectRoomsFromFolders(tmpDir)

	roomMap := make(map[string]bool)
	for _, r := range rooms {
		roomMap[r.Name] = true
	}

	for _, m := range mappings {
		if !roomMap[m.room] {
			t.Errorf("folder %q should map to room %q, but room not found", m.folder, m.room)
		}
	}
}

func TestCountFiles(t *testing.T) {
	tmpDir := t.TempDir()

	subdirs := []string{"dir1", "dir2", "dir3"}
	for _, d := range subdirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, d), 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", d, err)
		}
	}

	files := []string{
		"file1.txt", "file2.md", "file3.go",
		"dir1/file4.py", "dir2/file5.js",
	}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create dir for %s: %v", f, err)
		}
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", f, err)
		}
	}

	d := NewDetector()
	count := d.CountFiles(tmpDir)

	if count != len(files) {
		t.Errorf("expected %d files, got %d", len(files), count)
	}
}

func TestDetectRoomsFromFolders_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	d := NewDetector()
	rooms := d.DetectRoomsFromFolders(tmpDir)

	if len(rooms) != 1 || rooms[0].Name != "general" {
		t.Errorf("expected single general room, got %v", rooms)
	}
}

func TestDetectRoomsFromFolders_NonexistentDir(t *testing.T) {
	d := NewDetector()
	rooms := d.DetectRoomsFromFolders("/nonexistent/path/that/does/not/exist")

	if len(rooms) != 1 || rooms[0].Name != "general" {
		t.Errorf("expected single general room for nonexistent dir, got %v", rooms)
	}
}

func TestRoomKeywords(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(tmpDir, "frontend"), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	d := NewDetector()
	rooms := d.DetectRoomsFromFolders(tmpDir)

	for _, r := range rooms {
		if len(r.Keywords) == 0 {
			continue
		}
		if r.Name == "frontend" {
			if len(r.Keywords) < 1 {
				t.Error("frontend room should have keywords")
			}
			found := slices.Contains(r.Keywords, "frontend")
			if !found {
				t.Error("frontend room should include 'frontend' in keywords")
			}
		}
	}
}
