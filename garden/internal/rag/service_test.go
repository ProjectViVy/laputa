package rag

import (
	"context"
	"errors"
	"testing"

	"github.com/dashimaki/garden/internal/pipeline"
	"github.com/dashimaki/laputa/governance"
	"github.com/dashimaki/mentle/facade"
)

type fakeGovernance struct {
	sections map[governance.SectionName]map[string]any
	reads    int
}

func (f *fakeGovernance) GetSection(_ context.Context, name governance.SectionName) (map[string]any, error) {
	f.reads++
	if value := f.sections[name]; value != nil {
		return value, nil
	}
	return map[string]any{}, nil
}

type fakeRetriever struct{ queries []string }

func (f *fakeRetriever) Retrieve(_ context.Context, q facade.RetrievalQuery) ([]facade.RetrievalHit, error) {
	f.queries = append(f.queries, q.Text)
	return []facade.RetrievalHit{{ID: "1", Content: "Garden uses governed pipelines", Wing: "technical", Room: "architecture", Score: .016, Channels: []string{"vector", "bm25"}}, {ID: "2", Content: "secret", Wing: "private", Room: "hidden", Score: .02, Channels: []string{"vector"}}}, nil
}
func (f *fakeRetriever) QueryEntity(context.Context, string, string, string) ([]facade.GraphFact, error) {
	return []facade.GraphFact{{Predicate: "uses", Object: "Mentle", Confidence: .9}}, nil
}
func (f *fakeRetriever) Timeline(context.Context, string) ([]facade.TimelineEvent, error) {
	return nil, nil
}

type fakePlanner struct{}

func (fakePlanner) Plan(context.Context, PlannerInput) (RetrievalPlan, error) {
	return RetrievalPlan{Queries: []string{"garden architecture"}, Entities: []string{"Garden"}}, nil
}
func (fakePlanner) Refine(context.Context, PlannerInput) (RetrievalPlan, error) {
	return RetrievalPlan{Stop: true}, nil
}
func (fakePlanner) Summarize(_ context.Context, _ string, e []Evidence) (string, error) {
	if len(e) == 0 {
		return "", nil
	}
	return "Governed context [" + e[0].ID + "]", nil
}

func TestResolveUsesGovernanceAndFiltersDeniedMemory(t *testing.T) {
	gov := &fakeGovernance{sections: map[governance.SectionName]map[string]any{
		governance.SectionIdentity:   {"role": "assistant"},
		governance.SectionCommitment: {"agentic_rag": map[string]any{"denied_wings": []any{"private"}}},
	}}
	cfg := pipeline.DefaultConfig()
	manager, err := pipeline.NewManager(cfg.Pipelines, "test")
	if err != nil {
		t.Fatal(err)
	}
	retriever := &fakeRetriever{}
	service, err := NewService(manager, PolicyResolver{Governance: gov}, retriever, fakePlanner{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Resolve(context.Background(), ResolveRequest{Intent: "How is Garden designed?"})
	if err != nil {
		t.Fatal(err)
	}
	if result.TraceID == "" || result.Context == "" || len(result.Evidence) == 0 {
		t.Fatalf("incomplete result: %+v", result)
	}
	if result.Mode != "advanced" {
		t.Fatalf("mode=%q", result.Mode)
	}
	for _, evidence := range result.Evidence {
		if evidence.Locator == "private/hidden/2" {
			t.Fatal("denied memory leaked")
		}
	}
	if len(retriever.queries) != 1 || retriever.queries[0] != "garden architecture" {
		t.Fatalf("queries=%v", retriever.queries)
	}
	if gov.reads != len(governanceSections) {
		t.Fatalf("governance reads=%d", gov.reads)
	}
}

func TestBasicResolveSkipsPlannerAndReturnsBasicMode(t *testing.T) {
	gov := &fakeGovernance{sections: map[governance.SectionName]map[string]any{governance.SectionIdentity: {"role": "assistant"}}}
	cfg := pipeline.DefaultConfig()
	manager, err := pipeline.NewManager(cfg.Pipelines, "test")
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(manager, PolicyResolver{Governance: gov}, &fakeRetriever{}, fakePlanner{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Resolve(context.Background(), ResolveRequest{Intent: "Garden", Mode: "basic"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != "basic" || result.TraceID == "" || result.Context == "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestResolvePolicyDefaultsAndOverrides(t *testing.T) {
	p := ResolvePolicy(map[string]map[string]any{string(governance.SectionIdentity): {"agentic_rag": map[string]any{"allowed_capabilities": []any{"hybrid"}}}, string(governance.SectionCommitment): {"agentic_rag": map[string]any{"external_context": "query_only", "denied_sources": []any{"diary"}}}})
	if !p.AllowedCapabilities["hybrid"] || p.AllowedCapabilities["llm"] {
		t.Fatalf("capabilities=%v", p.AllowedCapabilities)
	}
	if p.ExternalContext != "query_only" || !p.DeniedSources["diary"] {
		t.Fatalf("policy=%+v", p)
	}
}

func TestInvalidCitationFallsBack(t *testing.T) {
	evidence := []Evidence{{ID: "ev_001"}}
	if validCitations("claim [ev_999]", evidence) {
		t.Fatal("invalid citation accepted")
	}
	if !validCitations("claim [ev_001]", evidence) {
		t.Fatal("valid citation rejected")
	}
}

func TestDecodePlanToleratesProviderStringBooleans(t *testing.T) {
	plan, err := decodePlan([]byte(`{"queries":"garden","entities":["Garden"],"temporal":"false","stop":"true"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Queries) != 1 || plan.Queries[0] != "garden" || plan.Temporal || !plan.Stop {
		t.Fatalf("plan=%+v", plan)
	}
}

type failingPlanner struct{ fakePlanner }

func (failingPlanner) Plan(context.Context, PlannerInput) (RetrievalPlan, error) {
	return RetrievalPlan{}, errors.New("down")
}
func TestFallbackPlannerReportsPrimaryFailure(t *testing.T) {
	planner := FallbackPlanner{Primary: failingPlanner{}, Fallback: RulePlanner{}}
	plan, err := planner.Plan(context.Background(), PlannerInput{Intent: "garden"})
	if err == nil || len(plan.Queries) != 1 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}
