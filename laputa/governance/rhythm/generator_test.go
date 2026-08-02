package rhythm

import (
	"context"
	"testing"
)

func TestGeneratorGenerate_Mock(t *testing.T) {
	gen := NewMockGenerator()
	result, err := gen.Generate(context.Background(), RhythmDaily, "test prompt")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if result.Title == "" {
		t.Error("expected non-empty title")
	}
}
