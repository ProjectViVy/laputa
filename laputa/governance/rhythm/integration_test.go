//go:build integration

package rhythm

import (
	"context"
	"os"
	"strings"
	"testing"

	laputa "github.com/dashimaki/laputa/governance"
)

func TestIntegration_RhythmDaily(t *testing.T) {
	keysPath := os.Getenv("LAPUTA_KEYS_PATH")
	if keysPath == "" {
		keysPath = `C:\Users\Administrator\Desktop\keys.txt`
	}
	lines, err := os.ReadFile(keysPath)
	if err != nil {
		t.Skipf("keys file not found: %v", err)
	}
	parts := strings.Split(strings.ReplaceAll(string(lines), "\r\n", "\n"), "\n")
	var apiKey, baseURL, model string
	for i, p := range parts {
		p = strings.TrimSpace(p)
		switch i {
		case 0:
			apiKey = p
		case 2:
			baseURL = p
		case 4:
			model = p
		}
	}
	if apiKey == "" {
		t.Skip("api key empty")
	}
	if baseURL == "" {
		baseURL = "https://api.stepfun.com/step_plan/v1"
	}
	if model == "" {
		model = "step-3.7-flash"
	}

	ctx := context.Background()
	store, _ := laputa.NewFileStore(t.TempDir())
	engine := laputa.NewEngine(store)
	_ = engine.Initialize(ctx)

	gen, err := NewOpenAIGenerator(ctx, baseURL, apiKey, model)
	if err != nil {
		t.Fatalf("generator: %v", err)
	}

	re := NewEngine(engine, gen, Config{})
	if err := re.Run(ctx, RhythmDaily); err != nil {
		t.Fatalf("run: %v", err)
	}

	daily, _ := engine.GetSection(ctx, laputa.SectionDaily)
	reports, ok := daily["reports"].([]any)
	if !ok || len(reports) == 0 {
		t.Errorf("expected report, got %v", daily)
	}
	first := reports[0].(map[string]any)
	if first["title"] == "" || first["summary"] == "" {
		t.Errorf("expected non-empty report, got %v", first)
	}
	t.Logf("report: %+v", first)
}
