package persona

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Service is the Laputa Persona authority over one profile-level persona/
// directory. Authority bodies are Markdown files; JSON is used only for the
// review queue and JSONL history metadata.
type Service struct {
	root  string
	mu    sync.Mutex
	clock func() time.Time
}

// Open prepares the Persona root and removes only an abandoned initialization
// staging directory. Staging is never read as authority.
func Open(configDir string) (*Service, error) {
	root := filepath.Join(configDir, "persona")
	for _, dir := range []string{root, filepath.Join(root, "history"), filepath.Join(root, "requests")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, storageError("cannot create persona directory", err)
		}
	}
	staging := filepath.Join(root, ".init-staging")
	if _, err := os.Stat(staging); err == nil {
		if err := os.RemoveAll(staging); err != nil {
			return nil, storageError("cannot clean initialization staging", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, storageError("cannot inspect initialization staging", err)
	}
	return &Service{root: root}, nil
}

// Root returns the Persona authority root.
func (s *Service) Root() string { return s.root }

// SetClock installs a deterministic clock for tests. Production callers leave
// it nil and use UTC wall-clock time.
func (s *Service) SetClock(clock func() time.Time) { s.clock = clock }

func (s *Service) now() time.Time {
	if s.clock != nil {
		return s.clock().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) documentPath(kind Kind) string {
	return filepath.Join(s.root, kind.FileName())
}

func (s *Service) historyDir(kind Kind) string {
	return filepath.Join(s.root, "history", kind.FileName())
}

func (s *Service) requestPath(id string) string {
	return filepath.Join(s.root, "requests", id+".json")
}

// Status computes the profile-level setup state for all seven files.
func (s *Service) Status() (*StatusView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statusLocked()
}

func (s *Service) statusLocked() (*StatusView, error) {
	files := make(map[string]FileState, len(AllKinds))
	for _, kind := range AllKinds {
		state, err := s.fileStateLocked(kind)
		if err != nil {
			return nil, err
		}
		files[kind.String()] = state
	}
	status := StatusReady
	allAbsent := true
	for _, kind := range RequiredKinds {
		state := files[kind.String()]
		if state.Exists {
			allAbsent = false
		}
		if !state.Exists || !state.Valid {
			status = StatusIncomplete
		}
	}
	if allAbsent {
		status = StatusUninitialized
	}
	return &StatusView{Status: status, Files: files}, nil
}

func (s *Service) fileStateLocked(kind Kind) (FileState, error) {
	path := s.documentPath(kind)
	state := FileState{
		Kind:         kind,
		FileName:     kind.FileName(),
		Exists:       fileExists(path),
		ContentLimit: kind.ContentLimit(),
		Required:     kind.RequiredForReady(),
		ToolOnly:     kind == KindWorld,
	}
	if frozen, ok := kind.FrozenLimit(); ok {
		state.FrozenLimit = &frozen
	}
	pending, err := s.pendingCountLocked(kind)
	if err != nil {
		return FileState{}, err
	}
	state.PendingCount = pending
	if !state.Exists {
		state.Valid = !kind.RequiredForReady()
		if kind.RequiredForReady() {
			reason := "missing"
			state.Reason = &reason
		}
		return state, nil
	}
	doc, err := s.readDocumentUncheckedLocked(kind)
	if err != nil {
		var invalid *InvalidContentError
		if errors.As(err, &invalid) {
			state.Reason = &invalid.Reason
			state.Revision, err = s.latestRevisionLocked(kind)
			if err != nil {
				return FileState{}, err
			}
			return state, nil
		}
		return FileState{}, err
	}
	state.Revision = doc.Revision
	state.UpdatedAt = doc.UpdatedAt
	if doc.Content == "" {
		state.Reason = stringPtr("empty")
		return state, nil
	}
	history, err := s.listHistoryLocked(kind)
	if err != nil {
		return FileState{}, err
	}
	if len(history) > 0 && history[0].Revision == doc.Revision && history[0].ContentHash == doc.ContentHash {
		state.Valid = true
		return state, nil
	}
	state.Reason = stringPtr("history_mismatch")
	return state, nil
}

// GetDocument returns a full, explicit read of one authority file. WORLD is
// intentionally available here only as an explicit document operation.
func (s *Service) GetDocument(kind Kind) (*Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	view, err := s.statusLocked()
	if err != nil {
		return nil, err
	}
	if view.Status == StatusUninitialized {
		return nil, ErrUninitialized
	}
	return s.readDocumentCheckedLocked(kind)
}

func (s *Service) readDocumentCheckedLocked(kind Kind) (*Document, error) {
	doc, err := s.readDocumentUncheckedLocked(kind)
	if err != nil {
		return nil, err
	}
	doc.PendingCount, err = s.pendingCountLocked(kind)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *Service) readDocumentUncheckedLocked(kind Kind) (*Document, error) {
	if kind.String() == "unknown" {
		return nil, ErrKindForbidden
	}
	path := s.documentPath(kind)
	if !fileExists(path) {
		if kind.RequiredForReady() {
			return nil, &InvalidContentError{Kind: kind, Reason: "missing"}
		}
		return &Document{Kind: kind, FileName: kind.FileName(), ContentHash: ContentHash("")}, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, storageError("cannot read document", err)
	}
	if !utf8.Valid(raw) {
		return nil, &InvalidContentError{Kind: kind, Reason: "invalid_utf8"}
	}
	content := NormalizeMarkdown(string(raw))
	if err := validateContent(kind, content, kind.RequiredForReady()); err != nil {
		return nil, err
	}
	entries, err := s.listHistoryLocked(kind)
	if err != nil {
		return nil, err
	}
	revision := uint64(0)
	if len(entries) > 0 {
		revision = entries[0].Revision
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, storageError("cannot stat document", err)
	}
	updated := info.ModTime().UTC()
	return &Document{Kind: kind, FileName: kind.FileName(), Exists: true, Valid: true, Content: content, Revision: revision, ContentHash: ContentHash(content), UpdatedAt: &updated}, nil
}

// ListHistory returns immutable revision metadata newest-first.
func (s *Service) ListHistory(kind Kind) ([]HistoryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listHistoryLocked(kind)
}

// ReadHistory returns one immutable revision with its content and diff.
func (s *Service) ReadHistory(kind Kind, revision uint64) (*HistoryRevision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.listHistoryLocked(kind)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.Revision != revision {
			continue
		}
		content, err := os.ReadFile(filepath.Join(s.historyDir(kind), entry.Snapshot))
		if err != nil {
			return nil, storageError("cannot read history snapshot", err)
		}
		diff, err := os.ReadFile(filepath.Join(s.historyDir(kind), entry.Diff))
		if err != nil {
			return nil, storageError("cannot read history diff", err)
		}
		return &HistoryRevision{HistoryEntry: entry, Content: string(content), UnifiedDiff: string(diff)}, nil
	}
	return nil, &HistoryNotFoundError{Kind: kind, Revision: revision}
}

func (s *Service) listHistoryLocked(kind Kind) ([]HistoryEntry, error) {
	if kind.String() == "unknown" {
		return nil, ErrKindForbidden
	}
	path := filepath.Join(s.historyDir(kind), "log.jsonl")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, storageError("cannot read history log", err)
	}
	var entries []HistoryEntry
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry HistoryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, storageError("corrupted history line", err)
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Revision > entries[j].Revision })
	return entries, nil
}

