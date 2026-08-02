package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dashimaki/mentle/internal/config"
	"github.com/dashimaki/mentle/internal/embedder"
	"github.com/dashimaki/mentle/internal/miner"
	"github.com/dashimaki/mentle/internal/palace"
	"github.com/dashimaki/mentle/internal/search"
	govector "github.com/dashimaki/mentle/storage/govector"

	"github.com/spf13/cobra"
)

type HookInput struct {
	SessionID string `json:"session_id"`
	Timestamp string `json:"timestamp"`
	Wing      string `json:"wing"`
	Room      string `json:"room"`
	Content   string `json:"content"`
}

type HookOutput struct {
	Success bool   `json:"success"`
	Result  any    `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

type SessionState struct {
	SessionID    string `json:"session_id"`
	StartedAt    string `json:"started_at"`
	MessageCount int    `json:"message_count"`
	Wing         string `json:"wing"`
	LastSavedAt  string `json:"last_saved_at"`
}

const (
	SaveInterval = 15
)

func newHookCmd() *cobra.Command {
	var harness string
	var autoIngestDir string

	cmd := &cobra.Command{
		Use:   "hook [session-start|stop|precompact]",
		Short: "Run hook logic for Claude Code/Codex integration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			action := args[0]

			var input HookInput
			if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
				output := HookOutput{
					Success: false,
					Error:   fmt.Sprintf("failed to decode input: %v", err),
				}
				json.NewEncoder(os.Stdout).Encode(output)
				return err
			}

			var result any
			var errMsg string

			switch action {
			case "session-start":
				result, errMsg = handleSessionStart(input, harness)
			case "stop":
				result, errMsg = handleStop(input, harness, autoIngestDir)
			case "precompact":
				result, errMsg = handlePrecompact(input, harness, autoIngestDir)
			default:
				errMsg = fmt.Sprintf("unknown hook action: %s", action)
			}

			output := HookOutput{
				Success: errMsg == "",
				Result:  result,
				Error:   errMsg,
			}

			if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
				return fmt.Errorf("failed to encode output: %v", err)
			}

			if errMsg != "" {
				return fmt.Errorf("%s", errMsg)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&harness, "harness", "claude-code", "Harness type: claude-code or codex")
	cmd.Flags().StringVar(&autoIngestDir, "auto-ingest", "", "Directory to auto-ingest on stop/precompact")

	return cmd
}

func sessionStatePath(palacePath, sessionID string) string {
	return filepath.Join(palacePath, "sessions", sessionID+".json")
}

func loadSessionState(palacePath, sessionID string) (*SessionState, error) {
	path := sessionStatePath(palacePath, sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func saveSessionState(palacePath string, state *SessionState) error {
	path := sessionStatePath(palacePath, state.SessionID)
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func handleSessionStart(input HookInput, harness string) (map[string]any, string) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, fmt.Sprintf("config: %v", err)
	}

	palacePath := os.ExpandEnv(cfg.PalacePath)
	if strings.HasPrefix(palacePath, "~") {
		home, _ := os.UserHomeDir()
		palacePath = home + palacePath[1:]
	}

	sessionDir := filepath.Join(palacePath, "sessions")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return nil, fmt.Sprintf("create session dir: %v", err)
	}

	state := &SessionState{
		SessionID:    input.SessionID,
		StartedAt:    time.Now().Format(time.RFC3339),
		MessageCount: 0,
		Wing:         input.Wing,
	}

	if err := saveSessionState(palacePath, state); err != nil {
		return nil, fmt.Sprintf("write session state: %v", err)
	}

	return map[string]any{
		"session_id": input.SessionID,
		"status":     "initialized",
		"harness":    harness,
		"wing":       input.Wing,
	}, ""
}

func handleStop(input HookInput, harness string, autoIngestDir string) (map[string]any, string) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, fmt.Sprintf("config: %v", err)
	}

	palacePath := os.ExpandEnv(cfg.PalacePath)
	if strings.HasPrefix(palacePath, "~") {
		home, _ := os.UserHomeDir()
		palacePath = home + palacePath[1:]
	}

	// Load session state
	state, _ := loadSessionState(palacePath, input.SessionID)
	if state != nil {
		state.MessageCount++
	}

	// Check if we've hit the save interval
	shouldSave := input.Content != "" || (state != nil && state.MessageCount%SaveInterval == 0)

	if !shouldSave && autoIngestDir == "" {
		if state != nil {
			saveSessionState(palacePath, state)
		}
		return map[string]any{
			"session_id":    input.SessionID,
			"status":        "skipped (save interval not reached)",
			"message_count": state.MessageCount,
			"next_save_at":  (SaveInterval - (state.MessageCount % SaveInterval)) * 15,
		}, ""
	}

	// Save content
	if input.Content != "" {
		wing := input.Wing
		if wing == "" {
			wing = "default"
		}
		room := input.Room
		if room == "" {
			room = "session-notes"
		}

		if saved, errMsg := saveContentToPalace(palacePath, cfg, wing, room, input.Content); !saved {
			return nil, errMsg
		}
	}

	// Auto-ingest directory if specified
	autoIngested := false
	if autoIngestDir != "" {
		if _, errMsg := ingestDirectory(palacePath, cfg, autoIngestDir, input.Wing); errMsg != "" {
			return nil, errMsg
		}
		autoIngested = true
	}

	// Update state
	if state != nil {
		state.LastSavedAt = time.Now().Format(time.RFC3339)
		saveSessionState(palacePath, state)
	}

	return map[string]any{
		"session_id":    input.SessionID,
		"status":        "saved",
		"content_saved": input.Content != "",
		"auto_ingested": autoIngested,
		"message_count": state.MessageCount,
	}, ""
}

func handlePrecompact(input HookInput, harness string, autoIngestDir string) (map[string]any, string) {
	if input.Content == "" {
		return map[string]any{
			"session_id": input.SessionID,
			"status":     "no content to save before compaction",
		}, ""
	}

	cfg, err := config.Load("")
	if err != nil {
		return nil, fmt.Sprintf("config: %v", err)
	}

	palacePath := os.ExpandEnv(cfg.PalacePath)
	if strings.HasPrefix(palacePath, "~") {
		home, _ := os.UserHomeDir()
		palacePath = home + palacePath[1:]
	}

	wing := input.Wing
	if wing == "" {
		wing = "default"
	}
	room := input.Room
	if room == "" {
		room = "precompact-save"
	}

	if saved, errMsg := saveContentToPalace(palacePath, cfg, wing, room, input.Content); !saved {
		return nil, errMsg
	}

	// Update session state
	state, _ := loadSessionState(palacePath, input.SessionID)
	if state != nil {
		state.MessageCount++
		state.LastSavedAt = time.Now().Format(time.RFC3339)
		saveSessionState(palacePath, state)
	}

	return map[string]any{
		"session_id": input.SessionID,
		"status":     "emergency saved",
		"wing":       wing,
		"room":       room,
	}, ""
}

// saveContentToPalace stores content into the palace vector store.
func saveContentToPalace(palacePath string, cfg *config.Config, wing, room, content string) (bool, string) {
	ctx := context.Background()
	emb, err := embedder.New("", cfg.GetModelsDir())
	if err != nil {
		return false, fmt.Sprintf("embedder: %v", err)
	}
	defer emb.Close()

	vectorDB, err := govector.NewStore(palacePath+"/vectors.db", 384)
	if err != nil {
		return false, fmt.Sprintf("vector store: %v", err)
	}

	searcher := search.NewSearcher(vectorDB, emb)

	if err := searcher.Store(ctx, palace.Drawer{
		Wing:    wing,
		Room:    room,
		Content: content,
	}); err != nil {
		return false, fmt.Sprintf("store: %v", err)
	}
	return true, ""
}

// ingestDirectory mines a directory into the palace.
func ingestDirectory(palacePath string, cfg *config.Config, dir, wing string) (bool, string) {
	ctx := context.Background()
	emb, err := embedder.New("", cfg.GetModelsDir())
	if err != nil {
		return false, fmt.Sprintf("embedder: %v", err)
	}
	defer emb.Close()

	vectorDB, err := govector.NewStore(palacePath+"/vectors.db", 384)
	if err != nil {
		return false, fmt.Sprintf("vector store: %v", err)
	}

	searcher := search.NewSearcher(vectorDB, emb)

	// Load gitignore if present
	m := miner.NewMiner(searcher)
	m.LoadGitignore(dir)

	if err := m.MineProject(ctx, dir, wing); err != nil {
		return false, fmt.Sprintf("mine: %v", err)
	}
	return true, ""
}
