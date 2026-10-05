// Package cli provides the mempalace-go command-line interface.
// It supports init, mine, search, wake-up, status, repair, compress, split, and hook commands.
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/ProjectViVy/laputa/mentle/facade"
	"github.com/ProjectViVy/laputa/mentle/internal/config"
	"github.com/ProjectViVy/laputa/mentle/internal/embedder"
	"github.com/ProjectViVy/laputa/mentle/internal/layers"
	"github.com/ProjectViVy/laputa/mentle/internal/miner"
	"github.com/ProjectViVy/laputa/mentle/internal/room"
	"github.com/ProjectViVy/laputa/mentle/internal/search"
	govector "github.com/ProjectViVy/laputa/mentle/storage/govector"
	"github.com/spf13/cobra"
)

var palacePath string

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mempalace-go",
		Short: "Give your AI a memory. No API key required.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if palacePath != "" {
				os.Setenv("MEMPALACE_PALACE_PATH", palacePath)
			}
			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&palacePath, "palace", "", "Palace path")
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newMineCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newWakeUpCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newRepairCmd())
	cmd.AddCommand(newCompressCmd())
	cmd.AddCommand(newSplitCmd())
	cmd.AddCommand(newHookCmd())
	cmd.AddCommand(newMcpCmd())
	cmd.AddCommand(newInstructionsCmd())
	cmd.AddCommand(newOnboardCmd())
	cmd.AddCommand(newBenchCmd())
	return cmd
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [dir]",
		Short: "Initialize the memory palace",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load("")
			if err != nil {
				return err
			}

			palaceDir := os.ExpandEnv(cfg.PalacePath)
			if len(args) > 0 {
				palaceDir = args[0]
			} else if palacePath != "" {
				palaceDir = palacePath
			}

			if err := os.MkdirAll(palaceDir, 0755); err != nil {
				return err
			}

			// Embeddings use hugot with ONNX models

			fmt.Printf("Initialized MemPalace at %s\n", palaceDir)
			return nil
		},
	}
}

func newMineCmd() *cobra.Command {
	var mode string
	cmd := &cobra.Command{
		Use:   "mine [directory]",
		Short: "Mine project files or conversations into the palace",
		Long:  "Mine files into the palace. Use --mode to choose mining mode: 'projects' (default) or 'convos'",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if mode == "" {
				mode = "projects"
			}
			if mode != "projects" && mode != "convos" {
				return fmt.Errorf("invalid mode: %s (must be 'projects' or 'convos')", mode)
			}

			cfg, err := config.Load("")
			if err != nil {
				return err
			}

			ctx := context.Background()

			svc := &facade.Service{}
			if err := svc.Init(ctx, facade.Options{}); err != nil {
				return err
			}
			defer svc.Close()

			if mode == "convos" {
				m := miner.NewMiner(svc)
				cm := miner.NewConversationMiner(m)
				return cm.MineConversations(ctx, args[0], "")
			}

			m := miner.NewMiner(svc)

			roomDetector, err := room.NewConfigBasedRoomDetector(args[0])
			if err != nil {
				fmt.Printf("Warning: could not load room config: %v\n", err)
			} else {
				m.SetRoomDetector(roomDetector)
			}

			palaceDir := os.ExpandEnv(cfg.PalacePath)
			if err := miner.LoadMtimeIndex(palaceDir); err != nil {
				fmt.Printf("Warning: could not load mtime index: %v\n", err)
			}
			if err := miner.LoadContentHashIndex(palaceDir); err != nil {
				fmt.Printf("Warning: could not load content hash index: %v\n", err)
			}

			if err := m.MineProject(ctx, args[0], ""); err != nil {
				return err
			}

			if err := miner.SaveMtimeIndex(palaceDir); err != nil {
				fmt.Printf("Warning: could not save mtime index: %v\n", err)
			}
			if err := miner.SaveContentHashIndex(palaceDir); err != nil {
				fmt.Printf("Warning: could not save content hash index: %v\n", err)
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "projects", "Mining mode: 'projects' or 'convos'")
	return cmd
}

func newSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search [query]",
		Short: "Search the memory palace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load("")
			if err != nil {
				return err
			}

			ctx := context.Background()

			emb, err := embedder.New("", cfg.GetModelsDir())
			if err != nil {
				return err
			}
			defer emb.Close()

			store, err := govector.NewStore(os.ExpandEnv(cfg.PalacePath)+"/vectors.db", 384)
			if err != nil {
				return err
			}

			searcher := search.NewSearcher(store, emb)
			results, err := searcher.Search(ctx, args[0], "", "", 5)
			if err != nil {
				return err
			}

			for _, r := range results {
				fmt.Printf("[%s/%s] %s\n", r.Wing, r.Room, truncate(r.Content, 200))
			}
			return nil
		},
	}
}

func newWakeUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "wake-up",
		Short: "Show L0 + L1 context",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load("")
			if err != nil {
				return err
			}

			ctx := context.Background()

			emb, err := embedder.New("", cfg.GetModelsDir())
			if err != nil {
				return err
			}
			defer emb.Close()

			store, err := govector.NewStore(os.ExpandEnv(cfg.PalacePath)+"/vectors.db", 384)
			if err != nil {
				return err
			}

			searcher := search.NewSearcher(store, emb)
			stack := layers.NewMemoryStack(cfg, searcher)

			wing, _ := cmd.Flags().GetString("wing")
			text, err := stack.WakeUp(ctx, wing)
			if err != nil {
				return err
			}
			fmt.Println(text)
			return nil
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show palace status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load("")
			if err != nil {
				return err
			}
			fmt.Printf("Palace: %s\n", cfg.PalacePath)
			fmt.Printf("Collection: %s\n", cfg.CollectionName)
			return nil
		},
	}
}

func newRepairCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repair",
		Short: "Rebuild disposable indexes from canonical memory",
		Long:  "Build and verify derived vector/BM25 indexes from canonical SQLite without moving the authority database",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			svc := &facade.Service{}
			if err := svc.Init(ctx, facade.Options{}); err != nil {
				return err
			}
			defer svc.Close()
			fmt.Printf("Rebuilding derived indexes from canonical SQLite at %s\n", svc.PalacePath)
			if err := svc.RebuildDerivedIndexes(ctx); err != nil {
				return err
			}
			health, err := svc.IndexHealth(ctx)
			if err != nil {
				return err
			}
			fmt.Printf("Rebuild complete: %d active memories, vector=%d, bm25=%d\n", health.CanonicalActiveCount, health.VectorActiveCount, health.BM25Count)
			return nil
		},
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
