package locomo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dashimaki/mentle/benchmarks"
	"github.com/dashimaki/mentle/internal/dialect"
	"github.com/dashimaki/mentle/internal/embedder"
	"github.com/dashimaki/mentle/internal/search"
	govector "github.com/dashimaki/mentle/storage/govector"
)

type Sample struct {
	SampleID       string            `json:"sample_id"`
	Conversation   map[string]any    `json:"conversation"`
	SessionSummary map[string]string `json:"session_summary,omitempty"`
	QA             []QAItem          `json:"qa"`
}

type QAItem struct {
	Question string   `json:"question"`
	Answer   string   `json:"answer"`
	Category int      `json:"category"`
	Evidence []string `json:"evidence"`
}

type Result struct {
	Mode           string             `json:"mode"`
	TopK           int                `json:"top_k"`
	Granularity    string             `json:"granularity"`
	TotalQA        int                `json:"total_qa"`
	AvgRecall      float64            `json:"avg_recall"`
	PerfRecall     float64            `json:"perfect_recall"`
	ZeroRecall     float64            `json:"zero_recall"`
	PartialRecall  float64            `json:"partial_recall"`
	CategoryRecall map[string]float64 `json:"category_recall"`
	F1Score        float64            `json:"f1_score,omitempty"`
}

var categoryNames = map[int]string{
	1: "Single-hop", 2: "Temporal", 3: "Temporal-inference",
	4: "Open-domain", 5: "Adversarial",
}

func Run(dataFile string, topK int, mode string, granularity string, limit int, llmRerank bool) (*Result, error) {
	ctx := context.Background()
	data, err := os.ReadFile(dataFile)
	if err != nil {
		return nil, err
	}
	var samples []Sample
	if err := json.Unmarshal(data, &samples); err != nil {
		return nil, err
	}
	if limit > 0 && len(samples) > limit {
		samples = samples[:limit]
	}
	result := &Result{
		Mode: mode, TopK: topK, Granularity: granularity,
		CategoryRecall: make(map[string]float64),
	}
	encoder := dialect.NewEncoder()
	totalRecall := 0.0
	totalF1 := 0.0
	perfectCount, zeroCount, partialCount := 0, 0, 0
	catRecalls := make(map[int][]float64)

	for _, sample := range samples {
		sessions := extractSessions(sample.Conversation)
		emb, err := embedder.New("", "")
		if err != nil {
			continue
		}
		store, _, err := buildCorpus(ctx, emb, sessions, mode, encoder, granularity)
		if err != nil {
			emb.Close()
			continue
		}
		searcher := search.NewSearcher(store, emb)
		for _, qa := range sample.QA {
			results, err := searcher.Search(ctx, qa.Question, "", "", topK)
			if err != nil {
				continue
			}
			evidenceIDs := mapEvidenceToIDs(qa.Evidence, granularity, sessions)
			var retrievedIDs []string
			for _, r := range results {
				retrievedIDs = append(retrievedIDs, r.Metadata["id"])
			}
			recall := benchmarks.FractionalRecall(retrievedIDs, evidenceIDs)
			totalRecall += recall
			if recall >= 1.0 {
				perfectCount++
			} else if recall == 0.0 {
				zeroCount++
			} else {
				partialCount++
			}
			catRecalls[qa.Category] = append(catRecalls[qa.Category], recall)
			if llmRerank && qa.Answer != "" {
				f1 := computeF1(results, qa.Answer)
				totalF1 += f1
			}
		}
		emb.Close()
	}
	totalQA := 0
	for _, recalls := range catRecalls {
		totalQA += len(recalls)
	}
	result.TotalQA = totalQA
	if totalQA > 0 {
		result.AvgRecall = totalRecall / float64(totalQA)
		result.PerfRecall = float64(perfectCount) / float64(totalQA)
		result.ZeroRecall = float64(zeroCount) / float64(totalQA)
		result.PartialRecall = float64(partialCount) / float64(totalQA)
	}
	for cat, recalls := range catRecalls {
		if len(recalls) > 0 {
			sum := 0.0
			for _, r := range recalls {
				sum += r
			}
			result.CategoryRecall[categoryNames[cat]] = sum / float64(len(recalls))
		}
	}
	if llmRerank && totalQA > 0 {
		result.F1Score = totalF1 / float64(totalQA)
	}
	return result, nil
}

