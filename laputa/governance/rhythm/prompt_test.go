package rhythm

import (
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	snapshot := map[string]any{
		"identity": map[string]any{
			"role": "assistant",
		},
		"memory_md": map[string]any{
			"summary": "user prefers concise output",
		},
		"history_md": map[string]any{
			"timeline": []map[string]any{
				{"event": "started project"},
			},
		},
	}

	prompt := BuildPrompt(RhythmDaily, snapshot)
	if !strings.Contains(prompt, "daily") {
		t.Errorf("prompt should mention rhythm kind")
	}
	if !strings.Contains(prompt, "concise output") {
		t.Errorf("prompt should include memory_md summary")
	}
}
