//go:build e2e && external_e2e

package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dashimaki/garden/internal/rag"
)

func TestExternalPlannerEndToEnd(t *testing.T) {
	baseURL, key, model := os.Getenv("GARDEN_RAG_BASE_URL"), os.Getenv("GARDEN_RAG_API_KEY"), os.Getenv("GARDEN_RAG_MODEL")
	if baseURL == "" || key == "" || model == "" {
		t.Skip("external planner environment is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	planner := &rag.OpenAIPlanner{BaseURL: baseURL, APIKey: key, Model: model}
	plan, err := planner.Plan(ctx, rag.PlannerInput{Intent: "Find evidence about Garden pipeline governance", Governance: map[string]map[string]any{"01-identity": {"role": "governed assistant"}}})
	if err != nil {
		t.Fatalf("external planner: %v", err)
	}
	if len(plan.Queries) == 0 {
		t.Fatalf("external planner returned no queries: %+v", plan)
	}
}