func (s *Service) latestRevisionLocked(kind Kind) (uint64, error) {
	entries, err := s.listHistoryLocked(kind)
	if err != nil {
		return 0, err
	}
	if len(entries) == 0 {
		return 0, nil
	}
	return entries[0].Revision, nil
}

func (s *Service) pendingCountLocked(kind Kind) (int, error) {
	requests, err := s.listRequestsLocked(nil)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, req := range requests {
		if req.Kind == kind && req.State == RequestPending {
			count++
		}
	}
	return count, nil
}

// ListRequests is the service-level spelling for review listing.
func (s *Service) ListRequests(kind *Kind) ([]ChangeRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listRequestsLocked(kind)
}

func (s *Service) listRequestsLocked(kind *Kind) ([]ChangeRequest, error) {
	dir := filepath.Join(s.root, "requests")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, storageError("cannot list reviews", err)
	}
	var requests []ChangeRequest
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, storageError("cannot read review", err)
		}
		var req ChangeRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, storageError("corrupted review entry", err)
		}
		if kind == nil || *kind == req.Kind {
			requests = append(requests, req)
		}
	}
	sort.Slice(requests, func(i, j int) bool {
		if requests[i].CreatedAt.Equal(requests[j].CreatedAt) {
			return requests[i].ID > requests[j].ID
		}
		return requests[i].CreatedAt.After(requests[j].CreatedAt)
	})
	return requests, nil
}

