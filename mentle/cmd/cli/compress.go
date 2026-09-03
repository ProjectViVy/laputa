package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/dashimaki/mentle/facade"
	"github.com/dashimaki/mentle/internal/dialect"
	"github.com/spf13/cobra"
)

func newCompressCmd() *cobra.Command {
	var wingFilter string
	var dryRun bool
	var configPath string

	cmd := &cobra.Command{
		Use:   "compress [wing]",
		Short: "Compress canonical memories using AAAK Dialect",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				wingFilter = args[0]
			}
			encoder := dialect.NewEncoder()
			if configPath != "" {
				if err := loadEntityConfig(encoder, configPath); err != nil {
					return fmt.Errorf("load entity config: %w", err)
				}
			}

			svc := &facade.Service{}
			if err := svc.Init(context.Background(), facade.Options{}); err != nil {
				return err
			}
			defer svc.Close()
			page, err := svc.ListMemories(context.Background(), facade.ListMemoryOptions{Limit: 200, Status: "active"})
			if err != nil {
				return fmt.Errorf("list canonical memories: %w", err)
			}
			if len(page.Items) == 0 {
				fmt.Println("No canonical memories to compress")
				return nil
			}

			processed := 0
			for _, memory := range page.Items {
				wing, _ := memory.Metadata["wing"].(string)
				room, _ := memory.Metadata["room"].(string)
				if wingFilter != "" && wing != wingFilter {
					continue
				}
				compressed := encoder.Compress(memory.Content, map[string]string{"wing": wing, "room": room})
				stats := encoder.CompressionStats(memory.Content, compressed)
				if dryRun {
					fmt.Printf("Memory: %s\n", memory.ID)
					fmt.Printf("  Wing: %s, Room: %s\n", wing, room)
					fmt.Printf("  Original: %d chars, ~%d tokens\n", stats.OriginalChars, stats.OriginalTokensEst)
					fmt.Printf("  Compressed: %d chars, ~%d tokens\n", stats.SummaryChars, stats.SummaryTokensEst)
					fmt.Printf("  Ratio: %.2fx\n  -> %s\n\n", stats.SizeRatio, compressed)
				} else {
					fmt.Printf("%s: %s\n", memory.ID, compressed)
				}
				processed++
			}
			if processed == 0 {
				fmt.Println("No canonical memories matched")
				return nil
			}
			fmt.Printf("Processed: %d canonical memories\n", processed)
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
