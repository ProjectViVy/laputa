package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ProjectViVy/laputa/laputa/actmem"
	"github.com/ProjectViVy/laputa/laputa/evolution"
)

// actmemCallerScope derives the trusted scope for HTTP principals from the
// configured profile + host workspace binding — never from request data.
// read/user/operator principals read the owner projection (the caller IS the
// subject); agent reads the union bound to ActmemWorkspace.
func (s *Server) actmemCallerScope(principal Principal) (evolution.Scope, error) {
	if principal == PrincipalAgent {
		kind := evolution.ScopePersonal
		if s.ActmemWorkspace != "" {
			kind = evolution.ScopeWorkspace
		}
		scope := evolution.Scope{SubjectID: s.ProfileID, Kind: kind, WorkspaceID: s.ActmemWorkspace}
		if err := scope.Validate(); err != nil {
			return evolution.Scope{}, err
		}
		return scope, nil
	}
	scope := evolution.Scope{SubjectID: s.ProfileID, Kind: evolution.ScopePersonal}
	if err := scope.Validate(); err != nil {
		return evolution.Scope{}, err
	}
	return scope, nil
}

type actmemView struct {
	Revision     uint64            `json:"revision"`
	UpdatedAt    string            `json:"updated_at"`
	Markdown     string            `json:"markdown"`
	Truncated    bool              `json:"truncated"`
	Unclassified bool              `json:"unclassified,omitempty"`
	Entries      []evolution.Entry `json:"entries,omitempty"`
	Sections     struct {
		Pulse string `json:"pulse"`
		Recap string `json:"recap"`
		Work  string `json:"work"`
	} `json:"sections"`
}

func (s *Server) actmemReadPrincipal(w http.ResponseWriter, r *http.Request) (Principal, bool) {
	return s.requireReadPrincipal(w, r, PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalOperator)
}

func parseActmemMaxChars(raw string) (int, error) {
	if raw == "" {
		return actmem.ACTMEMReadCapChars, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > actmem.ACTMEMReadCapChars {
		return 0, errors.New("max_chars must be between 1 and 1200")
	}
	return value, nil
}

func makeActmemView(document actmem.ActmemDocument, maxChars int) actmemView {
	view := actmemView{Revision: document.Revision, UpdatedAt: document.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"), Unclassified: document.Unclassified}
	view.Markdown = truncateActmem(document.Markdown, maxChars)
	view.Truncated = len([]rune(document.Markdown)) > len([]rune(view.Markdown))
	remaining := maxChars
	view.Sections.Pulse, remaining = takeActmem(document.Pulse, remaining)
	view.Sections.Recap, remaining = takeActmem(document.Recap, remaining)
	view.Sections.Work, remaining = takeActmem(document.Work, remaining)
	return view
}

func takeActmem(value string, budget int) (string, int) {
	if budget <= 0 {
		return "", 0
	}
	value = truncateActmem(value, budget)
	return value, budget - len([]rune(value))
}

func truncateActmem(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max-1]) + "…"
}

func (s *Server) handleActmemRead(w http.ResponseWriter, r *http.Request) {
	principal, ok := s.actmemReadPrincipal(w, r)
	if !ok {
		return
	}
	if s.Actmem == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("ACTMEM service unavailable"))
		return
	}
	for key := range r.URL.Query() {
		if key != "max_chars" {
			writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("unknown ACTMEM read query parameter: "+key))
			return
		}
	}
	maxChars, err := parseActmemMaxChars(r.URL.Query().Get("max_chars"))
	if err != nil {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	if principal == PrincipalAgent {
		scope, scopeErr := s.actmemCallerScope(principal)
		if scopeErr != nil {
			writeErrorWithCode(w, http.StatusInternalServerError, "actmem_scope_error", scopeErr)
			return
		}
		result, readErr := s.Actmem.ReadScoped(scope, evolution.ReadRequest{})
		if readErr != nil {
			writeActmemError(w, readErr)
			return
		}
		entries := result.Entries
		if maxChars < actmem.ACTMEMReadCapChars && len(entries) > 0 {
			entries = budgetEntries(entries, maxChars)
		}
		writeJSON(w, http.StatusOK, actmemView{Revision: result.Revision, Entries: entries})
		return
	}
	document, err := s.Actmem.Read()
	if err != nil {
		writeActmemError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, makeActmemView(document, maxChars))
}

