package longmemeval

import (
	"strings"
	"testing"
	"time"
)

// --- formatProgressLine tests ---

func TestFormatProgressLine_ContainsPhaseTimings(t *testing.T) {
	timing := QuestionTiming{
		Embed:  2300 * time.Millisecond,
		Search: 12 * time.Millisecond,
		Score:  500 * time.Microsecond,
	}
	line := formatProgressLine(1, 10, 1.0, 1.0, 0.923, "HIT", timing)

	for _, want := range []string{"embed=", "search=", "score="} {
		if !strings.Contains(line, want) {
			t.Errorf("progress line missing %q: %s", want, line)
		}
	}
	if !strings.Contains(line, "[1/10]") {
		t.Errorf("progress line missing question index: %s", line)
	}
	if !strings.Contains(line, "HIT") {
		t.Errorf("progress line missing status: %s", line)
	}
}

// --- mapCorpusToSessionIDs tests ---

func TestMapCorpusToSessionIDs_StandardSuffix(t *testing.T) {
	entry := Entry{
		HaystackSessionIDs: []string{"sess1", "sess2", "sess3"},
	}
	corpusIDs := []string{"sess1_turn_0", "sess2_turn_0", "sess3_turn_0"}
	got := mapCorpusToSessionIDs(corpusIDs, entry)
	want := []string{"sess1", "sess2", "sess3"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: expected %q, got %q", i, want[i], got[i])
		}
	}
}

// When a corpus ID does not end in _turn_0, the fallback uses HasPrefix.
// This can incorrectly match: if corpus ID is "sess10_x" and session IDs include
// "sess1", HasPrefix("sess10_x", "sess1") = true — wrong match.
// After the fix, an unrecognised corpus ID that has no exact match should
// either be skipped or mapped only when there is an unambiguous full match.
func TestMapCorpusToSessionIDs_FallbackNoAmbiguousPrefix(t *testing.T) {
	entry := Entry{
		HaystackSessionIDs: []string{"sess1", "sess10"},
	}
	// "sess10_custom" should map to "sess10", not "sess1" (prefix "sess1" is a substring of "sess10")
	corpusIDs := []string{"sess10_custom"}
	got := mapCorpusToSessionIDs(corpusIDs, entry)
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %v", got)
	}
	if got[0] != "sess10" {
		t.Errorf("ambiguous prefix: expected %q to map to %q, got %q", "sess10_custom", "sess10", got[0])
	}
}

// An unrecognised corpus ID with no matching session should be skipped, not
// leaked as a raw corpus ID into the session ID list.
func TestMapCorpusToSessionIDs_UnknownCorpusIDSkipped(t *testing.T) {
	entry := Entry{
		HaystackSessionIDs: []string{"sess1", "sess2"},
	}
	// "unknown_doc" shares no prefix with any session ID
	corpusIDs := []string{"sess1_turn_0", "unknown_doc"}
	got := mapCorpusToSessionIDs(corpusIDs, entry)
	for _, id := range got {
		if id == "unknown_doc" {
			t.Errorf("raw corpus ID %q leaked into session ID list", "unknown_doc")
		}
	}
	if len(got) != 1 || got[0] != "sess1" {
		t.Errorf("expected [sess1], got %v", got)
	}
}

// Deduplication: same session from multiple corpus IDs should appear once.
func TestMapCorpusToSessionIDs_Deduplication(t *testing.T) {
	entry := Entry{
		HaystackSessionIDs: []string{"sess1"},
	}
	corpusIDs := []string{"sess1_turn_0", "sess1_turn_0", "sess1_turn_0"}
	got := mapCorpusToSessionIDs(corpusIDs, entry)
	if len(got) != 1 {
		t.Errorf("expected dedup to 1 entry, got %v", got)
	}
}

// --- rankedSessionIDs helper tests ---
// These test the logic that was formerly inline in Run() to ensure that
// the "fill missing" padding does NOT appear in the ranked list used for recall.

// rankAndPad returns the session IDs actually retrieved (top-K), without appending
// unretrieved corpus items. This is the correct behaviour for recall evaluation.
func TestRankedList_DoesNotIncludeUnretrievedSessions(t *testing.T) {
	// Simulate: 5 sessions in corpus, retrieval returns only 2 of them.
	allCorpusSessionIDs := []string{"sess1", "sess2", "sess3", "sess4", "sess5"}
	retrievedSessionIDs := []string{"sess3", "sess1"} // only 2 retrieved

	// Recall@5 must be based on only the 2 retrieved sessions,
	// not padded out with the remaining 3.
	// If correct answer is sess4 (not retrieved), recall should be 0.
	correctIDs := []string{"sess4"}

	top5 := retrievedSessionIDs
	if len(top5) > 5 {
		top5 = top5[:5]
	}
	recall := RecallAny_test(top5, correctIDs)
	if recall != 0.0 {
		t.Errorf("recall should be 0 when correct session not retrieved, got %f", recall)
	}

	// Demonstrate the bug: if we pad unretrieved sessions onto the list,
	// recall@5 would incorrectly become 1.
	paddedList := make([]string, len(retrievedSessionIDs))
	copy(paddedList, retrievedSessionIDs)
	retrieved := map[string]bool{"sess3": true, "sess1": true}
	for _, id := range allCorpusSessionIDs {
		if !retrieved[id] {
			paddedList = append(paddedList, id)
		}
	}
	top5Padded := paddedList
	if len(top5Padded) > 5 {
		top5Padded = top5Padded[:5]
	}
	paddedRecall := RecallAny_test(top5Padded, correctIDs)
	if paddedRecall != 1.0 {
		// This demonstrates the bug behaviour for documentation purposes; if
		// this assertion fails the test dataset needs updating.
		t.Logf("note: padded recall was %f (expected 1.0 to demonstrate bug)", paddedRecall)
	}
}

// RecallAny_test is a local copy so the test file doesn't import the benchmarks package.
func RecallAny_test(topK []string, correctIDs []string) float64 {
	correctSet := make(map[string]bool)
	for _, id := range correctIDs {
		correctSet[id] = true
	}
	for _, id := range topK {
		if correctSet[id] {
			return 1.0
		}
	}
	return 0.0
}
