package longmemeval

import (
	"testing"
)

// TestBuildSessionCache_DeduplicatesTexts verifies that identical session texts
// produce a single cache entry (not N copies).
func TestBuildSessionCache_DeduplicatesTexts(t *testing.T) {
	entries := []Entry{
		{
			HaystackSessionIDs: []string{"s1", "s2"},
			HaystackSessions: [][]any{
				{map[string]any{"role": "user", "content": "hello world"}},
				{map[string]any{"role": "user", "content": "hello world"}}, // duplicate
			},
		},
	}

	texts := collectUniqueSessionTexts(entries, "default", nil)
	if len(texts) != 1 {
		t.Errorf("expected 1 unique text, got %d", len(texts))
	}
}

// TestCollectUniqueSessionTexts_MultipleEntries verifies texts from multiple
// entries are deduplicated across entries.
func TestCollectUniqueSessionTexts_MultipleEntries(t *testing.T) {
	entries := []Entry{
		{
			HaystackSessionIDs: []string{"s1"},
			HaystackSessions: [][]any{
				{map[string]any{"role": "user", "content": "shared session text"}},
			},
		},
		{
			HaystackSessionIDs: []string{"s2"},
			HaystackSessions: [][]any{
				{map[string]any{"role": "user", "content": "shared session text"}},
			},
		},
		{
			HaystackSessionIDs: []string{"s3"},
			HaystackSessions: [][]any{
				{map[string]any{"role": "user", "content": "unique text"}},
			},
		},
	}

	texts := collectUniqueSessionTexts(entries, "default", nil)
	if len(texts) != 2 {
		t.Errorf("expected 2 unique texts, got %d: %v", len(texts), texts)
	}
}