// budgetEntries keeps whole entries within the read budget, dropping the
// largest bodies first — a read projection, never a write.
func budgetEntries(entries []evolution.Entry, maxChars int) []evolution.Entry {
	total := 0
	for _, e := range entries {
		total += len([]rune(e.Body))
	}
	if total <= maxChars {
		return entries
	}
	kept := make([]evolution.Entry, len(entries))
	copy(kept, entries)
	for i := len(kept) - 1; i >= 0 && total > maxChars; i-- {
		body := len([]rune(kept[i].Body))
		allowed := maxChars - (total - body)
		if allowed < 0 {
			allowed = 0
		}
		kept[i].Body = truncateActmem(kept[i].Body, allowed)
		total = (total - body) + len([]rune(kept[i].Body))
	}
	return kept
}

type actmemQueryRequest struct {
	Query    string   `json:"query"`
	Sections []string `json:"sections"`
	MaxHits  int      `json:"max_hits"`
	MaxChars int      `json:"max_chars"`
}

func (s *Server) handleActmemQuery(w http.ResponseWriter, r *http.Request) {
	principal, ok := s.actmemReadPrincipal(w, r)
	if !ok {
		return
	}
	if s.Actmem == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("ACTMEM service unavailable"))
		return
	}
	if !personaContentTypeOK(r) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
		return
	}
	var body actmemQueryRequest
	if err := decodeJSON(w, r, 64<<10, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	scope, err := s.actmemCallerScope(principal)
	if err != nil {
		writeErrorWithCode(w, http.StatusInternalServerError, "actmem_scope_error", err)
		return
	}
	result, err := s.Actmem.Query(scope, actmem.QueryOptions{Query: body.Query, Sections: body.Sections, MaxHits: body.MaxHits, MaxChars: body.MaxChars})
	if err != nil {
		writeActmemError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Owner whole-save only: {markdown, base_revision}. Agents patch Work
// through maintenance; nobody writes free-text sections anymore.
type actmemSaveRequest struct {
	BaseRevision *uint64 `json:"base_revision"`
	Markdown     *string `json:"markdown"`
}

func (s *Server) handleActmemPut(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePrincipal(w, r, PrincipalUser); !ok {
		return
	}
	if s.Actmem == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("ACTMEM service unavailable"))
		return
	}
	if !personaContentTypeOK(r) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
		return
	}
	var body actmemSaveRequest
	if err := decodeJSON(w, r, 4<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	if body.BaseRevision == nil || body.Markdown == nil {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("markdown and base_revision are required"))
		return
	}
	result, err := s.Actmem.Save(*body.Markdown, *body.BaseRevision)
	if err != nil {
		writeActmemError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"changed": result.Changed, "revision": result.Revision})
}

type actmemMaintenanceRequest struct {
	Operation string `json:"operation"`
	// system_append: caller supplies event identity; the store allocs the id.
	// Scope is derived from the caller, never supplied: strict decoding
	// rejects subject_id/kind/workspace_id fields in this struct.
	Entry *struct {
		Section   string                `json:"section"`
		Field     string                `json:"field,omitempty"`
		SessionID string                `json:"session_id"`
		EventID   string                `json:"event_id,omitempty"`
		Body      string                `json:"body"`
		Sources   []evolution.SourceRef `json:"sources,omitempty"`
	} `json:"entry,omitempty"`
	// work_patch: scoped Work changes against base_revision.
	WorkPatch *evolution.WorkPatch `json:"work_patch,omitempty"`
	// fold_session.
	SessionKey string `json:"session_key,omitempty"`
}

