package actmem

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ProjectViVy/laputa/laputa/evolution"
	"gopkg.in/yaml.v3"
)

// Spec section 5.1 write operations on the v2 head. One in-process writer
// serializes everything under s.mu; a second writer process is unsupported
// and must not open the same authority directory for mutation.

// AppendEntry is the system append: the host supplies a fully-scoped ring
// entry; the store allocates the id, stamps occurred_at, dedupes a retained
// event id and evicts oldest entries on ring overflow.
func (s *Store) AppendEntry(entry evolution.Entry) (evolution.ActivityResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, err := s.load()
	if err != nil {
		return evolution.ActivityResult{}, err
	}
	if !h.classified() {
		return evolution.ActivityResult{}, newError("actmem_unclassified_head", "unclassified ACTMEM head accepts no scoped writes", nil)
	}
	if entry.Section != evolution.SectionPulse && entry.Section != evolution.SectionRecap {
		return evolution.ActivityResult{}, newError("actmem_invalid_edit", "system append targets pulse or recap", nil)
	}
	if err := entry.Scope.Validate(); err != nil {
		return evolution.ActivityResult{}, newError("actmem_invalid_edit", "entry scope: "+err.Error(), nil)
	}
	entry.Body = normalizeSection(entry.Body)
	itemCap := PulseItemCapChars
	if entry.Section == evolution.SectionRecap {
		itemCap = RecapItemCapChars
	}
	entry.Body = truncateChars(entry.Body, itemCap)
	ring := h.doc.Sections[entry.Section]
	if entry.Body == "" {
		return evolution.ActivityResult{Changed: false, Revision: h.revision, Entries: []evolution.Entry{}}, nil
	}
	if entry.EventID != "" {
		for _, existing := range ring {
			if existing.Meta.EventID == entry.EventID {
				return evolution.ActivityResult{Changed: false, Revision: h.revision, Entries: []evolution.Entry{entryFromStored(existing)}}, nil
			}
		}
	}
	if err := rejectReservedDelimiters(entry.Body); err != nil {
		return evolution.ActivityResult{}, err
	}
	entry.ID, err = newEntryID()
	if err != nil {
		return evolution.ActivityResult{}, newError("actmem_storage_error", "cannot allocate entry id", err)
	}
	entry.OccurredAt = s.now().Format("2006-01-02T15:04:05.000Z07:00")
	ring = append(ring, storedFromEntry(entry))
	for sectionChars(ring) > ACTMEMRingCapChars && len(ring) > 1 {
		ring = ring[1:]
	}
	h.doc.Sections[entry.Section] = ring
	if err := s.commitHead(&h); err != nil {
		return evolution.ActivityResult{}, err
	}
	return evolution.ActivityResult{Changed: true, Revision: h.revision, Entries: []evolution.Entry{entry}}, nil
}

