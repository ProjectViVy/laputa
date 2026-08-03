package cognitive

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testMemrules = `---
version: 2
updated: 2026-08-03T00:00:00Z
---

# Memory Rules

## R1 — Evidence primacy
Mentle raw material and evidence remain the primary source of truth.

## R2 — Claim distinction
Confirmed fact, observation, inference, and hypothesis are distinct categories.

## R3 — Contradiction handling
New contradictory evidence does not silently overwrite prior understanding.
`

const testWorld = `# WORLD

## [environment] Development machine
- status: confirmed
- confidence: high
- scope: dev, infra
- source: user
- updated: 2026-08-01T10:00:00Z

Windows 11, 64GB RAM, Go 1.26, Node 24.

## [project] LAPUTA repository
- status: observed
- confidence: medium
- scope: dev, project
- source: mem_abc123
- updated: 2026-08-02T14:30:00Z

Monorepo with three Go modules: laputa, mentle, garden.

## [people] Team
- status: inferred
- confidence: low
- scope: org
- source: autodream
- updated: 2026-08-02T16:00:00Z

Single developer working on the project.
`

func TestLoadMemRules(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MEMRULES.MD")
	if err := os.WriteFile(path, []byte(testMemrules), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadMemRules(path)
	if err != nil {
		t.Fatalf("LoadMemRules: %v", err)
	}

	if m.Version != "2" {
		t.Errorf("version = %q, want %q", m.Version, "2")
	}
	if len(m.Rules) != 3 {
		t.Fatalf("got %d rules, want 3", len(m.Rules))
	}
	if m.Rules[0].ID != "R1" {
		t.Errorf("rules[0].ID = %q, want R1", m.Rules[0].ID)
	}
	if m.Rules[0].Title != "Evidence primacy" {
		t.Errorf("rules[0].Title = %q, want %q", m.Rules[0].Title, "Evidence primacy")
	}
	if m.Rules[0].Text != "Mentle raw material and evidence remain the primary source of truth." {
		t.Errorf("rules[0].Text = %q", m.Rules[0].Text)
	}
}

func TestLoadMemRulesMissing(t *testing.T) {
	_, err := LoadMemRules(filepath.Join(t.TempDir(), "nonexistent.md"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestWorldStoreProject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "WORLD.MD")
	if err := os.WriteFile(path, []byte(testWorld), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := LoadWorld(path)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	if len(w.Claims) != 3 {
		t.Fatalf("got %d claims, want 3", len(w.Claims))
	}

	devClaims := w.Project([]string{"dev"}, 10000)
	if len(devClaims) != 2 {
		t.Fatalf("scope=dev: got %d claims, want 2", len(devClaims))
	}

	orgClaims := w.Project([]string{"org"}, 10000)
	if len(orgClaims) != 1 {
		t.Fatalf("scope=org: got %d claims, want 1", len(orgClaims))
	}
	if orgClaims[0].Title != "Team" {
		t.Errorf("org claim title = %q, want Team", orgClaims[0].Title)
	}
}

func TestWorldStoreProjectBudget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "WORLD.MD")
	if err := os.WriteFile(path, []byte(testWorld), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := LoadWorld(path)
	if err != nil {
		t.Fatal(err)
	}

	limited := w.Project(nil, 40)
	if len(limited) == 0 {
		t.Fatal("expected at least 1 claim within budget")
	}
	total := 0
	for _, c := range limited {
		total += len([]rune(c.Text))
	}
	if total > 40 {
		t.Errorf("budget exceeded: %d chars > 40", total)
	}
}

func TestWorldStoreSaveProtection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "WORLD.MD")
	if err := os.WriteFile(path, []byte(testWorld), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := LoadWorld(path)
	if err != nil {
		t.Fatal(err)
	}

	err = w.Save("autodream")
	if err != ErrConfirmedProtected {
		t.Fatalf("expected ErrConfirmedProtected, got %v", err)
	}

	err = w.Save("user")
	if err != nil {
		t.Fatalf("user save should succeed: %v", err)
	}
}

func TestWorldStoreSerializeRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "WORLD.MD")
	if err := os.WriteFile(path, []byte(testWorld), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := LoadWorld(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := w.Save("user"); err != nil {
		t.Fatal(err)
	}

	w2, err := LoadWorld(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(w2.Claims) != len(w.Claims) {
		t.Fatalf("round-trip: got %d claims, want %d", len(w2.Claims), len(w.Claims))
	}
	if w2.Claims[0].Domain != "environment" {
		t.Errorf("round-trip domain = %q, want environment", w2.Claims[0].Domain)
	}
	if w2.Claims[0].Status != ClaimConfirmed {
		t.Errorf("round-trip status = %q, want confirmed", w2.Claims[0].Status)
	}
	if !w2.Claims[0].Updated.Equal(time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("round-trip updated = %v", w2.Claims[0].Updated)
	}
}

func TestDefaultMemRulesParse(t *testing.T) {
	m := DefaultMemRules()
	if len(m.Rules) != 7 {
		t.Fatalf("got %d rules, want 7", len(m.Rules))
	}
	for i, want := range []string{"R1", "R2", "R3", "R4", "R5", "R6", "R7"} {
		if m.Rules[i].ID != want {
			t.Errorf("rules[%d].ID = %q, want %q", i, m.Rules[i].ID, want)
		}
		if m.Rules[i].Text == "" {
			t.Errorf("rules[%d].Text empty", i)
		}
	}
	if m.Version != "1" {
		t.Errorf("version = %q, want 1", m.Version)
	}
}

func TestInitializeDirCreatesDefaults(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cognitive")
	if err := InitializeDir(dir); err != nil {
		t.Fatalf("InitializeDir: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, MemRulesFileName))
	if err != nil {
		t.Fatalf("MEMRULES.MD not created: %v", err)
	}
	if string(data) != DefaultMemRulesText {
		t.Errorf("MEMRULES.MD content mismatch")
	}
	data, err = os.ReadFile(filepath.Join(dir, WorldFileName))
	if err != nil {
		t.Fatalf("WORLD.MD not created: %v", err)
	}
	if string(data) != DefaultWorldText {
		t.Errorf("WORLD.MD content mismatch")
	}
}

func TestInitializeDirIdempotent(t *testing.T) {
	dir := t.TempDir()
	customRules := "# Memory Rules\n\n## R1 — Custom\nUser edit.\n"
	customWorld := "# WORLD\n\nuser content\n"
	if err := os.WriteFile(filepath.Join(dir, MemRulesFileName), []byte(customRules), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, WorldFileName), []byte(customWorld), 0644); err != nil {
		t.Fatal(err)
	}

	if err := InitializeDir(dir); err != nil {
		t.Fatalf("InitializeDir: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, MemRulesFileName))
	if string(data) != customRules {
		t.Errorf("MEMRULES.MD was overwritten")
	}
	data, _ = os.ReadFile(filepath.Join(dir, WorldFileName))
	if string(data) != customWorld {
		t.Errorf("WORLD.MD was overwritten")
	}
}

func TestWorldStoreProjectConfidenceOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "WORLD.MD")
	if err := os.WriteFile(path, []byte(testWorld), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := LoadWorld(path)
	if err != nil {
		t.Fatal(err)
	}

	claims := w.Project(nil, 10000)
	if len(claims) != 3 {
		t.Fatalf("got %d claims, want 3", len(claims))
	}
	wantOrder := []string{"high", "medium", "low"}
	for i, want := range wantOrder {
		if claims[i].Confidence != want {
			t.Errorf("claims[%d].Confidence = %q, want %q", i, claims[i].Confidence, want)
		}
	}

	// Budget that only fits the highest-confidence claim.
	tight := w.Project(nil, 40)
	if len(tight) != 1 || tight[0].Confidence != "high" {
		t.Errorf("tight budget: got %d claims, want the single high-confidence claim", len(tight))
	}
}

func TestWorldStoreConcurrentProjectSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "WORLD.MD")
	if err := os.WriteFile(path, []byte(testWorld), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := LoadWorld(path)
	if err != nil {
		t.Fatal(err)
	}
	if w.Total() != 3 {
		t.Fatalf("Total() = %d, want 3", w.Total())
	}

	done := make(chan struct{})
	for i := 0; i < 4; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 50; j++ {
				w.Project([]string{"dev"}, 1000)
				w.Project(nil, 1000)
			}
		}()
	}
	for i := 0; i < 2; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 10; j++ {
				_ = w.Save("user")
			}
		}()
	}
	for i := 0; i < 6; i++ {
		<-done
	}
}
