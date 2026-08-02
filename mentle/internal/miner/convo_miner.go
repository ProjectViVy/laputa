package miner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dashimaki/mentle/internal/extractor"
	"github.com/dashimaki/mentle/internal/palace"
)

type ConversationMiner struct {
	miner     *Miner
	extractor *extractor.Extractor
}

func NewConversationMiner(miner *Miner) *ConversationMiner {
	return &ConversationMiner{
		miner:     miner,
		extractor: extractor.NewExtractor(),
	}
}

type Exchange struct {
	User      string
	Assistant string
}

func detectFormat(filename string, content []byte) string {
	filenameLower := strings.ToLower(filename)
	if strings.Contains(filenameLower, "claude") {
		return "claude"
	}
	if strings.Contains(filenameLower, "chatgpt") || strings.Contains(filenameLower, "gpt") {
		return "chatgpt"
	}
	if strings.Contains(filenameLower, "slack") {
		return "slack"
	}

	if strings.Contains(string(content), `"role"`) && strings.Contains(string(content), `"content"`) {
		return "chatgpt"
	}
	if strings.Contains(string(content), `"user"`) && strings.Contains(string(content), `"text"`) {
		return "slack"
	}
	if strings.Contains(string(content), "Human:") || strings.Contains(string(content), "Assistant:") || strings.Contains(string(content), "### Response") {
		return "claude"
	}

	return "unknown"
}

func parseClaudeExchanges(content []byte) []Exchange {
	var exchanges []Exchange
	text := string(content)

	humanAssistantRegex := regexp.MustCompile(`(?im)^Human:\s*(.+?)(?=^Assistant:|$)\s*Assistant:\s*(.+?)(?=^Human:|$)`)
	matches := humanAssistantRegex.FindAllStringSubmatch(text, -1)
	for _, m := range matches {
		exchanges = append(exchanges, Exchange{
			User:      strings.TrimSpace(m[1]),
			Assistant: strings.TrimSpace(m[2]),
		})
	}

	if len(exchanges) == 0 {
		responseRegex := regexp.MustCompile(`(?im)^### Response\s*\n\s*(.+?)(?=^### |$)`)
		userRegex := regexp.MustCompile(`(?im)^### (?:Human|User|Q)\s*\n\s*(.+?)(?=^### Response|$)`)
		responseMatches := responseRegex.FindAllStringSubmatch(text, -1)
		userMatches := userRegex.FindAllStringSubmatch(text, -1)

		minLen := min(len(userMatches), len(responseMatches))
		for i := range minLen {
			user := ""
			assistant := ""
			if i < len(userMatches) {
				user = strings.TrimSpace(userMatches[i][1])
			}
			if i < len(responseMatches) {
				assistant = strings.TrimSpace(responseMatches[i][1])
			}
			if user != "" || assistant != "" {
				exchanges = append(exchanges, Exchange{User: user, Assistant: assistant})
			}
		}
	}

	return exchanges
}

func parseChatGPTExchanges(content []byte) []Exchange {
	var exchanges []Exchange

	var messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	if err := json.Unmarshal(content, &messages); err == nil {
		var currentUser string
		for _, msg := range messages {
			msgContent := strings.TrimSpace(msg.Content)
			if msgContent == "" {
				continue
			}
			switch strings.ToLower(msg.Role) {
			case "user", "human":
				if currentUser != "" {
					exchanges = append(exchanges, Exchange{User: currentUser, Assistant: ""})
				}
				currentUser = msgContent
			case "assistant", "ai", "bot":
				if currentUser != "" {
					exchanges = append(exchanges, Exchange{User: currentUser, Assistant: msgContent})
					currentUser = ""
				} else {
					exchanges = append(exchanges, Exchange{User: "", Assistant: msgContent})
				}
			case "system":
			default:
				if currentUser != "" && msgContent != "" {
					exchanges = append(exchanges, Exchange{User: currentUser, Assistant: msgContent})
					currentUser = ""
				}
			}
		}
		if currentUser != "" {
			exchanges = append(exchanges, Exchange{User: currentUser, Assistant: ""})
		}
		return exchanges
	}

	var rawData []map[string]any
	if err := json.Unmarshal(content, &rawData); err == nil {
		for _, item := range rawData {
			if role, ok := item["role"].(string); ok {
				if content, ok := item["content"].(string); ok {
					content = strings.TrimSpace(content)
					if content == "" {
						continue
					}
					switch strings.ToLower(role) {
					case "user", "human":
						exchanges = append(exchanges, Exchange{User: content, Assistant: ""})
					case "assistant", "ai":
						if len(exchanges) > 0 && exchanges[len(exchanges)-1].Assistant == "" {
							exchanges[len(exchanges)-1].Assistant = content
						} else {
							exchanges = append(exchanges, Exchange{User: "", Assistant: content})
						}
					}
				}
			}
		}
	}

	return exchanges
}

