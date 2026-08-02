// Package longmemeval implements the LongMemEval benchmark for long-term memory retrieval.
package longmemeval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/dashimaki/mentle/benchmarks"
	"github.com/dashimaki/mentle/internal/dialect"
	"github.com/dashimaki/mentle/internal/embedder"
	"github.com/dashimaki/mentle/internal/search"
	govector "github.com/dashimaki/mentle/storage/govector"
)

type Entry struct {
	QuestionID         string   `json:"question_id"`
	QuestionType       string   `json:"question_type"`
	Question           string   `json:"question"`
	Answer             any      `json:"answer"`
	AnswerSessionIDs   []string `json:"answer_session_ids"`
	QuestionDate       string   `json:"question_date"`
	HaystackSessions   [][]any  `json:"haystack_sessions"`
	HaystackSessionIDs []string `json:"haystack_session_ids"`
	HaystackDates      []string `json:"haystack_dates"`
}

type Result struct {
	Mode        string                `json:"mode"`
	Granularity string                `json:"granularity"`
	TotalQ      int                   `json:"total_questions"`
	Recall5     float64               `json:"recall_5"`
	Recall10    float64               `json:"recall_10"`
	NDCG10      float64               `json:"ndcg_10"`
	PerType     map[string]TypeResult `json:"per_type"`
	PerK        map[int]KResult       `json:"per_k"`
}

type TypeResult struct {
	Count  int     `json:"count"`
	R5     float64 `json:"r5"`
	R10    float64 `json:"r10"`
	NDCG10 float64 `json:"ndcg_10"`
}

type KResult struct {
	RecallAny float64 `json:"recall_any"`
	RecallAll float64 `json:"recall_all"`
	NDCG      float64 `json:"ndcg"`
}

var evalKs = []int{1, 3, 5, 10, 30, 50}

// QuestionTiming holds per-phase wall-clock durations for a single benchmark question.
type QuestionTiming struct {
	Embed  time.Duration // corpus build: batch embedding + vector store writes
	Search time.Duration // vector similarity search
	Score  time.Duration // recall/NDCG scoring
}

// formatProgressLine returns the progress string printed after each question.
func formatProgressLine(idx, total int, r5, r10, ndcg10 float64, status string, t QuestionTiming) string {
	return fmt.Sprintf("[%d/%d] embed=%v search=%v score=%v R@5=%.0f R@10=%.0f NDCG@10=%.3f %s",
		idx, total, t.Embed.Round(time.Millisecond), t.Search.Round(time.Millisecond), t.Score.Round(time.Microsecond),
		r5, r10, ndcg10, status)
}

