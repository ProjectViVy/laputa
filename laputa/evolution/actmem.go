package evolution

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ACTMEM v2 grammar (contracts.md section 3). One ACTMEM.MD per subject:
// YAML front matter, then exactly the Pulse/Recap/Work sections in order,
// each entry enclosed by full-line HTML-comment delimiters.

const ActmemSchemaV2 = "laputa.actmem/v2"

// The contract owns the body caps; laputa/actmem re-exports them so the
// storage package stays the single importer of the grammar.
const (
	ActmemRingCapChars  = 1600 // Pulse/Recap each
	ActmemWorkCapChars  = 1600
	ActmemReadCapChars  = 1200
	ActmemCapsuleCap    = 800
	ActmemPulseEntryCap = 280
	ActmemRecapEntryCap = 200
)

var (
	actmemEntryIDRe    = regexp.MustCompile(`^e_[0-9a-f]{32}$`)
	actmemOpenPrefix   = "<!-- actmem-entry:"
	actmemClosePrefix  = "<!-- /actmem-entry:"
	actmemOpenLineRe   = regexp.MustCompile(`^<!-- actmem-entry:(e_[0-9a-f]{32}) -->$`)
	actmemCloseLineFmt = "<!-- /actmem-entry:%s -->"
)

// ActmemEntryMeta is the header metadata record for one entry: the Entry
// fields minus the body.
type ActmemEntryMeta struct {
	ID         string       `json:"id" yaml:"-"`
	Section    EntrySection `json:"section" yaml:"section"`
	Field      WorkField    `json:"field" yaml:"field"`
	Scope      Scope        `json:"scope" yaml:"scope"`
	SessionID  string       `json:"session_id" yaml:"session_id"`
	EventID    string       `json:"event_id" yaml:"event_id"`
	OccurredAt string       `json:"occurred_at" yaml:"occurred_at"`
	Sources    []SourceRef  `json:"sources" yaml:"sources"`
}

// ActmemEntry is one body entry: metadata plus its Markdown body.
type ActmemEntry struct {
	Meta ActmemEntryMeta
	Body string
}

// ActmemDocument is a parsed ACTMEM v2 document.
type ActmemDocument struct {
	Schema   string
	Revision uint64
	Updated  string
	Sections map[EntrySection][]ActmemEntry
}

// actmemHeader is the front-matter shape.
type actmemHeader struct {
	Schema   string                     `yaml:"schema"`
	Revision uint64                     `yaml:"revision"`
	Updated  string                     `yaml:"updated"`
	Entries  map[string]ActmemEntryMeta `yaml:"entries"`
}

var actmemHeaderKeys = map[string]bool{"schema": true, "revision": true, "updated": true, "entries": true}
var actmemMetaKeys = map[string]bool{"section": true, "field": true, "scope": true, "session_id": true, "event_id": true, "occurred_at": true, "sources": true}
var actmemScopeKeys = map[string]bool{"subject_id": true, "kind": true, "workspace_id": true}
var actmemSourceKeys = map[string]bool{"source_id": true, "record_id": true, "revision": true, "scope": true}

func actmemFormat(format string, args ...any) error {
	return &ContractError{Code: ErrActmemFormat, Message: fmt.Sprintf(format, args...)}
}

// ParseActmemDocument parses an ACTMEM v2 document. Line endings are
// normalized to LF at admission. Invalid input yields actmem_format_error.
func ParseActmemDocument(text string) (ActmemDocument, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return ActmemDocument{}, actmemFormat("document must open with ---")
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return ActmemDocument{}, actmemFormat("front matter never closed")
	}
	meta, revision, updated, err := parseActmemHeader(strings.Join(lines[1:end], "\n"))
	if err != nil {
		return ActmemDocument{}, err
	}
	doc, err := parseActmemBody(lines[end+1:], meta)
	if err != nil {
		return ActmemDocument{}, err
	}
	doc.Schema = ActmemSchemaV2
	doc.Revision = revision
	doc.Updated = updated
	return doc, nil
}