// ApplyWorkPatch applies scoped Work changes against base_revision. Entries
// outside the caller's admitted scope are untouched and unrevealed; a stale
// base conflicts with no automatic overwrite.
func (s *Store) ApplyWorkPatch(caller evolution.Scope, patch evolution.WorkPatch) (evolution.ActivityResult, error) {
	if err := caller.Validate(); err != nil {
		return evolution.ActivityResult{}, newError("actmem_invalid_edit", "caller scope: "+err.Error(), nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	h, err := s.load()
	if err != nil {
		return evolution.ActivityResult{}, err
	}
	if !h.classified() {
		return evolution.ActivityResult{}, newError("actmem_unclassified_head", "unclassified ACTMEM head accepts no scoped writes", nil)
	}
	if h.revision != patch.BaseRevision {
		return evolution.ActivityResult{}, revisionConflict(patch.BaseRevision, h.revision)
	}
	work := append([]evolution.ActmemEntry(nil), h.doc.Sections[evolution.SectionWork]...)
	byID := map[string]int{}
	for i, e := range work {
		byID[e.Meta.ID] = i
	}
	var changed []evolution.Entry
	for _, change := range patch.Changes {
		if err := change.Validate(); err != nil {
			return evolution.ActivityResult{}, newError("actmem_invalid_edit", err.Error(), nil)
		}
		switch change.Kind {
		case evolution.WorkChangeAdd:
			if err := rejectReservedDelimiters(change.Body); err != nil {
				return evolution.ActivityResult{}, err
			}
			entry := evolution.Entry{
				Section:    evolution.SectionWork,
				Field:      change.Field,
				Scope:      caller,
				Body:       normalizeSection(change.Body),
				Sources:    change.Sources,
				OccurredAt: s.now().Format("2006-01-02T15:04:05.000Z07:00"),
			}
			entry.ID, err = newEntryID()
			if err != nil {
				return evolution.ActivityResult{}, newError("actmem_storage_error", "cannot allocate entry id", err)
			}
			work = append(work, storedFromEntry(entry))
			byID[entry.ID] = len(work) - 1
			changed = append(changed, entry)
		default:
			idx, ok := byID[change.EntryID]
			if !ok || !visibleTo(work[idx].Meta.Scope, caller) {
				return evolution.ActivityResult{}, newError("actmem_invalid_edit", "unknown or invisible work entry "+change.EntryID, nil)
			}
			stored := work[idx]
			if stored.Meta.Field != change.Field {
				return evolution.ActivityResult{}, newError("actmem_invalid_edit", "work field mismatch for "+change.EntryID, nil)
			}
			switch change.Kind {
			case evolution.WorkChangeReplace:
				if err := rejectReservedDelimiters(change.Body); err != nil {
					return evolution.ActivityResult{}, err
				}
				stored.Body = normalizeSection(change.Body)
				if len(change.Sources) > 0 {
					stored.Meta.Sources = change.Sources
				}
				work[idx] = stored
				changed = append(changed, entryFromStored(stored))
			case evolution.WorkChangeComplete, evolution.WorkChangeDrop:
				work = append(work[:idx], work[idx+1:]...)
				delete(byID, change.EntryID)
				for id, i := range byID {
					if i > idx {
						byID[id] = i - 1
					}
				}
			}
		}
	}
	if sectionChars(work) > ACTMEMWorkCapChars {
		return evolution.ActivityResult{}, newError("actmem_cap_exceeded", "work exceeds its capacity", nil)
	}
	h.doc.Sections[evolution.SectionWork] = work
	if err := s.commitHead(&h); err != nil {
		return evolution.ActivityResult{}, err
	}
	return evolution.ActivityResult{Changed: len(changed) > 0, Revision: h.revision, Entries: changed}, nil
}

// Save is the authenticated owner's whole-document write. The markdown must
// be valid v2 within caps; the stored revision is head+1 regardless of the
// value the body declares. Invalid input leaves the file byte-identical.
func (s *Store) Save(markdown string, baseRevision uint64) (evolution.ActivityResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, err := s.load()
	if err != nil {
		return evolution.ActivityResult{}, err
	}
	if h.revision != baseRevision {
		return evolution.ActivityResult{}, revisionConflict(baseRevision, h.revision)
	}
	doc, err := evolution.ParseActmemDocument(markdown)
	if err != nil {
		return evolution.ActivityResult{}, newError("actmem_format_error", err.Error(), nil)
	}
	for section, entries := range doc.Sections {
		limit := ACTMEMRingCapChars
		if section == evolution.SectionWork {
			limit = ACTMEMWorkCapChars
		}
		if sectionChars(entries) > limit {
			return evolution.ActivityResult{}, newError("actmem_cap_exceeded", string(section)+" exceeds its capacity", nil)
		}
	}
	doc.Revision = h.revision + 1
	doc.Updated = s.now().Format("2006-01-02T15:04:05.000Z07:00")
	if err := writeAtomic(s.HeadPath(), []byte(doc.Render())); err != nil {
		return evolution.ActivityResult{}, newError("actmem_storage_error", "cannot atomically write ACTMEM", err)
	}
	return evolution.ActivityResult{Changed: true, Revision: doc.Revision, Entries: flattenEntries(doc)}, nil
}

// FoldSession archives one session's Pulse/Recap entries archive-first:
// write the deterministic capsule(s), then remove exactly the recorded
// source entries from the head. A crash can leave entries in both places;
// a retry completes removal of the archived set only — entries appended
// meanwhile are never consumed by the stale fold. A capsule whose recorded
// source body no longer matches the head entry is an explicit conflict.
func (s *Store) FoldSession(sessionKey string) ([]CapsuleSummary, error) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return nil, newError("actmem_invalid_edit", "session_key is required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	h, err := s.load()
	if err != nil {
		return nil, err
	}
	if !h.classified() {
		return nil, newError("actmem_unclassified_head", "unclassified ACTMEM head accepts no scoped writes", nil)
	}
	var sources []evolution.ActmemEntry
	byID := map[string]evolution.ActmemEntry{}
	for _, section := range []evolution.EntrySection{evolution.SectionPulse, evolution.SectionRecap} {
		for _, e := range h.doc.Sections[section] {
			if e.Meta.SessionID == sessionKey {
				sources = append(sources, e)
				byID[e.Meta.ID] = e
			}
		}
	}
	if len(sources) == 0 {
		return []CapsuleSummary{}, nil
	}
	toRemove := map[string]bool{}
	archivedIDs := map[string]bool{}
	var summaries []CapsuleSummary
	// Phase 1: any capsule this session already recorded wins removal; its
	// sources are the interrupted fold's exact set.
	existing, err := s.sessionCapsules(sessionKey)
	if err != nil {
		return nil, err
	}
	for _, capsule := range existing {
		for _, recorded := range capsule.sources {
			headEntry, present := byID[recorded.Meta.ID]
			if !present {
				continue // already removed or evicted before the crash
			}
			if headEntry.Body != recorded.Body {
				return nil, newError("actmem_fold_conflict", "source entry "+recorded.Meta.ID+" changed since "+capsule.name+" was archived", nil)
			}
			toRemove[recorded.Meta.ID] = true
			archivedIDs[recorded.Meta.ID] = true
		}
		summaries = append(summaries, capsule.summary)
	}
	// Phase 2: fold whatever sources remain unarchived — but not on a call
	// that just completed an interrupted fold; the later entries belong to
	// the next fold, not the stale one.
	if len(toRemove) == 0 {
		var remaining []evolution.ActmemEntry
		for _, e := range sources {
			if !archivedIDs[e.Meta.ID] {
				remaining = append(remaining, e)
			}
		}
		for _, chunk := range chunkFoldSources(sessionKey, remaining) {
			name := foldCapsuleName(entriesOf(chunk))
			path := filepath.Join(s.CapsulesDir(), name)
			raw, readErr := os.ReadFile(path)
			switch {
			case readErr == nil:
				capsule, parseErr := parseFoldCapsule(name, string(raw))
				if parseErr != nil || capsule.digest != foldDigest(sessionKey, chunk) {
					return nil, newError("actmem_fold_conflict", "existing capsule "+name+" does not match the source set", nil)
				}
			case errors.Is(readErr, os.ErrNotExist):
				if err := writeAtomic(path, []byte(renderFoldCapsule(sessionKey, chunk))); err != nil {
					return nil, newError("actmem_storage_error", "cannot write capsule", err)
				}
			default:
				return nil, newError("actmem_io_error", "cannot inspect capsule path", readErr)
			}
			for _, e := range chunk {
				toRemove[e.Meta.ID] = true
			}
			summaries = append(summaries, CapsuleSummary{Name: name, SessionKey: sessionKey, CreatedAt: foldCreatedAt(chunk), Chars: len([]rune(renderFoldCapsule(sessionKey, chunk)))})
		}
	}
	if s.crashBeforeHeadCommit {
		return nil, newError("actmem_storage_error", "injected crash before head commit", nil)
	}
	for _, section := range []evolution.EntrySection{evolution.SectionPulse, evolution.SectionRecap} {
		var kept []evolution.ActmemEntry
		for _, e := range h.doc.Sections[section] {
			if !toRemove[e.Meta.ID] {
				kept = append(kept, e)
			}
		}
		h.doc.Sections[section] = kept
	}
	if err := s.commitHead(&h); err != nil {
		return nil, err
	}
	return summaries, nil
}

type sessionCapsule struct {
	name    string
	summary CapsuleSummary
	sources []evolution.ActmemEntry
}

// sessionCapsules reads this session's well-formed fold capsules.
func (s *Store) sessionCapsules(sessionKey string) ([]sessionCapsule, error) {
	entries, err := os.ReadDir(s.CapsulesDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, newError("actmem_io_error", "cannot list capsules", err)
	}
	var out []sessionCapsule
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "fold-") || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(s.CapsulesDir(), entry.Name()))
		if err != nil {
			return nil, newError("actmem_io_error", "cannot read capsule", err)
		}
		capsule, err := parseFoldCapsule(entry.Name(), string(raw))
		if err != nil || capsule.sessionKey != sessionKey {
			continue // foreign or legacy capsules are not fold records
		}
		out = append(out, sessionCapsule{
			name:    entry.Name(),
			sources: capsule.sources,
			summary: CapsuleSummary{Name: entry.Name(), SessionKey: sessionKey, CreatedAt: capsule.createdAt, Chars: len([]rune(string(raw)))},
		})
	}
	return out, nil
}

