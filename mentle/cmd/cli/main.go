// Package cli provides the mempalace-go command-line interface.
// It supports init, mine, search, wake-up, status, repair, compress, split, and hook commands.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dashimaki/mentle/internal/config"
	"github.com/dashimaki/mentle/internal/embedder"
	"github.com/dashimaki/mentle/internal/layers"
	"github.com/dashimaki/mentle/internal/miner"
	"github.com/dashimaki/mentle/internal/palace"
	"github.com/dashimaki/mentle/internal/room"
	"github.com/dashimaki/mentle/internal/search"
	"github.com/dashimaki/mentle/pkg/wal"
	govector "github.com/dashimaki/mentle/storage/govector"
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

			for _, dir := range []string{palaceDir, palaceDir + "/wal"} {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
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

			emb, err := embedder.New("", cfg.GetModelsDir())
			if err != nil {
				return fmt.Errorf("embedder: %w", err)
			}
			defer emb.Close()

			store, err := govector.NewStore(os.ExpandEnv(cfg.PalacePath)+"/vectors.db", 384)
			if err != nil {
				return err
			}

			searcher := search.NewSearcher(store, emb)

			if mode == "convos" {
				m := miner.NewMiner(searcher)
				cm := miner.NewConversationMiner(m)
				return cm.MineConversations(ctx, args[0], "")
			}

			m := miner.NewMiner(searcher)

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
		Short: "Rebuild palace vector index",
		Long:  "Scan stored drawer files and rebuild the vector index",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load("")
			if err != nil {
				return err
			}

			palaceDir := os.ExpandEnv(cfg.PalacePath)
			fmt.Printf("Repairing palace at %s\n", palaceDir)

			if _, err := os.Stat(palaceDir); os.IsNotExist(err) {
				fmt.Printf("  No palace found at %s\n", palaceDir)
				return nil
			}

			walDir := filepath.Join(palaceDir, "wal")
			entries, err := os.ReadDir(walDir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("Nothing to repair - no wal directory found")
					return nil
				}
				return fmt.Errorf("read wal dir: %w", err)
			}

			if len(entries) == 0 {
				fmt.Println("Nothing to repair - wal directory is empty")
				return nil
			}

			backupDir := palaceDir + ".backup"
			if _, err := os.Stat(backupDir); err == nil {
				fmt.Printf("Removing old backup at %s\n", backupDir)
				os.RemoveAll(backupDir)
			}
			fmt.Printf("Creating backup at %s\n", backupDir)
			if err := os.Rename(palaceDir, backupDir); err != nil {
				return fmt.Errorf("create backup: %w", err)
			}

			os.MkdirAll(palaceDir, 0755)
			os.MkdirAll(walDir, 0755)

			vectorsPath := filepath.Join(palaceDir, "vectors.db")
			store, err := govector.NewStore(vectorsPath, 1024)
			if err != nil {
				return fmt.Errorf("create new vector store: %w", err)
			}
			defer store.Close()

			ctx := context.Background()
			emb, err := embedder.New("", cfg.GetModelsDir())
			if err != nil {
				return fmt.Errorf("embedder: %w", err)
			}
			defer emb.Close()

			searcher := search.NewSearcher(store, emb)

			fmt.Printf("Re-indexing %d drawer files...\n", len(entries))

			walNew, err := wal.NewWAL(palaceDir)
			if err != nil {
				return fmt.Errorf("create new WAL: %w", err)
			}

			newWalDir := filepath.Join(palaceDir, "wal")

			indexed := 0
			skipped := 0
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				drawerPath := filepath.Join(walDir, entry.Name())
				content, err := os.ReadFile(drawerPath)
				if err != nil {
					fmt.Printf("  Warning: failed to read %s: %v\n", entry.Name(), err)
					skipped++
					continue
				}

				var drawerEntry wal.Entry
				if err := json.Unmarshal(content, &drawerEntry); err != nil {
					drawerEntry = wal.Entry{
						DrawerID: entry.Name(),
						Content:  string(content),
						Wing:     "default",
						Room:     "general",
					}
				}

				newDrawerPath := filepath.Join(newWalDir, entry.Name())
				if err := os.WriteFile(newDrawerPath, content, 0644); err != nil {
					fmt.Printf("  Warning: failed to write %s: %v\n", entry.Name(), err)
					skipped++
					continue
				}

				if err := walNew.LogAdd(drawerEntry); err != nil {
					fmt.Printf("  Warning: failed to log %s: %v\n", entry.Name(), err)
				}

				drawer := palace.Drawer{
					ID:         drawerEntry.DrawerID,
					Content:    drawerEntry.Content,
					Wing:       drawerEntry.Wing,
					Room:       drawerEntry.Room,
					SourceFile: drawerEntry.DrawerID,
					AddedBy:    "mempalace-go-repair",
				}
				if err := searcher.Store(ctx, drawer); err != nil {
					fmt.Printf("  Warning: failed to index %s: %v\n", entry.Name(), err)
					skipped++
					continue
				}

				indexed++
				if indexed%100 == 0 {
					fmt.Printf("  Indexed %d/%d...\n", indexed, len(entries))
				}
			}

			fmt.Printf("\nRepair complete. %d drawers re-indexed, %d skipped.\n", indexed, skipped)
			fmt.Printf("Backup saved at %s\n", backupDir)
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