func Run(dataFile string, mode string, granularity string, limit int, skip int, topK int) (*Result, error) {
	ctx := context.Background()
	data, err := os.ReadFile(dataFile)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	if skip > 0 && skip <= len(entries) {
		entries = entries[skip:]
	}
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	result := &Result{
		Mode: mode, Granularity: granularity,
		PerType: make(map[string]TypeResult),
		PerK:    make(map[int]KResult),
	}
	for _, k := range evalKs {
		result.PerK[k] = KResult{}
	}

	encoder := dialect.NewEncoder()
	totalQ := 0
	hits5 := 0
	hits10 := 0
	totalNDCG10 := 0.0
	perTypeResults := make(map[string]*TypeResult)

	// embedWorkers is the number of parallel corpus-embedding goroutines.
	// Each worker loads its own hugot session (~90 MB RAM). Set to 1 to disable.
	const embedWorkers = 4

	// Create a pool of embedders — one per worker. Loading the ONNX model is
	// done once per worker here, amortised across all questions.
	fmt.Fprintf(os.Stderr, "Loading %d embedder(s)...\n", embedWorkers)
	embs := make([]*embedder.Embedder, embedWorkers)
	for i := range embs {
		e, err := embedder.New("", "")
		if err != nil {
			for j := range i {
				embs[j].Close()
			}
			return nil, fmt.Errorf("create embedder %d: %w", i, err)
		}
		embs[i] = e
	}
	defer func() {
		for _, e := range embs {
			e.Close()
		}
	}()

	// The search embedder is the first worker's embedder (search is serial).
	searchEmb := embs[0]

	// Pre-compute a cross-question session embedding cache. Many sessions appear
	// in multiple questions; embedding them once and reusing the vectors avoids
	// redundant GEMM work across the benchmark run.
	fmt.Fprintf(os.Stderr, "Pre-computing session embedding cache...\n")
	cacheStart := time.Now()
	sessionCache, err := BuildSessionCache(ctx, embs, entries, mode, encoder)
	if err != nil {
		return nil, fmt.Errorf("build session cache: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Cache built: %d unique sessions in %v\n", len(sessionCache), time.Since(cacheStart).Round(time.Millisecond))

	type corpusResult struct {
		store     *govector.Store
		corpusIDs []string
		embedDur  time.Duration
		entry     Entry
	}

	// Phase 1: build all corpora in parallel across the worker pool.
	// Each worker reads from the shared sessionCache (read-only, safe).
	embedStart := time.Now()
	cResults, err := runWorkerPoolWithResource(entries, embs,
		func(emb *embedder.Embedder, entry Entry) (corpusResult, error) {
			t0 := time.Now()
			store, ids, err := buildCorpus(ctx, emb, entry, mode, encoder, sessionCache)
			dur := time.Since(t0)
			return corpusResult{store: store, corpusIDs: ids, embedDur: dur, entry: entry}, err
		},
	)
	totalEmbedWall := time.Since(embedStart)
	if err != nil {
		return nil, fmt.Errorf("parallel corpus build: %w", err)
	}
	_ = totalEmbedWall // available for summary logging if desired

	// Phase 2: search + score serially (fast — dominated by embed).
	for _, cr := range cResults {
		entry := cr.entry
		if cr.store == nil {
			fmt.Fprintf(os.Stderr, "Error building corpus for entry %s\n", entry.QuestionID)
			continue
		}
		if len(cr.corpusIDs) == 0 {
			fmt.Fprintf(os.Stderr, "Warning: entry %s has 0 corpus IDs\n", entry.QuestionID)
		}

		t1 := time.Now()
		searcher := search.NewSearcher(cr.store, searchEmb)
		results, err := searcher.Search(ctx, entry.Question, "", "", topK)
		searchDur := time.Since(t1)
		cr.store.Close() // release file handles and temp storage after each entry
		if err != nil {
			continue
		}

		t2 := time.Now()
		var retrievedIDs []string
		for _, r := range results {
			retrievedIDs = append(retrievedIDs, r.Metadata["corpus_id"])
		}

		retrievedSessionIDs := mapCorpusToSessionIDs(retrievedIDs, entry)
		top5 := retrievedSessionIDs
		if len(top5) > 5 {
			top5 = top5[:5]
		}
		top10 := retrievedSessionIDs
		if len(top10) > 10 {
			top10 = top10[:10]
		}

		r5 := benchmarks.RecallAny(top5, entry.AnswerSessionIDs)
		r10 := benchmarks.RecallAny(top10, entry.AnswerSessionIDs)
		// Use session IDs (not raw corpus IDs) for NDCG so relevance comparisons
		// against entry.AnswerSessionIDs (which are session IDs) work correctly.
		allHaystackSessionIDs := mapCorpusToSessionIDs(cr.corpusIDs, entry)
		ndcg10 := benchmarks.NDCG(retrievedSessionIDs, entry.AnswerSessionIDs, allHaystackSessionIDs, 10)
		scoreDur := time.Since(t2)

		if r5 > 0 {
			hits5++
		}
		if r10 > 0 {
			hits10++
		}
		totalNDCG10 += ndcg10
		totalQ++

		for _, k := range evalKs {
			topKIDs := retrievedSessionIDs
			if k < len(topKIDs) {
				topKIDs = topKIDs[:k]
			}
			rAny := benchmarks.RecallAny(topKIDs, entry.AnswerSessionIDs)
			rAll := benchmarks.RecallAll(topKIDs, entry.AnswerSessionIDs)
			ndcg := benchmarks.NDCG(retrievedSessionIDs, entry.AnswerSessionIDs, allHaystackSessionIDs, k)
			kr := result.PerK[k]
			kr.RecallAny += rAny
			kr.RecallAll += rAll
			kr.NDCG += ndcg
			result.PerK[k] = kr
		}

		tr, exists := perTypeResults[entry.QuestionType]
		if !exists {
			tr = &TypeResult{}
			perTypeResults[entry.QuestionType] = tr
		}
		tr.Count++
		tr.R5 += r5
		tr.R10 += r10
		tr.NDCG10 += ndcg10

		status := "miss"
		if r5 > 0 {
			status = "HIT"
		}
		timing := QuestionTiming{Embed: cr.embedDur, Search: searchDur, Score: scoreDur}
		fmt.Println(formatProgressLine(totalQ, len(entries), r5, r10, ndcg10, status, timing))
	}

	result.TotalQ = totalQ
	if totalQ > 0 {
		result.Recall5 = float64(hits5) / float64(totalQ) * 100
		result.Recall10 = float64(hits10) / float64(totalQ) * 100
		result.NDCG10 = totalNDCG10 / float64(totalQ)
		for typ, tr := range perTypeResults {
			if tr.Count > 0 {
				result.PerType[typ] = TypeResult{
					Count: tr.Count, R5: tr.R5 / float64(tr.Count) * 100,
					R10: tr.R10 / float64(tr.Count) * 100, NDCG10: tr.NDCG10 / float64(tr.Count),
				}
			}
		}
		for k := range result.PerK {
			kr := result.PerK[k]
			kr.RecallAny /= float64(totalQ)
			kr.RecallAll /= float64(totalQ)
			kr.NDCG /= float64(totalQ)
			result.PerK[k] = kr
		}
	}
	return result, nil
}

func buildCorpus(ctx context.Context, emb *embedder.Embedder, entry Entry, mode string, encoder *dialect.Encoder, cache map[string][]float32) (*govector.Store, []string, error) {
	store, err := govector.NewStore(filepath.Join(os.TempDir(), fmt.Sprintf("bench_%d", time.Now().UnixNano())), 384)
	if err != nil {
		return nil, nil, err
	}

	// First pass: collect all texts
	type sessionInfo struct {
		sessionID string
		text      string
	}
	var sessions []sessionInfo

	for sessIdx, session := range entry.HaystackSessions {
		if sessIdx >= len(entry.HaystackSessionIDs) {
			continue
		}
		sessionID := entry.HaystackSessionIDs[sessIdx]

		var userTurns []string
		for _, turnAny := range session {
			turn, _ := turnAny.(map[string]any)
			if turn == nil {
				continue
			}
			if role, ok := turn["role"].(string); ok && role == "user" {
				if content, ok := turn["content"].(string); ok {
					userTurns = append(userTurns, content)
				}
			}
		}

		if len(userTurns) == 0 {
			continue
		}

		sessionText := strings.Join(userTurns, " ")
		if mode == "aaak" {
			sessionText = encoder.Compress(sessionText, map[string]string{})
		}
		sessions = append(sessions, sessionInfo{sessionID: sessionID, text: sessionText})
	}

	if len(sessions) == 0 {
		return store, nil, nil
	}

	// Second pass: resolve vectors — use cache if available, embed the rest.
	vectors := make([][]float32, len(sessions))
	var toEmbed []int // indices that still need embedding
	for i, s := range sessions {
		if cache != nil {
			if v, ok := cache[s.text]; ok {
				vectors[i] = v
				continue
			}
		}
		toEmbed = append(toEmbed, i)
	}
	if len(toEmbed) > 0 {
		missingTexts := make([]string, len(toEmbed))
		for j, idx := range toEmbed {
			missingTexts[j] = sessions[idx].text
		}
		vecs, err := emb.CreateEmbeddings(ctx, missingTexts)
		if err != nil {
			return nil, nil, fmt.Errorf("batch embed: %w", err)
		}
		for j, idx := range toEmbed {
			vectors[idx] = vecs[j]
		}
	}

	// Third pass: store vectors
	var corpusIDs []string
	points := make([]govector.Point, len(vectors))
	for i, vec := range vectors {
		corpusID := fmt.Sprintf("%s_turn_0", sessions[i].sessionID)
		corpusIDs = append(corpusIDs, corpusID)
		points[i] = govector.Point{
			ID:     fmt.Sprintf("doc_%d", i),
			Vector: vec,
			Payload: map[string]any{
				"corpus_id": corpusID, "session_id": sessions[i].sessionID,
				"sess_idx": i, "content": sessions[i].text,
			},
		}
	}
	if err := store.AddBatch(points); err != nil {
		return nil, nil, err
	}

	return store, corpusIDs, nil
}

func mapCorpusToSessionIDs(corpusIDs []string, entry Entry) []string {
	var sessionIDs []string
	seen := make(map[string]bool)
	for _, cid := range corpusIDs {
		// Fast path: corpus ID is sessionID + "_turn_0" (the format buildCorpus uses).
		sessionID := strings.TrimSuffix(cid, "_turn_0")
		if sessionID != cid {
			// Suffix was stripped — verify the resulting session ID actually exists.
			found := slices.Contains(entry.HaystackSessionIDs, sessionID)
			if !found {
				// Stripped form isn't a known session; fall through to full scan.
				sessionID = cid
			}
		}

		if sessionID == cid {
			// Fallback: find the longest session ID that is a prefix of this corpus ID.
			// Using the longest match avoids ambiguity (e.g. "sess1" vs "sess10").
			best := ""
			for _, s := range entry.HaystackSessionIDs {
				if strings.HasPrefix(cid, s) && len(s) > len(best) {
					best = s
				}
			}
			if best == "" {
				// No session matches this corpus ID — skip it rather than leaking
				// the raw corpus ID into the session list.
				continue
			}
			sessionID = best
		}

		if !seen[sessionID] {
			seen[sessionID] = true
			sessionIDs = append(sessionIDs, sessionID)
		}
	}
	return sessionIDs
}

func PrintResults(result *Result) {
	fmt.Println("\n=== LongMemEval Results ===")
	fmt.Printf("Mode: %s, Granularity: %s\n", result.Mode, result.Granularity)
	fmt.Printf("Total Questions: %d\n", result.TotalQ)
	fmt.Printf("Recall@5:  %.1f%%\n", result.Recall5)
	fmt.Printf("Recall@10: %.1f%%\n", result.Recall10)
	fmt.Printf("NDCG@10:   %.3f\n", result.NDCG10)

	fmt.Println("\nPer-K Results:")
	for _, k := range evalKs {
		if kr, ok := result.PerK[k]; ok {
			fmt.Printf("  K=%2d: RecallAny=%.3f, RecallAll=%.3f, NDCG=%.3f\n",
				k, kr.RecallAny, kr.RecallAll, kr.NDCG)
		}
	}

	if len(result.PerType) > 0 {
		fmt.Println("\nPer-Type Results:")
		for typ, tr := range result.PerType {
			fmt.Printf("  %-30s n=%3d R@5=%.1f%% R@10=%.1f%% NDCG@10=%.3f\n",
				typ, tr.Count, tr.R5, tr.R10, tr.NDCG10)
		}
	}
}

func SaveResults(result *Result, outputPath string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}