func parseSlackExchanges(content []byte) []Exchange {
	var exchanges []Exchange

	var messages []struct {
		User string `json:"user"`
		Text string `json:"text"`
	}

	if err := json.Unmarshal(content, &messages); err == nil {
		var currentUser string
		var currentText string
		for _, msg := range messages {
			text := strings.TrimSpace(msg.Text)
			if text == "" {
				continue
			}
			user := strings.TrimSpace(msg.User)
			if currentUser == "" {
				currentUser = user
				currentText = text
			} else if user != currentUser {
				exchanges = append(exchanges, Exchange{User: currentText, Assistant: ""})
				currentUser = user
				currentText = text
			} else {
				currentText += " " + text
			}
		}
		if currentText != "" {
			exchanges = append(exchanges, Exchange{User: currentText, Assistant: ""})
		}
		return exchanges
	}

	var slackData struct {
		Messages []struct {
			User string `json:"user"`
			Text string `json:"text"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(content, &slackData); err == nil {
		var currentUser string
		var currentText string
		for _, msg := range slackData.Messages {
			text := strings.TrimSpace(msg.Text)
			if text == "" {
				continue
			}
			user := strings.TrimSpace(msg.User)
			if currentUser == "" {
				currentUser = user
				currentText = text
			} else if user != currentUser {
				exchanges = append(exchanges, Exchange{User: currentText, Assistant: ""})
				currentUser = user
				currentText = text
			} else {
				currentText += " " + text
			}
		}
		if currentText != "" {
			exchanges = append(exchanges, Exchange{User: currentText, Assistant: ""})
		}
	}

	return exchanges
}

func buildExchangeText(exchange Exchange) string {
	var parts []string
	if exchange.User != "" {
		parts = append(parts, fmt.Sprintf("User: %s", exchange.User))
	}
	if exchange.Assistant != "" {
		parts = append(parts, fmt.Sprintf("Assistant: %s", exchange.Assistant))
	}
	return strings.Join(parts, "\n")
}

func (cm *ConversationMiner) MineConversations(ctx context.Context, dir, wing string) error {
	if wing == "" {
		wing = filepath.Base(dir)
	}

	var totalFiles int
	var totalExchanges int
	var stored int

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".json" && ext != ".jsonl" && ext != ".txt" && ext != ".md" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		format := detectFormat(path, content)
		if format == "unknown" {
			return nil
		}

		totalFiles++

		var exchanges []Exchange
		switch format {
		case "claude":
			exchanges = parseClaudeExchanges(content)
		case "chatgpt":
			exchanges = parseChatGPTExchanges(content)
		case "slack":
			exchanges = parseSlackExchanges(content)
		}

		for _, exchange := range exchanges {
			totalExchanges++
			exchangeText := buildExchangeText(exchange)
			if exchangeText == "" {
				continue
			}

			memories := cm.extractor.ExtractMemories(exchangeText, 0.3)
			if len(memories) == 0 {
				memories = []extractor.Memory{{
					Content:    exchangeText,
					MemoryType: "conversation",
				}}
			}

			for _, mem := range memories {
				// Detect room from memory type and content
				room := detectConversationRoom(mem.MemoryType, exchangeText)

				drawer := palace.Drawer{
					ID:         generateID(),
					Content:    mem.Content,
					Wing:       wing,
					Room:       room,
					SourceFile: path,
					AddedBy:    "convo-miner",
					FiledAt:    time.Now(),
					Metadata: map[string]string{
						"format":      format,
						"memory_type": mem.MemoryType,
					},
				}

				if err := cm.miner.searcher.Store(ctx, drawer); err == nil {
					stored++
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	fmt.Printf("Processed %d conversation files, %d exchanges, stored %d memories into wing '%s'\n",
		totalFiles, totalExchanges, stored, wing)
	return nil
}

// detectConversationRoom determines the room from memory type and content.
func detectConversationRoom(memoryType, content string) string {
	lower := strings.ToLower(content)

	switch memoryType {
	case "decision":
		return "decisions"
	case "preference":
		return "preferences"
	case "milestone":
		return "milestones"
	case "problem":
		return "problems"
	case "emotional":
		return "reflections"
	}

	// Topic-based detection from content
	if containsAny(lower, "technical", "architecture", "api", "database", "server") {
		return "technical"
	}
	if containsAny(lower, "planning", "design", "requirements", "spec") {
		return "planning"
	}
	if containsAny(lower, "decided", "chose", "switched", "migrated") {
		return "decisions"
	}
	if containsAny(lower, "prefer", "always", "never", "style") {
		return "preferences"
	}
	if containsAny(lower, "bug", "error", "fix", "issue", "debug") {
		return "problems"
	}

	return "general"
}
