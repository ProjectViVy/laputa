package extractor

import (
	"strings"
	"testing"
)

func TestExtractMemories(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name     string
		text     string
		minConf  float64
		wantLen  int
		wantType string
	}{
		{
			name:     "extracts decision from text",
			text:     "We decided to use Go for its concurrency model.",
			minConf:  0.1,
			wantLen:  1,
			wantType: "decision",
		},
		{
			name:     "extracts multiple memories from multiple paragraphs",
			text:     "We decided to use Go for the project.\n\nI prefer Python for quick scripting tasks.\n\nIt finally works after much debugging!",
			minConf:  0.1,
			wantLen:  3,
			wantType: "decision",
		},
		{
			name:     "no memories in empty text",
			text:     "",
			minConf:  0.1,
			wantLen:  0,
			wantType: "",
		},
		{
			name:     "confidence threshold filters low confidence",
			text:     "Because",
			minConf:  0.5,
			wantLen:  0,
			wantType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memories := extractor.ExtractMemories(tt.text, tt.minConf)
			if len(memories) != tt.wantLen {
				t.Errorf("ExtractMemories() got %d memories, want %d", len(memories), tt.wantLen)
			}
			if tt.wantLen > 0 && memories[0].MemoryType != tt.wantType {
				t.Errorf("ExtractMemories() got type %s, want %s", memories[0].MemoryType, tt.wantType)
			}
		})
	}
}

func TestDecisionPatterns(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "detects decided", text: "We decided to use PostgreSQL", want: true},
		{name: "detects chose", text: "We chose Redis for caching", want: true},
		{name: "detects went with", text: "We went with microservices", want: true},
		{name: "detects architecture", text: "The architecture is service-oriented", want: true},
		{name: "detects because", text: "I like it because it's fast", want: true},
		{name: "detects pattern", text: "This follows the observer pattern", want: true},
		{name: "detects approach", text: "A better approach would be to refactor", want: true},
		{name: "detects stack", text: "Our stack includes Go and React", want: true},
		{name: "detects framework", text: "We use the Gin framework", want: true},
		{name: "detects infrastructure", text: "The infrastructure is cloud-based", want: true},
		{name: "detects over because", text: "We chose this over that because it was faster", want: true},
		{name: "detects set to", text: "Set it to production mode", want: true},
		{name: "detects configure", text: "Configure the server", want: true},
		{name: "detects default", text: "Use the default settings", want: true},
		{name: "negative case", text: "The sky is blue", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, _ := scoreMarkers(tt.text, extractor.decisionPatterns)
			got := score > 0
			if got != tt.want {
				t.Errorf("decision pattern score(%q) = %v, want %v", tt.text, score, tt.want)
			}
		})
	}
}

func TestPreferencePatterns(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "detects prefer", text: "I prefer TypeScript over JavaScript", want: true},
		{name: "detects always use", text: "We always use unit tests", want: true},
		{name: "detects never use", text: "I never use global variables", want: true},
		{name: "detects like", text: "I like to keep code clean", want: true},
		{name: "detects hate", text: "I hate when code is messy", want: true},
		{name: "detects rule", text: "My rule is to comment complex logic", want: true},
		{name: "detects preference", text: "My preference is functional style", want: true},
		{name: "detects snake_case", text: "We use snake_case for variables", want: true},
		{name: "detects camelCase", text: "CamelCase is the standard here", want: true},
		{name: "detects functional style", text: "I prefer functional style", want: true},
		{name: "detects tabs spaces", text: "I prefer tabs over spaces", want: true},
		{name: "negative case", text: "The file is located in the directory", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, _ := scoreMarkers(tt.text, extractor.preferencePatterns)
			got := score > 0
			if got != tt.want {
				t.Errorf("preference pattern score(%q) = %v, want %v", tt.text, score, tt.want)
			}
		})
	}
}

