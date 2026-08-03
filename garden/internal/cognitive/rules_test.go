package cognitive

import (
	"os"
	"path/filepath"
	"testing"

	lpcog "github.com/dashimaki/laputa/governance/cognitive"
)

func TestLoadMissingFallsBackToBuiltin(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "MEMRULES.MD"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Origin() != "builtin" {
		t.Errorf("origin = %q, want builtin", s.Origin())
	}
	if len(s.Rules()) != 7 {
		t.Errorf("got %d rules, want 7", len(s.Rules()))
	}
	if !s.MatchesDefault() {
		t.Error("builtin rules should match default")
	}
	if s.Version() != "1" {
		t.Errorf("version = %q, want 1", s.Version())
	}
}

func TestLoadFileAndDetectManualEdit(t *testing.T) {
	// Unmodified default file matches.
	dir := t.TempDir()
	if err := lpcog.InitializeDir(dir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, lpcog.MemRulesFileName)
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Origin() != "file" {
		t.Errorf("origin = %q, want file", s.Origin())
	}
	if !s.MatchesDefault() {
		t.Error("unmodified default file should match")
	}

	// Human edit breaks the match.
	custom := "# Memory Rules\n\n## R1 — Custom rule\nUser-authored text.\n"
	if err := os.WriteFile(path, []byte(custom), 0644); err != nil {
		t.Fatal(err)
	}
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
	if s.MatchesDefault() {
		t.Error("edited file must not match default")
	}
	if len(s.Rules()) != 1 || s.Rules()[0].ID != "R1" || s.Rules()[0].Title != "Custom rule" {
		t.Errorf("unexpected rules after reload: %+v", s.Rules())
	}
	if s.HashPrefix() == "" || len(s.HashPrefix()) != 16 {
		t.Errorf("HashPrefix = %q, want 16 hex chars", s.HashPrefix())
	}
}

func TestLoadUnreadableFileErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MEMRULES.MD")
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for unreadable path")
	}
}
