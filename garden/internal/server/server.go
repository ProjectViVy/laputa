package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dashimaki/garden/agentapi"
	"github.com/dashimaki/garden/console"
	"github.com/dashimaki/garden/internal/activity"
	"github.com/dashimaki/garden/internal/evolution"
	"github.com/dashimaki/garden/internal/ingest"
	"github.com/dashimaki/garden/internal/mailbox"
	"github.com/dashimaki/garden/internal/pipeline"
	"github.com/dashimaki/garden/internal/recall"
	"github.com/dashimaki/garden/internal/report"
	"github.com/dashimaki/laputa/actmem"
	"github.com/dashimaki/laputa/persona"
	"github.com/dashimaki/mentle/facade"
	"github.com/google/uuid"
)

// Server exposes garden CRUD over HTTP.
type Server struct {
	AgentAPI     *agentapi.Service
	ProfileID    string
	Facade       *facade.Service
	FastRecall   *recall.FastService
	DeepRecall   *recall.DeepService
	TraceStore   *recall.TraceStore
	Evolution    *evolution.Service
	Activity     *activity.Store
	Checkpointer *activity.Checkpointer
	Pipelines    *pipeline.Manager
	Ingestions   *ingest.Service
	Reports      *report.Service
	Mailbox      *mailbox.Store
	Persona      *persona.Service
	Actmem       *actmem.Store
	Capabilities CapabilityConfig
	Materials    MaterialsProvider
	Components   map[string]string
	Addr         string
	// now is an optional clock seam for deterministic time-window handlers.
	// Production servers leave it nil and use UTC wall-clock time.
	now        func() time.Time
	httpServer *http.Server
}

// ListenAndServe starts the HTTP server and blocks until it stops.
func (s *Server) ListenAndServe() error {
	s.httpServer = &http.Server{Addr: s.Addr, Handler: s.HTTPHandler(), ReadHeaderTimeout: 10 * time.Second}
	return s.httpServer.ListenAndServe()
}