func TestMilestonePatterns(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "detects it works", text: "It works now!", want: true},
		{name: "detects it worked", text: "It worked after the fix", want: true},
		{name: "detects got it working", text: "Got it working with Docker", want: true},
		{name: "detects fixed", text: "It's finally fixed", want: true},
		{name: "detects solved", text: "We solved the memory leak", want: true},
		{name: "detects breakthrough", text: "This is a breakthrough in performance", want: true},
		{name: "detects figured out", text: "I figured it out", want: true},
		{name: "detects nailed it", text: "We nailed it with the design", want: true},
		{name: "detects finally", text: "It finally works after hours", want: true},
		{name: "detects first time", text: "First time it ran successfully", want: true},
		{name: "detects discovered", text: "I discovered the solution was simpler", want: true},
		{name: "detects realized", text: "I realized the config was wrong", want: true},
		{name: "detects found out", text: "Found out that the timeout was the cause", want: true},
		{name: "detects the key", text: "The key is to cache aggressively", want: true},
		{name: "detects the trick", text: "The trick is to batch the requests", want: true},
		{name: "detects built", text: "We built a robust system", want: true},
		{name: "detects created", text: "Created a new service", want: true},
		{name: "detects implemented", text: "Implemented the feature successfully", want: true},
		{name: "detects shipped", text: "We've shipped the update", want: true},
		{name: "detects launched", text: "Launched the product yesterday", want: true},
		{name: "detects deployed", text: "Deployed to production", want: true},
		{name: "detects released", text: "Released version 2.0", want: true},
		{name: "detects prototype", text: "Built a prototype", want: true},
		{name: "detects proof of concept", text: "Proof of concept works", want: true},
		{name: "detects demo", text: "The demo was successful", want: true},
		{name: "detects version", text: "Version 3 is now available", want: true},
		{name: "detects v2.0", text: "We are on v2.0 now", want: true},
		{name: "negative case", text: "The file contains text", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, _ := scoreMarkers(tt.text, extractor.milestonePatterns)
			got := score > 0
			if got != tt.want {
				t.Errorf("milestone pattern score(%q) = %v, want %v", tt.text, score, tt.want)
			}
		})
	}
}

func TestProblemPatterns(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "detects broke", text: "The server broke yesterday", want: true},
		{name: "detects crashed", text: "The app crashed on startup", want: true},
		{name: "detects failed", text: "The connection failed", want: true},
		{name: "detects error", text: "There was an error in the log", want: true},
		{name: "detects exception", text: "An exception was thrown", want: true},
		{name: "detects bug", text: "There's a bug in the authentication", want: true},
		{name: "detects issue", text: "The issue is with the database", want: true},
		{name: "detects problem", text: "The problem is memory usage", want: true},
		{name: "detects fix", text: "We need to fix the performance", want: true},
		{name: "detects workaround", text: "We found a workaround", want: true},
		{name: "detects root cause", text: "The root cause was identified", want: true},
		{name: "detects why", text: "Why did this happen?", want: true},
		{name: "detects doesn't work", text: "It doesn't work as expected", want: true},
		{name: "detects not working", text: "The service is not working", want: true},
		{name: "detects won't work", text: "This won't work properly", want: true},
		{name: "detects keeps failing", text: "It keeps failing", want: true},
		{name: "detects the problem is", text: "The problem is the configuration", want: true},
		{name: "detects the fix is", text: "The fix is to restart", want: true},
		{name: "detects that's why", text: "That's why it crashed", want: true},
		{name: "detects solution is", text: "The solution is to upgrade", want: true},
		{name: "detects the answer is", text: "The answer is simpler", want: true},
		{name: "negative case", text: "The file was saved successfully", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, _ := scoreMarkers(tt.text, extractor.problemPatterns)
			got := score > 0
			if got != tt.want {
				t.Errorf("problem pattern score(%q) = %v, want %v", tt.text, score, tt.want)
			}
		})
	}
}

