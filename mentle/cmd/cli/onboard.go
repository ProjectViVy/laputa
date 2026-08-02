package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dashimaki/mentle/internal/config"
	"github.com/dashimaki/mentle/internal/registry"
	"github.com/spf13/cobra"
)

func newOnboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "onboard",
		Short: "Interactive first-run setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)

			fmt.Println("Welcome to mempalace! Let's set up your memory palace.")
			fmt.Println()

			// Mode selection
			fmt.Println("What mode will you use? (work/personal/combo)")
			mode := promptInput(reader, "Mode", "combo")

			// Palace path
			defaultPath := "~/.mempalace/palace"
			palacePath := promptInput(reader, "Palace path", defaultPath)

			// People
			fmt.Println("\nEnter people you work with (comma-separated, or empty to skip):")
			peopleInput := promptInput(reader, "People", "")

			// Projects
			fmt.Println("Enter active projects (comma-separated, or empty to skip):")
			projectsInput := promptInput(reader, "Projects", "")

			// Wings
			fmt.Println("Enter topic wings (comma-separated, or empty to skip):")
			wingsInput := promptInput(reader, "Topic wings", "")

			// Expand paths
			if strings.HasPrefix(palacePath, "~") {
				home, _ := os.UserHomeDir()
				palacePath = strings.Replace(palacePath, "~", home, 1)
			}

			// Create directories
			if err := os.MkdirAll(palacePath, 0755); err != nil {
				return fmt.Errorf("create palace dir: %w", err)
			}

			// Create config
			cfg := &config.Config{
				PalacePath:     palacePath,
				CollectionName: "mempalace_drawers",
				ModelName:      "sentence-transformers/all-MiniLM-L6-v2",
			}

			// Parse people
			var people []string
			if peopleInput != "" {
				for p := range strings.SplitSeq(peopleInput, ",") {
					p = strings.TrimSpace(p)
					if p != "" {
						people = append(people, p)
					}
				}
			}

			// Parse projects
			var projects []string
			if projectsInput != "" {
				for p := range strings.SplitSeq(projectsInput, ",") {
					p = strings.TrimSpace(p)
					if p != "" {
						projects = append(projects, p)
					}
				}
			}

			// Parse wings
			var wings []string
			if wingsInput != "" {
				for w := range strings.SplitSeq(wingsInput, ",") {
					w = strings.TrimSpace(w)
					if w != "" {
						wings = append(wings, w)
					}
				}
			}

			// Create entity registry
			reg, err := registry.New(palacePath)
			if err != nil {
				return fmt.Errorf("create registry: %w", err)
			}

			for _, person := range people {
				reg.Add(person, "person", 0.95, "onboarding")
			}

			for _, project := range projects {
				reg.Add(project, "project", 0.9, "onboarding")
			}

			if err := reg.Save(); err != nil {
				return fmt.Errorf("save registry: %w", err)
			}

			// Save config
			configDir := filepath.Join(os.ExpandEnv(os.Getenv("HOME")), ".mempalace")
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return fmt.Errorf("create config dir: %w", err)
			}

			configPath := filepath.Join(configDir, "config.json")
			configData, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal config: %w", err)
			}

			if err := os.WriteFile(configPath, configData, 0644); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			// Create identity file
			identityPath := filepath.Join(filepath.Dir(palacePath), "identity.txt")
			if _, err := os.Stat(identityPath); os.IsNotExist(err) {
				fmt.Println("\nWould you like to create an identity file? (yes/no)")
				if promptYesNo(reader, "yes") {
					fmt.Println("Enter a brief description of yourself (for L0 memory layer):")
					identity := promptInput(reader, "Identity", "")
					if identity != "" {
						if err := os.WriteFile(identityPath, []byte(identity), 0644); err != nil {
							fmt.Printf("Warning: failed to write identity file: %v\n", err)
						}
					}
				}
			}

			// Create WAL directory
			walPath := filepath.Join(palacePath, "wal")
			if err := os.MkdirAll(walPath, 0755); err != nil {
				return fmt.Errorf("create WAL dir: %w", err)
			}

			// Create diary directory
			diaryPath := filepath.Join(palacePath, "diary")
			if err := os.MkdirAll(diaryPath, 0755); err != nil {
				return fmt.Errorf("create diary dir: %w", err)
			}

			// Create sessions directory
			sessionsPath := filepath.Join(palacePath, "sessions")
			if err := os.MkdirAll(sessionsPath, 0755); err != nil {
				return fmt.Errorf("create sessions dir: %w", err)
			}

			fmt.Println("\nSetup complete!")
			fmt.Printf("Palace path: %s\n", palacePath)
			fmt.Printf("Mode: %s\n", mode)
			if len(people) > 0 {
				fmt.Printf("People: %s\n", strings.Join(people, ", "))
			}
			if len(projects) > 0 {
				fmt.Printf("Projects: %s\n", strings.Join(projects, ", "))
			}
			if len(wings) > 0 {
				fmt.Printf("Wings: %s\n", strings.Join(wings, ", "))
			}
			fmt.Printf("Registry: %d entities\n", reg.Count())
			fmt.Println("\nYou can now run 'mempalace-go server' to start the MCP server.")
			return nil
		},
	}
}

func promptInput(reader *bufio.Reader, prompt, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func promptYesNo(reader *bufio.Reader, defaultVal string) bool {
	input := promptInput(reader, "(yes/no)", defaultVal)
	return strings.ToLower(input) == "yes" || strings.ToLower(input) == "y"
}
