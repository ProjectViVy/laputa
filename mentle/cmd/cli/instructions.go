package cli

import (
	"fmt"

	"github.com/dashimaki/mentle/internal/instructions"
	"github.com/spf13/cobra"
)

func newInstructionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "instructions [init|search|mine|help|status]",
		Short: "Show instructions for a command",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			instruction, found := instructions.GetInstruction(name)
			if !found {
				return fmt.Errorf("invalid instruction name: %s", name)
			}
			fmt.Print(instruction)
			return nil
		},
	}
}
