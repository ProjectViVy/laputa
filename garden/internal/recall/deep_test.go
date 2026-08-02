package recall

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/dashimaki/garden/internal/arbiter"
	"github.com/dashimaki/laputa/governance"
	"github.com/dashimaki/mentle/facade"
)

type mockGraph struct {
	facts    []facade.GraphFact
	events   []facade.TimelineEvent
	err      error
	calls    int
	tlCalls  int
}

func (m *mockGraph) QueryEntity(_ context.Context, _ string, _ string, _ string) ([]facade.GraphFact, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.facts, nil
}

func (m *mockGraph) Timeline(_ context.Context, _ string) ([]facade.TimelineEvent, error) {
	m.tlCalls++
	if m.err != nil {
		return nil, m.err
	}
	return m.events, nil
}

type failingGraph struct{ t *testing.T }

func (f failingGraph) QueryEntity(context.Context, string, string, string) ([]facade.GraphFact, error) {
	f.t.Fatal("QueryEntity must not be called without kg capability")
	return nil, nil
}

func (f failingGraph) Timeline(context.Context, string) ([]facade.TimelineEvent, error) {
	f.t.Fatal("Timeline must not be called without timeline capability")
	return nil, nil
}

type mockSearcher struct {
	cards    []facade.MemoryCard
	evidence []facade.EvidenceFragment
	err      error
}

func (m mockSearcher) SearchCards(context.Context, facade.CardQuery) (facade.CardPage, error) {
	if m.err != nil {
		return facade.CardPage{}, m.err
	}
	return facade.CardPage{Cards: m.cards}, nil
}

func (m mockSearcher) ReadEvidence(context.Context, facade.EvidenceQuery) ([]facade.EvidenceFragment, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.evidence, nil
}

type fakeGovReader struct{}

func (fakeGovReader) GetSection(_ context.Context, _ governance.SectionName) (map[string]any, error) {
	return map[string]any{"_meta": map[string]any{"version": "test"}}, nil
}

func deepTestService(t *testing.T, graph GraphSource, searcher CardSearcher) *DeepService {
	t.Helper()
	traceStore, err := OpenTraceStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { traceStore.Close() })

	fast := &FastService{Gov: fakeGovReader{}, Searcher: searcher}
	return &DeepService{
		Fast:    fast,
		Graph:   graph,
		Arbiter: arbiter.New(),
		Traces:  traceStore,
	}
}

func TestDeepRecallRequiresTriggerReason(t *testing.T) {
	svc := deepTestService(t, nil, mockSearcher{})
	_, err := svc.Recall(context.Background(), DeepRequest{Query: "test"})
	if err == nil {
		t.Fatal("expected error for missing trigger_reason")
	}
}