func (s *Server) HTTPHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /v2/memories", s.handleCreateMemory)
	mux.HandleFunc("GET /v2/memories/{id}", s.handleGetMemory)
	mux.HandleFunc("GET /v2/memories", s.handleListMemories)
	mux.HandleFunc("PATCH /v2/memories/{id}", s.handleUpdateMemory)
	mux.HandleFunc("DELETE /v2/memories/{id}", s.handleDeleteMemory)
	mux.HandleFunc("POST /v2/ingest/sessions", s.handleSessionSubmit)
	mux.HandleFunc("GET /v2/ingestions/{id}", s.handleIngestionStatus)
	mux.HandleFunc("POST /v2/recall/bootstrap", s.handleBootstrap)
	mux.HandleFunc("GET /v2/reports", s.handleReportsList)
	mux.HandleFunc("GET /v2/reports/latest", s.handleLatestReport)
	mux.HandleFunc("POST /v2/reports/generate", s.handleReportGenerate)
	mux.HandleFunc("GET /v2/reports/orientation", s.handleReportOrientation)
	mux.HandleFunc("GET /v2/reports/modules", s.handleModulesList)
	mux.HandleFunc("POST /v2/reports/modules", s.handleModuleCreate)
	mux.HandleFunc("PATCH /v2/reports/modules/{id}", s.handleModuleUpdate)
	mux.HandleFunc("GET /v2/pipelines", s.handlePipelines)
	mux.HandleFunc("GET /v2/pipelines/{name}", s.handlePipeline)
	mux.HandleFunc("GET /v2/pipelines/{name}/runs", s.handlePipelineRuns)
	mux.HandleFunc("GET /v2/pipelines/{name}/runs/{trace_id}", s.handlePipelineRun)
	mux.HandleFunc("POST /v2/recall/fast", s.handleFastRecall)
	mux.HandleFunc("POST /v2/recall/deep", s.handleDeepRecall)
	mux.HandleFunc("GET /v2/recall/traces/{trace_id}", s.handleRecallTrace)
	mux.HandleFunc("POST /v2/activity/events", s.handleActivityEvents)
	mux.HandleFunc("GET /v2/activity/sessions/{session_id}", s.handleSessionActivity)
	mux.HandleFunc("POST /v2/evolution/runs", s.handleEvolutionStartRun)
	mux.HandleFunc("GET /v2/evolution/runs", s.handleEvolutionListRuns)
	mux.HandleFunc("GET /v2/evolution/runs/{run_id}", s.handleEvolutionGetRun)
	mux.HandleFunc("GET /v2/evolution/candidates/{candidate_id}", s.handleEvolutionGetCandidate)
	mux.HandleFunc("POST /v2/evolution/proposals", s.handleEvolutionCreateProposal)
	mux.HandleFunc("GET /v2/evolution/proposals", s.handleEvolutionListProposals)
	mux.HandleFunc("GET /v2/evolution/proposals/{proposal_id}", s.handleEvolutionGetProposal)
	mux.HandleFunc("POST /v2/evolution/proposals/{proposal_id}/review", s.handleEvolutionReviewProposal)
	mux.HandleFunc("GET /v2/evolution/events/{event_id}", s.handleEvolutionGetEvent)
	mux.HandleFunc("GET /v2/evolution/hub/status", s.handleEvolutionHubStatus)
	mux.HandleFunc("GET /v2/mailbox/inbox", s.handleMailboxInbox)
	mux.HandleFunc("GET /v2/mailbox/outbox", s.handleMailboxOutbox)
	mux.HandleFunc("GET /v2/mailbox/dead-letter", s.handleMailboxDeadLetter)
	mux.HandleFunc("GET /v2/persona/documents", s.handlePersonaDocuments)
	mux.HandleFunc("GET /v2/persona/documents/{kind}", s.handlePersonaDocumentV2)
	mux.HandleFunc("PUT /v2/persona/documents/{kind}", s.handlePersonaDocumentPutV2)
	mux.HandleFunc("POST /v2/persona/initialize", s.handlePersonaInitializeV2)
	mux.HandleFunc("POST /v2/persona/repair", s.handlePersonaRepairV2)
	mux.HandleFunc("GET /v2/persona/reviews", s.handlePersonaReviewsV2)
	mux.HandleFunc("POST /v2/persona/reviews", s.handlePersonaReviewCreateV2)
	mux.HandleFunc("GET /v2/persona/reviews/{id}", s.handlePersonaReviewV2)
	mux.HandleFunc("POST /v2/persona/reviews/{id}/approve", func(w http.ResponseWriter, r *http.Request) { s.handlePersonaReviewDecisionV2(w, r, true) })
	mux.HandleFunc("POST /v2/persona/reviews/{id}/reject", func(w http.ResponseWriter, r *http.Request) { s.handlePersonaReviewDecisionV2(w, r, false) })
	mux.HandleFunc("GET /v2/persona/history/{kind}", s.handlePersonaHistoryV2)
	mux.HandleFunc("GET /v2/persona/history/{kind}/{revision}", s.handlePersonaHistoryRevisionV2)
	mux.HandleFunc("GET /v2/actmem", s.handleActmemRead)
	mux.HandleFunc("POST /v2/actmem/query", s.handleActmemQuery)
	mux.HandleFunc("PUT /v2/actmem", s.handleActmemPut)
	mux.HandleFunc("POST /v2/actmem/maintenance", s.handleActmemMaintenance)
	mux.HandleFunc("GET /v2/actmem/capsules", s.handleActmemCapsules)
	mux.HandleFunc("GET /v2/actmem/capsules/{name}", s.handleActmemCapsule)
	mux.HandleFunc("DELETE /v2/actmem/capsules/{name}", s.handleActmemCapsuleDelete)
	mux.HandleFunc("POST /v2/mailbox/items/{id}/approve", s.handleMailboxApprove)
	mux.HandleFunc("POST /v2/mailbox/items/{id}/reject", s.handleMailboxReject)
	mux.HandleFunc("GET /v2/admin/overview", s.handleAdminOverview)
	mux.HandleFunc("GET /v2/admin/components", s.handleAdminComponents)
	mux.HandleFunc("GET /v2/admin/index-health", s.handleAdminIndexHealth)
	mux.HandleFunc("GET /v2/admin/context-manifest/{trace_id}", s.handleAdminContextManifest)
	mux.HandleFunc("GET /v2/admin/spool", s.handleAdminSpool)
	mux.HandleFunc("GET /v2/materials/cards", s.handleMaterialsCards)
	mux.HandleFunc("GET /v2/materials/cards/{id}/evidence", s.handleMaterialsEvidence)
	mux.HandleFunc("GET /v2/materials/collections", s.handleMaterialsCollections)
	mux.HandleFunc("/", s.spaHandler())

	return requestMiddleware(mux)
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) spaHandler() http.HandlerFunc {
	sub, _ := fs.Sub(console.Dist, "dist")
	fileServer := http.FileServer(http.FS(sub))
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/v2/") {
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", "GET")
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			http.NotFound(w, r)
			return
		}
		clean := strings.TrimPrefix(path, "/")
		if clean == "" {
			clean = "index.html"
		}
		if _, err := fs.Stat(sub, clean); err != nil {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	}
}

