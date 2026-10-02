// Package actmem implements Laputa's independent ACTMEM activity-memory
// authority. It deliberately has no dependency on Persona, Mentle, or the
// retired Governance section store.
package actmem

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/dashimaki/laputa/evolution"
	"gopkg.in/yaml.v3"
)

const (
	ACTMEMFileName     = "ACTMEM.MD"
	ACTMEMRingCapChars = evolution.ActmemRingCapChars
	ACTMEMWorkCapChars = evolution.ActmemWorkCapChars
	ACTMEMReadCapChars = evolution.ActmemReadCapChars
	ACTMEMCapsuleCap   = evolution.ActmemCapsuleCap
	PulseItemCapChars  = evolution.ActmemPulseEntryCap
	RecapItemCapChars  = evolution.ActmemRecapEntryCap
	CapsuleChunkChars  = 560
)

// Contract spellings retained for callers that mirror the DIVA names.
const (
	ACTMEM_FILE_NAME         = ACTMEMFileName
	ACTMEM_RING_CAP_CHARS    = ACTMEMRingCapChars
	ACTMEM_WORK_CAP_CHARS    = ACTMEMWorkCapChars
	ACTMEM_READ_CAP_CHARS    = ACTMEMReadCapChars
	ACTMEM_CAPSULE_CAP_CHARS = ACTMEMCapsuleCap
	PULSE_ITEM_CAP_CHARS     = PulseItemCapChars
	RECAP_ITEM_CAP_CHARS     = RecapItemCapChars
)

var WorkSections = []string{"Goal", "Open", "Next", "Constraints", "Pointers"}

// Error is an ACTMEM domain error with a stable machine-readable code.
type Error struct {
	Code    string
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return "laputa/actmem: " + e.Message
	}
	return fmt.Sprintf("laputa/actmem: %s: %v", e.Message, e.Cause)
}

func (e *Error) Unwrap() error { return e.Cause }

func newError(code, message string, cause error) *Error {
	return &Error{Code: code, Message: message, Cause: cause}
}

// CodeOf returns the stable error code for an ACTMEM failure.
func CodeOf(err error) string {
	var domain *Error
	if errors.As(err, &domain) {
		return domain.Code
	}
	return "actmem_storage_error"
}

// ActmemDocument is the parsed ACTMEM head's owner projection. Markdown is
// always the canonical rendered form. Unclassified marks a legacy head whose
// entries carry no scope and stay out of every Agent projection until the
// owner re-saves through the v2 kernel.
type ActmemDocument struct {
	Revision     uint64    `json:"revision"`
	UpdatedAt    time.Time `json:"updated_at"`
	Pulse        string    `json:"pulse"`
	Recap        string    `json:"recap"`
	Work         string    `json:"work"`
	Markdown     string    `json:"markdown"`
	Unclassified bool      `json:"unclassified"`
	entries      []evolution.Entry
}

// Entries returns the classified v2 entries (empty for unclassified heads).
func (d ActmemDocument) Entries() []evolution.Entry { return d.entries }

// Document is the shorter public name used by Garden adapters.
type Document = ActmemDocument

func EmptyDocument() ActmemDocument {
	document := ActmemDocument{UpdatedAt: time.Unix(0, 0).UTC(), Work: emptyWork()}
	document.Markdown = render(document)
	return document
}

func (d ActmemDocument) Empty() bool {
	return d.Revision == 0 && d.Pulse == "" && d.Recap == "" && d.Work == emptyWork()
}

type CapsuleSummary struct {
	Name       string    `json:"name"`
	SessionKey string    `json:"session_key"`
	CreatedAt  time.Time `json:"created_at"`
	Chars      int       `json:"chars"`
}

type CapsuleDocument struct {
	CapsuleSummary
	Markdown string `json:"markdown"`
}

type QueryOptions struct {
	Query    string
	Sections []string
	MaxHits  int
	MaxChars int
}

