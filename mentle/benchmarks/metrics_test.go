package benchmarks

import (
	"math"
	"testing"
)

// --- NDCG tests ---

// When the correct item is ranked 1st in a 1-item corpus, NDCG@k should be 1.0.
func TestNDCG_PerfectRank(t *testing.T) {
	retrieved := []string{"a", "b", "c"}
	correct := []string{"a"}
	corpus := []string{"a", "b", "c"}
	got := NDCG(retrieved, correct, corpus, 5)
	if math.Abs(got-1.0) > 1e-9 {
		t.Errorf("expected 1.0, got %f", got)
	}
}

// When the correct item is not in the top-K window, NDCG@k should be 0.
func TestNDCG_CorrectItemOutsideTopK(t *testing.T) {
	retrieved := []string{"b", "c", "d", "e", "f", "a"}
	correct := []string{"a"}
	corpus := []string{"a", "b", "c", "d", "e", "f"}
	got := NDCG(retrieved, correct, corpus, 5)
	if got != 0.0 {
		t.Errorf("expected 0.0 when correct item is outside top-K, got %f", got)
	}
}

// When there are 2 correct items in the corpus but none are retrieved in top-K,
// IDCG should reflect 2 perfect hits (not 0), so NDCG = 0 (not divide-by-zero returning 0).
// This tests that IDCG is built from the number of relevant docs in corpus, not from retrieved.
func TestNDCG_IDCGBuiltFromCorpusNotRetrieved(t *testing.T) {
	// 2 correct items exist in corpus; neither appears in the top-5 retrieved
	retrieved := []string{"c", "d", "e", "f", "g"}
	correct := []string{"a", "b"}
	corpus := []string{"a", "b", "c", "d", "e", "f", "g"}
	got := NDCG(retrieved, correct, corpus, 5)
	// IDCG@5 for 2 relevant docs = 1/log2(2) + 1/log2(3) > 0, so result should be 0.0 (not NaN or panic)
	if got != 0.0 {
		t.Errorf("expected 0.0 when no correct items retrieved, got %f", got)
	}
}

// When 1 of 2 correct items is at rank 2 (not rank 1), NDCG should be < 1.
// If IDCG only uses retrieved relevances (the bug), the "ideal" would just sort
// [0,1] -> [1,0], giving the same DCG as [0,1] (they're different), so NDCG
// would equal DCG([0,1]) / DCG([1,0]) which is < 1. But if IDCG uses the full
// 2 relevant docs, ideal is 2 hits at ranks 1,2, giving higher IDCG and lower NDCG.
func TestNDCG_PartialRetrievalWithTwoRelevantInCorpus(t *testing.T) {
	retrieved := []string{"c", "a", "d"} // 'a' at rank 2, 'b' not retrieved
	correct := []string{"a", "b"}
	corpus := []string{"a", "b", "c", "d"}
	got := NDCG(retrieved, correct, corpus, 5)

	// DCG: rel=[0,1,0], dcg = 0 + 1/log2(3) + 0 = 0.6309...
	// IDCG (correct): 2 relevant docs in corpus, ideal=[1,1,0,...], idcg = 1/log2(2)+1/log2(3) = 1+0.6309 = 1.6309
	// NDCG = 0.6309/1.6309 = 0.3869...
	// Bug version IDCG: only sorts retrieved relevances [0,1,0] -> [1,0,0], idcg=1/log2(2)=1.0
	// Bug version NDCG = 0.6309/1.0 = 0.6309 (inflated)
	expectedCorrect := 0.6309 / (1.0 + 0.6309)
	expectedBug := 0.6309 / 1.0

	if math.Abs(got-expectedBug) < 1e-3 && math.Abs(got-expectedCorrect) > 1e-3 {
		t.Errorf("NDCG appears to use bug IDCG: got %f (bug value ~%f, correct value ~%f)",
			got, expectedBug, expectedCorrect)
	}
	if math.Abs(got-expectedCorrect) > 1e-3 {
		t.Errorf("expected NDCG ~%f, got %f", expectedCorrect, got)
	}
}

// --- RecallAny tests (sanity) ---

func TestRecallAny_Hit(t *testing.T) {
	if RecallAny([]string{"a", "b"}, []string{"b", "c"}) != 1.0 {
		t.Error("expected hit")
	}
}

func TestRecallAny_Miss(t *testing.T) {
	if RecallAny([]string{"a", "b"}, []string{"c", "d"}) != 0.0 {
		t.Error("expected miss")
	}
}

// --- RecallAll tests ---

func TestRecallAll_AllPresent(t *testing.T) {
	if RecallAll([]string{"a", "b", "c"}, []string{"a", "b"}) != 1.0 {
		t.Error("expected 1.0")
	}
}

func TestRecallAll_SomeMissing(t *testing.T) {
	if RecallAll([]string{"a", "c"}, []string{"a", "b"}) != 0.0 {
		t.Error("expected 0.0 when one correct item missing")
	}
}
