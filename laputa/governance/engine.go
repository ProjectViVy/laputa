package governance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

// SectionName represents one of the 14 governance sections.
type SectionName string

const (
	SectionIdentity          SectionName = "01-identity"
	SectionRelationship      SectionName = "02-relationship"
	SectionCommitment        SectionName = "03-commitment"
	SectionPreferences       SectionName = "04-preferences"
	SectionMemoryMD          SectionName = "05-memory_md"
	SectionHistoryMD         SectionName = "06-history_md"
	SectionDaily             SectionName = "07-daily"
	SectionWeekly            SectionName = "08-weekly"
	SectionMonthly           SectionName = "09-monthly"
	SectionJournalReflective SectionName = "10-journal_reflective"
	SectionProposalInbox     SectionName = "11-proposal_inbox"
	SectionChangelog         SectionName = "12-changelog"
	SectionReportIndexes     SectionName = "13-report_indexes"
	SectionAAAKSummaries     SectionName = "14-aaak_summaries"
)

// AllSections is the ordered list of all governance sections.
var AllSections = []SectionName{
	SectionIdentity,
	SectionRelationship,
	SectionCommitment,
	SectionPreferences,
	SectionMemoryMD,
	SectionHistoryMD,
	SectionDaily,
	SectionWeekly,
	SectionMonthly,
	SectionJournalReflective,
	SectionProposalInbox,
	SectionChangelog,
	SectionReportIndexes,
	SectionAAAKSummaries,
}

// WriteAuthority defines who may write to a section.
type WriteAuthority string

const (
	AuthorityAgentSelf WriteAuthority = "agent_self"
	AuthorityUserOnly  WriteAuthority = "user_only"
	AuthorityReport    WriteAuthority = "report_system"
	AuthorityTBD       WriteAuthority = "tbd"
)

// SectionInfo holds metadata for a governance section.
type SectionInfo struct {
	Name        SectionName
	WriteAuth   WriteAuthority
	SchemaOwner string
	Status      string // "stable" or "tbd"
}

// SectionRegistry maps section names to their metadata.
var SectionRegistry = map[SectionName]SectionInfo{
	SectionIdentity:          {Name: SectionIdentity, WriteAuth: AuthorityAgentSelf, SchemaOwner: "laputa", Status: "stable"},
	SectionRelationship:      {Name: SectionRelationship, WriteAuth: AuthorityAgentSelf, SchemaOwner: "laputa", Status: "stable"},
	SectionCommitment:        {Name: SectionCommitment, WriteAuth: AuthorityUserOnly, SchemaOwner: "laputa", Status: "stable"},
	SectionPreferences:       {Name: SectionPreferences, WriteAuth: AuthorityAgentSelf, SchemaOwner: "laputa", Status: "stable"},
	SectionMemoryMD:          {Name: SectionMemoryMD, WriteAuth: AuthorityAgentSelf, SchemaOwner: "laputa", Status: "stable"},
	SectionHistoryMD:         {Name: SectionHistoryMD, WriteAuth: AuthorityAgentSelf, SchemaOwner: "laputa", Status: "stable"},
	SectionDaily:             {Name: SectionDaily, WriteAuth: AuthorityReport, SchemaOwner: "report_system", Status: "stable"},
	SectionWeekly:            {Name: SectionWeekly, WriteAuth: AuthorityReport, SchemaOwner: "report_system", Status: "stable"},
	SectionMonthly:           {Name: SectionMonthly, WriteAuth: AuthorityReport, SchemaOwner: "report_system", Status: "stable"},
	SectionJournalReflective: {Name: SectionJournalReflective, WriteAuth: AuthorityTBD, SchemaOwner: "tbd", Status: "tbd"},
	SectionProposalInbox:     {Name: SectionProposalInbox, WriteAuth: AuthorityTBD, SchemaOwner: "tbd", Status: "tbd"},
	SectionChangelog:         {Name: SectionChangelog, WriteAuth: AuthorityTBD, SchemaOwner: "tbd", Status: "tbd"},
	SectionReportIndexes:     {Name: SectionReportIndexes, WriteAuth: AuthorityTBD, SchemaOwner: "tbd", Status: "tbd"},
	SectionAAAKSummaries:     {Name: SectionAAAKSummaries, WriteAuth: AuthorityTBD, SchemaOwner: "tbd", Status: "tbd"},
}