type QueryHit struct {
	Section     string  `json:"section"`
	WorkSection *string `json:"work_section"`
	LineIndex   int     `json:"line_index"`
	Excerpt     string  `json:"excerpt"`
}

type QueryResult struct {
	Revision      uint64     `json:"revision"`
	Items         []QueryHit `json:"items"`
	ReturnedChars int        `json:"returned_chars"`
	Truncated     bool       `json:"truncated"`
}

// Store is one profile's independent <profile>/actmem authority.
//
// Concurrency contract: the mutex makes operations atomic within one
// process. A second process writer remains unsupported — another process
// holding the same directory could interleave atomic file writes and
// produce divergent heads; run one Store owner per profile directory.
type Store struct {
	root  string
	mu    sync.Mutex
	clock func() time.Time
	// crashBeforeHeadCommit is a test hook simulating a crash between the
	// archive write and the head commit of a fold.
	crashBeforeHeadCommit bool
}

// ActmemStore is the DIVA-compatible name.
type ActmemStore = Store

// New creates a store without touching the filesystem. Reads of a missing
// head return the canonical empty document and do not create directories.
func New(configDir string) *Store {
	return &Store{root: filepath.Join(configDir, "actmem")}
}

func Open(configDir string) (*Store, error) { return New(configDir), nil }

func (s *Store) Root() string        { return s.root }
func (s *Store) HeadPath() string    { return filepath.Join(s.root, ACTMEMFileName) }
func (s *Store) CapsulesDir() string { return filepath.Join(s.root, "capsules") }

// SetClock installs a deterministic clock for tests.
func (s *Store) SetClock(clock func() time.Time) { s.clock = clock }

func (s *Store) now() time.Time {
	if s.clock != nil {
		return s.clock().UTC()
	}
	return time.Now().UTC()
}

func (s *Store) Read() (ActmemDocument, error) {
	h, err := s.load()
	if err != nil {
		return ActmemDocument{}, err
	}
	return h.projection(), nil
}

// load reads the head from disk. A missing head is the canonical empty v2
// document and creates nothing.
func (s *Store) load() (head, error) {
	raw, err := os.ReadFile(s.HeadPath())
	if errors.Is(err, os.ErrNotExist) {
		return emptyHead(), nil
	}
	if err != nil {
		return head{}, newError("actmem_io_error", "cannot read ACTMEM", err)
	}
	return loadHead(string(raw))
}

func (s *Store) ListCapsules() ([]CapsuleSummary, error) {
	entries, err := os.ReadDir(s.CapsulesDir())
	if errors.Is(err, os.ErrNotExist) {
		return []CapsuleSummary{}, nil
	}
	if err != nil {
		return nil, newError("actmem_io_error", "cannot list capsules", err)
	}
	result := make([]CapsuleSummary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
			continue
		}
		path := filepath.Join(s.CapsulesDir(), entry.Name())
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, newError("actmem_io_error", "cannot read capsule", readErr)
		}
		document, parseErr := parseCapsule(entry.Name(), string(raw))
		if parseErr != nil {
			return nil, parseErr
		}
		result = append(result, document.CapsuleSummary)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].Name > result[j].Name
		}
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (s *Store) ReadCapsule(name string) (CapsuleDocument, error) {
	if !validCapsuleName(name) {
		return CapsuleDocument{}, newError("actmem_capsule_invalid", "capsule is not part of the server projection", nil)
	}
	summaries, err := s.ListCapsules()
	if err != nil {
		return CapsuleDocument{}, err
	}
	var found bool
	for _, summary := range summaries {
		if summary.Name == name {
			found = true
			break
		}
	}
	if !found {
		return CapsuleDocument{}, newError("actmem_capsule_invalid", "capsule is not part of the server projection", nil)
	}
	raw, err := os.ReadFile(filepath.Join(s.CapsulesDir(), name))
	if err != nil {
		return CapsuleDocument{}, newError("actmem_io_error", "cannot read capsule", err)
	}
	return parseCapsule(name, string(raw))
}

