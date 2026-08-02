package miner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// NormalizedMessage is the standard transcript format.
type NormalizedMessage struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp,omitempty"`
}

// NormalizedConversation is a collection of normalized messages.
type NormalizedConversation struct {
	Messages []NormalizedMessage `json:"messages"`
	Source   string              `json:"source"`
}

// TranscriptToNormalized converts a conversation to transcript format (> Role: content).
func (c *NormalizedConversation) TranscriptToNormalized() string {
	var lines []string
	for _, msg := range c.Messages {
		if msg.Content != "" {
			lines = append(lines, fmt.Sprintf("> %s: %s", msg.Role, msg.Content))
		}
	}
	return strings.Join(lines, "\n")
}

// NormalizeClaudeExport normalizes Claude.ai JSON export format.
// Supports both the array format and the nested message format.
func NormalizeClaudeExport(content []byte) (*NormalizedConversation, error) {
	result := &NormalizedConversation{Source: "claude"}

	// Try array of messages first
	var msgs []struct {
		Role      string `json:"role"`
		Content   string `json:"content"`
		Timestamp string `json:"timestamp"`
	}
	if err := json.Unmarshal(content, &msgs); err == nil {
		for _, m := range msgs {
			result.Messages = append(result.Messages, NormalizedMessage{
				Role:      m.Role,
				Content:   m.Content,
				Timestamp: m.Timestamp,
			})
		}
		return result, nil
	}

	// Try nested format
	var nested struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(content, &nested); err != nil {
		return nil, err
	}
	for _, m := range nested.Messages {
		result.Messages = append(result.Messages, NormalizedMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return result, nil
}

// NormalizeChatGPTExport normalizes ChatGPT conversations.json format.
func NormalizeChatGPTExport(content []byte) (*NormalizedConversation, error) {
	result := &NormalizedConversation{Source: "chatgpt"}

	// ChatGPT format is actually a map with "mapping" key
	var rawMap map[string]any
	if err := json.Unmarshal(content, &rawMap); err == nil {
		if mappingData, ok := rawMap["mapping"]; ok {
			if mappingMap, ok := mappingData.(map[string]any); ok {
				for _, v := range mappingMap {
					if entry, ok := v.(map[string]any); ok {
						if message, ok := entry["message"].(map[string]any); ok {
							var msg NormalizedMessage
							if author, ok := message["author"].(map[string]any); ok {
								if role, ok := author["role"].(string); ok {
									msg.Role = role
								}
							}
							if contentData, ok := message["content"].(map[string]any); ok {
								if parts, ok := contentData["parts"].([]any); ok {
									var partsStr []string
									for _, p := range parts {
										if s, ok := p.(string); ok {
											partsStr = append(partsStr, s)
										}
									}
									msg.Content = strings.Join(partsStr, " ")
								}
							}
							if ts, ok := message["create_time"].(float64); ok {
								msg.Timestamp = fmt.Sprintf("%v", ts)
							}
							if msg.Role != "" && msg.Content != "" {
								result.Messages = append(result.Messages, msg)
							}
						}
					}
				}
			}
		}
		return result, nil
	}

	return result, nil
}

// NormalizeClaudeCodeJSONL normalizes Claude Code JSONL format.
func NormalizeClaudeCodeJSONL(content []byte) (*NormalizedConversation, error) {
	result := &NormalizedConversation{Source: "claude-code"}

	lines := strings.SplitSeq(strings.TrimSpace(string(content)), "\n")
	for line := range lines {
		if line == "" {
			continue
		}
		var msg struct {
			Type      string `json:"type"`
			Message   string `json:"message"`
			Role      string `json:"role"`
			Content   string `json:"content"`
			Timestamp string `json:"timestamp"`
		}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		if msg.Message != "" {
			result.Messages = append(result.Messages, NormalizedMessage{
				Role:      msg.Type,
				Content:   msg.Message,
				Timestamp: msg.Timestamp,
			})
		} else if msg.Role != "" && msg.Content != "" {
			result.Messages = append(result.Messages, NormalizedMessage{
				Role:      msg.Role,
				Content:   msg.Content,
				Timestamp: msg.Timestamp,
			})
		}
	}
	return result, nil
}

// NormalizeSlackExport normalizes Slack JSON export format.
func NormalizeSlackExport(content []byte) (*NormalizedConversation, error) {
	result := &NormalizedConversation{Source: "slack"}

	var messages []struct {
		User      string  `json:"user"`
		Text      string  `json:"text"`
		Timestamp float64 `json:"ts"`
		BotID     string  `json:"bot_id"`
	}
	if err := json.Unmarshal(content, &messages); err != nil {
		return nil, err
	}

	for _, m := range messages {
		role := "user"
		if m.BotID != "" {
			role = "bot"
		}
		result.Messages = append(result.Messages, NormalizedMessage{
			Role:      role,
			Content:   m.Text,
			Timestamp: fmt.Sprintf("%v", m.Timestamp),
		})
	}
	return result, nil
}

// NormalizePlainText normalizes plain text with > markers.
func NormalizePlainText(content []byte) (*NormalizedConversation, error) {
	result := &NormalizedConversation{Source: "plain-text"}

	lines := strings.Split(string(content), "\n")
	var currentRole, currentContent string

	flush := func() {
		if currentRole != "" && currentContent != "" {
			result.Messages = append(result.Messages, NormalizedMessage{
				Role:    currentRole,
				Content: strings.TrimSpace(currentContent),
			})
		}
		currentRole = ""
		currentContent = ""
	}

	re := regexp.MustCompile(`^>\s*(\w+):\s*(.*)`)
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			flush()
			currentRole = matches[1]
			currentContent = matches[2]
		} else if currentRole != "" {
			if currentContent != "" {
				currentContent += "\n"
			}
			currentContent += line
		}
	}
	flush()

	return result, nil
}

