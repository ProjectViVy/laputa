package rhythm

import (
	"context"
	"testing"

	laputa "github.com/dashimaki/laputa/governance"
)

func TestRhythmEngineRun_Daily(t *testing.T) {
	ctx := context.Background()
	store, _ := laputa.NewFileStore(t.TempDir())
	engine := laputa.NewEngine(store)
	_ = engine.Initialize(ctx)

	rhythm := NewEngine(engine, NewMockGenerator(), Config{})
	err := rhythm.Run(ctx, RhythmDaily)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	daily, _ := engine.GetSection(ctx, laputa.SectionDaily)
	reports, ok := daily["reports"].([]any)
	if !ok || len(reports) == 0 {
		t.Errorf("expected daily report to be written, got %v", daily)
	}
}
