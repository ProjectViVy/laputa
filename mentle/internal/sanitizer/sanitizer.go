// Package sanitizer provides input validation and sanitization for MCP tools.
// It blocks path traversal, null bytes, and enforces size limits.
package sanitizer

import (
	"fmt"
	"strings"
)

const (
	MaxNameLength    = 128
	MaxContentLength = 100000
)

// SanitizeName validates and sanitizes a name (wing, room, agent).
// It blocks path traversal, null bytes, and enforces safe characters.
func SanitizeName(name, fieldName string) (string, error) {
	if name == "" {
		return "", nil // Empty is allowed for optional fields
	}
	if len(name) > MaxNameLength {
		return "", fmt.Errorf("%s too long (max %d chars)", fieldName, MaxNameLength)
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", fmt.Errorf("%s contains invalid characters", fieldName)
	}
	if strings.Contains(name, "\x00") {
		return "", fmt.Errorf("%s contains null bytes", fieldName)
	}
	// Allow only safe characters: alphanumeric, hyphens, underscores, dots, spaces
	for _, r := range name {
		if !isSafeChar(r) {
			return "", fmt.Errorf("%s contains invalid character: %c", fieldName, r)
		}
	}
	return strings.TrimSpace(name), nil
}

// SanitizeContent validates and truncates content.
func SanitizeContent(content, fieldName string) (string, error) {
	if content == "" {
		return "", nil
	}
	if strings.Contains(content, "\x00") {
		return "", fmt.Errorf("%s contains null bytes", fieldName)
	}
	if len(content) > MaxContentLength {
		return content[:MaxContentLength], nil
	}
	return content, nil
}

func isSafeChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_' || r == '.' || r == ' '
}
