package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dashimaki/mentle/benchmarks/convomem"
	"github.com/dashimaki/mentle/benchmarks/locomo"
	"github.com/dashimaki/mentle/benchmarks/longmemeval"
	"github.com/dashimaki/mentle/benchmarks/membench"
	"github.com/spf13/cobra"
)

func newBenchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bench [longmemeval|locomo|membench|convomem]",
		Short: "Run memory retrieval benchmarks",
	}

	cmd.AddCommand(newLongMemEvalCmd())
	cmd.AddCommand(newLoCoMoCmd())
	cmd.AddCommand(newMemBenchCmd())
	cmd.AddCommand(newConvoMemCmd())

	return cmd
}

func newLongMemEvalCmd() *cobra.Command {
	var mode, granularity, outPath string
	var limit, skip, topK int

	cmd := &cobra.Command{
		Use:   "longmemeval <data.json>",
		Short: "Run LongMemEval benchmark (Recall@K, NDCG@K)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			result, err := longmemeval.Run(args[0], mode, granularity, limit, skip, topK)
			if err != nil {
				return err
			}

			longmemeval.PrintResults(result)
			fmt.Printf("\nTime: %v\n", time.Since(start))

			if outPath == "" {
				outPath = fmt.Sprintf("longmemeval_%s_%s.json", mode, time.Now().Format("20060102_150405"))
			}
			return longmemeval.SaveResults(result, outPath)
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "raw", "Retrieval mode: raw, aaak")
	cmd.Flags().StringVar(&granularity, "granularity", "session", "session or turn")
	cmd.Flags().IntVar(&topK, "top-k", 50, "Top-K retrieval")
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit to N questions (0=all)")
	cmd.Flags().IntVar(&skip, "skip", 0, "Skip first N questions")
	cmd.Flags().StringVar(&outPath, "out", "", "Output JSON path")

	return cmd
}

func newLoCoMoCmd() *cobra.Command {
	var mode, granularity, outPath string
	var topK, limit int
	var llmRerank bool

	cmd := &cobra.Command{
		Use:   "locomo <data.json>",
		Short: "Run LoCoMo benchmark (fractional recall, F1)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			result, err := locomo.Run(args[0], topK, mode, granularity, limit, llmRerank)
			if err != nil {
				return err
			}

			locomo.PrintResults(result)
			fmt.Printf("\nTime: %v\n", time.Since(start))

			if outPath == "" {
				outPath = fmt.Sprintf("locomo_%s_%s.json", mode, time.Now().Format("20060102_150405"))
			}
			return locomo.SaveResults(result, outPath)
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "raw", "Retrieval mode: raw, aaak")
	cmd.Flags().StringVar(&granularity, "granularity", "session", "dialog or session")
	cmd.Flags().IntVar(&topK, "top-k", 50, "Top-K retrieval")
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit to N conversations")
	cmd.Flags().BoolVar(&llmRerank, "llm-rerank", false, "Enable LLM reranking")
	cmd.Flags().StringVar(&outPath, "out", "", "Output JSON path")

	return cmd
}

func newMemBenchCmd() *cobra.Command {
	var category, topic, mode, outPath string
	var topK, limit int

	cmd := &cobra.Command{
		Use:   "membench <data_dir>",
		Short: "Run MemBench benchmark (Hit@K)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			result, err := membench.Run(args[0], category, topic, topK, mode, limit)
			if err != nil {
				return err
			}

			membench.PrintResults(result)
			fmt.Printf("\nTime: %v\n", time.Since(start))

			if outPath == "" {
				outPath = fmt.Sprintf("membench_%s_%s.json", mode, time.Now().Format("20060102_150405"))
			}
			return membench.SaveResults(result, outPath)
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "Run single category (default=all)")
	cmd.Flags().StringVar(&topic, "topic", "movie", "Topic filter: movie, food, book")
	cmd.Flags().StringVar(&mode, "mode", "raw", "Retrieval mode: raw, hybrid")
	cmd.Flags().IntVar(&topK, "top-k", 5, "Top-K retrieval")
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit items (0=all)")
	cmd.Flags().StringVar(&outPath, "out", "", "Output JSON path")

	return cmd
}

func newConvoMemCmd() *cobra.Command {
	var category, mode, cacheDir, outPath string
	var topK, limit int

	cmd := &cobra.Command{
		Use:   "convomem",
		Short: "Run ConvoMem benchmark (substring-match recall)",
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			result, err := convomem.Run(category, topK, mode, limit, cacheDir)
			if err != nil {
				return err
			}

			convomem.PrintResults(result)
			fmt.Printf("\nTime: %v\n", time.Since(start))

			if outPath == "" {
				outPath = fmt.Sprintf("convomem_%s_%s.json", mode, time.Now().Format("20060102_150405"))
			}
			return convomem.SaveResults(result, outPath)
		},
	}

	validCategories := append(convomem.Categories, "all")
	cmd.Flags().StringVar(&category, "category", "all", "Category: "+fmt.Sprint(validCategories))
	cmd.Flags().StringVar(&mode, "mode", "raw", "Retrieval mode: raw, aaak")
	cmd.Flags().IntVar(&topK, "top-k", 10, "Top-K retrieval")
	cmd.Flags().IntVar(&limit, "limit", 100, "Items per category")
	cmd.Flags().StringVar(&cacheDir, "cache-dir", filepath.Join(os.TempDir(), "convomem_cache"), "Cache directory")
	cmd.Flags().StringVar(&outPath, "out", "", "Output JSON path")

	return cmd
}
