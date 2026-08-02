// Package convomem implements the ConvoMem benchmark for conversation memory retrieval.
package convomem

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dashimaki/mentle/internal/dialect"
	"github.com/dashimaki/mentle/internal/embedder"
	"github.com/dashimaki/mentle/internal/search"
	govector "github.com/dashimaki/mentle/storage/govector"
)

const hfBaseURL = "https://huggingface.co/datasets/Salesforce/ConvoMem/resolve/main/core_benchmark/evidence_questions"

type EvidenceItem struct {
	Question         string            `json:"question"`
	Answer           string            `json:"answer"`
	MessageEvidences []EvidenceMessage `json:"message_evidences"`
	Conversations    []Conversation    `json:"conversations"`
}

type EvidenceMessage struct {
	Text    string `json:"text"`
	Speaker string `json:"speaker"`
}

type Conversation struct {
	Messages []EvidenceMessage `json:"messages"`
}

type EvidenceFile struct {
	EvidenceItems []EvidenceItem `json:"evidence_items"`
}

type Result struct {
	Category        string             `json:"category"`
	Mode            string             `json:"mode"`
	TopK            int                `json:"top_k"`
	TotalItems      int                `json:"total_items"`
	AvgRecall       float64            `json:"avg_recall"`
	Perfect         int                `json:"perfect"`
	Zero            int                `json:"zero"`
	Partial         int                `json:"partial"`
	CategoryRecalls map[string]float64 `json:"category_recalls,omitempty"`
}

var Categories = []string{
	"user_evidence", "assistant_facts_evidence", "changing_evidence",
	"abstention_evidence", "preference_evidence", "implicit_connection_evidence",
}

func Run(category string, topK int, mode string, limit int, cacheDir string) (*Result, error) {
	ctx := context.Background()
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "convomem_cache")
	}
	os.MkdirAll(cacheDir, 0755)

	result := &Result{Category: category, Mode: mode, TopK: topK, CategoryRecalls: make(map[string]float64)}
	cats := Categories
	if category != "" && category != "all" {
		cats = []string{category}
	}

	totalRecall := 0.0
	totalItems := 0
	encoder := dialect.NewEncoder()

	for _, cat := range cats {
		items, err := loadCategory(cat, cacheDir)
		if err != nil {
			fmt.Printf("Warning: failed to load %s: %v\n", cat, err)
			continue
		}
		if limit > 0 && len(items) > limit {
			items = items[:limit]
		}

		catRecall := 0.0
		for _, item := range items {
			recall, err := runItem(ctx, item, topK, mode, encoder)
			if err != nil {
				continue
			}
			catRecall += recall
			totalRecall += recall
			totalItems++
			if recall >= 1.0 {
				result.Perfect++
			} else if recall == 0.0 {
				result.Zero++
			} else {
				result.Partial++
			}
		}
		if len(items) > 0 {
			result.CategoryRecalls[cat] = catRecall / float64(len(items))
		}
		fmt.Printf("[%s] %d items, avg recall: %.3f\n", cat, len(items), catRecall/float64(max(1, len(items))))
	}

	result.TotalItems = totalItems
	if totalItems > 0 {
		result.AvgRecall = totalRecall / float64(totalItems)
	}
	return result, nil
}

func loadCategory(category, cacheDir string) ([]EvidenceItem, error) {
	cachePath := filepath.Join(cacheDir, category+".json")
	if data, err := os.ReadFile(cachePath); err == nil {
		var file EvidenceFile
		if err := json.Unmarshal(data, &file); err == nil {
			return file.EvidenceItems, nil
		}
	}

	url := fmt.Sprintf("%s/%s/1_evidence", hfBaseURL, category)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	os.WriteFile(cachePath, data, 0644)

	var file EvidenceFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	return file.EvidenceItems, nil
}

func runItem(ctx context.Context, item EvidenceItem, topK int, mode string, encoder *dialect.Encoder) (float64, error) {
	emb, err := embedder.New("", "")
	if err != nil {
		return 0, err
	}
	defer emb.Close()

	store, err := govector.NewStore(filepath.Join(os.TempDir(), fmt.Sprintf("bench_%d", time.Now().UnixNano())), 384)
	if err != nil {
		return 0, err
	}
	searcher := search.NewSearcher(store, emb)

	corpus := make([]string, 0)
	for _, conv := range item.Conversations {
		for _, msg := range conv.Messages {
			text := msg.Text
			if mode == "aaak" {
				text = encoder.Compress(text, map[string]string{})
			}
			corpus = append(corpus, text)
			vector, err := emb.CreateEmbedding(ctx, text)
			if err != nil {
				continue
			}
			payload := map[string]any{
				"speaker": msg.Speaker,
				"idx":     strconv.Itoa(len(corpus) - 1),
				"content": text,
			}
			store.Add(fmt.Sprintf("msg_%d", len(corpus)-1), vector, payload)
		}
	}

	results, err := searcher.Search(ctx, item.Question, "", "", topK)
	emb.Close()
	if err != nil {
		return 0, err
	}

	var retrievedIndices []int
	for _, r := range results {
		if idxStr, ok := r.Metadata["idx"]; ok {
			if i, err := strconv.Atoi(idxStr); err == nil {
				retrievedIndices = append(retrievedIndices, i)
			}
		}
	}

	evidenceTexts := make(map[string]bool)
	for _, e := range item.MessageEvidences {
		evidenceTexts[strings.ToLower(strings.TrimSpace(e.Text))] = true
	}

	found := 0
	for evText := range evidenceTexts {
		for _, idx := range retrievedIndices {
			if idx < len(corpus) {
				retText := strings.ToLower(strings.TrimSpace(corpus[idx]))
				if strings.Contains(retText, evText) || strings.Contains(evText, retText) {
					found++
					break
				}
			}
		}
	}

	if len(evidenceTexts) == 0 {
		return 1.0, nil
	}
	return float64(found) / float64(len(evidenceTexts)), nil
}

func PrintResults(result *Result) {
	fmt.Println("\n=== ConvoMem Results ===")
	fmt.Printf("Category: %s\n", result.Category)
	fmt.Printf("Mode: %s, Top-K: %d\n", result.Mode, result.TopK)
	fmt.Printf("Total Items: %d\n", result.TotalItems)
	fmt.Printf("Average Recall: %.3f\n", result.AvgRecall)
	fmt.Printf("Perfect (1.0): %d (%.1f%%)\n", result.Perfect, float64(result.Perfect)/float64(max(1, result.TotalItems))*100)
	fmt.Printf("Zero (0.0): %d (%.1f%%)\n", result.Zero, float64(result.Zero)/float64(max(1, result.TotalItems))*100)
	fmt.Printf("Partial (0-1): %d (%.1f%%)\n", result.Partial, float64(result.Partial)/float64(max(1, result.TotalItems))*100)
	if len(result.CategoryRecalls) > 0 {
		fmt.Println("\nPer-Category Recall:")
		for cat, recall := range result.CategoryRecalls {
			fmt.Printf("  %-35s %.3f\n", cat, recall)
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