func (s *Server) handleCreateMemory(w http.ResponseWriter, r *http.Request) {
	if s.Facade == nil {
		writeHandlerError(w, facade.ErrUnavailable)
		return
	}
	principal, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalAgent)
	if !ok {
		return
	}
	if !personaContentTypeOK(r) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
		return
	}
	raw, err := readBody(w, r, 64<<10)
	if err != nil {
		writeRequestError(w, err)
		return
	}
	var body facade.CreateMemoryRequest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeRequestError(w, err)
		return
	}
	body.Actor = auditLabel(r, principal)
	body.RequestID = w.Header().Get("X-Garden-Request-ID")
	hash := fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
	result, err := s.Facade.CreateMemoryWithIndexStatus(r.Context(), body, r.Header.Get("Idempotency-Key"), hash)
	if err != nil {
		writeMemoryError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) handleGetMemory(w http.ResponseWriter, r *http.Request) {
	if s.Facade == nil {
		writeHandlerError(w, facade.ErrUnavailable)
		return
	}
	if _, ok := s.requireReadPrincipal(w, r, PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalAutodream, PrincipalOperator); !ok {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("id is required"))
		return
	}
	memory, err := s.Facade.GetMemory(r.Context(), id)
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, memory)
}

func (s *Server) handleListMemories(w http.ResponseWriter, r *http.Request) {
	if s.Facade == nil {
		writeHandlerError(w, facade.ErrUnavailable)
		return
	}
	if _, ok := s.requireReadPrincipal(w, r, PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalAutodream, PrincipalOperator); !ok {
		return
	}
	for key := range r.URL.Query() {
		if key != "limit" && key != "cursor" && key != "status" && key != "kind" {
			writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("unknown memory query parameter: "+key))
			return
		}
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			writeError(w, http.StatusBadRequest, errors.New("limit must be between 1 and 200"))
			return
		}
		limit = parsed
	}
	page, err := s.Facade.ListMemories(r.Context(), facade.ListMemoryOptions{Limit: limit, Cursor: r.URL.Query().Get("cursor"), Status: r.URL.Query().Get("status"), Kind: r.URL.Query().Get("kind")})
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) handleDeleteMemory(w http.ResponseWriter, r *http.Request) {
	if s.Facade == nil {
		writeHandlerError(w, facade.ErrUnavailable)
		return
	}
	principal, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalAgent)
	if !ok {
		return
	}
	if !personaContentTypeOK(r) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("id is required"))
		return
	}
	var body facade.DeleteMemoryRequest
	if err := decodeJSON(w, r, 16<<10, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	if body.ExpectedVersion <= 0 || strings.TrimSpace(body.Reason) == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("expected_version and reason are required"))
		return
	}
	body.Actor = auditLabel(r, principal)
	body.RequestID = w.Header().Get("X-Garden-Request-ID")
	result, err := s.Facade.DeleteMemory(r.Context(), id, body.ExpectedVersion, body.Actor, body.RequestID, body.Reason)
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleUpdateMemory(w http.ResponseWriter, r *http.Request) {
	if s.Facade == nil {
		writeHandlerError(w, facade.ErrUnavailable)
		return
	}
	principal, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalAgent)
	if !ok {
		return
	}
	if !personaContentTypeOK(r) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("id is required"))
		return
	}
	var body facade.UpdateMemoryRequest
	if err := decodeJSON(w, r, 64<<10, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	if body.ExpectedVersion == nil || *body.ExpectedVersion <= 0 || strings.TrimSpace(body.Reason) == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("expected_version, reason and a mutable field are required"))
		return
	}
	if body.Content == nil && body.Tags == nil {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("expected_version, reason and a mutable field are required"))
		return
	}
	body.Actor = auditLabel(r, principal)
	body.RequestID = w.Header().Get("X-Garden-Request-ID")
	result, err := s.Facade.UpdateMemoryWithIndexStatus(r.Context(), id, body)
	if err != nil {
		writeMemoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleSessionSubmit(w http.ResponseWriter, r *http.Request) {
	principal, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalAgent, PrincipalAutodream)
	if !ok {
		return
	}
	if s.AgentAPI == nil {
		writeError(w, http.StatusServiceUnavailable, facade.ErrUnavailable)
		return
	}
	if !personaContentTypeOK(r) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", errors.New("content type must be application/json"))
		return
	}
	var body agentapi.SessionSubmitRequest
	if err := decodeJSON(w, r, 4<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	accepted, err := s.AgentAPI.SubmitSession(r.Context(), agentapi.Principal(principal), body)
	if err != nil {
		var domainErr *agentapi.Error
		if errors.As(err, &domainErr) {
			switch domainErr.Code {
			case "event_conflict":
				writeErrorWithCode(w, http.StatusConflict, "memory_idempotency_conflict", domainErr)
			case "unavailable":
				writeError(w, http.StatusServiceUnavailable, facade.ErrUnavailable)
			case "invalid_request":
				writeErrorWithCode(w, http.StatusBadRequest, "invalid_request", domainErr)
			default:
				writeHandlerError(w, domainErr)
			}
		} else {
			writeHandlerError(w, err)
		}
		return
	}
	writeJSON(w, http.StatusAccepted, accepted)
}
func (s *Server) handleIngestionStatus(w http.ResponseWriter, r *http.Request) {
	if s.Ingestions == nil {
		writeError(w, http.StatusServiceUnavailable, facade.ErrUnavailable)
		return
	}
	status, err := s.Ingestions.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	components := map[string]string{"garden": "ok", "laputa": "ok"}
	for name, value := range s.Components {
		components[name] = value
		if value != "ok" {
			status = "degraded"
		}
	}
	indexHealth, healthErr := s.indexHealth(r.Context(), PrincipalRead)
	if healthErr != nil {
		status = "degraded"
		components["mentle"] = "unavailable"
	} else {
		components["mentle"] = indexHealth.Status
		if indexHealth.Status != "ok" {
			status = "degraded"
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": status, "components": components, "index_health": indexHealth, "api_contract": "garden-hermes/1", "source": "live"})
}

func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	principal, ok := s.requireReadPrincipal(w, r, PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalAutodream, PrincipalOperator)
	if !ok {
		return
	}
	// The HTTP contract accepts recall inputs only. Host binding is not an
	// authenticated client claim and must not be decoded from this request.
	var body struct {
		SessionID   string `json:"session_id"`
		Intent      string `json:"intent"`
		BudgetChars int    `json:"budget_chars"`
	}
	if err := decodeJSON(w, r, 1<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	if strings.TrimSpace(body.SessionID) == "" {
		writeError(w, http.StatusBadRequest, errors.New("session_id is required"))
		return
	}
	if body.BudgetChars == 0 {
		body.BudgetChars = 8000
	}
	if body.BudgetChars < 256 || body.BudgetChars > 64000 {
		writeError(w, http.StatusBadRequest, errors.New("budget_chars must be between 256 and 64000"))
		return
	}
	intent := body.Intent
	if strings.TrimSpace(intent) == "" {
		intent = "current session bootstrap context"
	}
	if s.AgentAPI == nil || strings.TrimSpace(s.ProfileID) == "" {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "unavailable", errors.New("Garden runtime unavailable"))
		return
	}
	view, err := s.AgentAPI.Bootstrap(r.Context(), agentapi.Principal(principal), agentapi.BootstrapRequest{
		Binding: agentapi.Binding{ProfileID: s.ProfileID, AgentID: "garden-http", Platform: "http", SessionID: body.SessionID},
		SessionID: body.SessionID, Intent: intent, BudgetChars: body.BudgetChars,
	})
	if err != nil {
		var domainErr *agentapi.Error
		if errors.As(err, &domainErr) {
			status := http.StatusBadRequest
			switch domainErr.Code {
			case "authentication_required":
				status = http.StatusUnauthorized
			case "principal_forbidden":
				status = http.StatusForbidden
			case "unavailable":
				status = http.StatusServiceUnavailable
			}
			writeErrorWithCode(w, status, domainErr.Code, domainErr)
		} else {
			writeErrorWithCode(w, http.StatusInternalServerError, "internal_error", errors.New("bootstrap unavailable"))
		}
		return
	}
	writeJSON(w, http.StatusOK, view)
}
func (s *Server) handleLatestReport(w http.ResponseWriter, r *http.Request) {
	if s.Reports == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("report service unavailable"))
		return
	}
	value, err := s.Reports.Latest(r.Context(), r.URL.Query().Get("cadence"))
	if errors.Is(err, report.ErrNotFound) {
		_, _ = s.Reports.Generate(r.Context(), r.URL.Query().Get("cadence"), time.Now().UTC())
		value, err = s.Reports.Latest(r.Context(), r.URL.Query().Get("cadence"))
	}
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) handlePipelines(w http.ResponseWriter, r *http.Request) {
	if s.Pipelines == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("pipeline runtime unavailable"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revision": s.Pipelines.Revision(), "pipelines": s.Pipelines.Definitions()})
}

func (s *Server) handlePipeline(w http.ResponseWriter, r *http.Request) {
	if s.Pipelines == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("pipeline runtime unavailable"))
		return
	}
	name := r.PathValue("name")
	for _, definition := range s.Pipelines.Definitions() {
		if definition.Name == name {
			writeJSON(w, http.StatusOK, map[string]any{"revision": s.Pipelines.Revision(), "pipeline": definition})
			return
		}
	}
	writeError(w, http.StatusNotFound, errors.New("pipeline not found"))
}

