package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Print MCP setup instructions for Claude CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			palacePath := ""
			if cmd.Flags().Changed("palace") {
				palacePath, _ = cmd.Flags().GetString("palace")
			}

			header := "MemPalace MCP quick setup:"
			if palacePath != "" {
				header = fmt.Sprintf("MemPalace MCP quick setup (palace: %s):", palacePath)
			}

			serverCmd := "./mempalace-go server"
			if palacePath != "" {
				serverCmd = fmt.Sprintf("%s --palace %s", serverCmd, palacePath)
			}

			mcpCmd := fmt.Sprintf("claude mcp add mempalace -- %s", serverCmd)

			fmt.Printf("%s\n", header)
			fmt.Printf("  %s\n\n", mcpCmd)
			fmt.Printf("Run the server directly:\n")
			fmt.Printf("  %s\n", serverCmd)
			return nil
		},
	}
	cmd.Flags().String("palace", "", "Palace path")
	return cmd
}