func validateContent(kind Kind, content string, requireNonempty bool) error {
	if kind.String() == "unknown" {
		return ErrKindForbidden
	}
	if !utf8.ValidString(content) {
		return &InvalidContentError{Kind: kind, Reason: "invalid_utf8"}
	}
	normalized := NormalizeMarkdown(content)
	if requireNonempty && normalized == "" {
		return &InvalidContentError{Kind: kind, Reason: "empty"}
	}
	if VisibleLen(normalized) > kind.ContentLimit() {
		return &CapExceededError{Kind: kind, Limit: kind.ContentLimit()}
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// Initialize validates the complete input before touching live state, then
// installs five staged documents and their first history revision under a
// guarded same-volume operation.
func (s *Service) Initialize(init Initialization, actor string, source WriteSource, reason string) (*WriteOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	view, err := s.statusLocked()
	if err != nil {
		return nil, err
	}
	switch view.Status {
	case StatusReady:
		return nil, ErrAlreadyInitialized
	case StatusIncomplete:
		return nil, ErrIncomplete
	}
	if s.hasRequiredHistoryArtifactsLocked() {
		return nil, ErrIncomplete
	}
	if actor == "" {
		actor = "user"
	}
	if source == "" {
		source = SourceInit
	}
	inputs := init.Map()
	contents := make(map[Kind]string, len(RequiredKinds))
	for _, kind := range RequiredKinds {
		content := NormalizeMarkdown(inputs[kind])
		if kind == KindUser {
			content = wrapUserSection(content)
		}
		if err := validateContent(kind, content, true); err != nil {
			return nil, err
		}
		contents[kind] = content
	}

	staging := filepath.Join(s.root, ".init-staging")
	if err := os.RemoveAll(staging); err != nil {
		return nil, storageError("cannot reset initialization staging", err)
	}
	if err := os.MkdirAll(staging, 0o700); err != nil {
		return nil, storageError("cannot create initialization staging", err)
	}
	now := s.now()
	for _, kind := range RequiredKinds {
		if err := writeInitialArtifacts(staging, kind, contents[kind], actor, source, reason, now); err != nil {
			_ = os.RemoveAll(staging)
			return nil, err
		}
	}

	installedDocs := make([]string, 0, len(RequiredKinds))
	installedHistory := make([]string, 0, len(RequiredKinds))
	rollback := func() {
		for _, path := range installedDocs {
			_ = os.Remove(path)
		}
		for _, path := range installedHistory {
			_ = os.RemoveAll(path)
		}
		_ = os.RemoveAll(staging)
	}
	for _, kind := range RequiredKinds {
		if fileExists(s.documentPath(kind)) {
			rollback()
			return nil, ErrIncomplete
		}
		if _, err := os.Stat(s.historyDir(kind)); err == nil {
			rollback()
			return nil, ErrIncomplete
		} else if !os.IsNotExist(err) {
			rollback()
			return nil, storageError("cannot inspect history path", err)
		}
	}
	for _, kind := range RequiredKinds {
		historyPath := s.historyDir(kind)
		if err := os.Rename(filepath.Join(staging, "history", kind.FileName()), historyPath); err != nil {
			rollback()
			return nil, storageError("cannot install initial history", err)
		}
		installedHistory = append(installedHistory, historyPath)
	}
	for _, kind := range RequiredKinds {
		docPath := s.documentPath(kind)
		if err := os.Rename(filepath.Join(staging, kind.FileName()), docPath); err != nil {
			rollback()
			return nil, storageError("cannot install initial document", err)
		}
		installedDocs = append(installedDocs, docPath)
	}
	if err := os.RemoveAll(staging); err != nil {
		return nil, storageError("cannot remove initialization staging", err)
	}
	return &WriteOutcome{Changed: true}, nil
}

// Repair reconstructs only owner-selected invalid/missing required artifacts.
func (s *Service) Repair(input RepairInput) (*StatusView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	view, err := s.statusLocked()
	if err != nil {
		return nil, err
	}
	if view.Status != StatusIncomplete || len(input.Documents) == 0 {
		return nil, ErrRepairNotRequired
	}
	contents := make(map[Kind]string, len(input.Documents))
	for kind, body := range input.Documents {
		if !kind.RequiredForReady() {
			return nil, ErrKindForbidden
		}
		state, ok := view.Files[kind.String()]
		if !ok || state.Valid {
			return nil, ErrRepairNotRequired
		}
		content := NormalizeMarkdown(body)
		if kind == KindUser {
			content = wrapUserSection(content)
		}
		if err := validateContent(kind, content, true); err != nil {
			return nil, err
		}
		contents[kind] = content
	}
	// Repair may touch several authority documents and stale pending review
	// records. Capture the complete Persona tree so an I/O failure in any
	// later document restores the exact pre-repair state.
	backup, err := snapshotPersonaTree(s.root)
	if err != nil {
		return nil, storageError("cannot snapshot Persona before repair", err)
	}
	for _, kind := range RequiredKinds {
		content, ok := contents[kind]
		if !ok {
			continue
		}
		base, err := s.latestRevisionLocked(kind)
		if err != nil {
			return nil, err
		}
		if _, err := s.writeDocumentCoreLocked(kind, content, base, "user", SourceInit, input.Reason, true, ""); err != nil {
			_ = restorePersonaTree(s.root, backup)
			return nil, err
		}
	}
	return s.statusLocked()
}

// Write is the user-direct service operation. Existing revision zero never
// acts as a wildcard; only a genuinely missing optional document can start at
// revision zero.
func (s *Service) Write(kind Kind, content string, baseRevision uint64, actor string, source WriteSource, reason string) (*WriteOutcome, error) {
	return s.SaveUserDocument(kind, content, baseRevision, actor, source, reason)
}

// SaveUserDocument applies a complete user-authored document.
func (s *Service) SaveUserDocument(kind Kind, content string, baseRevision uint64, actor string, source WriteSource, reason string) (*WriteOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireReadyLocked(); err != nil {
		return nil, err
	}
	if actor == "" {
		actor = "user"
	}
	if source == "" {
		source = SourceUserDirect
	}
	if source != SourceUserDirect && source != SourceHistoryResave {
		return nil, ErrKindForbidden
	}
	if source == SourceUserDirect && s.contentExistsInHistoryLocked(kind, content) {
		source = SourceHistoryResave
	}
	return s.writeDocumentCoreLocked(kind, content, baseRevision, actor, source, reason, false, "")
}

// SaveUserObservations updates only USER's Observations section.
func (s *Service) SaveUserObservations(content string, baseRevision uint64, reason string) (*WriteOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireReadyLocked(); err != nil {
		return nil, err
	}
	current, err := s.readDocumentUncheckedLocked(KindUser)
	if err != nil {
		return nil, err
	}
	proposed := ReplaceMarkdownSection(current.Content, "Observations", content)
	return s.writeDocumentCoreLocked(KindUser, proposed, baseRevision, "user", SourceUserDirect, reason, false, "")
}

// SaveAgentP16 is the only direct agent write operation: DREAM, DARK, or
// USER's Observations section.
func (s *Service) SaveAgentP16(kind Kind, content, reason string) (*WriteOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireReadyLocked(); err != nil {
		return nil, err
	}
	current, err := s.readDocumentUncheckedLocked(kind)
	if err != nil {
		return nil, err
	}
	var proposed string
	switch kind {
	case KindDream, KindDark:
		proposed = content
	case KindUser:
		proposed = ReplaceMarkdownSection(current.Content, "Observations", content)
	default:
		return nil, ErrKindForbidden
	}
	return s.writeDocumentCoreLocked(kind, proposed, current.Revision, "agent", SourceAgentP16, reason, false, "")
}

// SaveAgentP16WithRevision is the HTTP-facing form that enforces the caller's
// exact base revision at the commit boundary.
func (s *Service) SaveAgentP16WithRevision(kind Kind, content string, baseRevision uint64, reason string) (*WriteOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireReadyLocked(); err != nil {
		return nil, err
	}
	current, err := s.readDocumentUncheckedLocked(kind)
	if err != nil {
		return nil, err
	}
	var proposed string
	switch kind {
	case KindDream, KindDark:
		proposed = content
	case KindUser:
		proposed = ReplaceMarkdownSection(current.Content, "Observations", content)
	default:
		return nil, ErrKindForbidden
	}
	return s.writeDocumentCoreLocked(kind, proposed, baseRevision, "agent", SourceAgentP16, reason, false, "")
}

// CreateRequest persists a protected Persona proposal. The caller must supply
// a trusted actor; HTTP adapters derive it from capability claims.
func (s *Service) CreateRequest(req ChangeRequest) (*ChangeRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireReadyLocked(); err != nil {
		return nil, err
	}
	if !requestKindAllowed(req.Actor, req.Kind) {
		return nil, ErrKindForbidden
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" {
		return nil, &TypedError{Code: "persona_invalid_content", Message: "review reason is required"}
	}
	if req.ID == "" {
		req.ID = newRequestID()
	}
	if !validRequestID(req.ID) {
		return nil, ErrKindForbidden
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = s.now()
	} else {
		req.CreatedAt = req.CreatedAt.UTC()
	}
	req.State = RequestPending
	current, err := s.readDocumentUncheckedLocked(req.Kind)
	if err != nil {
		return nil, err
	}
	if current.Revision != req.BaseRevision || current.ContentHash != req.BaseHash {
		return nil, &RevisionConflictError{Expected: req.BaseRevision, Current: current.Revision}
	}
	requests, err := s.listRequestsLocked(&req.Kind)
	if err != nil {
		return nil, err
	}
	for _, existing := range requests {
		if existing.State == RequestPending {
			return nil, &RequestExistsError{Kind: req.Kind}
		}
	}
	req.ProposedMarkdown = NormalizeMarkdown(req.ProposedMarkdown)
	if err := validateContent(req.Kind, req.ProposedMarkdown, true); err != nil {
		return nil, err
	}
	if err := validateRequestScope(req.Kind, req.Actor, current.Content, req.ProposedMarkdown); err != nil {
		return nil, err
	}
	if err := s.saveRequestLocked(req); err != nil {
		return nil, err
	}
	return &req, nil
}

// CreateReview is the clean-break spelling for CreateRequest.
func (s *Service) CreateReview(input PersonaReviewCreate) (*ChangeRequest, error) {
	return s.CreateRequest(ChangeRequest{Kind: input.Kind, BaseRevision: input.BaseRevision, BaseHash: input.BaseHash, ProposedMarkdown: input.ProposedMarkdown, Actor: input.Actor, Reason: input.Reason})
}

// ListReviews provides domain-side filtering for the clean-break adapter.
func (s *Service) ListReviews(kind *Kind, state *RequestState, limit int) ([]ChangeRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.listRequestsLocked(kind)
	if err != nil {
		return nil, err
	}
	if state != nil {
		filtered := items[:0]
		for _, item := range items {
			if item.State == *state {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *Service) GetReview(id string) (*ChangeRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadRequestLocked(id)
}

// AcceptRequest applies a pending proposal and preserves its actor/source/
// reason in history. A drifted base is persisted as stale before returning.
func (s *Service) AcceptRequest(id string) (*WriteOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireReadyLocked(); err != nil {
		return nil, err
	}
	req, err := s.loadRequestLocked(id)
	if err != nil {
		return nil, err
	}
	if req.State != RequestPending {
		return nil, &RequestStaleError{ID: id}
	}
	current, err := s.readDocumentUncheckedLocked(req.Kind)
	if err != nil {
		return nil, err
	}
	if current.Revision != req.BaseRevision || current.ContentHash != req.BaseHash {
		req.State = RequestStale
		now := s.now()
		req.DecidedAt = &now
		if saveErr := s.saveRequestLocked(*req); saveErr != nil {
			return nil, saveErr
		}
		return nil, &RequestStaleError{ID: id}
	}
	if err := validateRequestScope(req.Kind, req.Actor, current.Content, req.ProposedMarkdown); err != nil {
		return nil, err
	}
	source := SourceAgentP5Accepted
	actor := "agent"
	if req.Actor == ActorAutodream {
		source = SourceAutodreamP5
		actor = "autodream"
	}
	out, err := s.writeDocumentCoreLocked(req.Kind, req.ProposedMarkdown, req.BaseRevision, actor, source, req.Reason, false, id)
	if err != nil {
		return nil, err
	}
	req.State = RequestAccepted
	now := s.now()
	req.DecidedAt = &now
	if err := s.saveRequestLocked(*req); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) ApproveRequest(id string) (*WriteOutcome, error) { return s.AcceptRequest(id) }

// RejectRequest marks a pending proposal rejected without changing Persona.
func (s *Service) RejectRequest(id string) (*WriteOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireReadyLocked(); err != nil {
		return nil, err
	}
	req, err := s.loadRequestLocked(id)
	if err != nil {
		return nil, err
	}
	if req.State != RequestPending {
		return nil, &RequestStaleError{ID: id}
	}
	req.State = RequestRejected
	now := s.now()
	req.DecidedAt = &now
	if err := s.saveRequestLocked(*req); err != nil {
		return nil, err
	}
	return &WriteOutcome{Changed: false}, nil
}

func (s *Service) loadRequestLocked(id string) (*ChangeRequest, error) {
	if !validRequestID(id) {
		return nil, &RequestNotFoundError{ID: id}
	}
	raw, err := os.ReadFile(s.requestPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &RequestNotFoundError{ID: id}
		}
		return nil, storageError("cannot read review", err)
	}
	var req ChangeRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, storageError("corrupted review entry", err)
	}
	return &req, nil
}

func (s *Service) saveRequestLocked(req ChangeRequest) error {
	if !validRequestID(req.ID) {
		return ErrKindForbidden
	}
	raw, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return storageError("cannot marshal review", err)
	}
	if err := writeAtomicBytes(s.requestPath(req.ID), raw); err != nil {
		return storageError("cannot write review", err)
	}
	return nil
}

func (s *Service) requireReadyLocked() error {
	view, err := s.statusLocked()
	if err != nil {
		return err
	}
	switch view.Status {
	case StatusUninitialized:
		return ErrUninitialized
	case StatusIncomplete:
		return ErrIncomplete
	default:
		return nil
	}
}

type fileBackup struct {
	path   string
	data   []byte
	exists bool
}

func (s *Service) writeDocumentCoreLocked(kind Kind, content string, baseRevision uint64, actor string, source WriteSource, reason string, force bool, acceptingRequest string) (*WriteOutcome, error) {
	normalized := NormalizeMarkdown(content)
	if err := validateContent(kind, normalized, true); err != nil {
		return nil, err
	}
	current, err := s.currentDocumentForWriteLocked(kind, force)
	if err != nil {
		return nil, err
	}
	if current.Revision != baseRevision {
		return nil, &RevisionConflictError{Expected: baseRevision, Current: current.Revision}
	}
	if !force && current.Exists && current.ContentHash == ContentHash(normalized) {
		current.PendingCount, _ = s.pendingCountLocked(kind)
		return &WriteOutcome{Document: *current, Changed: false}, nil
	}
	backups, err := s.stalePendingLocked(kind, acceptingRequest)
	if err != nil {
		return nil, err
	}
	if _, err := s.persistRevisionLocked(kind, current, normalized, baseRevision, actor, source, reason); err != nil {
		_ = restoreBackups(backups)
		return nil, err
	}
	doc, err := s.readDocumentUncheckedLocked(kind)
	if err != nil {
		_ = restoreBackups(backups)
		return nil, err
	}
	doc.PendingCount, err = s.pendingCountLocked(kind)
	if err != nil {
		return nil, err
	}
	return &WriteOutcome{Document: *doc, Changed: true}, nil
}

func (s *Service) currentDocumentForWriteLocked(kind Kind, force bool) (*Document, error) {
	doc, err := s.readDocumentUncheckedLocked(kind)
	if err == nil {
		return doc, nil
	}
	var invalid *InvalidContentError
	if !force || !errors.As(err, &invalid) {
		return nil, err
	}
	revision, revErr := s.latestRevisionLocked(kind)
	if revErr != nil {
		return nil, revErr
	}
	if invalid.Reason == "missing" {
		return &Document{Kind: kind, FileName: kind.FileName(), Revision: revision, ContentHash: ContentHash("")}, nil
	}
	return &Document{Kind: kind, FileName: kind.FileName(), Exists: fileExists(s.documentPath(kind)), Revision: revision, ContentHash: ContentHash("")}, nil
}

// persistRevisionLocked creates immutable history, appends and syncs its
// metadata, and atomically replaces the live document. Every failure rolls
// back artifacts created by this operation.
func (s *Service) persistRevisionLocked(kind Kind, current *Document, content string, baseRevision uint64, actor string, source WriteSource, reason string) (*Document, error) {
	revision := current.Revision + 1
	dir := s.historyDir(kind)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, storageError("cannot create history directory", err)
	}
	entry := HistoryEntry{Revision: revision, ContentHash: ContentHash(content), Snapshot: fmt.Sprintf("%d.md", revision), Diff: fmt.Sprintf("%d.diff", revision), Actor: actor, Source: string(source), Reason: strings.TrimSpace(reason), BaseRevision: baseRevision, CreatedAt: s.now()}
	snapshotPath := filepath.Join(dir, entry.Snapshot)
	diffPath := filepath.Join(dir, entry.Diff)
	if err := writeImmutable(snapshotPath, []byte(content)); err != nil {
		return nil, storageError("cannot create history snapshot", err)
	}
	cleanupHistory := func() {
		_ = os.Remove(snapshotPath)
		_ = os.Remove(diffPath)
	}
	if err := writeImmutable(diffPath, []byte(UnifiedDiff(current.Content, content, kind.FileName()))); err != nil {
		cleanupHistory()
		return nil, storageError("cannot create history diff", err)
	}
	logPath := filepath.Join(dir, "log.jsonl")
	oldLog, logExisted, err := readOptionalFile(logPath)
	if err != nil {
		cleanupHistory()
		return nil, storageError("cannot read history log", err)
	}
	line, err := json.Marshal(entry)
	if err != nil {
		cleanupHistory()
		return nil, storageError("cannot marshal history entry", err)
	}
	if err := appendSynced(logPath, append(line, '\n')); err != nil {
		cleanupHistory()
		return nil, storageError("cannot append history log", err)
	}
	rollbackLog := func() {
		_ = restoreOptionalFile(logPath, oldLog, logExisted)
		cleanupHistory()
	}
	if err := writeAtomicBytes(s.documentPath(kind), []byte(content)); err != nil {
		rollbackLog()
		return nil, storageError("cannot install Persona document", err)
	}
	return &Document{Kind: kind, FileName: kind.FileName(), Exists: true, Valid: true, Content: content, Revision: revision, ContentHash: entry.ContentHash}, nil
}

func (s *Service) stalePendingLocked(kind Kind, exceptID string) ([]fileBackup, error) {
	requests, err := s.listRequestsLocked(&kind)
	if err != nil {
		return nil, err
	}
	var backups []fileBackup
	rollback := func() { _ = restoreBackups(backups) }
	for _, req := range requests {
		if req.ID == exceptID || req.State != RequestPending {
			continue
		}
		path := s.requestPath(req.ID)
		old, existed, readErr := readOptionalFile(path)
		if readErr != nil {
			rollback()
			return nil, storageError("cannot read pending review", readErr)
		}
		backups = append(backups, fileBackup{path: path, data: old, exists: existed})
		req.State = RequestStale
		now := s.now()
		req.DecidedAt = &now
		if err := s.saveRequestLocked(req); err != nil {
			rollback()
			return nil, err
		}
	}
	return backups, nil
}

func (s *Service) contentExistsInHistoryLocked(kind Kind, content string) bool {
	normalized := NormalizeMarkdown(content)
	entries, err := s.listHistoryLocked(kind)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		raw, err := os.ReadFile(filepath.Join(s.historyDir(kind), entry.Snapshot))
		if err == nil && NormalizeMarkdown(string(raw)) == normalized {
			return true
		}
	}
	return false
}