func (s *Store) DeleteCapsule(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.ReadCapsule(name); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(s.CapsulesDir(), name)); err != nil {
		return newError("actmem_storage_error", "cannot delete capsule", err)
	}
	return nil
}

func parse(raw string) (ActmemDocument, error) {
	if !utf8.ValidString(raw) {
		return ActmemDocument{}, newError("actmem_malformed", "ACTMEM is not valid UTF-8", nil)
	}
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) < 9 || lines[0] != "---" {
		return ActmemDocument{}, newError("actmem_malformed", "missing front matter", nil)
	}
	if !strings.HasPrefix(lines[1], "revision: ") || !strings.HasPrefix(lines[2], "updated_at: ") || lines[3] != "---" {
		return ActmemDocument{}, newError("actmem_malformed", "invalid front matter", nil)
	}
	revision, err := strconv.ParseUint(strings.TrimPrefix(lines[1], "revision: "), 10, 64)
	if err != nil {
		return ActmemDocument{}, newError("actmem_malformed", "invalid revision", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, strings.TrimPrefix(lines[2], "updated_at: "))
	if err != nil {
		return ActmemDocument{}, newError("actmem_malformed", "invalid updated_at", err)
	}
	if lines[4] != "" || lines[5] != "# ACTMEM" {
		return ActmemDocument{}, newError("actmem_malformed", "invalid ACTMEM heading", nil)
	}
	body := strings.Join(lines[6:], "\n")
	pulse, rest, ok := takeSection(body, "## Pulse", "## Recap")
	if !ok {
		return ActmemDocument{}, newError("actmem_malformed", "missing Pulse or Recap heading", nil)
	}
	recap, workBody, ok := takeSection(rest, "## Recap", "## Work")
	if !ok {
		return ActmemDocument{}, newError("actmem_malformed", "missing Recap or Work heading", nil)
	}
	workBody = strings.TrimPrefix(workBody, "## Work")
	work, err := normalizeWork(strings.TrimLeft(workBody, "\n"))
	if err != nil {
		return ActmemDocument{}, err
	}
	document := ActmemDocument{Revision: revision, UpdatedAt: updatedAt.UTC(), Pulse: normalizeSection(pulse), Recap: normalizeSection(recap), Work: work}
	if err := validateCaps(document); err != nil {
		return ActmemDocument{}, err
	}
	document.Markdown = render(document)
	return document, nil
}

func takeSection(body, heading, next string) (string, string, bool) {
	body = strings.TrimPrefix(body, "\n")
	if !strings.HasPrefix(body, heading+"\n") {
		return "", "", false
	}
	body = strings.TrimPrefix(body, heading+"\n")
	marker := "\n" + next + "\n"
	index := strings.Index(body, marker)
	if index < 0 {
		return "", "", false
	}
	return body[:index], body[index+1:], true
}

func render(document ActmemDocument) string {
	return fmt.Sprintf("---\nrevision: %d\nupdated_at: %s\n---\n\n# ACTMEM\n\n## Pulse\n%s\n\n## Recap\n%s\n\n## Work\n%s\n", document.Revision, document.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"), strings.TrimSpace(document.Pulse), strings.TrimSpace(document.Recap), strings.TrimSpace(document.Work))
}

func validateCaps(document ActmemDocument) error {
	for _, item := range []struct {
		name  string
		value string
		limit int
	}{
		{"pulse", document.Pulse, ACTMEMRingCapChars},
		{"recap", document.Recap, ACTMEMRingCapChars},
		{"work", document.Work, ACTMEMWorkCapChars},
	} {
		if runeLen(item.value) > item.limit {
			return newError("actmem_cap_exceeded", item.name+" exceeds its capacity", nil)
		}
	}
	return nil
}

func emptyWork() string { return renderWork(map[string]string{}) }

func normalizeWork(raw string) (string, error) {
	sections, err := splitWork(raw)
	if err != nil {
		return "", err
	}
	return renderWork(sections), nil
}

func splitWork(raw string) (map[string]string, error) {
	sections := map[string]string{}
	var current string
	var body []string
	flush := func() {
		if current != "" {
			sections[current] = normalizeSection(strings.Join(body, "\n"))
			body = nil
		}
	}
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "### ") {
			flush()
			canonical, err := canonicalWorkSection(strings.TrimPrefix(line, "### "))
			if err != nil {
				return nil, err
			}
			if _, exists := sections[canonical]; exists {
				return nil, newError("actmem_malformed", "duplicate Work section "+canonical, nil)
			}
			current = canonical
			continue
		}
		if current == "" {
			if strings.TrimSpace(line) != "" {
				return nil, newError("actmem_malformed", "Work content is outside a registered subsection", nil)
			}
			continue
		}
		body = append(body, line)
	}
	flush()
	for _, section := range WorkSections {
		if _, ok := sections[section]; !ok {
			sections[section] = ""
		}
	}
	return sections, nil
}

