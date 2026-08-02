package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dashimaki/mentle/internal/config"
	"github.com/dashimaki/mentle/internal/dialect"
	"github.com/dashimaki/mentle/pkg/wal"
	"github.com/spf13/cobra"
)

func newCompressCmd() *cobra.Command {
	var wingFilter string
	var dryRun bool
	var configPath string

	cmd := &cobra.Command{
		Use:   "compress [wing]",
		Short: "Compress drawers using AAAK Dialect",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				wingFilter = args[0]
			}

			cfg, err := config.Load("")
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			palaceDir := os.ExpandEnv(cfg.PalacePath)
			walDir := filepath.Join(palaceDir, "wal")

			if _, err := os.Stat(walDir); os.IsNotExist(err) {
				fmt.Printf("No wal directory found at %s\n", walDir)
				return nil
			}

			encoder := dialect.NewEncoder()

			if configPath != "" {
				if err := loadEntityConfig(encoder, configPath); err != nil {
					return fmt.Errorf("load entity config: %w", err)
				}
			}

			entries, err := os.ReadDir(walDir)
			if err != nil {
				return fmt.Errorf("read wal directory: %w", err)
			}

			if len(entries) == 0 {
				fmt.Println("No drawers to compress")
				return nil
			}

			var totalOrigTokens, totalCompTokens int
			var totalOrigChars, totalCompChars int
			processed := 0
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

				if wingFilter != "" && drawerEntry.Wing != wingFilter {
					continue
				}

				metadata := map[string]string{
					"wing": drawerEntry.Wing,
					"room": drawerEntry.Room,
				}

				compressed := encoder.Compress(drawerEntry.Content, metadata)
				stats := encoder.CompressionStats(drawerEntry.Content, compressed)

				if dryRun {
					fmt.Printf("Drawer: %s\n", entry.Name())
					fmt.Printf("  Wing: %s, Room: %s\n", drawerEntry.Wing, drawerEntry.Room)
					fmt.Printf("  Original: %d chars, ~%d tokens\n", stats.OriginalChars, stats.OriginalTokensEst)
					fmt.Printf("  Compressed: %d chars, ~%d tokens\n", stats.SummaryChars, stats.SummaryTokensEst)
					fmt.Printf("  Ratio: %.2fx\n", stats.SizeRatio)
					fmt.Printf("  -> %s\n\n", compressed)
				} else {
					compressedPath := drawerPath + ".compressed"
					if err := os.WriteFile(compressedPath, []byte(compressed), 0644); err != nil {
						fmt.Printf("  Warning: failed to write %s: %v\n", compressedPath, err)
						skipped++
						continue
					}
				}

				totalOrigTokens += stats.OriginalTokensEst
				totalCompTokens += stats.SummaryTokensEst
				totalOrigChars += stats.OriginalChars
				totalCompChars += stats.SummaryChars
				processed++
			}

			if processed == 0 {
				fmt.Println("No drawers processed")
				return nil
			}

			if dryRun {
				fmt.Println("=== DRY RUN SUMMARY ===")
			} else {
				fmt.Println("=== COMPRESSION COMPLETE ===")
			}
			fmt.Printf("Processed: %d drawers\n", processed)
			if skipped > 0 {
				fmt.Printf("Skipped: %d\n", skipped)
			}
			fmt.Printf("Total original: %d chars, ~%d tokens\n", totalOrigChars, totalOrigTokens)
			fmt.Printf("Total compressed: %d chars, ~%d tokens\n", totalCompChars, totalCompTokens)
			if totalCompTokens > 0 {
				overallRatio := float64(totalOrigTokens) / float64(totalCompTokens)
				fmt.Printf("Overall compression ratio: %.2fx\n", overallRatio)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show compression stats without writing files")
	cmd.Flags().StringVar(&configPath, "config", "", "Path to entity config JSON file")
	cmd.Flags().StringVar(&wingFilter, "wing", "", "Filter by wing")

	return cmd
}

func loadEntityConfig(encoder *dialect.Encoder, configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var entityConfig struct {
		Entities map[string]string `json:"entities"`
	}

	if err := json.Unmarshal(data, &entityConfig); err != nil {
		return err
	}

	encoder.SetEntityCodes(entityConfig.Entities)

	return nil
}