// commitHead bumps the revision and persists the canonical render.
func (s *Store) commitHead(h *head) error {
	h.doc.Revision = h.revision + 1
	h.doc.Updated = s.now().Format("2006-01-02T15:04:05.000Z07:00")
	markdown := h.doc.Render()
	if err := writeAtomic(s.HeadPath(), []byte(markdown)); err != nil {
		return newError("actmem_storage_error", "cannot atomically write ACTMEM", err)
	}
	h.revision = h.doc.Revision
	h.updated = h.doc.Updated
	h.markdown = markdown
	return nil
}

func revisionConflict(expected, actual uint64) error {
	return newError("actmem_revision_conflict", fmt.Sprintf("expected revision %d, actual %d", expected, actual), nil)
}

func sectionChars(entries []evolution.ActmemEntry) int {
	total := 0
	for _, e := range entries {
		total += len([]rune(e.Body))
	}
	return total
}

func flattenEntries(doc evolution.ActmemDocument) []evolution.Entry {
	h := head{doc: doc}
	return h.entries()
}

// rejectReservedDelimiters refuses bodies carrying a full-line actmem entry
// delimiter; they would corrupt the container grammar.
func rejectReservedDelimiters(body string) error {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "<!-- actmem-entry:") || strings.HasPrefix(line, "<!-- /actmem-entry:") {
			return newError("actmem_format_error", "body contains a reserved entry delimiter", nil)
		}
	}
	return nil
}