func (s *Service) hasRequiredHistoryArtifactsLocked() bool {
	for _, kind := range RequiredKinds {
		entries, err := os.ReadDir(s.historyDir(kind))
		if err == nil && len(entries) > 0 {
			return true
		}
	}
	return false
}

func writeInitialArtifacts(root string, kind Kind, content, actor string, source WriteSource, reason string, now time.Time) error {
	historyDir := filepath.Join(root, "history", kind.FileName())
	if err := os.MkdirAll(historyDir, 0o700); err != nil {
		return storageError("cannot create staged history directory", err)
	}
	if err := writeImmutable(filepath.Join(root, kind.FileName()), []byte(content)); err != nil {
		return storageError("cannot stage Persona document", err)
	}
	entry := HistoryEntry{Revision: 1, ContentHash: ContentHash(content), Snapshot: "1.md", Diff: "1.diff", Actor: actor, Source: string(source), Reason: strings.TrimSpace(reason), BaseRevision: 0, CreatedAt: now}
	if err := writeImmutable(filepath.Join(historyDir, entry.Snapshot), []byte(content)); err != nil {
		return storageError("cannot stage initial snapshot", err)
	}
	if err := writeImmutable(filepath.Join(historyDir, entry.Diff), []byte(UnifiedDiff("", content, kind.FileName()))); err != nil {
		return storageError("cannot stage initial diff", err)
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return storageError("cannot marshal initial history", err)
	}
	if err := writeImmutable(filepath.Join(historyDir, "log.jsonl"), append(line, '\n')); err != nil {
		return storageError("cannot stage initial history log", err)
	}
	return nil
}

