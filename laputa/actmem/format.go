package actmem

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dashimaki/laputa/evolution"
)

// The v2 storage model: laputa/actmem serializes the head in the contract
// grammar via laputa/evolution. There is one parser, not two.
//
// Legacy heads (the pre-v2 front matter without schema/entries) remain
// owner-readable but are unclassified: every scoped projection excludes them
// until the owner re-saves the document through the v2 write kernel.

// head is the in-memory form of the on-disk ACTMEM.MD.
type head struct {
	doc      evolution.ActmemDocument // classified v2 entries
	legacy   *ActmemDocument          // unclassified owner projection
	missing  bool                     // no file on disk
	revision uint64
	updated  string
	markdown string
}

// loadHead parses the on-disk bytes. A document declaring the v2 schema is
// classified and its grammar errors propagate verbatim; anything else tries
// the legacy parser and is marked unclassified rather than silently
// upgraded.
func loadHead(raw string) (head, error) {
	if !utf8.ValidString(raw) {
		return head{}, newError("actmem_malformed", "ACTMEM is not valid UTF-8", nil)
	}
	if declaresV2Schema(raw) {
		doc, err := evolution.ParseActmemDocument(raw)
		if err != nil {
			return head{}, newError("actmem_format_error", err.Error(), nil)
		}
		return head{doc: doc, revision: doc.Revision, updated: doc.Updated, markdown: doc.Render()}, nil
	}
	legacy, err := parse(raw)
	if err != nil {
		return head{}, err
	}
	return head{legacy: &legacy, revision: legacy.Revision, updated: legacy.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"), markdown: legacy.Markdown}, nil
}

// declaresV2Schema reports whether the front matter names the v2 schema —
// the probe only inspects the schema line, never content.
func declaresV2Schema(raw string) bool {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	for i := 1; i < len(lines) && i < 64; i++ {
		line := lines[i]
		if line == "---" {
			return false
		}
		if strings.HasPrefix(line, "schema:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "schema:")) == evolution.ActmemSchemaV2 ||
				strings.Contains(strings.TrimSpace(strings.TrimPrefix(line, "schema:")), "actmem/")
		}
	}
	return false
}

func emptyHead() head {
	doc := evolution.ActmemDocument{
		Schema:  evolution.ActmemSchemaV2,
		Updated: "1970-01-01T00:00:00Z",
		Sections: map[evolution.EntrySection][]evolution.ActmemEntry{
			evolution.SectionPulse: {},
			evolution.SectionRecap: {},
			evolution.SectionWork:  {},
		},
	}
	return head{doc: doc, missing: true, markdown: doc.Render()}
}

func (h head) classified() bool { return h.legacy == nil }

// entries flattens the v2 document into section-ordered contract entries.
func (h head) entries() []evolution.Entry {
	var out []evolution.Entry
	for _, section := range []evolution.EntrySection{evolution.SectionPulse, evolution.SectionRecap, evolution.SectionWork} {
		for _, stored := range h.doc.Sections[section] {
			out = append(out, entryFromStored(stored))
		}
	}
	return out
}

func entryFromStored(stored evolution.ActmemEntry) evolution.Entry {
	return evolution.Entry{
		ID:         stored.Meta.ID,
		Section:    stored.Meta.Section,
		Field:      stored.Meta.Field,
		Scope:      stored.Meta.Scope,
		SessionID:  stored.Meta.SessionID,
		EventID:    stored.Meta.EventID,
		OccurredAt: stored.Meta.OccurredAt,
		Body:       stored.Body,
		Sources:    stored.Meta.Sources,
	}
}

func storedFromEntry(entry evolution.Entry) evolution.ActmemEntry {
	return evolution.ActmemEntry{
		Meta: evolution.ActmemEntryMeta{
			ID:         entry.ID,
			Section:    entry.Section,
			Field:      entry.Field,
			Scope:      entry.Scope,
			SessionID:  entry.SessionID,
			EventID:    entry.EventID,
			OccurredAt: entry.OccurredAt,
			Sources:    entry.Sources,
		},
		Body: entry.Body,
	}
}

// projection builds the owner-facing section strings. Work keeps its
// familiar field headings so the view stays recognizable.
func (h head) projection() ActmemDocument {
	if h.legacy != nil {
		out := *h.legacy
		out.Unclassified = true
		return out
	}
	var pulse, recap, work strings.Builder
	for _, stored := range h.doc.Sections[evolution.SectionPulse] {
		writeProjectedBody(&pulse, stored.Body)
	}
	for _, stored := range h.doc.Sections[evolution.SectionRecap] {
		writeProjectedBody(&recap, stored.Body)
	}
	first := true
	for _, field := range []evolution.WorkField{evolution.FieldGoal, evolution.FieldOpen, evolution.FieldNext, evolution.FieldConstraints, evolution.FieldPointers} {
		if !first {
			work.WriteString("\n\n")
		}
		first = false
		work.WriteString("### " + workFieldTitle(field))
		for _, stored := range h.doc.Sections[evolution.SectionWork] {
			if stored.Meta.Field == field {
				work.WriteString("\n")
				work.WriteString(stored.Body)
			}
		}
	}
	document := ActmemDocument{
		Revision: h.doc.Revision,
		Pulse:    pulse.String(),
		Recap:    recap.String(),
		Work:     work.String(),
		Markdown: h.doc.Render(),
	}
	document.UpdatedAt = parseUpdated(h.doc.Updated)
	document.entries = h.entries()
	return document
}

func writeProjectedBody(b *strings.Builder, body string) {
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	b.WriteString(body)
}

func workFieldTitle(field evolution.WorkField) string {
	switch field {
	case evolution.FieldOpen:
		return "Open"
	case evolution.FieldNext:
		return "Next"
	case evolution.FieldConstraints:
		return "Constraints"
	case evolution.FieldPointers:
		return "Pointers"
	default:
		return "Goal"
	}
}

func parseUpdated(raw string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Unix(0, 0).UTC()
	}
	return parsed.UTC()
}

// newEntryID allocates a fresh actmem entry id; the writer, never the
// model, assigns it.
func newEntryID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "e_" + hex.EncodeToString(raw[:]), nil
}