func TestDeepRecallNoKGWithoutCapability(t *testing.T) {
	svc := deepTestService(t, failingGraph{t}, mockSearcher{
		cards: []facade.MemoryCard{{ID: "mem_1", Kind: "fact", Summary: "test", CandidateScore: 0.8}},
	})
	resp, err := svc.Recall(context.Background(), DeepRequest{
		Query:         "test",
		TriggerReason: "verification",
		Entities:      []string{"garden"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Assertions) != 0 {
		t.Fatalf("assertions=%v, want none", resp.Assertions)
	}
}

func TestDeepRecallExpandsKGWithCapability(t *testing.T) {
	graph := &mockGraph{facts: []facade.GraphFact{
		{Predicate: "leads", Object: "alice", ValidFrom: "2026-01-01", Confidence: 0.9},
	}}
	svc := deepTestService(t, graph, mockSearcher{
		cards: []facade.MemoryCard{{ID: "mem_1", Kind: "fact", Summary: "test", CandidateScore: 0.8}},
	})
	resp, err := svc.Recall(context.Background(), DeepRequest{
		Query:         "test",
		TriggerReason: "verification",
		Capabilities:  []string{"kg"},
		Entities:      []string{"garden"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if graph.calls != 1 {
		t.Fatalf("kg calls=%d, want 1", graph.calls)
	}
	if len(resp.Assertions) != 1 {
		t.Fatalf("assertions=%d, want 1", len(resp.Assertions))
	}
	if resp.Assertions[0].Subject != "garden" || resp.Assertions[0].Object != "alice" {
		t.Fatalf("assertion=%+v", resp.Assertions[0])
	}
	if resp.Assertions[0].Status != "active" {
		t.Fatalf("status=%s", resp.Assertions[0].Status)
	}
}

func TestDeepRecallTimelineExpansion(t *testing.T) {
	graph := &mockGraph{events: []facade.TimelineEvent{
		{Predicate: "released", Object: "v1.0", ValidFrom: "2026-01-15"},
		{Predicate: "released", Object: "v2.0", ValidFrom: "2026-06-01"},
	}}
	svc := deepTestService(t, graph, mockSearcher{
		cards: []facade.MemoryCard{{ID: "mem_1", Kind: "fact", Summary: "test", CandidateScore: 0.8}},
	})
	resp, err := svc.Recall(context.Background(), DeepRequest{
		Query:         "releases",
		TriggerReason: "history check",
		Capabilities:  []string{"timeline"},
		Entities:      []string{"garden"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if graph.tlCalls != 1 {
		t.Fatalf("timeline calls=%d, want 1", graph.tlCalls)
	}
	if len(resp.Assertions) != 2 {
		t.Fatalf("assertions=%d, want 2", len(resp.Assertions))
	}
}

func TestDeepRecallAlwaysEmitsTrace(t *testing.T) {
	svc := deepTestService(t, nil, mockSearcher{
		cards: []facade.MemoryCard{{ID: "mem_1", Kind: "fact", Summary: "test", CandidateScore: 0.8}},
	})
	resp, err := svc.Recall(context.Background(), DeepRequest{
		Query:         "test",
		TriggerReason: "verification",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Trace.TraceID == "" {
		t.Fatal("trace ID is empty")
	}
	if resp.Trace.TriggerReason != "verification" {
		t.Fatalf("trace=%+v", resp.Trace)
	}
	if resp.RecallTraceID == nil || *resp.RecallTraceID != resp.Trace.TraceID {
		t.Fatal("RecallTraceID not linked")
	}

	got, err := svc.Traces.Get(context.Background(), resp.Trace.TraceID)
	if err != nil {
		t.Fatalf("trace not persisted: %v", err)
	}
	if got.Query != "test" {
		t.Fatalf("persisted trace=%+v", got)
	}
}

func TestDeepRecallFallbackOnKGError(t *testing.T) {
	graph := &mockGraph{err: errors.New("kg timeout")}
	svc := deepTestService(t, graph, mockSearcher{
		cards: []facade.MemoryCard{{ID: "mem_1", Kind: "fact", Summary: "test", CandidateScore: 0.8}},
	})
	resp, err := svc.Recall(context.Background(), DeepRequest{
		Query:         "test",
		TriggerReason: "verification",
		Capabilities:  []string{"kg"},
		Entities:      []string{"garden"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Trace.Degraded {
		t.Fatal("expected degraded trace")
	}
	if len(resp.Warnings) == 0 {
		t.Fatal("expected warnings")
	}
	if resp.Mode != "deep" {
		t.Fatalf("mode=%s", resp.Mode)
	}
}

func TestDeepRecallFallbackOnSeedError(t *testing.T) {
	svc := deepTestService(t, nil, mockSearcher{err: errors.New("mentle down")})
	svc.Fast.Searcher = nil
	svc.Fast.Gov = nil

	_, err := svc.Recall(context.Background(), DeepRequest{
		Query:         "test",
		TriggerReason: "verification",
	})
	if err != nil {
		t.Fatal("should not return error; should return degraded response")
	}
}

func TestDeepRecallBudgetEnforced(t *testing.T) {
	svc := deepTestService(t, nil, mockSearcher{})
	_, err := svc.Recall(context.Background(), DeepRequest{
		Query:         "test",
		TriggerReason: "verification",
		BudgetChars:   100,
	})
	if err == nil {
		t.Fatal("expected error for budget below minimum")
	}
}

func TestDeepRecallPlannerOptional(t *testing.T) {
	svc := deepTestService(t, nil, mockSearcher{
		cards: []facade.MemoryCard{{ID: "mem_1", Kind: "fact", Summary: "test", CandidateScore: 0.8}},
	})
	resp, err := svc.Recall(context.Background(), DeepRequest{
		Query:         "test",
		TriggerReason: "verification",
		UsePlanner:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Trace.TraceID == "" {
		t.Fatal("trace missing")
	}
}