func validateRequestScope(kind Kind, actor RequestActor, current, proposed string) error {
	if kind == KindUser {
		if actor == ActorAgent && mustSection(current, "Observations") != mustSection(proposed, "Observations") {
			return ErrKindForbidden
		}
		if actor == ActorAutodream && userPreferences(current) != userPreferences(proposed) {
			return ErrKindForbidden
		}
	}
	if kind == KindWorld && !worldUserContentPreserved(current, proposed) {
		return &WorldProtectedError{}
	}
	if kind == KindWorld && !worldEntryGateAllows(actor, current, proposed) {
		return &WorldEntryGateError{}
	}
	return nil
}

func mustSection(content, heading string) string {
	section, ok := ExtractMarkdownSection(content, heading)
	if !ok {
		return ""
	}
	return section
}

func requestKindAllowed(actor RequestActor, kind Kind) bool {
	switch actor {
	case ActorAgent:
		return kind == KindIdentity || kind == KindRelationship || kind == KindRedline || kind == KindUser || kind == KindWorld
	case ActorAutodream:
		return kind == KindIdentity || kind == KindRelationship || kind == KindUser || kind == KindWorld || kind == KindDark
	default:
		return false
	}
}

func validRequestID(id string) bool {
	return id != "" && len(id) <= 200 && id != "." && id != ".." && !strings.ContainsAny(id, `/\\`)
}

