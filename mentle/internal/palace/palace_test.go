package palace

import (
	"strings"
	"testing"
	"time"
)

func TestNewWing(t *testing.T) {
	name := "TestWing"
	wing := NewWing(name)

	if wing.Name != name {
		t.Errorf("expected Name %q, got %q", name, wing.Name)
	}
	if wing.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if time.Since(wing.CreatedAt) > time.Second {
		t.Error("CreatedAt should be recent")
	}
}

func TestNewDrawer(t *testing.T) {
	now := time.Now()
	drawer := Drawer{
		ID:         "test-id",
		Content:    "test content",
		Wing:       "test-wing",
		Room:       "test-room",
		SourceFile: "test.go",
		ChunkIndex: 0,
		AddedBy:    "tester",
		FiledAt:    now,
		Metadata:   map[string]string{"key": "value"},
	}

	if drawer.ID != "test-id" {
		t.Errorf("expected ID %q, got %q", "test-id", drawer.ID)
	}
	if drawer.Content != "test content" {
		t.Errorf("expected Content %q, got %q", "test content", drawer.Content)
	}
	if drawer.Wing != "test-wing" {
		t.Errorf("expected Wing %q, got %q", "test-wing", drawer.Wing)
	}
	if drawer.Room != "test-room" {
		t.Errorf("expected Room %q, got %q", "test-room", drawer.Room)
	}
	if drawer.SourceFile != "test.go" {
		t.Errorf("expected SourceFile %q, got %q", "test.go", drawer.SourceFile)
	}
	if drawer.ChunkIndex != 0 {
		t.Errorf("expected ChunkIndex 0, got %d", drawer.ChunkIndex)
	}
	if drawer.AddedBy != "tester" {
		t.Errorf("expected AddedBy %q, got %q", "tester", drawer.AddedBy)
	}
	if !drawer.FiledAt.Equal(now) {
		t.Errorf("expected FiledAt %v, got %v", now, drawer.FiledAt)
	}
	if drawer.Metadata["key"] != "value" {
		t.Errorf("expected Metadata[key] %q, got %q", "value", drawer.Metadata["key"])
	}
}

func TestSanitizeName_ValidInputs(t *testing.T) {
	validNames := []string{
		"a",
		"Abc",
		"abc123",
		"abc 123",
		"abc_123",
		"abc-123",
		"abc'123",
		"a b c",
		"A B C",
		"test_user",
		"test-user",
		"don't",
		"name.with.dots",
		"Name With Spaces And Numbers 123",
		strings.Repeat("a", 128),
		"a" + strings.Repeat("b", 126) + "z",
	}

	for _, name := range validNames {
		result, err := SanitizeName(name, "TestField")
		if err != nil {
			t.Errorf("SanitizeName(%q) returned error: %v", name, err)
		}
		if result != name {
			t.Errorf("SanitizeName(%q) = %q, want %q", name, result, name)
		}
	}
}

func TestSanitizeName_InvalidInputs(t *testing.T) {
	invalidNames := []struct {
		name        string
		description string
	}{
		{"", "empty string"},
		{strings.Repeat("a", 129), "too long"},
		{"../etc/passwd", "path traversal"},
		{"/absolute/path", "absolute path"},
		{"path\\with\\backslash", "backslash"},
		{"path;with;semicolon", "semicolon"},
		{"path$with$dollar", "dollar sign"},
		{"path`with`backtick", "backtick"},
		{"path|with|pipe", "pipe"},
		{"path&with&ampersand", "ampersand"},
		{"path<with>brackets", "brackets"},
		{"path\nwithnewline", "newline"},
		{"path\twith\ttab", "tab"},
	}

	for _, tc := range invalidNames {
		t.Run(tc.description, func(t *testing.T) {
			_, err := SanitizeName(tc.name, "TestField")
			if err == nil {
				t.Errorf("SanitizeName(%q) should return error for %s", tc.name, tc.description)
			}
		})
	}
}

func TestSanitizeName_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		fieldName   string
		expectError bool
	}{
		{"validName", "CustomField", false},
		{"", "EmptyField", true},
		{strings.Repeat("x", 128), "MaxLenField", false},
		{strings.Repeat("x", 129), "TooLongField", true},
	}

	for _, tc := range tests {
		t.Run(tc.name+"_"+tc.fieldName, func(t *testing.T) {
			_, err := SanitizeName(tc.name, tc.fieldName)
			if tc.expectError && err == nil {
				t.Errorf("SanitizeName(%q, %q) expected error", tc.name, tc.fieldName)
			}
			if !tc.expectError && err != nil {
				t.Errorf("SanitizeName(%q, %q) unexpected error: %v", tc.name, tc.fieldName, err)
			}
		})
	}
}

func TestSanitizeContent(t *testing.T) {
	t.Run("valid content", func(t *testing.T) {
		result, err := SanitizeContent("test content", 100)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != "test content" {
			t.Errorf("expected %q, got %q", "test content", result)
		}
	})

	t.Run("empty content", func(t *testing.T) {
		_, err := SanitizeContent("", 100)
		if err == nil {
			t.Error("expected error for empty content")
		}
	})

	t.Run("content too long", func(t *testing.T) {
		_, err := SanitizeContent("long content", 5)
		if err == nil {
			t.Error("expected error for content exceeding max length")
		}
	})

	t.Run("maxLength zero means no limit", func(t *testing.T) {
		longContent := strings.Repeat("a", 10000)
		result, err := SanitizeContent(longContent, 0)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != longContent {
			t.Error("content should not be limited when maxLength is 0")
		}
	})
}
