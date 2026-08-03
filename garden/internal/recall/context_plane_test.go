package recall

// Context Plane test matrix (ADR-0004 §6): invariants for what may and may
// not appear in a ContextView.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dashimaki/laputa/governance"
	"github.com/dashimaki/laputa/governance/cognitive"
	"github.com/dashimaki/mentle/facade"
)

// mapGov returns per-section data, unlike fakeGov which returns the same
// payload for every section.
type mapGov map[governance.SectionName]map[string]any

func (m mapGov) GetSection(_ context.Context, name governance.SectionName) (map[string]any, error) {
	if data, ok := m[name]; ok {
		return data, nil
	}
	return map[string]any{}, nil
}

const worldFixtureText = `# WORLD

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

func worldFixture(t *testing.T) *cognitive.WorldStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "WORLD.MD")
	if err := os.WriteFile(path, []byte(worldFixtureText), 0644); err != nil {
		t.Fatal(err)
	}
	w, err := cognitive.LoadWorld(path)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestFrozenCoreInContext(t *testing.T) {
	gov := mapGov{
		governance.SectionIdentity:     {"_meta": map[string]any{"version": "frozen-id-v1"}},
		governance.SectionRelationship: {"_meta": map[string]any{"version": "frozen-rel-v1"}},
		governance.SectionCommitment:   {"_meta": map[string]any{"version": "frozen-com-v1"}},
		governance.SectionPreferences:  {"_meta": map[string]any{"version": "frozen-pref-v1"}},
	}
	svc := &FastService{Gov: gov}
	view, err := svc.Recall(context.Background(), FastRequest{Query: "bootstrap", BudgetChars: 6000})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(view.Context, "frozen core:") {
		t.Fatalf("context missing frozen core line: %q", view.Context)
	}
	for _, marker := range []string{"frozen-id-v1", "frozen-rel-v1", "frozen-com-v1", "frozen-pref-v1"} {
		if !strings.Contains(view.Context, marker) {
			t.Errorf("context missing frozen ref %q: %q", marker, view.Context)
		}
		if view.Governance.FrozenRefs == nil {
			t.Fatal("projection missing FrozenRefs")
		}
	}
}

func TestSTMInContext(t *testing.T) {
	gov := mapGov{
		governance.SectionMemoryMD: {"refs": []any{"mem_a", "mem_b"}},
	}
	svc := &FastService{Gov: gov}
	view, err := svc.Recall(context.Background(), FastRequest{Query: "session state", BudgetChars: 6000})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(view.Context, "working set: mem_a, mem_b") {
		t.Fatalf("context missing working set refs: %q", view.Context)
	}
	if len([]rune(view.Context)) > 6000 {
		t.Errorf("context exceeds budget")
	}
}

func TestWorldScopedProjection(t *testing.T) {
	world := worldFixture(t)

	svc := &FastService{Gov: mapGov{}, World: world}
	view, err := svc.Recall(context.Background(), FastRequest{Query: "environment", Scope: "dev", BudgetChars: 6000})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.World) != 2 {
		t.Fatalf("scope=dev: got %d world claims, want 2", len(view.World))
	}
	for _, claim := range view.World {
		if !contains(claim.Scopes, "dev") {
			t.Errorf("claim %q lacks dev scope", claim.Title)
		}
	}
	if strings.Contains(view.Context, "Single developer working on the project.") {
		t.Error("org-scoped claim leaked into dev-scoped context")
	}
	if !strings.Contains(view.Context, "world projection:") {
		t.Errorf("context missing world block: %q", view.Context)
	}

	// No scope: highest confidence first.
	view, err = svc.Recall(context.Background(), FastRequest{Query: "environment", BudgetChars: 6000})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.World) == 0 || view.World[0].Confidence != "high" {
		t.Fatalf("no-scope projection should lead with high confidence: %+v", view.World)
	}
}

func TestMemrulesNeverInContext(t *testing.T) {
	rulesText := cognitive.DefaultMemRulesText +
		"\n## R8 — Unique tripwire rule\nUNIQUE_MEMRULE_SENTENCE must never surface in context.\n"
	path := filepath.Join(t.TempDir(), "MEMRULES.MD")
	if err := os.WriteFile(path, []byte(rulesText), 0644); err != nil {
		t.Fatal(err)
	}
	rules, err := cognitive.LoadMemRules(path)
	if err != nil {
		t.Fatal(err)
	}
	var needles []string
	for _, rule := range rules.Rules {
		needles = append(needles, rule.Text)
	}

	world := worldFixture(t)
	searcher := &fakeSearcher{
		cards:    []facade.MemoryCard{{ID: "mem_1", Kind: "fact", CandidateScore: 0.9}},
		evidence: []facade.EvidenceFragment{{CardID: "mem_1", Excerpt: "ordinary evidence", Validity: "active"}},
	}

	exits := map[string]*FastService{
		"normal":      {Gov: mapGov{}, Searcher: searcher, World: world},
		"mentle_down": {Gov: mapGov{}, World: world},
		"search_err":  {Gov: mapGov{}, Searcher: &fakeSearcher{cardErr: context.DeadlineExceeded}, World: world},
	}
	for name, svc := range exits {
		view, err := svc.Recall(context.Background(), FastRequest{Query: "rules", BudgetChars: 6000})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		payload, _ := json.Marshal(view)
		for _, needle := range needles {
			if strings.Contains(view.Context, needle) || strings.Contains(string(payload), needle) {
				t.Errorf("%s: rule text leaked into ContextView: %q", name, needle)
			}
		}
		if strings.Contains(view.Context, "R8") || strings.Contains(string(payload), "UNIQUE_MEMRULE_SENTENCE") {
			t.Errorf("%s: rule identifier leaked into ContextView", name)
		}
	}
}

func TestFullWorldNeverInContext(t *testing.T) {
	world := &cognitive.WorldStore{}
	markers := make([]string, 5)
	for i := range markers {
		marker := strings.Repeat(string(rune('A'+i)), 20) + "_CLAIM_MARKER_" + string(rune('0'+i))
		markers[i] = marker
		world.Claims = append(world.Claims, cognitive.WorldClaim{
			Domain:     "test",
			Title:      marker,
			Status:     cognitive.ClaimObserved,
			Confidence: "high",
			Scopes:     []string{"dev"},
			Source:     "user",
			Text:       marker + " " + strings.Repeat("x", 79-len(marker)),
		})
	}
	svc := &FastService{Gov: mapGov{}, World: world}

	// Budget 256 (the minimum) cannot fit all five ~100-char claims.
	view, err := svc.Recall(context.Background(), FastRequest{Query: "everything", BudgetChars: 256})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.World) >= 5 {
		t.Fatalf("full world was projected: %d claims", len(view.World))
	}
	missing := 0
	for _, marker := range markers {
		if !strings.Contains(view.Context, marker) {
			missing++
		}
	}
	if missing == 0 {
		t.Error("context contains every world claim; wholesale injection detected")
	}
	if len([]rune(view.Context)) > 256 {
		t.Errorf("context exceeds budget: %d runes", len([]rune(view.Context)))
	}
}

func TestReportsNeverInContext(t *testing.T) {
	gov := mapGov{
		governance.SectionDaily:   {"content": "DAILY_REPORT_MARKER"},
		governance.SectionWeekly:  {"content": "WEEKLY_REPORT_MARKER"},
		governance.SectionMonthly: {"content": "MONTHLY_REPORT_MARKER"},
	}
	svc := &FastService{Gov: gov}
	view, err := svc.Recall(context.Background(), FastRequest{Query: "reports", BudgetChars: 6000})
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(view)
	for _, marker := range []string{"DAILY_REPORT_MARKER", "WEEKLY_REPORT_MARKER", "MONTHLY_REPORT_MARKER"} {
		if strings.Contains(view.Context, marker) || strings.Contains(string(payload), marker) {
			t.Errorf("report content leaked into ContextView: %s", marker)
		}
	}
}

type capturingSearcher struct {
	cards    []facade.MemoryCard
	evidence []facade.EvidenceFragment
	query    facade.EvidenceQuery
	captured bool
}

func (c *capturingSearcher) SearchCards(_ context.Context, _ facade.CardQuery) (facade.CardPage, error) {
	return facade.CardPage{Cards: c.cards}, nil
}

func (c *capturingSearcher) ReadEvidence(_ context.Context, q facade.EvidenceQuery) ([]facade.EvidenceFragment, error) {
	c.query = q
	c.captured = true
	return c.evidence, nil
}

func TestEvidenceBounded(t *testing.T) {
	searcher := &capturingSearcher{
		cards:    []facade.MemoryCard{{ID: "mem_1", Kind: "fact", CandidateScore: 0.9}},
		evidence: []facade.EvidenceFragment{{CardID: "mem_1", Excerpt: strings.Repeat("e", 2000), Validity: "active"}},
	}
	svc := &FastService{Gov: mapGov{}, Searcher: searcher}
	view, err := svc.Recall(context.Background(), FastRequest{Query: "evidence", BudgetChars: 8000})
	if err != nil {
		t.Fatal(err)
	}
	if !searcher.captured {
		t.Fatal("ReadEvidence was not called")
	}
	if searcher.query.PerItemBudget != 800 {
		t.Errorf("PerItemBudget = %d, want 800", searcher.query.PerItemBudget)
	}
	if searcher.query.TotalBudget != 4000 {
		t.Errorf("TotalBudget = %d, want 4000", searcher.query.TotalBudget)
	}
	if len([]rune(view.Context)) > 8000 {
		t.Errorf("context exceeds request budget")
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