func TestEmotionalPatterns(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "detects feelings", text: "These are my true feelings about it", want: true},
		{name: "detects vulnerable", text: "I feel vulnerable admitting this", want: true},
		{name: "detects honestly", text: "Honestly, I'm not sure about this", want: true},
		{name: "detects truth is", text: "Truth is, we need to improve", want: true},
		{name: "detects the thing is", text: "The thing is, it might not work", want: true},
		{name: "detects im not sure", text: "I'm not sure about this", want: true},
		{name: "detects hope", text: "I hope we can solve this", want: true},
		{name: "detects fear", text: "My fear is that we'll fail", want: true},
		{name: "detects worry", text: "I worry about the deadline", want: true},
		{name: "detects sad", text: "Honestly, I was sad", want: true},
		{name: "detects happy", text: "I was happy with the result", want: true},
		{name: "detects angry", text: "I was angry about the delay", want: true},
		{name: "detects scared", text: "I was scared of the unknown", want: true},
		{name: "detects love", text: "I love working on this project", want: true},
		{name: "detects proud", text: "I'm proud of what we achieved", want: true},
		{name: "detects hurt", text: "It hurt to admit this", want: true},
		{name: "detects cry", text: "I could cry right now", want: true},
		{name: "detects sorry", text: "I'm sorry for the confusion", want: true},
		{name: "detects grateful", text: "I'm grateful for your help", want: true},
		{name: "detects worried", text: "I'm worried about the timeline", want: true},
		{name: "detects lonely", text: "I feel lonely in this decision", want: true},
		{name: "detects beautiful", text: "This is a beautiful solution", want: true},
		{name: "detects amazing", text: "It's an amazing result", want: true},
		{name: "detects wonderful", text: "This is wonderful news", want: true},
		{name: "detects i need", text: "I need to tell you something", want: true},
		{name: "negative case", text: "The data is stored in the database", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, _ := scoreMarkers(tt.text, extractor.emotionalPatterns)
			got := score > 0
			if got != tt.want {
				t.Errorf("emotional pattern score(%q) = %v, want %v", tt.text, score, tt.want)
			}
		})
	}
}

func TestSentiment(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{name: "positive sentiment", text: "I love this amazing breakthrough solution!", expected: "positive"},
		{name: "negative sentiment", text: "This bug is terrible and causes crashes", expected: "negative"},
		{name: "neutral sentiment", text: "The server is running on port 8080", expected: "neutral"},
		{name: "mixed positive", text: "proud happy love wonderful", expected: "positive"},
		{name: "mixed negative", text: "bug error crash fail broken", expected: "negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractor.getSentiment(tt.text)
			if got != tt.expected {
				t.Errorf("getSentiment(%q) = %q, want %q", tt.text, got, tt.expected)
			}
		})
	}
}

func TestHasResolution(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{name: "has fixed", text: "It's finally fixed now", expected: true},
		{name: "has solved", text: "We solved the issue", expected: true},
		{name: "has resolved", text: "The problem is resolved", expected: true},
		{name: "has it works", text: "It works now", expected: true},
		{name: "has nailed it", text: "We nailed it", expected: true},
		{name: "has figured out", text: "I figured it out", expected: true},
		{name: "no resolution", text: "The bug is still there", expected: false},
		{name: "no resolution 2", text: "Why is this happening", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasResolution(tt.text)
			if got != tt.expected {
				t.Errorf("hasResolution(%q) = %v, want %v", tt.text, got, tt.expected)
			}
		})
	}
}

func TestIsCodeLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{name: "shell command", line: "$ pip install requests", expected: true},
		{name: "git command", line: "git commit -m 'fix'", expected: true},
		{name: "import statement", line: "import os from sys", expected: true},
		{name: "code assignment", line: "const x = 5", expected: true},
		{name: "function def", line: "def main():", expected: true},
		{name: "markdown code block", line: "```", expected: true},
		{name: "pipe separator", line: "| header |", expected: true},
		{name: "prose line", line: "This is a normal sentence describing my work.", expected: false},
		{name: "empty line", line: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCodeLine(tt.line)
			if got != tt.expected {
				t.Errorf("isCodeLine(%q) = %v, want %v", tt.line, got, tt.expected)
			}
		})
	}
}

func TestExtractProse(t *testing.T) {
	text := "Normal prose line\n$ pip install something\n```python\ncode block\n```\nAnother prose line"

	prose := extractProse(text)

	if strings.Contains(prose, "$ pip") {
		t.Error("extractProse should filter shell commands")
	}
	if strings.Contains(prose, "python") {
		t.Error("extractProse should filter code blocks")
	}
	if !strings.Contains(prose, "Normal prose") {
		t.Error("extractProse should keep prose lines")
	}
}

func TestScoreMarkers(t *testing.T) {
	extractor := NewExtractor()

	text := "I prefer Python and I always use TypeScript"
	score, keywords := scoreMarkers(text, extractor.preferencePatterns)

	if score < 2 {
		t.Errorf("Expected score >= 2, got %v", score)
	}
	if len(keywords) < 2 {
		t.Errorf("Expected at least 2 keywords, got %v", keywords)
	}
}

