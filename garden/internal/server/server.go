package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dashimaki/garden/console"
	"github.com/dashimaki/garden/internal/activity"
	"github.com/dashimaki/garden/internal/authority"
	"github.com/dashimaki/garden/internal/crud"
	"github.com/dashimaki/garden/internal/evolution"
	"github.com/dashimaki/garden/internal/ingest"
	"github.com/dashimaki/garden/internal/pipeline"
	"github.com/dashimaki/garden/internal/rag"
	"github.com/dashimaki/garden/internal/recall"
	"github.com/dashimaki/garden/internal/report"
	"github.com/dashimaki/garden/internal/router"
	"github.com/dashimaki/laputa/governance"
	"github.com/dashimaki/mentle/facade"
	"github.com/google/uuid"
)

// Server exposes garden CRUD over HTTP.
type Server struct {
	Handler        *crud.Handler
	Resolver       rag.Resolver
	FastRecall     *recall.FastService
	DeepRecall     *recall.DeepService
	TraceStore     *recall.TraceStore
	Evolution      *evolution.Service
	Activity       *activity.Store
	Checkpointer   *activity.Checkpointer
	Pipelines      *pipeline.Manager
	Ingestions     *ingest.Service
	Reports        *report.Service
	Governed       *governance.GovernedService
	GovernedWriter *authority.GovernedWriter
	Materials      MaterialsProvider
	Components     map[string]string
	Addr           string
	httpServer     *http.Server
}

// ListenAndServe starts the HTTP server and blocks until it stops.
func (s *Server) ListenAndServe() error {
	s.httpServer = &http.Server{Addr: s.Addr, Handler: s.HTTPHandler(), ReadHeaderTimeout: 10 * time.Second}
	return s.httpServer.ListenAndServe()
}

func (s *Server) HTTPHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/memories", s.handleWrite)
	mux.HandleFunc("GET /v1/memories/{key}", s.handleRead)
	mux.HandleFunc("GET /v1/memories", s.handleList)
	mux.HandleFunc("PATCH /v1/memories/{key}", s.handleUpdate)
	mux.HandleFunc("DELETE /v1/memories/{key}", s.handleForget)
	mux.HandleFunc("POST /v1/sessions", s.handleSessionSubmit)
	mux.HandleFunc("GET /v1/ingestions/{id}", s.handleIngestionStatus)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /v1/context/resolve", s.handleResolveContext)
	mux.HandleFunc("POST /v1/context/bootstrap", s.handleBootstrap)
	mux.HandleFunc("GET /v1/reports/latest", s.handleLatestReport)
	mux.HandleFunc("GET /v1/pipelines", s.handlePipelines)
	mux.HandleFunc("GET /v1/pipelines/{name}", s.handlePipeline)
	mux.HandleFunc("GET /v1/pipelines/{name}/runs", s.handlePipelineRuns)
	mux.HandleFunc("GET /v1/pipelines/{name}/runs/{trace_id}", s.handlePipelineRun)
	mux.HandleFunc("POST /v2/recall/fast", s.handleFastRecall)
	mux.HandleFunc("POST /v2/recall/deep", s.handleDeepRecall)
	mux.HandleFunc("GET /v2/recall/traces/{trace_id}", s.handleRecallTrace)
	mux.HandleFunc("POST /v2/activity/events", s.handleActivityEvents)
	mux.HandleFunc("GET /v2/activity/sessions/{session_id}", s.handleSessionActivity)
	mux.HandleFunc("POST /v2/governance/projection", s.handleGovernanceProjection)
	mux.HandleFunc("POST /v2/governance/mutations", s.handleGovernanceMutation)
	mux.HandleFunc("GET /v2/governance/audit", s.handleGovernanceAudit)
	mux.HandleFunc("POST /v2/evolution/runs", s.handleEvolutionStartRun)
	mux.HandleFunc("GET /v2/evolution/runs/{run_id}", s.handleEvolutionGetRun)
	mux.HandleFunc("GET /v2/evolution/candidates/{candidate_id}", s.handleEvolutionGetCandidate)
	mux.HandleFunc("POST /v2/evolution/proposals", s.handleEvolutionCreateProposal)
	mux.HandleFunc("GET /v2/evolution/proposals/{proposal_id}", s.handleEvolutionGetProposal)
	mux.HandleFunc("POST /v2/evolution/proposals/{proposal_id}/review", s.handleEvolutionReviewProposal)
	mux.HandleFunc("GET /v2/evolution/events/{event_id}", s.handleEvolutionGetEvent)
	mux.HandleFunc("GET /v2/admin/overview", s.handleAdminOverview)
	mux.HandleFunc("GET /v2/admin/components", s.handleAdminComponents)
	mux.HandleFunc("GET /v2/admin/context-manifest/{trace_id}", s.handleAdminContextManifest)
	mux.HandleFunc("GET /v2/admin/spool", s.handleAdminSpool)
	mux.HandleFunc("GET /v2/admin/audit", s.handleAdminAudit)
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

