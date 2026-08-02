package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

type transcriptFormat int

const (
	formatUnknown transcriptFormat = iota
	formatClaude
	formatChatGPT
	formatSlack
)

type session struct {
	timestamp string
	content   strings.Builder
}

var (
	outputDir   string
	dryRun      bool
	minSessions int
)

func newSplitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "split [directory]",
		Short: "Split mega transcript files into per-session files",
		Args:  cobra.ExactArgs(1),
		RunE:  runSplit,
	}
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Output directory (default: same as input)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be split without creating files")
	cmd.Flags().IntVar(&minSessions, "min-sessions", 2, "Minimum sessions required to split")
	return cmd
}

func runSplit(cmd *cobra.Command, args []string) error {
	dir := args[0]
	if minSessions < 1 {
		minSessions = 1
	}

	files, err := findTranscriptFiles(dir)
	if err != nil {
		return fmt.Errorf("finding transcripts: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No transcript files found")
		return nil
	}

	fmt.Printf("Found %d transcript file(s)\n", len(files))

	for _, file := range files {
		sessions, format, err := splitFile(file)
		if err != nil {
			fmt.Printf("Error splitting %s: %v\n", file, err)
			continue
		}

		if len(sessions) < minSessions {
			fmt.Printf("Skipping %s: only %d session(s) found (min: %d)\n",
				filepath.Base(file), len(sessions), minSessions)
			continue
		}

		fmt.Printf("%s (%s): %d sessions\n", filepath.Base(file), formatName(format), len(sessions))

		if dryRun {
			for i, s := range sessions {
				preview := s.content.String()
				if len(preview) > 50 {
					preview = preview[:50] + "..."
				}
				fmt.Printf("  Session %d [%s]: %s\n", i+1, s.timestamp, preview)
			}
		} else {
			outDir := outputDir
			if outDir == "" {
				outDir = filepath.Dir(file)
			}
			outDir = filepath.Join(outDir, "split")
			if err := os.MkdirAll(outDir, 0755); err != nil {
				return fmt.Errorf("creating output dir: %w", err)
			}

			base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
			for i, s := range sessions {
				filename := fmt.Sprintf("%s_session_%d_%s.txt", base, i+1, sessions[i].timestamp)
				filename = sanitizeFilename(filename)
				outPath := filepath.Join(outDir, filename)
				if err := os.WriteFile(outPath, []byte(s.content.String()), 0644); err != nil {
					return fmt.Errorf("writing session file: %w", err)
				}
			}
			fmt.Printf("  -> %s/\n", outDir)
		}
	}

	return nil
}

func findTranscriptFiles(dir string) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		format := detectFormat(path)
		if format != formatUnknown {
			files = append(files, path)
		}
	}
	return files, nil
}

func detectFormat(path string) transcriptFormat {
	name := strings.ToLower(filepath.Base(path))

	if strings.Contains(name, "claude") {
		return formatClaude
	}
	if strings.Contains(name, "chatgpt") || strings.Contains(name, "openai") {
		return formatChatGPT
	}
	if strings.Contains(name, "slack") {
		return formatSlack
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return formatUnknown
	}

	lower := strings.ToLower(string(content))

	if strings.Contains(lower, "### response") || strings.Contains(lower, "human:") && strings.Contains(lower, "assistant:") {
		return formatClaude
	}

	var jsonCheck struct {
		Message string `json:"message"`
		Role    string `json:"role"`
	}
	if json.Unmarshal(content, &jsonCheck) == nil && jsonCheck.Message != "" && jsonCheck.Role != "" {
		return formatChatGPT
	}

	if strings.Contains(lower, `"user":`) && strings.Contains(lower, `"text":`) {
		return formatSlack
	}

	return formatUnknown
}

func formatName(f transcriptFormat) string {
	switch f {
	case formatClaude:
		return "Claude"
	case formatChatGPT:
		return "ChatGPT"
	case formatSlack:
		return "Slack"
	default:
		return "Unknown"
	}
}

func splitFile(path string) ([]session, transcriptFormat, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, formatUnknown, err
	}

	format := detectFormat(path)

	var sessions []session
	var current session

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()

		if ts := extractTimestamp(line); ts != "" {
			if current.content.Len() > 50 {
				sessions = append(sessions, current)
			}
			current = session{timestamp: ts}
		}

		if current.timestamp != "" {
			if current.content.Len() > 0 {
				current.content.WriteString("\n")
			}
			current.content.WriteString(line)
		}
	}

	if current.content.Len() > 50 {
		sessions = append(sessions, current)
	}

	return sessions, format, nil
}

func extractTimestamp(line string) string {
	patterns := []string{
		`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})`,
		`(\d{2}:\d{2}:\d{2})`,
		`(\d{2}:\d{2})`,
	}

	for _, pat := range patterns {
		re := regexp.MustCompile(pat)
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

func sanitizeFilename(name string) string {
	re := regexp.MustCompile(`[^\w\-.]+`)
	return re.ReplaceAllString(name, "_")
}