func storageError(message string, err error) error {
	return &TypedError{Code: "persona_storage_error", Message: message, Err: err}
}

func readOptionalFile(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return data, err == nil, err
}

func appendSynced(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func writeImmutable(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	ok = true
	return nil
}

func writeAtomicBytes(path string, data []byte) error {
	tmp := fmt.Sprintf("%s.tmp-%s", path, newRequestID())
	if err := writeFileSync(tmp, data); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func writeFileSync(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func restoreOptionalFile(path string, data []byte, existed bool) error {
	if !existed {
		return os.Remove(path)
	}
	return writeAtomicBytes(path, data)
}

func restoreBackups(backups []fileBackup) error {
	var first error
	for i := len(backups) - 1; i >= 0; i-- {
		backup := backups[i]
		if err := restoreOptionalFile(backup.path, backup.data, backup.exists); err != nil && !os.IsNotExist(err) && first == nil {
			first = err
		}
	}
	return first
}

// snapshotPersonaTree records all files in the authority root. It is used
// only around the bounded multi-file repair operation, not as a second
// authority or recovery log.
func snapshotPersonaTree(root string) ([]fileBackup, error) {
	var backups []fileBackup
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		backups = append(backups, fileBackup{path: path, data: data, exists: true})
		return nil
	})
	return backups, err
}

func restorePersonaTree(root string, backups []fileBackup) error {
	keep := make(map[string]struct{}, len(backups))
	for _, backup := range backups {
		keep[filepath.Clean(backup.path)] = struct{}{}
	}
	var extras []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if _, ok := keep[filepath.Clean(path)]; !ok {
			extras = append(extras, path)
		}
		return nil
	}); err != nil {
		return err
	}
	for _, path := range extras {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	for _, backup := range backups {
		if err := os.MkdirAll(filepath.Dir(backup.path), 0o700); err != nil {
			return err
		}
		if err := writeAtomicBytes(backup.path, backup.data); err != nil {
			return err
		}
	}
	return nil
}