func (s *Server) handleActmemMaintenance(w http.ResponseWriter, r *http.Request) {
	principal, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalAgent)
	if !ok {
		return
	}
	if s.Actmem == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("ACTMEM service unavailable"))
		return
	}
	if !personaContentTypeOK(r) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
		return
	}
	var body actmemMaintenanceRequest
	if err := decodeJSON(w, r, 4<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	scope, err := s.actmemCallerScope(principal)
	if err != nil {
		writeErrorWithCode(w, http.StatusInternalServerError, "actmem_scope_error", err)
		return
	}
	payload := map[string]any{}
	switch strings.TrimSpace(body.Operation) {
	case "system_append":
		if body.Entry == nil || strings.TrimSpace(body.Entry.SessionID) == "" || body.Entry.Body == "" {
			writeErrorWithCode(w, http.StatusBadRequest, "actmem_invalid_edit", errors.New("entry with session_id and body is required"))
			return
		}
		result, opErr := s.Actmem.AppendEntry(evolution.Entry{
			Section:   evolution.EntrySection(body.Entry.Section),
			Scope:     scope,
			Field:     evolution.WorkField(body.Entry.Field),
			SessionID: body.Entry.SessionID,
			EventID:   body.Entry.EventID,
			Body:      body.Entry.Body,
			Sources:   body.Entry.Sources,
		})
		if opErr != nil {
			writeActmemError(w, opErr)
			return
		}
		payload["result"] = map[string]any{"changed": result.Changed, "revision": result.Revision, "entries": result.Entries}
	case "work_patch":
		if body.WorkPatch == nil {
			writeErrorWithCode(w, http.StatusBadRequest, "actmem_invalid_edit", errors.New("work_patch is required"))
			return
		}
		result, opErr := s.Actmem.ApplyWorkPatch(scope, *body.WorkPatch)
		if opErr != nil {
			writeActmemError(w, opErr)
			return
		}
		payload["result"] = map[string]any{"changed": result.Changed, "revision": result.Revision, "entries": result.Entries}
	case "fold_session":
		if strings.TrimSpace(body.SessionKey) == "" {
			writeErrorWithCode(w, http.StatusBadRequest, "actmem_invalid_edit", errors.New("session_key is required"))
			return
		}
		capsules, opErr := s.Actmem.FoldSession(body.SessionKey)
		if opErr != nil {
			writeActmemError(w, opErr)
			return
		}
		payload["capsules"] = capsules
	default:
		writeErrorWithCode(w, http.StatusBadRequest, "actmem_invalid_edit", errors.New("unknown ACTMEM maintenance operation"))
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleActmemCapsules(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.actmemReadPrincipal(w, r); !ok {
		return
	}
	if s.Actmem == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("ACTMEM service unavailable"))
		return
	}
	items, err := s.Actmem.ListCapsules()
	if err != nil {
		writeActmemError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleActmemCapsule(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.actmemReadPrincipal(w, r); !ok {
		return
	}
	if s.Actmem == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("ACTMEM service unavailable"))
		return
	}
	document, err := s.Actmem.ReadCapsule(r.PathValue("name"))
	if err != nil {
		writeActmemError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, document)
}

func (s *Server) handleActmemCapsuleDelete(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePrincipal(w, r, PrincipalUser); !ok {
		return
	}
	if s.Actmem == nil {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "service_unavailable", errors.New("ACTMEM service unavailable"))
		return
	}
	name := r.PathValue("name")
	if err := s.Actmem.DeleteCapsule(name); err != nil {
		writeActmemError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": name, "deleted": true})
}

func writeActmemError(w http.ResponseWriter, err error) {
	code := actmem.CodeOf(err)
	status := http.StatusInternalServerError
	switch code {
	case "actmem_malformed", "actmem_invalid_edit", "actmem_invalid_entry", "actmem_format_error", "actmem_unclassified_head", "actmem_scope_forbidden":
		status = http.StatusBadRequest
	case "actmem_capsule_invalid":
		code = "actmem_capsule_not_found"
		status = http.StatusNotFound
	case "actmem_revision_conflict", "actmem_fold_conflict":
		status = http.StatusConflict
	case "actmem_cap_exceeded":
		status = http.StatusRequestEntityTooLarge
	case "actmem_io_error", "actmem_storage_error":
		code = "actmem_storage_error"
		status = http.StatusInternalServerError
	}
	writeErrorWithCode(w, status, code, err)
}