// foldCapsuleName is deterministic: same ordered source set, same name.
func foldCapsuleName(sources []evolution.Entry) string {
	digest := sha256.New()
	for _, e := range sources {
		digest.Write([]byte(e.ID))
		digest.Write([]byte{0})
	}
	return "fold-" + hex.EncodeToString(digest.Sum(nil))[:24] + ".md"
}

func foldDigest(sessionKey string, sources []evolution.ActmemEntry) string {
	digest := sha256.New()
	digest.Write([]byte(sessionKey))
	for _, e := range sources {
		digest.Write([]byte{0})
		digest.Write([]byte(e.Meta.ID))
		digest.Write([]byte{0})
		digest.Write([]byte(e.Body))
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func entriesOf(chunk []evolution.ActmemEntry) []evolution.Entry {
	var out []evolution.Entry
	for _, e := range chunk {
		out = append(out, entryFromStored(e))
	}
	return out
}

// foldCreatedAt is deterministic — the newest source's occurred_at — so a
// retry reproduces byte-identical capsule content.
func foldCreatedAt(sources []evolution.ActmemEntry) (latest time.Time) {
	for _, e := range sources {
		if t, err := time.Parse(time.RFC3339Nano, e.Meta.OccurredAt); err == nil && t.After(latest) {
			latest = t
		}
	}
	return latest.UTC()
}

// chunkFoldSources greedily packs ordered sources into capsules under the
// capsule cap, so a retry reproduces exactly the same chunking.
func chunkFoldSources(sessionKey string, sources []evolution.ActmemEntry) [][]evolution.ActmemEntry {
	var chunks [][]evolution.ActmemEntry
	var current []evolution.ActmemEntry
	for _, source := range sources {
		candidate := append(append([]evolution.ActmemEntry(nil), current...), source)
		if len(current) > 0 && len([]rune(renderFoldCapsule(sessionKey, candidate))) > ACTMEMCapsuleCap {
			chunks = append(chunks, current)
			current = []evolution.ActmemEntry{source}
			continue
		}
		current = candidate
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	return chunks
}

// renderFoldCapsule emits the capsule as strict YAML front matter (session,
// digest, per-entry metadata) plus delimited bodies, so scope metadata
// survives folding and a retry can byte-verify the archive.
func renderFoldCapsule(sessionKey string, sources []evolution.ActmemEntry) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "session_key: %s\n", strconvQuote(sessionKey))
	fmt.Fprintf(&b, "created_at: %s\n", strconvQuote(foldCreatedAt(sources).Format("2006-01-02T15:04:05.000Z07:00")))
	fmt.Fprintf(&b, "fold_digest: %s\n", strconvQuote(foldDigest(sessionKey, sources)))
	b.WriteString("entries:\n")
	for _, e := range sources {
		meta, _ := yaml.Marshal(map[string]evolution.ActmemEntryMeta{e.Meta.ID: e.Meta})
		for _, line := range strings.Split(strings.TrimRight(string(meta), "\n"), "\n") {
			b.WriteString("  " + line + "\n")
		}
	}
	b.WriteString("---\n\n# ACTMEM Capsule\n\n")
	for _, e := range sources {
		fmt.Fprintf(&b, "<!-- actmem-entry:%s -->\n%s\n<!-- /actmem-entry:%s -->\n", e.Meta.ID, e.Body, e.Meta.ID)
	}
	return b.String()
}

type foldCapsuleHeader struct {
	SessionKey string                               `yaml:"session_key"`
	CreatedAt  string                               `yaml:"created_at"`
	FoldDigest string                               `yaml:"fold_digest"`
	Entries    map[string]evolution.ActmemEntryMeta `yaml:"entries"`
}

type parsedFoldCapsule struct {
	sessionKey string
	digest     string
	createdAt  time.Time
	sources    []evolution.ActmemEntry
}

// parseFoldCapsule validates a capsule written by this kernel and returns
// its recorded fold record for retry matching.
func parseFoldCapsule(name, markdown string) (parsedFoldCapsule, error) {
	if !validCapsuleName(name) {
		return parsedFoldCapsule{}, newError("actmem_capsule_invalid", "invalid capsule name", nil)
	}
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	if len(lines) < 5 || lines[0] != "---" {
		return parsedFoldCapsule{}, newError("actmem_malformed", "capsule front matter is invalid", nil)
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return parsedFoldCapsule{}, newError("actmem_malformed", "capsule front matter never closed", nil)
	}
	var header foldCapsuleHeader
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &header); err != nil {
		return parsedFoldCapsule{}, newError("actmem_malformed", "capsule header is invalid", err)
	}
	if header.SessionKey == "" || header.FoldDigest == "" {
		return parsedFoldCapsule{}, newError("actmem_malformed", "capsule carries no fold record", nil)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, header.CreatedAt)
	if err != nil {
		return parsedFoldCapsule{}, newError("actmem_malformed", "capsule created_at is invalid", err)
	}
	parsed := parsedFoldCapsule{sessionKey: header.SessionKey, digest: header.FoldDigest, createdAt: createdAt.UTC()}
	// Delimited bodies after the heading.
	pos := end + 1
	for pos < len(lines) && strings.TrimSpace(lines[pos]) == "" {
		pos++
	}
	if pos < len(lines) && lines[pos] == "# ACTMEM Capsule" {
		pos++
	}
	for pos < len(lines) {
		line := lines[pos]
		if strings.TrimSpace(line) == "" {
			pos++
			continue
		}
		m := actmemOpenLineRe.FindStringSubmatch(line)
		if m == nil {
			return parsedFoldCapsule{}, newError("actmem_malformed", "capsule body is malformed", nil)
		}
		id := m[1]
		meta, ok := header.Entries[id]
		if !ok {
			return parsedFoldCapsule{}, newError("actmem_malformed", "capsule entry "+id+" has no metadata", nil)
		}
		meta.ID = id
		pos++
		closeLine := fmt.Sprintf("<!-- /actmem-entry:%s -->", id)
		var body []string
		closed := false
		for pos < len(lines) {
			if lines[pos] == closeLine {
				closed = true
				pos++
				break
			}
			body = append(body, lines[pos])
			pos++
		}
		if !closed {
			return parsedFoldCapsule{}, newError("actmem_malformed", "capsule entry "+id+" never closed", nil)
		}
		parsed.sources = append(parsed.sources, evolution.ActmemEntry{Meta: meta, Body: strings.TrimRight(strings.Join(body, "\n"), "\n")})
	}
	return parsed, nil
}

var actmemOpenLineRe = regexp.MustCompile(`^<!-- actmem-entry:(e_[0-9a-f]{32}) -->$`)

func strconvQuote(v string) string {
	raw, _ := yaml.Marshal(v)
	return strings.TrimSpace(string(raw))
}