func wrapUserSection(body string) string {
	return "## Preferences\n" + NormalizeMarkdown(body)
}

func userPreferences(content string) string {
	if section, ok := ExtractMarkdownSection(content, "Preferences"); ok {
		return NormalizeMarkdown(section)
	}
	return NormalizeMarkdown(content)
}

func worldUserContentPreserved(current, proposed string) bool {
	for _, segment := range protectedWorldSegments(current) {
		if !strings.Contains(proposed, strings.TrimSpace(segment)) {
			return false
		}
	}
	return true
}

func protectedWorldSegments(content string) []string {
	var protected []string
	var block string
	flush := func() {
		if strings.TrimSpace(block) == "" {
			return
		}
		if blockIsUserProtected(block) || !strings.HasPrefix(strings.TrimSpace(block), "## [") {
			protected = append(protected, strings.TrimSpace(block))
		}
		block = ""
	}
	for _, line := range strings.Split(NormalizeMarkdown(content), "\n") {
		if strings.HasPrefix(line, "## [") && block != "" {
			flush()
		}
		if block != "" {
			block += "\n"
		}
		block += line
	}
	flush()
	return protected
}

func blockIsUserProtected(block string) bool {
	hasConfirmed, hasUser := false, false
	for _, line := range strings.Split(block, "\n") {
		switch strings.TrimSpace(line) {
		case "- status: confirmed":
			hasConfirmed = true
		case "- source: user":
			hasUser = true
		}
	}
	return hasConfirmed && hasUser
}

