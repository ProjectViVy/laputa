// Package membench implements the MemBench benchmark for memory retrieval.
package membench

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dashimaki/mentle/internal/embedder"
	"github.com/dashimaki/mentle/internal/search"
	govector "github.com/dashimaki/mentle/storage/govector"
)

type MemBenchItem struct {
	TID           int     `json:"tid"`
	MessageList   []any   `json:"message_list"`
	QA            *QA     `json:"QA,omitempty"`
	TargetStepID  []any   `json:"target_step_id,omitempty"`
	TargetStepIDs [][]any `json:"target_step_ids,omitempty"`
}

type QA struct {
	Question    string            `json:"question"`
	Answer      string            `json:"answer"`
	Choices     map[string]string `json:"choices,omitempty"`
	GroundTruth string            `json:"ground_truth"`
}

type Result struct {
	Category        string             `json:"category"`
	TotalItems      int                `json:"total_items"`
	TotalHits       int                `json:"total_hits"`
	HitRate         float64            `json:"hit_rate"`
	CategoryResults map[string]float64 `json:"category_results,omitempty"`
}

func Run(dataDir string, category string, topic string, topK int, mode string, limit int) (*Result, error) {
	ctx := context.Background()
	categories := []string{
		"simple", "highlevel", "knowledge_update", "comparative",
		"conditional", "noisy", "aggregative", "highlevel_rec",
		"lowlevel_rec", "RecMultiSession", "post_processing",
	}
	if category != "" {
		categories = []string{category}
	}

	result := &Result{CategoryResults: make(map[string]float64)}
	totalItems := 0
	totalHits := 0

	for _, cat := range categories {
		filePath := filepath.Join(dataDir, cat+".json")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue
		}

		catHits, catTotal, err := runCategory(ctx, filePath, topic, topK, mode, limit)
		if err != nil {
			return nil, fmt.Errorf("category %s: %w", cat, err)
		}

		hitRate := 0.0
		if catTotal > 0 {
			hitRate = float64(catHits) / float64(catTotal) * 100
		}
		result.CategoryResults[cat] = hitRate
		totalItems += catTotal
		totalHits += catHits
		fmt.Printf("[%s] %d/%d (%.1f%%)\n", cat, catHits, catTotal, hitRate)
	}

	result.TotalItems = totalItems
	result.TotalHits = totalHits
	if totalItems > 0 {
		result.HitRate = float64(totalHits) / float64(totalItems) * 100
	}
	return result, nil
}

func runCategory(ctx context.Context, filePath, topic string, topK int, mode string, limit int) (int, int, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, 0, err
	}

	var itemsByTopic map[string][]MemBenchItem
	if err := json.Unmarshal(data, &itemsByTopic); err != nil {
		return 0, 0, fmt.Errorf("parse: %w", err)
	}

	if topic != "" {
		if items, ok := itemsByTopic[topic]; ok {
			itemsByTopic = map[string][]MemBenchItem{topic: items}
		}
	}

	emb, err := embedder.New("", "")
	if err != nil {
		return 0, 0, fmt.Errorf("embedder: %w", err)
	}
	defer emb.Close()

	hits := 0
	total := 0

	for topicName, items := range itemsByTopic {
		for _, item := range items {
			if limit > 0 && total >= limit {
				break
			}
			if item.QA == nil || len(item.TargetStepIDs) == 0 {
				continue
			}
			total++

			store, err := buildCorpus(ctx, emb, topicName, items)
			if err != nil {
				continue
			}
			searcher := search.NewSearcher(store, emb)
			targetSIDs := extractTargetSIDs(item.TargetStepIDs)

			results, err := searcher.Search(ctx, item.QA.Question, "", "", topK)
			if err != nil {
				continue
			}

			var retrievedSIDs []string
			for _, r := range results {
				if s, ok := r.Metadata["sid"]; ok {
					retrievedSIDs = append(retrievedSIDs, s)
				}
			}

			hit := false
			for _, sid := range retrievedSIDs {
				if targetSIDs[sid] {
					hit = true
					break
				}
			}
			if hit {
				hits++
			}
			if total%50 == 0 {
				fmt.Printf("  Progress: %d items, %d hits (%.1f%%)\n",
					total, hits, float64(hits)/float64(max(1, total))*100)
			}
		}
	}
	emb.Close()
	return hits, total, nil
}

func buildCorpus(ctx context.Context, emb *embedder.Embedder, topicName string, items []MemBenchItem) (*govector.Store, error) {
	store, err := govector.NewStore(filepath.Join(os.TempDir(), fmt.Sprintf("bench_%d", time.Now().UnixNano())), 384)
	if err != nil {
		return nil, err
	}

	globalIdx := 0
	for _, item := range items {
		sid := 0
		for _, session := range item.MessageList {
			turns := extractTurns(session)
			for tIdx, turn := range turns {
				text := formatTurn(turn, sid, tIdx)
				vector, err := emb.CreateEmbedding(ctx, text)
				if err != nil {
					continue
				}
				payload := map[string]any{
					"topic": topicName, "sid": fmt.Sprintf("%d", sid),
					"s_idx": sid, "t_idx": tIdx, "global_idx": globalIdx, "content": text,
				}
				store.Add(fmt.Sprintf("doc_%d", globalIdx), vector, payload)
				globalIdx++
			}
			sid++
		}
	}
	return store, nil
}

func extractTurns(session any) []map[string]any {
	var turns []map[string]any
	switch s := session.(type) {
	case []any:
		for _, t := range s {
			if turn, ok := t.(map[string]any); ok {
				turns = append(turns, turn)
			}
		}
	case map[string]any:
		turns = append(turns, s)
	}
	return turns
}

func formatTurn(turn map[string]any, sid, tIdx int) string {
	user := getString(turn, "user")
	asst := getString(turn, "assistant")
	timeStr := getString(turn, "time")
	text := fmt.Sprintf("[User] %s [Assistant] %s", user, asst)
	if timeStr != "" {
		text = fmt.Sprintf("[%s] %s", timeStr, text)
	}
	return text
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func extractTargetSIDs(targetStepIDs [][]any) map[string]bool {
	sids := make(map[string]bool)
	for _, step := range targetStepIDs {
		if len(step) >= 1 {
			switch s := step[0].(type) {
			case float64:
				sids[fmt.Sprintf("%.0f", s)] = true
			case string:
				sids[s] = true
			}
		}
	}
	return sids
}

func PrintResults(result *Result) {
	fmt.Println("\n=== MemBench Results ===")
	fmt.Printf("Overall Hit@K: %d/%d (%.1f%%)\n", result.TotalHits, result.TotalItems, result.HitRate)
	fmt.Println("\nPer-Category:")
	for cat, rate := range result.CategoryResults {
		fmt.Printf("  %-25s %.1f%%\n", cat, rate)
	}
}

func SaveResults(result *Result, outputPath string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}
