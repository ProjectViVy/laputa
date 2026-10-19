package actmem

import (
	"bytes"
	"io"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const recapSchema = "actmem/v3"

type recapMetadata struct {
	Schema string    `yaml:"schema"`
	Head   RecapHead `yaml:",inline"`
}

// Metadata precedes an exact, unrestricted Markdown body. The decoder never
// normalizes whitespace, CRLF, embedded headings, fences or delimiters in text.
func encodeRecap(head RecapHead) ([]byte, error) {
	header, err := yaml.Marshal(recapMetadata{Schema: recapSchema, Head: head})
	if err != nil {
		return nil, recapUnavailable("cannot encode Recap metadata", err)
	}
	return []byte("---\n" + string(header) + "---\n" + head.Text), nil
}
func decodeRecap(raw []byte) (RecapHead, error) {
	fail := func(message string, cause error) (RecapHead, error) {
		return RecapHead{}, recapUnavailable(message, cause)
	}
	if !utf8.Valid(raw) || !bytes.HasPrefix(raw, []byte("---\n")) {
		return fail("invalid UTF-8 Recap front matter", nil)
	}
	end := bytes.Index(raw[4:], []byte("\n---\n"))
	if end < 0 {
		return fail("missing Recap front matter delimiter", nil)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw[4 : 4+end]))
	decoder.KnownFields(true)
	var metadata recapMetadata
	if err := decoder.Decode(&metadata); err != nil {
		return fail("invalid Recap metadata", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fail("trailing Recap metadata", err)
	}
	head := metadata.Head
	head.Text = string(raw[4+end+5:])
	if metadata.Schema != recapSchema || !validRecapBinding(head.Binding) || head.Revision == 0 || head.OperationID == "" || !validRecapDigest(head.OperationDigest) || !validRecapDigest(head.Digest) || head.UpdatedAt.IsZero() {
		return fail("incomplete Recap metadata", nil)
	}
	if (head.Revision == 1 && head.PreviousDigest != "") || (head.Revision > 1 && !validRecapDigest(head.PreviousDigest)) {
		return fail("invalid previous Recap digest", nil)
	}
	if recapHeadDigest(head) != head.Digest {
		return fail("Recap content digest mismatch", nil)
	}
	switch head.Cause {
	case "summary":
		if head.Sources.Through <= head.Sources.After || strings.TrimSpace(head.Sources.Digest) == "" || head.CoveredThrough != head.Sources.Through || head.RestoredFrom != 0 {
			return fail("invalid summary metadata", nil)
		}
	case "reconcile":
		if head.Sources.After != head.CoveredThrough || head.Sources.Through != head.CoveredThrough || strings.TrimSpace(head.Sources.Digest) == "" || head.RestoredFrom != 0 {
			return fail("invalid reconcile metadata", nil)
		}
	case "edit", "clear", "restore":
		if head.Sources != (SourceRange{}) {
			return fail("unexpected Recap source range", nil)
		}
		if head.Cause == "restore" {
			if head.RestoredFrom == 0 || head.RestoredFrom >= head.Revision {
				return fail("invalid restore revision", nil)
			}
		} else if head.RestoredFrom != 0 {
			return fail("unexpected restore revision", nil)
		}
		if head.Cause == "clear" && head.Text != "" {
			return fail("clear must contain empty text", nil)
		}
	default:
		return fail("invalid Recap cause", nil)
	}
	return head, nil
}
func declaresV3Schema(raw string) bool {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return false
	}
	for _, line := range strings.Split(normalized[4:], "\n") {
		if line == "---" {
			return false
		}
		if strings.HasPrefix(line, "schema:") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "schema:")), "\"'") == recapSchema
		}
	}
	return false
}
