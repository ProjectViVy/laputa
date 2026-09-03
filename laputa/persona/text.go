package persona

import (
	"fmt"
	"strings"

	"github.com/rivo/uniseg"
	"golang.org/x/text/unicode/norm"
)

// NormalizeMarkdown applies the DIVA-verified canonical form:
// CRLF/CR -> LF, NFC unicode normalization, then trim outer whitespace.
func NormalizeMarkdown(content string) string {
	replaced := strings.ReplaceAll(content, "\r\n", "\n")
	replaced = strings.ReplaceAll(replaced, "\r", "\n")
	return strings.TrimSpace(norm.NFC.String(replaced))
}

// VisibleLen counts non-whitespace grapheme clusters after normalization.
func VisibleLen(content string) int {
	count := 0
	gr := uniseg.NewGraphemes(NormalizeMarkdown(content))
	for gr.Next() {
		if !allWhitespace(gr.Str()) {
			count++
		}
	}
	return count
}

func allWhitespace(cluster string) bool {
	for _, r := range cluster {
		if !isUnicodeSpace(r) {
			return false
		}
	}
	return true
}

func isUnicodeSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r',
		0x85, 0xA0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000:
		return true
	}
	return r >= 0x2000 && r <= 0x200A
}

// ExtractMarkdownSection returns the body beneath an exact "## heading" marker.
func ExtractMarkdownSection(content, heading string) (string, bool) {
	marker := "## " + heading
	lines := strings.Split(NormalizeMarkdown(content), "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimRight(line, " \t") == marker {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return "", false
	}
	var body []string
	for _, line := range lines[start:] {
		if strings.HasPrefix(line, "## ") {
			break
		}
		body = append(body, line)
	}
	return strings.TrimSpace(strings.Join(body, "\n")), true
}

// ReplaceMarkdownSection replaces (or appends) the body under "## heading".
func ReplaceMarkdownSection(content, heading, body string) string {
	normalized := NormalizeMarkdown(content)
	marker := "## " + heading
	lines := strings.Split(normalized, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimRight(line, " \t") == marker {
			start = i
			break
		}
	}
	if start < 0 {
		if normalized == "" {
			return NormalizeMarkdown(marker + "\n" + strings.TrimSpace(body))
		}
		return NormalizeMarkdown(normalized + "\n\n" + marker + "\n" + strings.TrimSpace(body))
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	trimmed := strings.TrimSpace(body)
	out := make([]string, 0, len(lines)+2)
	out = append(out, lines[:start+1]...)
	if trimmed != "" {
		out = append(out, strings.Split(trimmed, "\n")...)
	}
	out = append(out, lines[end:]...)
	return NormalizeMarkdown(strings.Join(out, "\n"))
}

// UnifiedDiff renders a line-granular unified diff between two contents.
// It finds the common prefix and suffix, then emits one hunk covering the
// changed middle. Header lines carry the authority file name.
func UnifiedDiff(before, after, fileName string) string {
	a := strings.Split(NormalizeMarkdown(before), "\n")
	b := strings.Split(NormalizeMarkdown(after), "\n")
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix && a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}
	aStart, aEnd := prefix, len(a)-suffix
	bStart, bEnd := prefix, len(b)-suffix
	var sb strings.Builder
	sb.WriteString("--- " + fileName + "\n")
	sb.WriteString("+++ " + fileName + "\n")
	if aStart == aEnd && bStart == bEnd {
		return ""
	}
	context := 3
	ctxStart := prefix - context
	if ctxStart < 0 {
		ctxStart = 0
	}
	leadCtx := a[ctxStart:prefix]
	aChanged := a[aStart:aEnd]
	bChanged := b[bStart:bEnd]
	tailCtx := a[aEnd:min(aEnd+context, len(a))]
	sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", ctxStart+1, len(leadCtx)+len(aChanged)+len(tailCtx), ctxStart+1, len(leadCtx)+len(bChanged)+len(tailCtx)))
	for _, line := range leadCtx {
		sb.WriteString(" " + line + "\n")
	}
	for _, line := range aChanged {
		sb.WriteString("-" + line + "\n")
	}
	for _, line := range bChanged {
		sb.WriteString("+" + line + "\n")
	}
	for _, line := range tailCtx {
		sb.WriteString(" " + line + "\n")
	}
	return sb.String()
}