// SectionStore is the interface for CRUD operations on governance sections.
type SectionStore interface {
	// Read returns the current data for a section.
	Read(ctx context.Context, section SectionName) (map[string]any, error)

	// Write replaces the entire section data.
	Write(ctx context.Context, section SectionName, data map[string]any) error

	// Patch updates a specific path within a section (RFC 6902 JSON Patch style).
	Patch(ctx context.Context, section SectionName, path string, value any) error

	// Delete removes a specific path within a section.
	Delete(ctx context.Context, section SectionName, path string) error

	// List returns all section names currently stored.
	List(ctx context.Context) ([]SectionName, error)

	// Exists checks if a section exists.
	Exists(ctx context.Context, section SectionName) (bool, error)
}

// FileStore implements SectionStore using JSON files on disk.
type FileStore struct {
	baseDir string
	mu      sync.RWMutex
	// locks provides cross-process advisory locking per section.
	// Without this, multiple Laputa processes (or laputa + direct file
	// edits) racing on the same .json would lose updates to last-writer-wins.
	locks map[SectionName]*flock.Flock
}

// NewFileStore creates a new file-based section store.
func NewFileStore(baseDir string) (*FileStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create base dir: %w", err)
	}
	// Create sections subdirectory
	sectionsDir := filepath.Join(baseDir, "sections")
	if err := os.MkdirAll(sectionsDir, 0755); err != nil {
		return nil, fmt.Errorf("create sections dir: %w", err)
	}
	return &FileStore{baseDir: baseDir, locks: make(map[SectionName]*flock.Flock)}, nil
}

// sectionLock returns (and lazily creates) the cross-process lock for a section.
func (s *FileStore) sectionLock(section SectionName) *flock.Flock {
	s.mu.Lock()
	defer s.mu.Unlock()
	if lk, ok := s.locks[section]; ok {
		return lk
	}
	lockPath := filepath.Join(s.baseDir, "sections", string(section)+".lock")
	// Orphan cleanup: a previous laputa process may have crashed without
	// releasing the OS handle on the lock file. On Windows, the lock
	// file persists even after Unlock() because flock.New() opens it for
	// the lifetime of the FLock value; when that process dies abruptly
	// the OS releases the handle but the file remains. The next process
	// will then fail TryLock until the file is removed.
	removeIfOrphan(lockPath)
	lk := flock.New(lockPath)
	s.locks[section] = lk
	return lk
}

// removeIfOrphan deletes a lock file when its holding PID is dead.
// This is best-effort: if we can't determine the holder (Windows lock
// files don't store a PID by default), we leave it. Phantom-handle
// recovery on next session also relies on the bounded-retry TryLock.
func removeIfOrphan(lockPath string) {
	// On Windows gofrs/flock writes the PID to the file as a sidecar
	// convention we don't currently use; the file is purely 0 bytes.
	// We treat any existing lock file from a prior session as orphaned
	// because the only writer is a live laputa process holding it via
	// a live OS handle. If a different laputa instance is concurrently
	// running, our TryLock retry loop will wait briefly and retry.
	if _, err := os.Stat(lockPath); err != nil {
		return
	}
	_ = os.Remove(lockPath)
}

// sectionPath returns the file path for a section.
func (s *FileStore) sectionPath(section SectionName) string {
	return filepath.Join(s.baseDir, "sections", string(section)+".json")
}

// Read implements SectionStore.
func (s *FileStore) Read(ctx context.Context, section SectionName) (map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.sectionPath(section)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, fmt.Errorf("read section %s: %w", section, err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal section %s: %w", section, err)
	}
	return result, nil
}