func (s *Server) handlePipelineRuns(w http.ResponseWriter, r *http.Request) {
	if s.Pipelines == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("pipeline runtime unavailable"))
		return
	}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, errors.New("limit must be between 1 and 100"))
			return
		}
		limit = parsed
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": s.Pipelines.Runs(r.PathValue("name"), limit)})
}

func (s *Server) handlePipelineRun(w http.ResponseWriter, r *http.Request) {
	if s.Pipelines == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("pipeline runtime unavailable"))
		return
	}
	trace, ok := s.Pipelines.RunByID(r.PathValue("name"), r.PathValue("trace_id"))
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("pipeline run not found"))
		return
	}
	writeJSON(w, http.StatusOK, trace)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// writeErrorWithCode writes the error envelope with an explicit code.
func writeErrorWithCode(w http.ResponseWriter, status int, code string, err error) {
	writeErrorWithDetails(w, status, code, err, map[string]any{})
}

func writeErrorWithDetails(w http.ResponseWriter, status int, code string, err error, details map[string]any) {
	retryable := status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
	if details == nil {
		details = map[string]any{}
	}
	writeJSON(w, status, map[string]any{"code": code, "message": err.Error(), "error": err.Error(), "retryable": retryable, "request_id": w.Header().Get("X-Garden-Request-ID"), "details": details})
}

