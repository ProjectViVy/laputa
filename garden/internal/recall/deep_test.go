package recall

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/dashimaki/garden/internal/arbiter"
	"github.com/dashimaki/mentle/facade"
)

type mockGraph struct {
	facts   []facade.GraphFact
	events  []facade.TimelineEvent
	err     error
	calls   int
	tlCalls int
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

func deepTestService(t *testing.T, graph GraphSource, searcher CardSearcher) *DeepService {
	t.Helper()
	traceStore, err := OpenTraceStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceStore.Close() })
	return &DeepService{Fast: &FastService{Searcher: searcher}, Graph: graph, Arbiter: arbiter.New(), Traces: traceStore}
}

func TestDeepRecallRequiresTriggerReason(t *testing.T) {
	if _, err := deepTestService(t, nil, &fakeSearcher{}).Recall(context.Background(), DeepRequest{Query: "test"}); err == nil {
		t.Fatal("expected error for missing trigger_reason")
	}
}

func TestDeepRecallNoKGWithoutCapability(t *testing.T) {
	resp, err := deepTestService(t, failingGraph{t}, &fakeSearcher{cards: []facade.MemoryCard{{ID: "mem_1", Kind: "fact"}}}).Recall(context.Background(), DeepRequest{Query: "test", TriggerReason: "verification", Entities: []string{"garden"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Assertions) != 0 {
		t.Fatalf("assertions=%v", resp.Assertions)
	}
	for _, source := range resp.Trace.SourceSet {
		if source == "world" || source == "governance" {
			t.Fatalf("retired source in trace: %v", resp.Trace.SourceSet)
		}
	}
}

func TestDeepRecallExpandsKGWithCapability(t *testing.T) {
	graph := &mockGraph{facts: []facade.GraphFact{{Predicate: "leads", Object: "alice", ValidFrom: "2026-01-01", Confidence: .9}}}
	resp, err := deepTestService(t, graph, &fakeSearcher{cards: []facade.MemoryCard{{ID: "mem_1", Kind: "fact"}}}).Recall(context.Background(), DeepRequest{Query: "test", TriggerReason: "verification", Capabilities: []string{"kg"}, Entities: []string{"garden"}})
	if err != nil || graph.calls != 1 || len(resp.Assertions) != 1 {
		t.Fatalf("resp=%+v err=%v calls=%d", resp, err, graph.calls)
	}
	if resp.Assertions[0].Subject != "garden" || resp.Assertions[0].Object != "alice" || resp.Assertions[0].Status != "active" {
		t.Fatalf("assertion=%+v", resp.Assertions[0])
	}
}

func TestDeepRecallTimelineExpansion(t *testing.T) {
	graph := &mockGraph{events: []facade.TimelineEvent{{Predicate: "released", Object: "v1", ValidFrom: "2026-01-15"}, {Predicate: "released", Object: "v2", ValidFrom: "2026-06-01"}}}
	resp, err := deepTestService(t, graph, &fakeSearcher{cards: []facade.MemoryCard{{ID: "mem_1", Kind: "fact"}}}).Recall(context.Background(), DeepRequest{Query: "releases", TriggerReason: "history", Capabilities: []string{"timeline"}, Entities: []string{"garden"}})
	if err != nil || graph.tlCalls != 1 || len(resp.Assertions) != 2 {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
}

func TestDeepRecallAlwaysEmitsTrace(t *testing.T) {
	svc := deepTestService(t, nil, &fakeSearcher{cards: []facade.MemoryCard{{ID: "mem_1", Kind: "fact"}}})
	resp, err := svc.Recall(context.Background(), DeepRequest{Query: "test", TriggerReason: "verification"})
	if err != nil || resp.Trace.TraceID == "" || resp.RecallTraceID == nil || *resp.RecallTraceID != resp.Trace.TraceID {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
	got, err := svc.Traces.Get(context.Background(), resp.Trace.TraceID)
	if err != nil || got.Query != "test" {
		t.Fatalf("trace=%+v err=%v", got, err)
	}
	if got.SourceSet[0] != "cards" && got.SourceSet[0] != "frozen_core" {
		t.Fatalf("source set=%v", got.SourceSet)
	}
}

func TestDeepRecallFallbackOnKGError(t *testing.T) {
	resp, err := deepTestService(t, &mockGraph{err: errors.New("kg timeout")}, &fakeSearcher{cards: []facade.MemoryCard{{ID: "mem_1", Kind: "fact"}}}).Recall(context.Background(), DeepRequest{Query: "test", TriggerReason: "verification", Capabilities: []string{"kg"}, Entities: []string{"garden"}})
	if err != nil || !resp.Trace.Degraded || len(resp.Warnings) == 0 || resp.Mode != "deep" {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
}

func TestDeepRecallFallbackOnSeedError(t *testing.T) {
	resp, err := deepTestService(t, nil, &fakeSearcher{cardErr: errors.New("mentle down")}).Recall(context.Background(), DeepRequest{Query: "test", TriggerReason: "verification"})
	if err != nil || !resp.Trace.Degraded {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
}

func TestDeepRecallBudgetAndPlanner(t *testing.T) {
	svc := deepTestService(t, nil, &fakeSearcher{})
	if _, err := svc.Recall(context.Background(), DeepRequest{Query: "test", TriggerReason: "verification", BudgetChars: 100}); err == nil {
		t.Fatal("expected budget error")
	}
	resp, err := svc.Recall(context.Background(), DeepRequest{Query: "test", TriggerReason: "verification", UsePlanner: true})
	if err != nil || resp.Trace.TraceID == "" {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
}