func renderWork(sections map[string]string) string {
	parts := make([]string, 0, len(WorkSections))
	for _, name := range WorkSections {
		body := strings.TrimSpace(sections[name])
		if body == "" {
			parts = append(parts, "### "+name)
		} else {
			parts = append(parts, "### "+name+"\n"+body)
		}
	}
	return strings.Join(parts, "\n\n")
}

func canonicalWorkSection(value string) (string, error) {
	value = strings.TrimSpace(value)
	for _, section := range WorkSections {
		if strings.EqualFold(section, value) {
			return section, nil
		}
	}
	return "", newError("actmem_invalid_edit", "unknown Work section: "+value, nil)
}

func normalizeSection(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "\r\n", "\n"))
}

func RecapFromFinalResponse(value string) string {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	inCode := false
	var paragraph []string
	for _, line := range strings.Split(normalized, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}
		if trimmed == "" {
			if len(paragraph) > 0 {
				break
			}
			continue
		}
		trimmed = strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		trimmed = strings.TrimSpace(strings.TrimLeft(trimmed, "-*>"))
		if trimmed != "" {
			paragraph = append(paragraph, trimmed)
		}
	}
	return truncateChars(strings.Join(paragraph, " "), RecapItemCapChars)
}

func recapFromFinalResponse(value string) string { return RecapFromFinalResponse(value) }

func truncateChars(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	if max <= 0 {
		return ""
	}
	return string(runes[:max-1]) + "…"
}

