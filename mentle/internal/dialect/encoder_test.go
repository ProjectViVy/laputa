package dialect

import (
	"slices"
	"strings"
	"testing"
)

func TestCompress(t *testing.T) {
	encoder := NewEncoder()

	tests := []struct {
		name     string
		text     string
		metadata map[string]string
		wantPart string
	}{
		{
			name:     "basic text",
			text:     "I decided to use the API for the new architecture project.",
			metadata: nil,
			wantPart: "0:",
		},
		{
			name:     "text with entities",
			text:     "John and Mary discussed the core principles of the system.",
			metadata: nil,
			wantPart: "0:",
		},
		{
			name:     "text with emotions",
			text:     "I was worried about the deployment but excited to see it work.",
			metadata: nil,
			wantPart: "0:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encoder.Compress(tt.text, tt.metadata)
			if !strings.HasPrefix(result, "0:") {
				t.Errorf("Compress() = %v, want prefix 0:", result)
			}
		})
	}
}

func TestDetectEmotions(t *testing.T) {
	encoder := NewEncoder()

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "love emotion",
			text:     "I love this approach to solving problems.",
			expected: []string{"love"},
		},
		{
			name:     "fear emotion",
			text:     "I fear this might not work correctly.",
			expected: []string{"fear"},
		},
		{
			name:     "hope emotion",
			text:     "I hope we can finish this soon.",
			expected: []string{"hope"},
		},
		{
			name:     "multiple emotions",
			text:     "I was worried and frustrated about the confusion.",
			expected: []string{"anx", "frust", "confuse"},
		},
		{
			name:     "no emotions",
			text:     "The system processed the request successfully.",
			expected: nil,
		},
		{
			name:     "happy joy emotion",
			text:     "I am so happy with the results!",
			expected: []string{"joy"},
		},
		{
			name:     "sad grief emotion",
			text:     "I feel sad about what happened.",
			expected: []string{"grief"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encoder.detectEmotions(tt.text)
			if len(tt.expected) == 0 && len(got) == 0 {
				return
			}
			if len(got) == 0 && len(tt.expected) > 0 {
				t.Errorf("detectEmotions() returned empty, want at least %v", tt.expected)
				return
			}
			found := slices.Contains(tt.expected, got[0])
			if !found && len(tt.expected) > 0 {
				t.Errorf("detectEmotions() first emotion = %v, want one of %v", got[0], tt.expected)
			}
		})
	}
}

func TestDetectFlags(t *testing.T) {
	encoder := NewEncoder()

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "decision flag",
			text:     "We decided to move forward with the plan.",
			expected: []string{"DECISION"},
		},
		{
			name:     "origin flag",
			text:     "I founded this project in 2020.",
			expected: []string{"ORIGIN"},
		},
		{
			name:     "core flag",
			text:     "This is the core principle we follow.",
			expected: []string{"CORE"},
		},
		{
			name:     "pivot flag",
			text:     "It was a turning point in the project.",
			expected: []string{"PIVOT"},
		},
		{
			name:     "technical flag",
			text:     "We need to fix the API and database issues.",
			expected: []string{"TECHNICAL"},
		},
		{
			name:     "multiple flags",
			text:     "I created the core architecture and deployed it.",
			expected: []string{"ORIGIN", "CORE", "TECHNICAL"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encoder.detectFlags(tt.text)
			if len(tt.expected) == 0 && len(got) == 0 {
				return
			}
			if len(got) == 0 && len(tt.expected) > 0 {
				t.Errorf("detectFlags() returned empty, want at least %v", tt.expected)
				return
			}
			found := slices.Contains(tt.expected, got[0])
			if !found && len(tt.expected) > 0 {
				t.Errorf("detectFlags() first flag = %v, want one of %v", got[0], tt.expected)
			}
		})
	}
}

func TestExtractTopics(t *testing.T) {
	encoder := NewEncoder()

	tests := []struct {
		name      string
		text      string
		maxTopics int
		minCount  int
	}{
		{
			name:      "extract programming topics",
			text:      "Python and Go are programming languages used for building software systems.",
			maxTopics: 3,
			minCount:  1,
		},
		{
			name:      "skip stop words",
			text:      "The quick brown fox jumps over the lazy dog.",
			maxTopics: 3,
			minCount:  0,
		},
		{
			name:      "boost proper nouns",
			text:      "Python is different from Java and JavaScript in many ways.",
			maxTopics: 3,
			minCount:  1,
		},
		{
			name:      "empty text",
			text:      "",
			maxTopics: 3,
			minCount:  0,
		},
		{
			name:      "short words filtered",
			text:      "I am in on at to of for.",
			maxTopics: 3,
			minCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encoder.extractTopics(tt.text, tt.maxTopics)
			if len(got) < tt.minCount {
				t.Errorf("extractTopics() returned %v, want at least %v items", got, tt.minCount)
			}
			if len(got) > tt.maxTopics {
				t.Errorf("extractTopics() returned %v, want at most %v items", got, tt.maxTopics)
			}
		})
	}
}