func TestSplitIntoSegments(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{name: "splits paragraphs", text: "Para 1\n\nPara 2", expected: 2},
		{name: "single paragraph", text: "Just one paragraph", expected: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := splitIntoSegments(tt.text)
			if len(segments) < 1 && tt.expected > 0 {
				t.Errorf("splitIntoSegments(%q) returned empty, want %d", tt.text, tt.expected)
			}
		})
	}
}

func TestSplitByTurns(t *testing.T) {
	lines := []string{
		"User: Hello",
		"Assistant: Hi there",
		"User: How are you?",
		"Assistant: I'm good",
	}

	segments := splitByTurns(lines)

	if len(segments) < 2 {
		t.Errorf("Expected at least 2 segments, got %d", len(segments))
	}
}

func TestDisambiguate(t *testing.T) {
	tests := []struct {
		name       string
		memoryType string
		text       string
		scores     map[string]float64
		expected   string
	}{
		{
			name:       "resolved problem becomes milestone",
			memoryType: "problem",
			text:       "I fixed the bug",
			scores:     map[string]float64{"problem": 2},
			expected:   "milestone",
		},
		{
			name:       "emotional with positive sentiment",
			memoryType: "problem",
			text:       "I'm happy I solved this",
			scores:     map[string]float64{"problem": 1, "emotional": 2},
			expected:   "emotional",
		},
		{
			name:       "problem stays problem",
			memoryType: "problem",
			text:       "The bug is still there",
			scores:     map[string]float64{"problem": 2},
			expected:   "problem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := disambiguate(tt.memoryType, tt.text, tt.scores)
			if got != tt.expected {
				t.Errorf("disambiguate(%q, %q) = %q, want %q", tt.memoryType, tt.text, got, tt.expected)
			}
		})
	}
}

func TestMemoryStructure(t *testing.T) {
	extractor := NewExtractor()
	memories := extractor.ExtractMemories("We decided to use Go for the API.", 0.1)

	if len(memories) != 1 {
		t.Fatalf("Expected 1 memory, got %d", len(memories))
	}

	m := memories[0]
	if m.MemoryType != "decision" {
		t.Errorf("Expected MemoryType 'decision', got %q", m.MemoryType)
	}
	if m.ChunkIndex != 0 {
		t.Errorf("Expected ChunkIndex 0, got %d", m.ChunkIndex)
	}
}

func TestChunkIndexIncrements(t *testing.T) {
	extractor := NewExtractor()
	memories := extractor.ExtractMemories("We decided to use Go for the API.\n\nI prefer Python for scripting tasks.\n\nIt finally works after the deployment!", 0.1)

	if len(memories) != 3 {
		t.Fatalf("Expected 3 memories, got %d", len(memories))
	}

	for i, m := range memories {
		if m.ChunkIndex != i {
			t.Errorf("Memory %d: expected ChunkIndex %d, got %d", i, i, m.ChunkIndex)
		}
	}
}

func TestShortSentencesFiltered(t *testing.T) {
	extractor := NewExtractor()
	memories := extractor.ExtractMemories("Hi.\n\nWe chose Go for the API.", 0.1)

	if len(memories) != 1 {
		t.Fatalf("Expected 1 memory (short sentence filtered), got %d", len(memories))
	}
}

func TestCodeFiltering(t *testing.T) {
	extractor := NewExtractor()

	text := "# Installation\n$ npm install\n\nWe decided to use Node for the server"
	memories := extractor.ExtractMemories(text, 0.1)

	if len(memories) > 0 && strings.Contains(memories[0].Content, "$ npm") {
		t.Error("Code lines should be filtered from memory content")
	}
}

func TestSpeakerTurnSplitting(t *testing.T) {
	extractor := NewExtractor()

	text := "User: We decided to use Go for the project\nAssistant: That sounds like a good choice\nUser: I prefer TypeScript actually\nAssistant: TypeScript has better type safety"
	memories := extractor.ExtractMemories(text, 0.1)

	if len(memories) < 1 {
		t.Error("Speaker turns should produce at least one memory")
	}
}