func (s *Server) handleWrite(w http.ResponseWriter, r *http.Request) {
	raw, err := readBody(w, r, 64<<10)
	if err != nil {
		writeRequestError(w, err)
		return
	}
	var shape map[string]json.RawMessage
	if err := json.Unmarshal(raw, &shape); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	_, legacyKey := shape["key"]
	_, legacyValue := shape["value"]
	_, canonical := shape["content"]
	if (legacyKey || legacyValue) && canonical {
		writeError(w, http.StatusBadRequest, errors.New("canonical and legacy fields cannot be mixed"))
		return
	}
	if canonical {
		var body facade.CreateMemoryRequest
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			writeRequestError(w, err)
			return
		}
		body.Actor = auditActor(r)
		body.RequestID = w.Header().Get("X-Garden-Request-ID")
		hash := fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
		memory, err := s.Handler.CreateMemory(r.Context(), body, r.Header.Get("Idempotency-Key"), hash)
		if err != nil {
			writeHandlerError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, memory)
		return
	}
	var body struct {
		Key   string         `json:"key"`
		Value string         `json:"value"`
		Meta  map[string]any `json:"meta,omitempty"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if body.Key == "" {
		writeError(w, http.StatusBadRequest, errors.New("key is required"))
		return
	}

	id, err := s.Handler.Write(router.WithActor(r.Context(), mapActorRole(auditActor(r))), body.Key, body.Value, body.Meta)
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": id})
}

func (s *Server) handleRead(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, errors.New("key is required"))
		return
	}

	if strings.HasPrefix(key, "mem_") {
		memory, err := s.Handler.GetMemory(r.Context(), key)
		if err != nil {
			writeHandlerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, memory)
		return
	}
	record, err := s.Handler.Read(r.Context(), key)
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("view") == "canonical" {
		limit := 50
		if raw := r.URL.Query().Get("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 1 || parsed > 200 {
				writeError(w, http.StatusBadRequest, errors.New("limit must be between 1 and 200"))
				return
			}
			limit = parsed
		}
		page, err := s.Handler.ListMemories(r.Context(), facade.ListMemoryOptions{Limit: limit, Cursor: r.URL.Query().Get("cursor"), Status: r.URL.Query().Get("status"), Kind: r.URL.Query().Get("kind")})
		if err != nil {
			writeHandlerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, page)
		return
	}
	prefix := r.URL.Query().Get("prefix")
	if prefix == "" {
		prefix = "section:"
	}

	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			writeError(w, http.StatusBadRequest, errors.New("limit must be a non-negative integer"))
			return
		}
		limit = parsed
	}

	records, err := s.Handler.List(r.Context(), prefix, limit)
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	if records == nil {
		records = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"records": records})
}

func (s *Server) handleForget(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, errors.New("key is required"))
		return
	}

	if strings.HasPrefix(key, "mem_") {
		result, err := s.Handler.DeleteMemory(r.Context(), key, auditActor(r), w.Header().Get("X-Garden-Request-ID"))
		if err != nil {
			writeHandlerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	ok, err := s.Handler.Forget(router.WithActor(r.Context(), mapActorRole(auditActor(r))), key)
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": ok})
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("key")
	if !strings.HasPrefix(id, "mem_") {
		writeError(w, http.StatusBadRequest, errors.New("PATCH requires a canonical memory id"))
		return
	}
	var body facade.UpdateMemoryRequest
	if err := decodeJSON(w, r, 64<<10, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	body.Actor = auditActor(r)
	body.RequestID = w.Header().Get("X-Garden-Request-ID")
	memory, err := s.Handler.UpdateMemory(r.Context(), id, body)
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, memory)
}

func (s *Server) handleSessionSubmit(w http.ResponseWriter, r *http.Request) {
	if s.Ingestions == nil {
		writeError(w, http.StatusServiceUnavailable, facade.ErrUnavailable)
		return
	}
	var body ingest.SubmitRequest
	if err := decodeJSON(w, r, 4<<20, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	accepted, err := s.Ingestions.Submit(r.Context(), body)
	if err != nil {
		writeHandlerError(w, err)
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
	writeJSON(w, http.StatusOK, map[string]any{"status": status, "components": components, "api_contract": "garden-hermes/1"})
}

func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SessionID   string `json:"session_id"`
		Intent      string `json:"intent"`
		BudgetChars int    `json:"budget_chars"`
	}
	if err := decodeJSON(w, r, 1<<20, &body); err != nil {
		writeRequestError(w, err)
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

	if s.FastRecall != nil {
		view, err := s.FastRecall.Recall(r.Context(), recall.FastRequest{Query: intent, Scope: "", BudgetChars: body.BudgetChars, SessionID: body.SessionID})
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"trace_id": view.TraceID, "context": view.Context, "evidence": view.Evidence, "degraded": view.Degraded, "warnings": view.Warnings})
		return
	}

	if s.Resolver == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("pipeline unavailable"))
		return
	}
	result, err := s.Resolver.Resolve(r.Context(), rag.ResolveRequest{Intent: intent, SessionID: body.SessionID, Mode: "basic"})
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	runes := []rune(result.Context)
	if len(runes) > body.BudgetChars {
		result.Context = string(runes[:body.BudgetChars])
	}
	writeJSON(w, http.StatusOK, map[string]any{"trace_id": result.TraceID, "context": result.Context, "evidence": result.Evidence, "degraded": result.Degraded, "warnings": result.Warnings})
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

func (s *Server) handleResolveContext(w http.ResponseWriter, r *http.Request) {
	if s.Resolver == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("agentic RAG unavailable"))
		return
	}
	var request rag.ResolveRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(request.Intent) == "" {
		writeError(w, http.StatusBadRequest, errors.New("intent is required"))
		return
	}
	result, err := s.Resolver.Resolve(r.Context(), request)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
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

func writeError(w http.ResponseWriter, status int, err error) {
	code := "invalid_request"
	retryable := false
	switch status {
	case http.StatusNotFound:
		code = "memory_not_found"
	case http.StatusConflict:
		code = "version_conflict"
	case http.StatusRequestEntityTooLarge:
		code = "payload_too_large"
	case http.StatusTooManyRequests:
		code = "busy"
		retryable = true
	case http.StatusInternalServerError:
		code = "internal_error"
		retryable = true
	case http.StatusServiceUnavailable:
		code = "mentle_unavailable"
		retryable = true
	case http.StatusGatewayTimeout:
		code = "timeout"
		retryable = true
	}
	if errors.Is(err, facade.ErrIdempotencyConflict) || errors.Is(err, ingest.ErrEventConflict) {
		code = "idempotency_conflict"
	}
	if errors.Is(err, report.ErrNotFound) {
		code = "report_not_found"
	}
	writeJSON(w, status, map[string]any{"error": err.Error(), "code": code, "retryable": retryable, "request_id": w.Header().Get("X-Garden-Request-ID"), "details": map[string]any{}})
}

func writeHandlerError(w http.ResponseWriter, err error) {
	if errors.Is(err, facade.ErrMemoryNotFound) || errors.Is(err, ingest.ErrNotFound) || errors.Is(err, report.ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if errors.Is(err, governance.ErrUnauthorized) {
		writeError(w, http.StatusForbidden, err)
		return
	}
	if errors.Is(err, facade.ErrVersionConflict) || errors.Is(err, facade.ErrIdempotencyConflict) || errors.Is(err, ingest.ErrEventConflict) {
		writeError(w, http.StatusConflict, err)
		return
	}
	if errors.Is(err, facade.ErrUnavailable) {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}
	if strings.Contains(err.Error(), "exceeds 64 KiB") {
		writeError(w, http.StatusRequestEntityTooLarge, err)
		return
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unknown key prefix"),
		strings.Contains(msg, "invalid governance key"),
		strings.Contains(msg, "unknown section"),
		strings.Contains(msg, "governance list prefix"),
		strings.Contains(msg, "required"), strings.Contains(msg, "invalid"),
		strings.Contains(msg, "at least one"), strings.Contains(msg, "content_hash"),
		strings.Contains(msg, "phase must"):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
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
	var raw json.RawMessage
	if err := json.NewDecoder(reader).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}
func decodeJSON(w http.ResponseWriter, r *http.Request, max int64, dst any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, max))
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
func writeRequestError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		writeError(w, http.StatusRequestEntityTooLarge, err)
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
