package miner

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	// Add a small delay to ensure different timestamps
	time.Sleep(1 * time.Millisecond)
	id2 := generateID()
	if id1 == id2 {
		t.Errorf("generateID should produce unique IDs, got %s twice", id1)
	}
	if len(id1) < 5 || id1[:2] != "d_" {
		t.Errorf("generateID should start with 'd_', got %s", id1)
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"claude format", `{"role": "user", "content": "hello"}`, "claude"},
		{"chatgpt format", `{"message": {"role": "user", "content": "hello"}}`, "chatgpt"},
		{"unknown format", `{"data": "test"}`, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectFormat([]byte(tt.content))
			if result != tt.expected {
				t.Errorf("DetectFormat(%q) = %q, want %q", tt.content, result, tt.expected)
			}
		})
	}
}

func TestNormalizeClaudeExport(t *testing.T) {
	valid := []byte(`{"messages": [{"role": "user", "content": "hello"}], "source": "test"}`)
	result, err := NormalizeClaudeExport(valid)
	if err != nil {
		t.Errorf("NormalizeClaudeExport with valid JSON returned error: %v", err)
	}
	if len(result.Messages) != 1 || result.Messages[0].Role != "user" {
		t.Errorf("NormalizeClaudeExport parsed incorrectly: %+v", result)
	}
}

func TestNormalizeClaudeExportInvalid(t *testing.T) {
	invalid := []byte(`{invalid json}`)
	_, err := NormalizeClaudeExport(invalid)
	if err == nil {
		t.Errorf("NormalizeClaudeExport with invalid JSON should return error")
	}
}

func TestChunkContent(t *testing.T) {
	// Test short content
	content := "short"
	chunks := chunkContent(content, 100, 20)
	if len(chunks) != 1 || chunks[0] != "short" {
		t.Errorf("chunkContent(\"short\", 100, 20) = %v, want [%q]", chunks, "short")
	}

	// Test content that fits exactly
	content = strings.Repeat("x", 100)
	chunks = chunkContent(content, 100, 20)
	if len(chunks) != 1 || chunks[0] != content {
		t.Errorf("chunkContent(exact fit, 100, 20) = %v, want [%q]", chunks, content)
	}

	// Test content requiring chunking without boundaries
	content = strings.Repeat("x", 250)       // 250 chars
	chunks = chunkContent(content, 800, 100) // Use same values as actual call in Miner.MineProject
	if len(chunks) != 1 {                    // 250 < 800, so should be 1 chunk
		t.Errorf("chunkContent(250x, 800, 100) = %v, want 1 chunk", chunks)
	}
	if len(chunks[0]) != 250 {
		t.Errorf("chunkContent first chunk incorrect length: got %d, want 250", len(chunks[0]))
	}

	// Test with smaller chunk size to force splitting
	content = strings.Repeat("x", 900) // 900 chars
	chunks = chunkContent(content, 800, 100)
	if len(chunks) != 2 {
		t.Errorf("chunkContent(900x, 800, 100) = %v, want 2 chunks", chunks)
	}
	// First chunk should be ~800 chars (trying to break at boundary)
	// Second chunk should be remainder
	if len(chunks[0]) < 700 || len(chunks[0]) > 900 {
		t.Errorf("chunkContent first chunk length unreasonable: got %d", len(chunks[0]))
	}
	if len(chunks[1]) < 0 || len(chunks[1]) > 200 {
		t.Errorf("chunkContent second chunk length unreasonable: got %d", len(chunks[1]))
	}

	// Test empty content
	content = ""
	chunks = chunkContent(content, 800, 100)
	if len(chunks) != 0 {
		t.Errorf("chunkContent empty string = %v, want []", chunks)
	}

	// Test content with only spaces
	content = "   \n\t\n   "
	chunks = chunkContent(content, 800, 100)
	if len(chunks) != 0 {
		t.Errorf("chunkContent whitespace only = %v, want []", chunks)
	}
}

func TestDetectRoomFromContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "API endpoints",
			content:  "package api\n\n// Handler handles API requests\nfunc Handler() {\n\t// server endpoint",
			expected: "backend",
		},
		{
			name:     "React component",
			content:  "import React from 'react';\n\nexport const Button = () => {\n  return <div className='component'>",
			expected: "frontend",
		},
		{
			name:     "Test file",
			content:  "package main\n\nimport \"testing\"\n\nfunc TestSomething(t *testing.T) {\n\t// test case",
			expected: "testing",
		},
		{
			name:     "Documentation",
			content:  "# README\n\nThis project documentation describes how to use the wiki and docs.",
			expected: "documentation",
		},
		{
			name:     "Docker/deploy",
			content:  "FROM golang:1.21\n# Docker deployment\n# k8s infrastructure",
			expected: "infrastructure",
		},
		{
			name:     "Short/no match",
			content:  "x",
			expected: "general",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectRoomFromContent(tt.content)
			if result != tt.expected {
				t.Errorf("DetectRoomFromContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestContentHashIndex(t *testing.T) {
	content := "hello world test content"
	hash1 := GenerateContentHash(content)
	hash2 := GenerateContentHash("different content")
	hash3 := GenerateContentHash(content)

	if hash1 == hash2 {
		t.Error("different content should produce different hashes")
	}
	if hash1 != hash3 {
		t.Error("same content should produce same hash")
	}
	if len(hash1) != 64 {
		t.Errorf("SHA256 hex hash should be 64 chars, got %d", len(hash1))
	}
}

func TestIsContentAlreadyMined(t *testing.T) {
	ResetContentHashIndex()

	path := "/test/file.go"
	hash := GenerateContentHash("test content")

	RegisterContentHash(path, hash)

	if !IsContentAlreadyMined(path, hash) {
		t.Error("expected content to be already mined")
	}

	differentHash := GenerateContentHash("different content")
	if IsContentAlreadyMined(path, differentHash) {
		t.Error("expected different content to not be already mined")
	}

	if IsContentAlreadyMined("/test/other.go", hash) {
		t.Error("expected unknown file to not be already mined")
	}
}

func TestDetectRoomFromPath_Filename(t *testing.T) {
	result := detectRoomFromPath("/project/src/ButtonComponent.tsx")
	if result != "frontend" {
		t.Errorf("expected 'frontend' from filename pattern, got %q", result)
	}
}

func TestDetectRoomFromPath_APIServer(t *testing.T) {
	result := detectRoomFromPath("/project/api/handler.go")
	if result != "backend" {
		t.Errorf("expected 'backend', got %q", result)
	}
}

func TestDetectRoomFromPath_DocsDir(t *testing.T) {
	result := detectRoomFromPath("/project/docs/readme.md")
	if result != "documentation" {
		t.Errorf("expected 'documentation', got %q", result)
	}
}

func TestDetectRoomFromPath_TestDir(t *testing.T) {
	result := detectRoomFromPath("/project/tests/unit_test.go")
	if result != "testing" {
		t.Errorf("expected 'testing', got %q", result)
	}
}

func TestDetectRoomFromPath_General(t *testing.T) {
	result := detectRoomFromPath("/project/misc/random.xyz")
	if result != "general" {
		t.Errorf("expected 'general' fallback, got %q", result)
	}
}

type mockRoomDetector struct {
	result string
}

func (m *mockRoomDetector) DetectRoom(filePath string, content string, projectPath string) string {
	return m.result
}

func TestMinerSetRoomDetector(t *testing.T) {
	ResetContentHashIndex()
	m := NewMiner(nil)
	detector := &mockRoomDetector{result: "custom-room"}
	m.SetRoomDetector(detector)

	if m.roomDetector == nil {
		t.Error("expected roomDetector to be set")
	}

	result := m.roomDetector.DetectRoom("/some/path", "content", "/some")
	if result != "custom-room" {
		t.Errorf("expected 'custom-room', got %q", result)
	}
}

func TestSaveAndLoadContentHashIndex(t *testing.T) {
	tmpDir := t.TempDir()

	ResetContentHashIndex()

	RegisterContentHash("/test/file1.go", "hash1")
	RegisterContentHash("/test/file2.go", "hash2")

	err := SaveContentHashIndex(tmpDir)
	if err != nil {
		t.Fatalf("SaveContentHashIndex() error = %v", err)
	}

	ResetContentHashIndex()

	err = LoadContentHashIndex(tmpDir)
	if err != nil {
		t.Fatalf("LoadContentHashIndex() error = %v", err)
	}

	if !IsContentAlreadyMined("/test/file1.go", "hash1") {
		t.Error("expected file1 to be in loaded index")
	}

	if !IsContentAlreadyMined("/test/file2.go", "hash2") {
		t.Error("expected file2 to be in loaded index")
	}

	if IsContentAlreadyMined("/test/other.go", "hash") {
		t.Error("expected unknown file to not be in loaded index")
	}
}

func TestLoadContentHashIndex_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	ResetContentHashIndex()

	err := LoadContentHashIndex(tmpDir)
	if err != nil {
		t.Errorf("expected no error for missing file, got %v", err)
	}
}

func TestDetectRoomCombined(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		content  string
		expected string
	}{
		{
			name:     "path match takes priority",
			path:     "/project/frontend/button.tsx",
			content:  "this is all about to API server handlers",
			expected: "frontend",
		},
		{
			name:     "content match when path is generic",
			path:     "/project/src/utils.go",
			content:  "package api\n\n// Handler handles API requests for server",
			expected: "backend",
		},
		{
			name:     "general fallback",
			path:     "/project/misc/notes.txt",
			content:  "random notes",
			expected: "general",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectRoomCombined(tt.path, tt.content)
			if result != tt.expected {
				t.Errorf("DetectRoomCombined() = %q, want %q", result, tt.expected)
			}
		})
	}
}