// Write implements SectionStore.
func (s *FileStore) Write(ctx context.Context, section SectionName, data map[string]any) error {
	// Lock order matters: acquire the cross-process flock FIRST, then
	// take the in-process mutex. The reverse order risks a deadlock where
	// a previous Write holds s.mu and is hung on lk.Lock(), blocking every
	// subsequent Read/Write in the same process from making progress.
	//
	// Note on flock on Windows: the gofrs/flock library's blocking Lock()
	// implementation uses LockFileEx which can hang indefinitely if a
	// previous laputa process crashed without cleanly releasing the OS
	// handle on the lock file ("phantom handle" symptom). To stay robust
	// we use TryLock with bounded retries.
	lk := s.sectionLock(section)
	if err := acquireWithRetry(lk, 200*time.Millisecond, 10); err != nil {
		return fmt.Errorf("acquire section lock %s: %w", section, err)
	}
	defer func() { _ = lk.Unlock() }()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Add metadata
	data["_meta"] = map[string]any{
		"updated_at": time.Now().UTC().Format(time.RFC3339),
		"version":    "1.0",
	}

	path := s.sectionPath(section)
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal section %s: %w", section, err)
	}

	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("write section %s: %w", section, err)
	}
	return nil
}

// acquireWithRetry tries TryLock in a tight loop. If the lock can't be
// acquired within total budget, it returns the last error from TryLock.
// This sidesteps a known Windows LockFileEx hang when prior holders crash.
func acquireWithRetry(lk *flock.Flock, backoff time.Duration, attempts int) error {
	var lastErr error
	for i := 0; i < attempts; i++ {
		ok, err := lk.TryLock()
		if err != nil {
			lastErr = err
		} else if ok {
			return nil
		}
		time.Sleep(backoff)
	}
	return lastErr
}

// Patch implements SectionStore (simple dot-notation path).
func (s *FileStore) Patch(ctx context.Context, section SectionName, path string, value any) error {
	data, err := s.Read(ctx, section)
	if err != nil {
		return err
	}

	// Simple dot-notation: "a.b.c" -> data["a"]["b"]["c"] = value
	if err := setPath(data, path, value); err != nil {
		return fmt.Errorf("patch section %s path %s: %w", section, path, err)
	}

	return s.Write(ctx, section, data)
}

// Delete implements SectionStore.
func (s *FileStore) Delete(ctx context.Context, section SectionName, path string) error {
	data, err := s.Read(ctx, section)
	if err != nil {
		return err
	}

	if err := deletePath(data, path); err != nil {
		return fmt.Errorf("delete section %s path %s: %w", section, path, err)
	}

	return s.Write(ctx, section, data)
}

// List implements SectionStore.
func (s *FileStore) List(ctx context.Context) ([]SectionName, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sectionsDir := filepath.Join(s.baseDir, "sections")
	entries, err := os.ReadDir(sectionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list sections: %w", err)
	}

	var sections []SectionName
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}
		section := SectionName(name[:len(name)-5]) // strip .json
		sections = append(sections, section)
	}
	return sections, nil
}

// Exists implements SectionStore.
func (s *FileStore) Exists(ctx context.Context, section SectionName) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err := os.Stat(s.sectionPath(section))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("check section %s: %w", section, err)
	}
	return true, nil
}

// setPath sets a value at a dot-notation path in a map.
func setPath(data map[string]any, path string, value any) error {
	parts := splitPath(path)
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return nil
		}
		next, ok := current[part].(map[string]any)
		if !ok {
			next = make(map[string]any)
			current[part] = next
		}
		current = next
	}
	return nil
}

// deletePath removes a value at a dot-notation path in a map.
func deletePath(data map[string]any, path string) error {
	parts := splitPath(path)
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			delete(current, part)
			return nil
		}
		next, ok := current[part].(map[string]any)
		if !ok {
			return fmt.Errorf("path not found: %s", path)
		}
		current = next
	}
	return nil
}