func writeError(w http.ResponseWriter, status int, err error) {
	code := "invalid_request"
	switch status {
	case http.StatusNotFound:
		code = "memory_not_found"
	case http.StatusConflict:
		code = "version_conflict"
	case http.StatusRequestEntityTooLarge:
		code = "payload_too_large"
	case http.StatusTooManyRequests:
		code = "busy"
	case http.StatusInternalServerError:
		code = "internal_error"
	case http.StatusServiceUnavailable:
		code = "service_unavailable"
	case http.StatusGatewayTimeout:
		code = "timeout"
	}
	if errors.Is(err, facade.ErrIdempotencyConflict) || errors.Is(err, ingest.ErrEventConflict) {
		code = "idempotency_conflict"
	}
	if errors.Is(err, report.ErrNotFound) {
		code = "report_not_found"
	}
	writeErrorWithDetails(w, status, code, err, map[string]any{})
}

func writeHandlerError(w http.ResponseWriter, err error) {
	if errors.Is(err, facade.ErrMemoryNotFound) || errors.Is(err, ingest.ErrNotFound) || errors.Is(err, report.ErrNotFound) || errors.Is(err, mailbox.ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if errors.Is(err, mailbox.ErrIllegalTransition) {
		writeError(w, http.StatusConflict, err)
		return
	}
	if errors.Is(err, facade.ErrVersionConflict) || errors.Is(err, facade.ErrIdempotencyConflict) || errors.Is(err, ingest.ErrEventConflict) {
		code := "memory_version_conflict"
		if errors.Is(err, facade.ErrIdempotencyConflict) || errors.Is(err, ingest.ErrEventConflict) {
			code = "memory_idempotency_conflict"
		}
		writeErrorWithCode(w, http.StatusConflict, code, err)
		return
	}
	if errors.Is(err, facade.ErrEmbeddingDimensionMismatch) || errors.Is(err, facade.ErrEmbeddingMetricMismatch) || errors.Is(err, facade.ErrEmbeddingIdentityMismatch) {
		writeErrorWithCode(w, http.StatusConflict, "memory_embedding_identity_mismatch", err)
		return
	}
	if errors.Is(err, facade.ErrIndexHealthUnavailable) {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "index_health_unavailable", errors.New("live index probes unavailable"))
		return
	}
	if errors.Is(err, facade.ErrReadOnly) {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "memory_read_only", errors.New("memory writes unavailable in lexical-only mode"))
		return
	}
	if errors.Is(err, facade.ErrUnavailable) {
		writeErrorWithCode(w, http.StatusServiceUnavailable, "memory_unavailable", err)
		return
	}
	if strings.Contains(err.Error(), "exceeds 64 KiB") {
		writeErrorWithCode(w, http.StatusRequestEntityTooLarge, "memory_content_too_large", err)
		return
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unknown key prefix"),
		strings.Contains(msg, "unknown section"),
		strings.Contains(msg, "required"), strings.Contains(msg, "invalid"),
		strings.Contains(msg, "at least one"), strings.Contains(msg, "content_hash"),
		strings.Contains(msg, "phase must"):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}

