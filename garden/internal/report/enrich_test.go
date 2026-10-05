package report

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ProjectViVy/laputa/mentle/facade"
)

type fakeArtifactGen struct {
	result *ArtifactResult
	err    error
}

func dailyReportFixture() Report {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	return Report{
		Cadence:     "daily",
		WindowStart: now.Add(-24 * time.Hour),
		WindowEnd:   now,
		SourceIDs:   []string{"mem_1"},
		Summary:     "deterministic summary",
		Completed:   []string{"fallback complete"},
		Decisions:   []string{"chose x"},
		Generator:   GeneratorDeterministic,
	}
}

func (f *fakeArtifactGen) GenerateArtifact(ctx context.Context, _ string, _ string) (*ArtifactResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return f.result, nil
}

func TestLLMEnricherFillsArtifact(t *testing.T) {
	long := strings.Repeat("x", 300)
	enricher := &LLMEnricher{Gen: &fakeArtifactGen{result: &ArtifactResult{
		Goals:     []string{"finish gate b", long},
		Completed: []string{"shipped dual write"},
		Decisions: []string{},
		OpenLoops: []string{"console page"},
	}}}
	r := dailyReportFixture()
	out, err := enricher.Enrich(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if out.Generator != GeneratorLLM {
		t.Fatalf("generator=%q", out.Generator)
	}
	if len(out.Goals) != 2 || out.Goals[0] != "finish gate b" {
		t.Fatalf("goals=%v", out.Goals)
	}
	if runes := []rune(out.Goals[1]); len(runes) != 241 || !strings.HasSuffix(out.Goals[1], "…") {
		t.Fatalf("long goal not truncated to 240 runes: %d", len(runes))
	}
	if len(out.Completed) != 1 || out.Completed[0] != "shipped dual write" {
		t.Fatalf("completed=%v", out.Completed)
	}
	if len(out.Decisions) != 1 || out.Decisions[0] != "chose x" {
		t.Fatalf("empty llm decisions must keep deterministic fallback: %v", out.Decisions)
	}
	if len(out.OpenLoops) != 1 || out.OpenLoops[0] != "console page" {
		t.Fatalf("openLoops=%v", out.OpenLoops)
	}
}

func TestLLMEnricherBoundsLists(t *testing.T) {
	many := make([]string, 15)
	for i := range many {
		many[i] = "goal"
	}
	enricher := &LLMEnricher{Gen: &fakeArtifactGen{result: &ArtifactResult{Goals: many}}}
	out, err := enricher.Enrich(context.Background(), dailyReportFixture())
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Goals) != 10 {
		t.Fatalf("goals=%d, want cap 10", len(out.Goals))
	}
}

func TestLLMEnricherErrorKeepsDeterministic(t *testing.T) {
	enricher := &LLMEnricher{Gen: &fakeArtifactGen{err: errors.New("llm refused")}}
	r := dailyReportFixture()
	out, err := enricher.Enrich(context.Background(), r)
	if err == nil {
		t.Fatal("expected error")
	}
	if out.Generator != GeneratorDeterministic {
		t.Fatalf("generator=%q", out.Generator)
	}
}

func TestLLMEnricherTimeout(t *testing.T) {
	blocking := &blockingArtifactGen{block: 200 * time.Millisecond}
	enricher := &LLMEnricher{Gen: blocking, Timeout: 10 * time.Millisecond}
	start := time.Now()
	_, err := enricher.Enrich(context.Background(), dailyReportFixture())
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed := time.Since(start); elapsed > 150*time.Millisecond {
		t.Fatalf("timeout not enforced: %v", elapsed)
	}
}

type blockingArtifactGen struct{ block time.Duration }

func (b *blockingArtifactGen) GenerateArtifact(ctx context.Context, _ string, _ string) (*ArtifactResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(b.block):
		return &ArtifactResult{}, nil
	}
}

func TestLLMEnricherUnknownCadence(t *testing.T) {
	enricher := &LLMEnricher{Gen: &fakeArtifactGen{result: &ArtifactResult{}}}
	r := dailyReportFixture()
	r.Cadence = "hourly"
	if _, err := enricher.Enrich(context.Background(), r); err == nil {
		t.Fatal("unknown cadence must error")
	}
}

func TestGenerateUsesEnricher(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	enricher := &LLMEnricher{Gen: &fakeArtifactGen{result: &ArtifactResult{Goals: []string{"ship reports"}, OpenLoops: []string{"wire console"}}}}
	svc, err := Open(filepath.Join(t.TempDir(), "garden.db"), fakeLister{items: []facade.Memory{{ID: "mem_1", Content: "work", UpdatedAt: now}}}, nil, enricher)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	if _, err := svc.Generate(context.Background(), "daily", now); err != nil {
		t.Fatal(err)
	}
	latest, err := svc.Latest(context.Background(), "daily")
	if err != nil {
		t.Fatal(err)
	}
	if latest.Generator != GeneratorLLM {
		t.Fatalf("generator=%q", latest.Generator)
	}
	if len(latest.Goals) != 1 || latest.Goals[0] != "ship reports" {
		t.Fatalf("goals=%v", latest.Goals)
	}
}

func TestGenerateEnrichFailureFallsBack(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	enricher := &LLMEnricher{Gen: &fakeArtifactGen{err: errors.New("model down")}}
	svc, err := Open(filepath.Join(t.TempDir(), "garden.db"), fakeLister{items: []facade.Memory{{ID: "mem_1", Content: "work", UpdatedAt: now}}}, nil, enricher)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	if _, err := svc.Generate(context.Background(), "daily", now); err != nil {
		t.Fatalf("enrich failure must not fail generation: %v", err)
	}
	latest, err := svc.Latest(context.Background(), "daily")
	if err != nil {
		t.Fatal(err)
	}
	if latest.Generator != GeneratorDeterministic {
		t.Fatalf("generator=%q", latest.Generator)
	}
}