// DetectFormat detects the export format from content.
func DetectFormat(content []byte) string {
	text := string(content)

	// Check for Claude Code JSONL (multiple JSON lines)
	if strings.Contains(text, "\n") {
		lines := strings.Split(strings.TrimSpace(text), "\n")
		if len(lines) > 1 {
			var obj map[string]any
			if err := json.Unmarshal([]byte(lines[0]), &obj); err == nil {
				if _, ok := obj["type"]; ok {
					return "claude-code-jsonl"
				}
			}
		}
	}

	// Check for Slack JSON array
	if strings.HasPrefix(strings.TrimSpace(text), "[") {
		var arr []map[string]any
		if err := json.Unmarshal(content, &arr); err == nil {
			if len(arr) > 0 {
				if _, hasUser := arr[0]["user"]; hasUser {
					if _, hasText := arr[0]["text"]; hasText {
						return "slack"
					}
				}
			}
		}
	}

	// Check for ChatGPT — either the full conversations.json mapping format
	// (contains "author" + "role") or the per-message wrapper format (top-level
	// "message" key whose value is a JSON object).
	if strings.Contains(text, `"author"`) && strings.Contains(text, `"role"`) {
		return "chatgpt"
	}
	var rawObj map[string]any
	if err := json.Unmarshal(content, &rawObj); err == nil {
		if msgVal, ok := rawObj["message"]; ok {
			if _, ok := msgVal.(map[string]any); ok {
				return "chatgpt"
			}
		}
	}

	// Check for Claude
	if strings.Contains(text, `"role"`) && strings.Contains(text, `"content"`) {
		return "claude"
	}

	// Check for plain text with > markers
	if strings.Contains(text, "> ") {
		return "plain-text"
	}

	return "unknown"
}

// Normalize auto-detects format and normalizes content.
func Normalize(content []byte) (*NormalizedConversation, error) {
	format := DetectFormat(content)

	switch format {
	case "claude":
		return NormalizeClaudeExport(content)
	case "chatgpt":
		return NormalizeChatGPTExport(content)
	case "claude-code-jsonl":
		return NormalizeClaudeCodeJSONL(content)
	case "slack":
		return NormalizeSlackExport(content)
	case "plain-text":
		return NormalizePlainText(content)
	default:
		return nil, fmt.Errorf("unknown format")
	}
}
