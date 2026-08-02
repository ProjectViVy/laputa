package entity

import (
	"strings"
	"testing"
)

func TestExtractCandidates(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name     string
		text     string
		expected map[string]int
	}{
		{
			name: "finds capitalized words appearing 3+ times",
			text: "Alice Bob Alice Bob Alice Charlie",
			expected: map[string]int{
				"Alice": 3,
			},
		},
		{
			name:     "ignores stop words",
			text:     "The And But The And But The",
			expected: map[string]int{},
		},
		{
			name: "mixed case words",
			text: "David said hello David said goodbye David was there",
			expected: map[string]int{
				"David": 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.ExtractCandidates(tt.text)
			for name, expectedCount := range tt.expected {
				if count, ok := result[name]; !ok || count != expectedCount {
					t.Errorf("expected %s to appear %d times, got %v", name, expectedCount, result)
					break
				}
			}
			for name := range result {
				if _, ok := tt.expected[name]; !ok && tt.expected != nil {
					delete(tt.expected, name)
				}
			}
		})
	}
}

func TestStopWords(t *testing.T) {
	d := NewDetector()

	text := "The quick brown fox jumps over the lazy dog The quick brown fox The"
	candidates := d.ExtractCandidates(text)

	for word := range candidates {
		if stopWords[strings.ToLower(word)] {
			t.Errorf("stop word %q should not be in candidates", word)
		}
	}
}

func TestScoreEntity(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name    string
		text    string
		entity  string
		wantPS  int
		wantPrS int
	}{
		{
			name:    "person verb pattern",
			text:    "Alice said hello to Bob",
			entity:  "Alice",
			wantPS:  2,
			wantPrS: 0,
		},
		{
			name:    "project verb pattern",
			text:    "We built Palantir pipeline last week",
			entity:  "Palantir",
			wantPS:  0,
			wantPrS: 2,
		},
		{
			name:    "direct address",
			text:    "Hey Alice thanks for helping Bob",
			entity:  "Alice",
			wantPS:  0,
			wantPrS: 0,
		},
		{
			name:    "versioned project",
			text:    "palantir-core v2 is released",
			entity:  "palantir",
			wantPS:  0,
			wantPrS: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps, prs := d.ScoreEntity(tt.entity, tt.text)
			if ps != tt.wantPS || prs != tt.wantPrS {
				t.Errorf("ScoreEntity(%q, %q) = (%d, %d), want (%d, %d)",
					tt.entity, tt.text, ps, prs, tt.wantPS, tt.wantPrS)
			}
		})
	}
}

func TestClassifyEntity(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name           string
		frequency      int
		personScore    int
		projectScore   int
		personSignals  []string
		projectSignals []string
		expectedType   string
	}{
		{
			name:           "person with high person score and two signal types",
			frequency:      5,
			personScore:    10,
			projectScore:   2,
			personSignals:  []string{"dialogue marker (2x)", "'Alice ...' action (3x)"},
			projectSignals: []string{},
			expectedType:   "person",
		},
		{
			name:           "person with one signal type only",
			frequency:      5,
			personScore:    10,
			projectScore:   2,
			personSignals:  []string{"dialogue marker (5x)"},
			projectSignals: []string{},
			expectedType:   "uncertain",
		},
		{
			name:           "project with high project score",
			frequency:      3,
			personScore:    1,
			projectScore:   8,
			personSignals:  []string{},
			projectSignals: []string{"project verb (4x)"},
			expectedType:   "project",
		},
		{
			name:           "uncertain with mixed signals",
			frequency:      5,
			personScore:    5,
			projectScore:   5,
			personSignals:  []string{"action (2x)"},
			projectSignals: []string{"verb (2x)"},
			expectedType:   "uncertain",
		},
		{
			name:           "no signals",
			frequency:      15,
			personScore:    0,
			projectScore:   0,
			personSignals:  []string{},
			projectSignals: []string{},
			expectedType:   "uncertain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity := d.ClassifyEntity(tt.name, tt.frequency, tt.personScore, tt.projectScore, tt.personSignals, tt.projectSignals)
			if entity.Type != tt.expectedType {
				t.Errorf("ClassifyEntity(%q, %d, %d, %d) type = %q, want %q",
					tt.name, tt.frequency, tt.personScore, tt.projectScore, entity.Type, tt.expectedType)
			}
		})
	}
}

func TestDetect(t *testing.T) {
	d := NewDetector()

	text := `
	Alice worked on the Palantir project. 
	Alice wrote code for Palantir. 
	Alice pushed the Palantir pipeline.
	We built Palantir architecture last week.
	`

	entities := d.Detect(text)

	if len(entities) == 0 {
		t.Error("expected to detect entities, got none")
	}

	found := make(map[string]bool)
	for _, e := range entities {
		found[e.Name] = true
	}

	if !found["Alice"] {
		t.Error("expected to detect Alice as person")
	}
	if !found["Palantir"] {
		t.Error("expected to detect Palantir as project")
	}
}

func BenchmarkExtractCandidates(b *testing.B) {
	d := NewDetector()
	text := "Alice Bob Charlie Alice Bob Charlie Alice Bob David Alice Bob Charlie David Alice Bob"
	for i := 0; i < b.N; i++ {
		d.ExtractCandidates(text)
	}
}
