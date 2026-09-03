package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/dashimaki/laputa/actmem"
)

type actmemView struct {
	Revision  uint64 `json:"revision"`
	UpdatedAt string `json:"updated_at"`
	Markdown  string `json:"markdown"`
	Truncated bool   `json:"truncated"`
	Sections  struct {
		Pulse string `json:"pulse"`
		Recap string `json:"recap"`
		Work  string `json:"work"`
	} `json:"sections"`
}

func (s *Server) actmemReadPrincipal(w http.ResponseWriter, r *http.Request) bool {
	_, ok := s.requireReadPrincipal(w, r, PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalOperator)
	return ok
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
	view := actmemView{Revision: document.Revision, UpdatedAt: document.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00")}
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
	if !s.actmemReadPrincipal(w, r) {
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
	document, err := s.Actmem.Read()
	if err != nil {
		writeActmemError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, makeActmemView(document, maxChars))
}

type actmemQueryRequest struct {
	Query    string   `json:"query"`
	Sections []string `json:"sections"`
	MaxHits  int      `json:"max_hits"`
	MaxChars int      `json:"max_chars"`
}

func (s *Server) handleActmemQuery(w http.ResponseWriter, r *http.Request) {
	if !s.actmemReadPrincipal(w, r) {
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
	result, err := s.Actmem.Query(actmem.QueryOptions{Query: body.Query, Sections: body.Sections, MaxHits: body.MaxHits, MaxChars: body.MaxChars})
	if err != nil {
		writeActmemError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type actmemPatchRequest struct {
	BaseRevision *uint64 `json:"base_revision"`
	Pulse        *string `json:"pulse"`
	Recap        *string `json:"recap"`
	Work         *string `json:"work"`
}

func (s *Server) handleActmemPut(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalAgent); !ok {
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
	var body actmemPatchRequest
	if err := decodeJSON(w, r, 4<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	if body.BaseRevision == nil {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("base_revision is required"))
		return
	}
	result, err := s.Actmem.Put(actmem.ActmemPatch{Pulse: body.Pulse, Recap: body.Recap, Work: body.Work, BaseRevision: *body.BaseRevision})
	if err != nil {
		writeActmemError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, actmemWritePayload(result))
}

type actmemMaintenanceRequest struct {
	Operation    string  `json:"operation"`
	SessionKey   string  `json:"session_key"`
	Content      string  `json:"content"`
	Section      string  `json:"section"`
	Replacement  string  `json:"replacement"`
	ItemIndex    *int    `json:"item_index"`
	BaseRevision *uint64 `json:"base_revision"`
}

func (s *Server) handleActmemMaintenance(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalAgent); !ok {
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
	var (
		result   actmem.WriteResult
		capsules []actmem.CapsuleSummary
		err      error
	)
	switch strings.TrimSpace(body.Operation) {
	case "append_pulse":
		if strings.TrimSpace(body.SessionKey) == "" {
			err = errors.New("session_key is required")
			break
		}
		result, err = s.Actmem.AppendPulse(body.SessionKey, body.Content)
	case "append_recap":
		if strings.TrimSpace(body.SessionKey) == "" {
			err = errors.New("session_key is required")
			break
		}
		result, err = s.Actmem.AppendRecap(body.SessionKey, body.Content)
	case "edit_work":
		if body.BaseRevision == nil {
			err = errors.New("base_revision is required")
		} else {
			result, err = s.Actmem.EditWork(body.Section, body.Replacement, *body.BaseRevision)
		}
	case "complete_open_item":
		if body.BaseRevision == nil || body.ItemIndex == nil {
			err = errors.New("item_index and base_revision are required")
		} else {
			result, err = s.Actmem.CompleteOpenItem(*body.ItemIndex, *body.BaseRevision)
		}
	case "drop_item":
		if body.BaseRevision == nil || body.ItemIndex == nil {
			err = errors.New("section, item_index and base_revision are required")
		} else {
			result, err = s.Actmem.DropItem(body.Section, *body.ItemIndex, *body.BaseRevision)
		}
	case "fold_session":
		if strings.TrimSpace(body.SessionKey) == "" {
			err = errors.New("session_key is required")
			break
		}
		capsules, err = s.Actmem.FoldSession(body.SessionKey)
		if err == nil {
			document, readErr := s.Actmem.Read()
			err = readErr
			result = actmem.WriteResult{Changed: len(capsules) > 0, Document: document}
		}
	default:
		err = errors.New("unknown ACTMEM maintenance operation")
	}
	if err != nil {
		if strings.HasPrefix(err.Error(), "unknown ACTMEM") || strings.Contains(err.Error(), "required") {
			writeErrorWithCode(w, http.StatusBadRequest, "actmem_invalid_edit", err)
		} else {
			writeActmemError(w, err)
		}
		return
	}
	payload := map[string]any{"result": actmemWritePayload(result)}
	if capsules != nil {
		payload["capsules"] = capsules
	}
	writeJSON(w, http.StatusOK, payload)
}

func actmemWritePayload(result actmem.WriteResult) map[string]any {
	return map[string]any{"changed": result.Changed, "revision": result.Document.Revision, "updated_at": result.Document.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00")}
}

func (s *Server) handleActmemCapsules(w http.ResponseWriter, r *http.Request) {
	if !s.actmemReadPrincipal(w, r) {
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
	if !s.actmemReadPrincipal(w, r) {
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
	if _, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalAgent); !ok {
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
	case "actmem_malformed", "actmem_invalid_edit":
		status = http.StatusBadRequest
	case "actmem_capsule_invalid":
		code = "actmem_capsule_not_found"
		status = http.StatusNotFound
	case "actmem_revision_conflict":
		status = http.StatusConflict
	case "actmem_cap_exceeded":
		status = http.StatusRequestEntityTooLarge
	case "actmem_io_error", "actmem_storage_error":
		code = "actmem_storage_error"
		status = http.StatusInternalServerError
	}
	writeErrorWithCode(w, status, code, err)
}
