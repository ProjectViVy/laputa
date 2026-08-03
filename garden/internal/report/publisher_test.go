package report

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/dashimaki/laputa/governance"
	"github.com/dashimaki/mentle/facade"
)

type fakeGateway struct {
	sections map[governance.SectionName]map[string]any
	requests []governance.MutationRequest
	mutErr   error
}

func (f *fakeGateway) GetSection(_ context.Context, section governance.SectionName) (map[string]any, error) {
	if data, ok := f.sections[section]; ok {
		return data, nil
	}
	return map[string]any{}, nil
}

func (f *fakeGateway) Mutate(_ context.Context, req governance.MutationRequest) error {
	if f.mutErr != nil {
		return f.mutErr
	}
	f.requests = append(f.requests, req)
	if f.sections == nil {
		f.sections = map[governance.SectionName]map[string]any{}
	}
	f.sections[req.Section] = req.Data
	return nil
}

func dailyReportFixture() Report {
	return Report{
		Cadence:       "daily",
		WindowStart:   time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC),
		SourceIDs:     []string{"mem_1"},
		SourceHash:    "sha256:fixture",
		Title:         "Daily Garden Memory Report",
		Summary:       "- did things\n",
		Highlights:    []string{"did things"},
		Completed:     []string{"did things"},
		Decisions:     []string{"chose x"},
		Goals:         []string{},
		OpenLoops:     []string{},
		OpenQuestions: []string{},
		SourceRefs:    []string{"mem_1"},
		Revision:      1,
		Generator:     GeneratorDeterministic,
		GeneratedAt:   time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC),
	}
}

func TestGovernedPublisherWritesAsReportSystem(t *testing.T) {
	gw := &fakeGateway{}
	pub := &GovernedPublisher{Gov: gw}
	if err := pub.Publish(context.Background(), dailyReportFixture()); err != nil {
		t.Fatal(err)
	}
	if len(gw.requests) != 1 {
		t.Fatalf("requests=%d", len(gw.requests))
	}
	req := gw.requests[0]
	if req.Section != governance.SectionDaily {
		t.Fatalf("section=%s", req.Section)
	}
	if req.Actor != governance.ActorReportSystem {
		t.Fatalf("actor=%s", req.Actor)
	}
	if req.Action != "write" {
		t.Fatalf("action=%s", req.Action)
	}
	reports, _ := req.Data["reports"].([]any)
	if len(reports) != 1 {
		t.Fatalf("reports=%d", len(reports))
	}
	entry, _ := reports[0].(map[string]any)
	if entry["source_hash"] != "sha256:fixture" || entry["generator"] != GeneratorDeterministic || entry["revision"] != 1 {
		t.Fatalf("entry=%v", entry)
	}
	decisions, _ := entry["decisions"].([]any)
	if len(decisions) != 1 || decisions[0] != "chose x" {
		t.Fatalf("decisions=%v", decisions)
	}
}

func TestGovernedPublisherSkipsDuplicateHash(t *testing.T) {
	gw := &fakeGateway{sections: map[governance.SectionName]map[string]any{
		governance.SectionDaily: {"reports": []any{map[string]any{"source_hash": "sha256:fixture", "title": "prior"}}},
	}}
	pub := &GovernedPublisher{Gov: gw}
	if err := pub.Publish(context.Background(), dailyReportFixture()); err != nil {
		t.Fatal(err)
	}
	if len(gw.requests) != 0 {
		t.Fatalf("duplicate hash must not mutate, requests=%d", len(gw.requests))
	}
}

func TestGovernedPublisherCapsSectionEntries(t *testing.T) {
	existing := make([]any, maxSectionReports)
	for i := range existing {
		existing[i] = map[string]any{"source_hash": "sha256:old", "seq": i}
	}
	gw := &fakeGateway{sections: map[governance.SectionName]map[string]any{
		governance.SectionDaily: {"reports": existing},
	}}
	pub := &GovernedPublisher{Gov: gw}
	if err := pub.Publish(context.Background(), dailyReportFixture()); err != nil {
		t.Fatal(err)
	}
	reports, _ := gw.requests[0].Data["reports"].([]any)
	if len(reports) != maxSectionReports {
		t.Fatalf("capped len=%d", len(reports))
	}
	first, _ := reports[0].(map[string]any)
	if first["seq"] != 1 {
		t.Fatalf("oldest entry was not dropped first: %v", first)
	}
	last, _ := reports[len(reports)-1].(map[string]any)
	if last["source_hash"] != "sha256:fixture" {
		t.Fatalf("new entry missing: %v", last)
	}
}

type capturingPublisher struct {
	reports []Report
	err     error
}

func (c *capturingPublisher) Publish(_ context.Context, r Report) error {
	if c.err != nil {
		return c.err
	}
	c.reports = append(c.reports, r)
	return nil
}

func TestGeneratePublishesOnlyOnNewInsert(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	pub := &capturingPublisher{}
	svc, err := Open(filepath.Join(t.TempDir(), "garden.db"), fakeLister{items: []facade.Memory{{ID: "mem_1", Content: "work", UpdatedAt: now}}}, pub, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	if _, err := svc.Generate(context.Background(), "daily", now); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Generate(context.Background(), "daily", now); err != nil {
		t.Fatal(err)
	}
	if len(pub.reports) != 1 {
		t.Fatalf("publish count=%d, want 1 (duplicate source hash must not republish)", len(pub.reports))
	}
	if pub.reports[0].Revision != 1 {
		t.Fatalf("published revision=%d", pub.reports[0].Revision)
	}
}

func TestGeneratePublishFailureNonFatal(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	pub := &capturingPublisher{err: errors.New("section write refused")}
	svc, err := Open(filepath.Join(t.TempDir(), "garden.db"), fakeLister{items: []facade.Memory{{ID: "mem_1", Content: "work", UpdatedAt: now}}}, pub, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	r, err := svc.Generate(context.Background(), "daily", now)
	if err != nil {
		t.Fatalf("publish failure must not fail generation: %v", err)
	}
	if r.SourceHash == "" {
		t.Fatal("report missing")
	}
	latest, err := svc.Latest(context.Background(), "daily")
	if err != nil || latest.SourceHash != r.SourceHash {
		t.Fatalf("latest=%+v err=%v", latest, err)
	}
}
