package pipeline

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

type testStep struct {
	id       string
	contract Contract
	run      func(State) (Result, error)
}
type testInterceptor struct{ before, after, failed int }

func (i *testInterceptor) BeforeStep(context.Context, Step, State) error { i.before++; return nil }
func (i *testInterceptor) AfterStep(context.Context, Step, State, Result) error {
	i.after++
	return nil
}
func (i *testInterceptor) OnError(_ context.Context, _ Step, _ State, err error) error {
	i.failed++
	return err
}

func (s testStep) ID() string                                         { return s.id }
func (s testStep) Contract() Contract                                 { return s.contract }
func (s testStep) Run(_ context.Context, state State) (Result, error) { return s.run(state) }

func TestRunnerContractsAndLoop(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Pipelines[0].Steps = []StepConfig{{ID: "a", Next: []string{"a"}}, {ID: "b"}}
	cfg.Pipelines[0].Capabilities = []string{"read"}
	cfg.Pipelines[0].MaxSteps = 4
	cfg.Pipelines[0].MaxVisitsPerStep = 2
	m, err := NewManager(cfg.Pipelines, "test")
	if err != nil {
		t.Fatal(err)
	}
	visits := 0
	_ = m.Register(testStep{id: "a", contract: Contract{Produces: []string{"ready"}, Capabilities: []string{"read"}}, run: func(s State) (Result, error) {
		visits++
		s["ready"] = true
		if visits == 1 {
			return Result{Next: "a"}, nil
		}
		return Result{}, nil
	}})
	_ = m.Register(testStep{id: "b", contract: Contract{Requires: []string{"ready"}}, run: func(s State) (Result, error) { s["done"] = true; return Result{}, nil }})
	trace, err := m.Run(context.Background(), "agentic_recall_v1", "trace", State{})
	if err != nil {
		t.Fatal(err)
	}
	if trace.Status != "ok" || visits != 2 {
		t.Fatalf("trace=%+v visits=%d", trace, visits)
	}
}

func TestRunnerRejectsCapability(t *testing.T) {
	d := Definition{Name: "p", Version: "1", Capabilities: []string{"read"}, MaxSteps: 2, MaxVisitsPerStep: 1, Steps: []StepConfig{{ID: "write"}}}
	m, _ := NewManager([]Definition{d}, "r")
	_ = m.Register(testStep{id: "write", contract: Contract{Capabilities: []string{"write"}}, run: func(State) (Result, error) { return Result{}, nil }})
	if _, err := m.Run(context.Background(), "p", "", State{}); err == nil {
		t.Fatal("expected capability rejection")
	}
}

func TestIdempotentStepRetriesOnce(t *testing.T) {
	d := Definition{Name: "p", Version: "1", Capabilities: []string{"read"}, MaxSteps: 2, MaxVisitsPerStep: 1, Steps: []StepConfig{{ID: "read", OnError: "retry"}}}
	m, _ := NewManager([]Definition{d}, "r")
	calls := 0
	_ = m.Register(testStep{id: "read", contract: Contract{Capabilities: []string{"read"}, Idempotent: true}, run: func(State) (Result, error) {
		calls++
		if calls == 1 {
			return Result{}, errors.New("transient")
		}
		return Result{}, nil
	}})
	if _, err := m.Run(context.Background(), "p", "", State{}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestExampleConfigLoads(t *testing.T) {
	cfg, revision, err := LoadConfig(filepath.Join("..", "..", "config", "pipelines.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if revision == "" || len(cfg.Pipelines) != 1 || cfg.Pipelines[0].Name != "agentic_recall_v1" {
		t.Fatalf("config=%+v revision=%q", cfg, revision)
	}
}

func TestInterceptorsWrapStepExecution(t *testing.T) {
	d := Definition{Name: "p", Version: "1", MaxSteps: 1, MaxVisitsPerStep: 1, Steps: []StepConfig{{ID: "a"}}}
	m, _ := NewManager([]Definition{d}, "r")
	_ = m.Register(testStep{id: "a", run: func(State) (Result, error) { return Result{}, nil }})
	interceptor := &testInterceptor{}
	m.Use(interceptor)
	if _, err := m.Run(context.Background(), "p", "", State{}); err != nil {
		t.Fatal(err)
	}
	if interceptor.before != 1 || interceptor.after != 1 || interceptor.failed != 0 {
		t.Fatalf("interceptor=%+v", interceptor)
	}
}