func runeLen(value string) int { return len([]rune(value)) }
func nonEmptyLines(value string) []string {
	var result []string
	for _, line := range strings.Split(value, "\n") {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result
}

func validCapsuleName(name string) bool {
	return name != "" && filepath.Base(name) == name && filepath.Ext(name) == ".md" && !strings.ContainsAny(name, `/\\`) && name != ".md" && !strings.Contains(name, "..")
}

// Query performs bounded literal, case-insensitive matching over the caller's
// visible entries. Unclassified heads and hidden scopes expose nothing.
func (s *Store) Query(caller evolution.Scope, options QueryOptions) (QueryResult, error) {
	query := strings.ToLower(normalizeSection(options.Query))
	if query == "" {
		return QueryResult{}, newError("actmem_invalid_edit", "query is required", nil)
	}
	if options.MaxHits == 0 {
		options.MaxHits = 20
	}
	if options.MaxChars == 0 {
		options.MaxChars = ACTMEMReadCapChars
	}
	if options.MaxHits < 1 || options.MaxHits > 50 || options.MaxChars < 1 || options.MaxChars > ACTMEMReadCapChars {
		return QueryResult{}, newError("actmem_invalid_edit", "query bounds are invalid", nil)
	}
	if err := caller.Validate(); err != nil {
		return QueryResult{}, newError("actmem_invalid_edit", "caller scope: "+err.Error(), nil)
	}
	h, err := s.load()
	if err != nil {
		return QueryResult{}, err
	}
	result := QueryResult{Revision: h.revision, Items: []QueryHit{}}
	if !h.classified() {
		return result, nil
	}
	allowed := map[evolution.EntrySection]bool{evolution.SectionPulse: true, evolution.SectionRecap: true, evolution.SectionWork: true}
	for _, section := range options.Sections {
		if !allowed[evolution.EntrySection(strings.ToLower(strings.TrimSpace(section)))] {
			return QueryResult{}, newError("actmem_invalid_edit", "unknown query section: "+section, nil)
		}
	}
	wanted := func(section evolution.EntrySection) bool {
		if len(options.Sections) == 0 {
			return true
		}
		for _, s := range options.Sections {
			if evolution.EntrySection(strings.ToLower(strings.TrimSpace(s))) == section {
				return true
			}
		}
		return false
	}
	for _, entry := range h.entries() {
		if !wanted(entry.Section) || !visibleTo(entry.Scope, caller) {
			continue
		}
		for index, line := range nonEmptyLines(entry.Body) {
			if !strings.Contains(strings.ToLower(line), query) {
				continue
			}
			hit := QueryHit{Section: string(entry.Section), LineIndex: index, Excerpt: line}
			if entry.Section == evolution.SectionWork {
				field := string(entry.Field)
				hit.WorkSection = &field
			}
			if !appendQueryHit(&result, hit, options.MaxHits, options.MaxChars) {
				return result, nil
			}
		}
	}
	return result, nil
}

func appendQueryHit(result *QueryResult, hit QueryHit, maxHits, maxChars int) bool {
	if len(result.Items) >= maxHits {
		result.Truncated = true
		return false
	}
	remaining := maxChars - result.ReturnedChars
	if remaining <= 0 {
		result.Truncated = true
		return false
	}
	if runeLen(hit.Excerpt) > remaining {
		hit.Excerpt = truncateChars(hit.Excerpt, remaining)
		result.Truncated = true
	}
	result.Items = append(result.Items, hit)
	result.ReturnedChars += runeLen(hit.Excerpt)
	if result.ReturnedChars >= maxChars {
		result.Truncated = true
		return false
	}
	return true
}

func parseCapsule(name, markdown string) (CapsuleDocument, error) {
	if !validCapsuleName(name) {
		return CapsuleDocument{}, newError("actmem_capsule_invalid", "invalid capsule name", nil)
	}
	if !utf8.ValidString(markdown) {
		return CapsuleDocument{}, newError("actmem_malformed", "capsule is not valid UTF-8", nil)
	}
	if runeLen(markdown) > ACTMEMCapsuleCap {
		return CapsuleDocument{}, newError("actmem_cap_exceeded", "capsule exceeds its capacity", nil)
	}
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) < 6 || lines[0] != "---" {
		return CapsuleDocument{}, newError("actmem_malformed", "capsule front matter is invalid", nil)
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return CapsuleDocument{}, newError("actmem_malformed", "capsule front matter is invalid", nil)
	}
	var header struct {
		SessionKey string `yaml:"session_key"`
		CreatedAt  string `yaml:"created_at"`
	}
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &header); err != nil {
		return CapsuleDocument{}, newError("actmem_malformed", "capsule metadata is invalid", err)
	}
	if header.SessionKey == "" {
		return CapsuleDocument{}, newError("actmem_malformed", "capsule session_key is invalid", nil)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, header.CreatedAt)
	if err != nil {
		return CapsuleDocument{}, newError("actmem_malformed", "capsule created_at is invalid", err)
	}
	rest := lines[end+1:]
	if len(rest) < 2 || rest[0] != "" || rest[1] != "# ACTMEM Capsule" {
		return CapsuleDocument{}, newError("actmem_malformed", "capsule heading is invalid", nil)
	}
	summary := CapsuleSummary{Name: name, SessionKey: header.SessionKey, CreatedAt: createdAt.UTC(), Chars: runeLen(markdown)}
	return CapsuleDocument{CapsuleSummary: summary, Markdown: markdown}, nil
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	random := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, random); err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.tmp-%s", path, hex.EncodeToString(random))
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	ok = true
	return nil
}
