package palace

import (
	"fmt"
	"regexp"
)

const MaxNameLength = 128

var safeNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_ .'-]{0,126}[a-zA-Z0-9]?$`)

func SanitizeName(value, fieldName string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s must be non-empty", fieldName)
	}
	if len(value) > MaxNameLength {
		return "", fmt.Errorf("%s exceeds max length %d", fieldName, MaxNameLength)
	}
	if safeNameRe.MatchString(value) {
		return value, nil
	}
	return "", fmt.Errorf("%s contains invalid characters", fieldName)
}

func SanitizeContent(value string, maxLength int) (string, error) {
	if value == "" {
		return "", fmt.Errorf("content must be non-empty")
	}
	if maxLength > 0 && len(value) > maxLength {
		return "", fmt.Errorf("content exceeds max length %d", maxLength)
	}
	return value, nil
}