func TestExtractKeySentence(t *testing.T) {
	encoder := NewEncoder()

	tests := []struct {
		name     string
		text     string
		maxLen   int
		hasWords []string
	}{
		{
			name:     "decision sentence",
			text:     "First sentence here. I decided to use the new approach. Another sentence follows.",
			maxLen:   60,
			hasWords: []string{"decided"},
		},
		{
			name:     "short preferred",
			text:     "Short sentence. This is a much longer sentence that goes on and on with many words and ideas.",
			maxLen:   60,
			hasWords: []string{"Short"},
		},
		{
			name:     "key word boosted",
			text:     "Some text. The key insight is important. More text here.",
			maxLen:   60,
			hasWords: []string{"key"},
		},
		{
			name:     "very short sentences ignored",
			text:     "Hi. Ok. Yes. No.",
			maxLen:   0,
			hasWords: nil,
		},
		{
			name:     "long sentence truncated",
			text:     "This is a very long sentence that contains way too many words to fit within the optimal length threshold for effective communication and engagement with the reader.",
			maxLen:   60,
			hasWords: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encoder.extractKeySentence(tt.text)
			if tt.maxLen == 0 && got != "" {
				t.Errorf("extractKeySentence() = %v, want empty for very short sentences", got)
			}
			if len(got) > tt.maxLen && tt.maxLen > 0 {
				t.Errorf("extractKeySentence() = %v (len %d), want max len %d", got, len(got), tt.maxLen)
			}
		})
	}
}

func TestDetectEntities(t *testing.T) {
	encoder := NewEncoder()

	encoder.entityCodes["Alice"] = "ALI"
	encoder.entityCodes["Bob"] = "BOB"

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "known entities",
			text:     "Alice and Bob worked together on the project.",
			expected: []string{"ALI"},
		},
		{
			name:     "no known entities fallback capitalized",
			text:     "Python and Java are programming languages.",
			expected: []string{"PYT", "JAV"},
		},
		{
			name:     "mixed entities",
			text:     "Alice used Python for the project.",
			expected: []string{"ALI"},
		},
		{
			name:     "no entities",
			text:     "this is all lowercase text.",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encoder.detectEntities(tt.text)
			if len(tt.expected) == 0 && len(got) == 0 {
				return
			}
			if len(got) > 0 && len(tt.expected) > 0 {
				if got[0] != tt.expected[0] {
					t.Errorf("detectEntities() first entity = %v, want %v", got[0], tt.expected[0])
				}
			}
		})
	}
}

func TestNewEncoder(t *testing.T) {
	encoder := NewEncoder()
	if encoder == nil {
		t.Errorf("NewEncoder() returned nil")
	}
	if encoder.entityCodes == nil {
		t.Errorf("NewEncoder() entityCodes is nil")
	}
	if encoder.skipNames == nil {
		t.Errorf("NewEncoder() skipNames is nil")
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "item exists",
			slice:    []string{"a", "b", "c"},
			item:     "b",
			expected: true,
		},
		{
			name:     "item not exists",
			slice:    []string{"a", "b", "c"},
			item:     "d",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "a",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.slice, tt.item)
			if got != tt.expected {
				t.Errorf("contains() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCompressFormat(t *testing.T) {
	encoder := NewEncoder()

	result := encoder.Compress("I decided to use the API for architecture.", nil)

	parts := strings.Split(result, "|")
	if len(parts) < 2 {
		t.Errorf("Compress() should have at least 2 parts, got %d: %v", len(parts), result)
	}

	if !strings.HasPrefix(parts[0], "0:") {
		t.Errorf("Compress() first part should start with 0:, got %v", parts[0])
	}
}

func TestCompressEmptyText(t *testing.T) {
	encoder := NewEncoder()

	result := encoder.Compress("", nil)

	if !strings.Contains(result, "???") {
		t.Errorf("Compress() empty text should have ??? for entities, got %v", result)
	}
	if !strings.Contains(result, "misc") {
		t.Errorf("Compress() empty text should have misc for topics, got %v", result)
	}
}

func TestEmotionCodeMapping(t *testing.T) {
	encoder := NewEncoder()

	texts := map[string]string{
		"I love this":         "love",
		"I hate this":         "rage",
		"I am worried":        "anx",
		"I feel sad":          "grief",
		"I am happy":          "joy",
		"I trust you":         "trust",
		"I hope for the best": "hope",
		"I fear the unknown":  "fear",
		"I am grateful":       "grat",
		"I am surprised":      "surprise",
		"I am curious":        "curious",
		"I feel relieved":     "relief",
		"I wonder about this": "wonder",
		"I am frustrated":     "frust",
		"I am confused":       "confuse",
		"I am excited":        "excite",
	}

	for text, expected := range texts {
		emotions := encoder.detectEmotions(text)
		if len(emotions) == 0 {
			t.Errorf("detectEmotions(%q) returned empty, want %q", text, expected)
		} else if emotions[0] != expected {
			t.Errorf("detectEmotions(%q)[0] = %q, want %q", text, emotions[0], expected)
		}
	}
}

func TestFlagDetection(t *testing.T) {
	encoder := NewEncoder()

	texts := map[string]string{
		"We decided to proceed":        "DECISION",
		"I founded this company":       "ORIGIN",
		"It was created from scratch":  "ORIGIN",
		"This is the core of our work": "CORE",
		"It is fundamental to success": "CORE",
		"It was a turning point":       "PIVOT",
		"This changed everything":      "PIVOT",
		"We need to fix the API":       "TECHNICAL",
		"The database needs tuning":    "TECHNICAL",
		"We will deploy on Friday":     "TECHNICAL",
	}

	for text, expected := range texts {
		flags := encoder.detectFlags(text)
		if len(flags) == 0 {
			t.Errorf("detectFlags(%q) returned empty, want %q", text, expected)
		} else if flags[0] != expected {
			t.Errorf("detectFlags(%q)[0] = %q, want %q", text, flags[0], expected)
		}
	}
}