func worldEntryGateAllows(actor RequestActor, current, proposed string) bool {
	currentProse, currentClaims := splitWorldContent(current)
	proposedProse, proposedClaims := splitWorldContent(proposed)
	for _, segment := range proposedProse {
		found := false
		for _, existing := range currentProse {
			if existing == segment {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	for _, block := range proposedClaims {
		if !validWorldEntry(block) {
			return false
		}
	}
	if actor == ActorAutodream {
		for identity, block := range currentClaims {
			if proposedClaims[identity] != block {
				return false
			}
		}
	}
	return true
}

func splitWorldContent(content string) ([]string, map[string]string) {
	var prose []string
	claims := make(map[string]string)
	var block string
	flush := func() {
		normalized := strings.TrimSpace(block)
		if normalized == "" {
			return
		}
		if identity, ok := worldEntryIdentity(normalized); ok {
			claims[identity] = normalized
		} else {
			prose = append(prose, normalized)
		}
		block = ""
	}
	for _, line := range strings.Split(NormalizeMarkdown(content), "\n") {
		if strings.HasPrefix(line, "## [") && block != "" {
			flush()
		}
		if block != "" {
			block += "\n"
		}
		block += line
	}
	flush()
	return prose, claims
}

func worldEntryIdentity(block string) (string, bool) {
	first := strings.Split(block, "\n")[0]
	heading := strings.TrimPrefix(first, "## [")
	if heading == first || !strings.Contains(heading, "] ") {
		return "", false
	}
	parts := strings.SplitN(heading, "] ", 2)
	domain, title := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	if domain == "" || title == "" {
		return "", false
	}
	return strings.ToLower(domain) + ":" + strings.ToLower(title), true
}

func validWorldEntry(block string) bool {
	if _, ok := worldEntryIdentity(block); !ok {
		return false
	}
	hasStatus, hasSource := false, false
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		hasStatus = hasStatus || strings.HasPrefix(trimmed, "- status: ")
		hasSource = hasSource || strings.HasPrefix(trimmed, "- source: ")
	}
	return hasStatus && hasSource
}

func stringPtr(value string) *string { return &value }

func newRequestID() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("req_%x", raw)
}