func parseActmemHeader(src string) (map[string]ActmemEntryMeta, uint64, string, error) {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(src), &node); err != nil {
		return nil, 0, "", actmemFormat("header YAML: %v", err)
	}
	if err := rejectYAMLExtensions(&node); err != nil {
		return nil, 0, "", err
	}
	var header actmemHeader
	if err := node.Decode(&header); err != nil {
		return nil, 0, "", actmemFormat("header: %v", err)
	}
	if err := checkYAMLKeys(&node, actmemHeaderKeys); err != nil {
		return nil, 0, "", err
	}
	if header.Schema != ActmemSchemaV2 {
		return nil, 0, "", actmemFormat("schema must be %q", ActmemSchemaV2)
	}
	if header.Updated == "" {
		return nil, 0, "", actmemFormat("updated required")
	}
	if header.Entries == nil {
		header.Entries = map[string]ActmemEntryMeta{}
	}
	if err := checkEntryMetaKeys(&node, header); err != nil {
		return nil, 0, "", err
	}
	for id, m := range header.Entries {
		m.ID = id
		if !actmemEntryIDRe.MatchString(id) {
			return nil, 0, "", actmemFormat("entry id %q must match e_[0-9a-f]{32}", id)
		}
		switch m.Section {
		case SectionPulse, SectionRecap:
			if m.Field != "" {
				return nil, 0, "", actmemFormat("ring entry %s carries no field", id)
			}
		case SectionWork:
			switch m.Field {
			case FieldGoal, FieldOpen, FieldNext, FieldConstraints, FieldPointers:
			default:
				return nil, 0, "", actmemFormat("work entry %s requires a work field", id)
			}
		default:
			return nil, 0, "", actmemFormat("entry %s unknown section %q", id, m.Section)
		}
		if err := m.Scope.Validate(); err != nil {
			return nil, 0, "", actmemFormat("entry %s scope: %v", id, err)
		}
		header.Entries[id] = m
	}
	return header.Entries, header.Revision, header.Updated, nil
}

// rejectYAMLExtensions walks the decoded node tree and rejects anchors,
// aliases and non-standard tags.
func rejectYAMLExtensions(n *yaml.Node) error {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.AliasNode {
		return actmemFormat("YAML aliases are not allowed")
	}
	if n.Style&yaml.TaggedStyle != 0 {
		return actmemFormat("YAML tags are not allowed")
	}
	for _, c := range n.Content {
		if err := rejectYAMLExtensions(c); err != nil {
			return err
		}
	}
	return nil
}

// checkYAMLKeys validates that the root mapping carries exactly the allowed
// header keys (all four required).
func checkYAMLKeys(doc *yaml.Node, allowed map[string]bool) error {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return actmemFormat("empty header")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return actmemFormat("header must be a mapping")
	}
	seen := map[string]bool{}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i].Value
		if !allowed[key] {
			return actmemFormat("unknown header key %q", key)
		}
		seen[key] = true
	}
	for key := range allowed {
		if !seen[key] {
			return actmemFormat("missing header key %q", key)
		}
	}
	return nil
}