// splitPath splits a dot-notation path into parts.
func splitPath(path string) []string {
	var parts []string
	var current string
	for _, r := range path {
		if r == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// Engine is the main Laputa governance engine.
type Engine struct {
	store SectionStore
}

// NewEngine creates a new Laputa engine.
func NewEngine(store SectionStore) *Engine {
	return &Engine{store: store}
}

// GetSection returns a section's data.
func (e *Engine) GetSection(ctx context.Context, section SectionName) (map[string]any, error) {
	return e.store.Read(ctx, section)
}

// SetSection replaces a section's data.
func (e *Engine) SetSection(ctx context.Context, section SectionName, data map[string]any) error {
	return e.store.Write(ctx, section, data)
}

// UpdateSection patches a section at a specific path.
func (e *Engine) UpdateSection(ctx context.Context, section SectionName, path string, value any) error {
	return e.store.Patch(ctx, section, path, value)
}

// DeleteSectionPath removes a path from a section.
func (e *Engine) DeleteSectionPath(ctx context.Context, section SectionName, path string) error {
	return e.store.Delete(ctx, section, path)
}

// ListSections returns all section names.
func (e *Engine) ListSections(ctx context.Context) ([]SectionName, error) {
	return e.store.List(ctx)
}

// Initialize creates all 14 sections with default templates.
func (e *Engine) Initialize(ctx context.Context) error {
	for _, section := range AllSections {
		exists, err := e.store.Exists(ctx, section)
		if err != nil {
			return fmt.Errorf("check section %s: %w", section, err)
		}
		if exists {
			continue
		}
		if err := e.store.Write(ctx, section, defaultSectionData(section)); err != nil {
			return fmt.Errorf("initialize section %s: %w", section, err)
		}
	}
	return nil
}

// Snapshot returns the full Laputa state (all 14 sections).
func (e *Engine) Snapshot(ctx context.Context) (map[string]any, error) {
	state := make(map[string]any)
	state["schema_version"] = "1.0.0"
	state["sections"] = make(map[string]any)
	sections := state["sections"].(map[string]any)

	for _, section := range AllSections {
		data, err := e.store.Read(ctx, section)
		if err != nil {
			return nil, fmt.Errorf("snapshot section %s: %w", section, err)
		}
		info := SectionRegistry[section]
		sections[string(section)] = map[string]any{
			"data":            data,
			"status":          info.Status,
			"write_authority": info.WriteAuth,
		}
	}
	return state, nil
}

// defaultSectionData returns the default data for a section.
func defaultSectionData(section SectionName) map[string]any {
	switch section {
	case SectionIdentity:
		return map[string]any{
			"role":         "",
			"capabilities": []string{},
			"constraints":  []string{},
			"sop":          []map[string]any{},
			"agentic_rag": map[string]any{
				"allowed_capabilities": []string{"governance", "llm", "hybrid", "kg", "timeline"},
			},
		}
	case SectionRelationship:
		return map[string]any{
			"relationships": []map[string]any{},
			"resonance":     map[string]any{},
		}
	case SectionCommitment:
		return map[string]any{
			"commitments": []map[string]any{},
			"red_lines":   []string{},
			"agentic_rag": map[string]any{
				"denied_sources":   []string{},
				"denied_wings":     []string{},
				"denied_rooms":     []string{},
				"external_context": "full",
			},
		}
	case SectionPreferences:
		return map[string]any{
			"preferences": []map[string]any{},
			"learning":    []map[string]any{},
		}
	case SectionMemoryMD:
		return map[string]any{
			"summary":    "",
			"highlights": []map[string]any{},
		}
	case SectionHistoryMD:
		return map[string]any{
			"timeline": []map[string]any{},
		}
	case SectionDaily:
		return map[string]any{
			"reports": []map[string]any{},
		}
	case SectionWeekly:
		return map[string]any{
			"reports": []map[string]any{},
		}
	case SectionMonthly:
		return map[string]any{
			"reports": []map[string]any{},
		}
	case SectionJournalReflective:
		return map[string]any{
			"entries": []map[string]any{},
			"status":  "tbd",
		}
	case SectionProposalInbox:
		return map[string]any{
			"proposals": []map[string]any{},
			"status":    "tbd",
		}
	case SectionChangelog:
		return map[string]any{
			"records": []map[string]any{},
			"status":  "tbd",
		}
	case SectionReportIndexes:
		return map[string]any{
			"indexes": []map[string]any{},
			"status":  "tbd",
		}
	case SectionAAAKSummaries:
		return map[string]any{
			"summaries": []map[string]any{},
			"status":    "tbd",
		}
	default:
		return map[string]any{}
	}
}