func writeMemoryError(w http.ResponseWriter, err error) {
	message := err.Error()
	switch {
	case strings.Contains(message, "memory content is required"),
		strings.Contains(message, "invalid memory kind"),
		strings.Contains(message, "invalid source type"),
		strings.Contains(message, "at least one mutable field"):
		writeErrorWithCode(w, http.StatusBadRequest, "memory_invalid_content", err)
	case strings.Contains(message, "memory content exceeds 64 KiB"):
		writeErrorWithCode(w, http.StatusRequestEntityTooLarge, "memory_content_too_large", err)
	default:
		writeHandlerError(w, err)
	}
}

func requestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Garden-Request-ID"))
		if id == "" {
			id = "req_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		}
		w.Header().Set("X-Garden-Request-ID", id)
		r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func readBody(w http.ResponseWriter, r *http.Request, max int64) ([]byte, error) {
	reader := http.MaxBytesReader(w, r.Body, max)
	decoder := json.NewDecoder(reader)
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, errors.New("request body must contain one JSON value")
		}
		return nil, err
	}
	return raw, nil
}
func decodeJSON(w http.ResponseWriter, r *http.Request, max int64, dst any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, max))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain one JSON value")
		}
		return err
	}
	return nil
}
func writeRequestError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		writeError(w, http.StatusRequestEntityTooLarge, err)
		return
	}
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) || errors.Is(err, io.EOF) {
		writeErrorWithCode(w, http.StatusBadRequest, "malformed_json", err)
		return
	}
	writeError(w, http.StatusBadRequest, err)
}
func auditActor(r *http.Request) string {
	value := strings.TrimSpace(r.Header.Get("X-Garden-Actor"))
	if value == "" {
		return "user_request"
	}
	return value
}