// checkEntryMetaKeys walks entry metadata mappings for unknown keys. yaml.v3
// already rejects duplicate mapping keys during Unmarshal.
func checkEntryMetaKeys(doc *yaml.Node, header actmemHeader) error {
	root := doc.Content[0]
	var entriesNode *yaml.Node
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "entries" {
			entriesNode = root.Content[i+1]
		}
	}
	if entriesNode == nil {
		return nil
	}
	if entriesNode.Kind != yaml.MappingNode {
		return actmemFormat("entries must be a mapping")
	}
	for i := 0; i+1 < len(entriesNode.Content); i += 2 {
		id := entriesNode.Content[i].Value
		metaNode := entriesNode.Content[i+1]
		if metaNode.Kind != yaml.MappingNode {
			return actmemFormat("entry %s metadata must be a mapping", id)
		}
		for j := 0; j+1 < len(metaNode.Content); j += 2 {
			key := metaNode.Content[j].Value
			val := metaNode.Content[j+1]
			if !actmemMetaKeys[key] {
				return actmemFormat("entry %s unknown metadata key %q", id, key)
			}
			switch key {
			case "scope":
				if err := checkMappingKeys(val, actmemScopeKeys, "entry "+id+" scope"); err != nil {
					return err
				}
			case "sources":
				if err := checkSourceListKeys(val, id); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func checkMappingKeys(n *yaml.Node, allowed map[string]bool, what string) error {
	if n.Kind != yaml.MappingNode {
		return actmemFormat("%s must be a mapping", what)
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if !allowed[n.Content[i].Value] {
			return actmemFormat("%s: unknown key %q", what, n.Content[i].Value)
		}
	}
	return nil
}

func checkSourceListKeys(n *yaml.Node, entryID string) error {
	if n.Kind != yaml.SequenceNode {
		return actmemFormat("entry %s sources must be a list", entryID)
	}
	for _, item := range n.Content {
		if err := checkMappingKeys(item, actmemSourceKeys, "entry "+entryID+" source"); err != nil {
			return err
		}
		for i := 0; i+1 < len(item.Content); i += 2 {
			if item.Content[i].Value == "scope" {
				if err := checkMappingKeys(item.Content[i+1], actmemScopeKeys, "entry "+entryID+" source scope"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// parseActmemBody scans the section body and binds each entry to its header
// record.
func parseActmemBody(lines []string, meta map[string]ActmemEntryMeta) (ActmemDocument, error) {
	doc := ActmemDocument{Sections: map[EntrySection][]ActmemEntry{
		SectionPulse: {}, SectionRecap: {}, SectionWork: {},
	}}
	order := []struct {
		marker  string
		section EntrySection
	}{{"## Pulse", SectionPulse}, {"## Recap", SectionRecap}, {"## Work", SectionWork}}

	pos := 0
	skipBlanks := func() {
		for pos < len(lines) && strings.TrimSpace(lines[pos]) == "" {
			pos++
		}
	}
	skipBlanks()
	for _, step := range order {
		if pos >= len(lines) || lines[pos] != step.marker {
			return doc, actmemFormat("expected %q", step.marker)
		}
		pos++
		for {
			skipBlanks()
			if pos >= len(lines) || strings.HasPrefix(lines[pos], "## ") {
				break
			}
			m := actmemOpenLineRe.FindStringSubmatch(lines[pos])
			if m == nil {
				return doc, actmemFormat("unassociated text in %s: %q", step.section, lines[pos])
			}
			pos++
			id := m[1]
			entryMeta, ok := meta[id]
			if !ok {
				return doc, actmemFormat("entry %s has no header record", id)
			}
			if entryMeta.Section != step.section {
				return doc, actmemFormat("entry %s declared %s but appears in %s", id, entryMeta.Section, step.section)
			}
			delete(meta, id)
			var body []string
			closeLine := fmt.Sprintf(actmemCloseLineFmt, id)
			closed := false
			for pos < len(lines) {
				line := lines[pos]
				if line == closeLine {
					closed = true
					pos++
					break
				}
				if strings.HasPrefix(line, actmemOpenPrefix) || strings.HasPrefix(line, actmemClosePrefix) {
					return doc, actmemFormat("reserved delimiter inside entry %s body", id)
				}
				body = append(body, line)
				pos++
			}
			if !closed {
				return doc, actmemFormat("entry %s never closed", id)
			}
			doc.Sections[step.section] = append(doc.Sections[step.section], ActmemEntry{Meta: entryMeta, Body: strings.TrimRight(strings.Join(body, "\n"), "\n")})
		}
	}
	skipBlanks()
	if pos < len(lines) {
		return doc, actmemFormat("trailing content after Work section: %q", lines[pos])
	}
	if len(meta) > 0 {
		orphans := make([]string, 0, len(meta))
		for id := range meta {
			orphans = append(orphans, id)
		}
		sort.Strings(orphans)
		return doc, actmemFormat("orphan header metadata: %s", strings.Join(orphans, ","))
	}
	return doc, nil
}

// Render emits the canonical v2 document text.
func (d ActmemDocument) Render() string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("schema: " + ActmemSchemaV2 + "\n")
	fmt.Fprintf(&b, "revision: %d\n", d.Revision)
	b.WriteString("updated: " + strconv.Quote(d.Updated) + "\n")
	ids := make([]string, 0)
	index := map[string]ActmemEntryMeta{}
	for _, entries := range d.Sections {
		for _, e := range entries {
			ids = append(ids, e.Meta.ID)
			index[e.Meta.ID] = e.Meta
		}
	}
	if len(ids) == 0 {
		b.WriteString("entries: {}\n")
	} else {
		b.WriteString("entries:\n")
	}
	sort.Strings(ids)
	for _, id := range ids {
		writeEntryMeta(&b, index[id])
	}
	b.WriteString("---\n")
	for _, section := range []EntrySection{SectionPulse, SectionRecap} {
		fmt.Fprintf(&b, "\n## %s\n", sectionTitle(section))
		for _, e := range d.Sections[section] {
			writeEntry(&b, e)
		}
	}
	b.WriteString("\n## Work\n")
	for _, field := range []WorkField{FieldGoal, FieldOpen, FieldNext, FieldConstraints, FieldPointers} {
		for _, e := range d.Sections[SectionWork] {
			if e.Meta.Field == field {
				writeEntry(&b, e)
			}
		}
	}
	return b.String()
}

func sectionTitle(s EntrySection) string {
	switch s {
	case SectionPulse:
		return "Pulse"
	case SectionRecap:
		return "Recap"
	default:
		return "Work"
	}
}

func writeEntry(b *strings.Builder, e ActmemEntry) {
	fmt.Fprintf(b, "\n<!-- actmem-entry:%s -->\n", e.Meta.ID)
	b.WriteString(e.Body)
	b.WriteString("\n")
	fmt.Fprintf(b, "<!-- /actmem-entry:%s -->\n", e.Meta.ID)
}

func writeEntryMeta(b *strings.Builder, m ActmemEntryMeta) {
	fmt.Fprintf(b, "  %s:\n", m.ID)
	fmt.Fprintf(b, "    section: %s\n", m.Section)
	fmt.Fprintf(b, "    field: %s\n", emptyOrQuoted(string(m.Field)))
	b.WriteString("    scope:\n")
	fmt.Fprintf(b, "      subject_id: %s\n", strconv.Quote(m.Scope.SubjectID))
	fmt.Fprintf(b, "      kind: %s\n", m.Scope.Kind)
	fmt.Fprintf(b, "      workspace_id: %s\n", strconv.Quote(m.Scope.WorkspaceID))
	fmt.Fprintf(b, "    session_id: %s\n", strconv.Quote(m.SessionID))
	fmt.Fprintf(b, "    event_id: %s\n", strconv.Quote(m.EventID))
	fmt.Fprintf(b, "    occurred_at: %s\n", strconv.Quote(m.OccurredAt))
	b.WriteString("    sources:")
	if len(m.Sources) == 0 {
		b.WriteString(" []\n")
		return
	}
	b.WriteString("\n")
	for _, s := range m.Sources {
		fmt.Fprintf(b, "      - source_id: %s\n", strconv.Quote(s.SourceID))
		fmt.Fprintf(b, "        record_id: %s\n", strconv.Quote(s.RecordID))
		fmt.Fprintf(b, "        revision: %d\n", s.Revision)
		b.WriteString("        scope:\n")
		fmt.Fprintf(b, "          subject_id: %s\n", strconv.Quote(s.Scope.SubjectID))
		fmt.Fprintf(b, "          kind: %s\n", s.Scope.Kind)
		fmt.Fprintf(b, "          workspace_id: %s\n", strconv.Quote(s.Scope.WorkspaceID))
	}
}

func emptyOrQuoted(s string) string {
	if s == "" {
		return `""`
	}
	return s
}

// ReadRequest is the scoped explicit-read DTO.
type ReadRequest struct {
	Sections []EntrySection `json:"sections"`
	MaxChars uint32         `json:"max_chars"`
}

// Validate enforces the closed section set and the explicit-read cap.
func (r ReadRequest) Validate() error {
	for _, s := range r.Sections {
		switch s {
		case SectionPulse, SectionRecap, SectionWork:
		default:
			return invalidSchema("unknown actmem section %q", s)
		}
	}
	if r.MaxChars > ActmemReadCapChars {
		return invalidSchema("max_chars exceeds explicit read cap")
	}
	return nil
}

// WorkChangeKind is the closed scoped Work edit vocabulary.
type WorkChangeKind string

const (
	WorkChangeAdd      WorkChangeKind = "add"
	WorkChangeReplace  WorkChangeKind = "replace"
	WorkChangeComplete WorkChangeKind = "complete"
	WorkChangeDrop     WorkChangeKind = "drop"
)

// WorkChange is one scoped Work edit. Add carries an empty entry_id; the
// writer allocates the id. Replace/complete/drop require a visible existing
// Work id; complete/drop carry no body. Scope and event metadata are never
// editable model fields.
type WorkChange struct {
	Kind    WorkChangeKind `json:"kind"`
	EntryID string         `json:"entry_id"`
	Field   WorkField      `json:"field"`
	Body    string         `json:"body"`
	Sources []SourceRef    `json:"sources"`
}

// Validate enforces the per-kind field shape.
func (c WorkChange) Validate() error {
	switch c.Kind {
	case WorkChangeAdd:
		if c.EntryID != "" {
			return invalidSchema("work add carries empty entry_id")
		}
		if c.Body == "" {
			return invalidSchema("work add requires a body")
		}
	case WorkChangeReplace:
		if c.EntryID == "" || c.Body == "" {
			return invalidSchema("work replace requires entry_id and body")
		}
	case WorkChangeComplete, WorkChangeDrop:
		if c.EntryID == "" {
			return invalidSchema("work %s requires entry_id", c.Kind)
		}
		if c.Body != "" {
			return invalidSchema("work %s carries no body", c.Kind)
		}
	default:
		return invalidSchema("unknown work change kind %q", c.Kind)
	}
	switch c.Field {
	case FieldGoal, FieldOpen, FieldNext, FieldConstraints, FieldPointers:
	default:
		return invalidSchema("unknown work field %q", c.Field)
	}
	for _, s := range c.Sources {
		if err := s.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// WorkPatch is the typed work_patch payload and the scoped Work edit request.
type WorkPatch struct {
	BaseRevision uint64       `json:"base_revision"`
	Changes      []WorkChange `json:"changes"`
}

// ActivityResult reports a scoped operation outcome. Entries are filtered to
// the caller's scope; hidden entries' content, ids and counts stay absent.
type ActivityResult struct {
	Changed  bool    `json:"changed"`
	Revision uint64  `json:"revision"`
	Entries  []Entry `json:"entries"`
}