func extractSessions(conv map[string]any) map[string][]map[string]string {
	sessions := make(map[string][]map[string]string)
	for key, val := range conv {
		if strings.HasSuffix(key, "_date_time") {
			continue
		}
		if msgs, ok := val.([]any); ok {
			var sessionMsgs []map[string]string
			for _, m := range msgs {
				if msg, ok := m.(map[string]any); ok {
					sessionMsgs = append(sessionMsgs, map[string]string{
						"speaker": getString(msg, "speaker"),
						"text":    getString(msg, "text"),
						"dia_id":  getString(msg, "dia_id"),
					})
				}
			}
			sessions[key] = sessionMsgs
		}
	}
	return sessions
}

func buildCorpus(ctx context.Context, emb *embedder.Embedder, sessions map[string][]map[string]string, mode string, encoder *dialect.Encoder, granularity string) (*govector.Store, []string, error) {
	store, err := govector.NewStore(filepath.Join(os.TempDir(), fmt.Sprintf("bench_%d", time.Now().UnixNano())), 384)
	if err != nil {
		return nil, nil, err
	}
	var corpus []string
	docIdx := 0
	for sessionName, msgs := range sessions {
		for _, msg := range msgs {
			text := msg["text"]
			if mode == "aaak" {
				text = encoder.Compress(text, map[string]string{})
			}
			corpus = append(corpus, text)
			var id string
			if granularity == "dialog" {
				id = msg["dia_id"]
			} else {
				id = sessionName
			}
			vector, err := emb.CreateEmbedding(ctx, text)
			if err != nil {
				continue
			}
			payload := map[string]any{
				"id": id, "session": sessionName, "idx": docIdx, "content": text,
			}
			store.Add(fmt.Sprintf("doc_%d", docIdx), vector, payload)
			docIdx++
		}
	}
	return store, corpus, nil
}

func mapEvidenceToIDs(evidence []string, granularity string, sessions map[string][]map[string]string) []string {
	var ids []string
	for _, ev := range evidence {
		if granularity == "dialog" {
			ids = append(ids, ev)
		} else {
			for sessionName, msgs := range sessions {
				for _, msg := range msgs {
					if msg["dia_id"] == ev {
						ids = append(ids, sessionName)
						break
					}
				}
			}
		}
	}
	return ids
}

func computeF1(results []search.Drawer, answer string) float64 {
	var text strings.Builder
	for _, r := range results {
		text.WriteString(r.Metadata["content"] + " ")
	}
	return benchmarks.F1Score(text.String(), answer)
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func PrintResults(result *Result) {
	fmt.Println("\n=== LoCoMo Results ===")
	fmt.Printf("Mode: %s, Top-K: %d, Granularity: %s\n", result.Mode, result.TopK, result.Granularity)
	fmt.Printf("Total QA pairs: %d\n", result.TotalQA)
	fmt.Printf("Average Recall: %.3f\n", result.AvgRecall)
	fmt.Printf("Perfect (1.0): %.1f%%\n", result.PerfRecall*100)
	fmt.Printf("Zero (0.0): %.1f%%\n", result.ZeroRecall*100)
	fmt.Printf("Partial (0-1): %.1f%%\n", result.PartialRecall*100)
	if result.F1Score > 0 {
		fmt.Printf("F1 Score: %.3f\n", result.F1Score)
	}
	if len(result.CategoryRecall) > 0 {
		fmt.Println("\nPer-Category Recall:")
		for cat, recall := range result.CategoryRecall {
			fmt.Printf("  %-25s %.3f\n", cat, recall)
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
