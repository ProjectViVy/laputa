package agentapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ProjectViVy/laputa/garden/internal/ingest"
	"github.com/ProjectViVy/laputa/garden/internal/recall"
	"github.com/ProjectViVy/laputa/garden/internal/runtimecore"
)

// Service is the trusted in-process entrypoint. The caller supplies a principal
// obtained from authentication independently of untrusted Binding metadata.
type Service struct{ runtime *runtimecore.Garden }

func NewService(runtime *runtimecore.Garden) *Service { return &Service{runtime: runtime} }

func eventPrefix(b Binding) string {
	return "agent/" + base64.RawURLEncoding.EncodeToString([]byte(b.ProfileID+"\x00"+b.AgentID+"\x00"+b.Platform+"\x00"+b.SessionID)) + "/"
}

// FastRecallRequest carries the same recall inputs as the domain FastService,
// with a mandatory external Agent binding.
type FastRecallRequest struct {
	Binding     Binding `json:"binding"`
	Query       string  `json:"query"`
	Scope       string  `json:"scope"`
	BudgetChars int     `json:"budget_chars"`
}

// CaptureStatus is the public asynchronous receipt status without storage handles.
type CaptureStatus struct {
	IngestionID string   `json:"ingestion_id"`
	Status      string   `json:"status"`
	MemoryIDs   []string `json:"memory_ids"`
	TraceID     string   `json:"trace_id,omitempty"`
	Warnings    []string `json:"warnings"`
	Error       *string  `json:"error"`
}

func failure(code, message string) *Error {
	return &Error{Code: code, Message: message, LegacyError: message, Details: map[string]any{}}
}
func (s *Service) check(binding Binding, principal Principal, op Operation) error {
	if s == nil || s.runtime == nil {
		return failure("unavailable", "Garden runtime unavailable")
	}
	if err := Authorize(s.runtime.ProfileID, binding, principal, op); err != nil {
		return err
	}
	if strings.TrimSpace(binding.AgentID) == "" || strings.TrimSpace(binding.Platform) == "" || (op != OpSearch && op != OpIndexHealth && strings.TrimSpace(binding.SessionID) == "") {
		return failure("invalid_binding", "agent_id, platform and session_id are required")
	}
	return nil
}
func convert[T any](value any) (T, error) {
	var result T
	raw, err := json.Marshal(value)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(raw, &result)
	return result, err
}
func (s *Service) recall(ctx context.Context, binding Binding, principal Principal, op Operation, query, scope string, budget int) (ContextView, error) {
	if err := s.check(binding, principal, op); err != nil {
		return ContextView{}, err
	}
	if s.runtime.FastRecall == nil {
		return ContextView{}, failure("unavailable", "fast recall unavailable")
	}
	view, err := s.runtime.FastRecall.Recall(ctx, recall.FastRequest{Query: query, Scope: scope, BudgetChars: budget, SessionID: binding.SessionID})
	if err != nil {
		return ContextView{}, failure("invalid_request", err.Error())
	}
	out, err := convert[ContextView](view)
	if err != nil {
		return ContextView{}, fmt.Errorf("convert recall: %w", err)
	}
	return out, nil
}
func (s *Service) FastRecall(ctx context.Context, principal Principal, req FastRecallRequest) (ContextView, error) {
	return s.recall(ctx, req.Binding, principal, OpSearch, req.Query, req.Scope, req.BudgetChars)
}
func (s *Service) Bootstrap(ctx context.Context, principal Principal, req BootstrapRequest) (BootstrapResponse, error) {
	if req.SessionID != "" && req.SessionID != req.Binding.SessionID {
		return BootstrapResponse{}, failure("invalid_binding", "session_id does not match binding")
	}
	if req.BudgetChars == 0 {
		req.BudgetChars = 8000
	}
	query := req.Intent
	if strings.TrimSpace(query) == "" {
		query = "current session bootstrap context"
	}
	view, err := s.recall(ctx, req.Binding, principal, OpBootstrap, query, "", req.BudgetChars)
	if err != nil {
		return BootstrapResponse{}, err
	}
	return BootstrapResponse{TraceID: view.TraceID, Context: view.Context, FrozenCore: view.FrozenCore, Evidence: view.Evidence, Degraded: view.Degraded, Warnings: view.Warnings}, nil
}
func (s *Service) Capture(ctx context.Context, principal Principal, req CaptureRequest) (CaptureReceipt, error) {
	if err := s.check(req.Binding, principal, OpCapture); err != nil {
		return CaptureReceipt{}, err
	}
	if req.Phase != CaptureCompleted && req.Phase != CaptureFailed && req.Phase != CaptureCanceled {
		return CaptureReceipt{}, failure("invalid_request", "terminal capture phase required")
	}
	if strings.TrimSpace(req.Provenance.RunID) == "" || req.Provenance.EventSeq == 0 {
		return CaptureReceipt{}, failure("invalid_request", "run_id and event_seq are required")
	}
	if s.runtime.Ingest == nil {
		return CaptureReceipt{}, failure("unavailable", "ingest unavailable")
	}
	event := eventPrefix(req.Binding) + req.Provenance.RunID + ":" + fmt.Sprint(req.Provenance.EventSeq)
	// The host phase is terminal provenance; ingest's session_end phase is its
	// own ingestion lifecycle and must not be confused with the host phase.
	accepted, err := s.runtime.Ingest.Submit(ctx, ingest.SubmitRequest{SessionID: req.Binding.SessionID, EventID: event, Phase: "session_end", Content: req.Content, ContentHash: req.ContentHash, Workspace: req.Binding.WorkspaceID, OccurredAt: req.OccurredAt})
	if errors.Is(err, ingest.ErrEventConflict) {
		return CaptureReceipt{}, failure("event_conflict", err.Error())
	}
	if err != nil {
		return CaptureReceipt{}, failure("invalid_request", err.Error())
	}
	// Ingest also deduplicates identical session content across distinct events.
	// That receipt cannot be reassigned to a different binding or terminal event.
	if accepted.EventID != event {
		return CaptureReceipt{}, failure("event_conflict", "content already captured for a different event")
	}
	return CaptureReceipt{IngestionID: accepted.IngestionID, SessionID: accepted.SessionID, EventID: accepted.EventID, Status: accepted.Status, Seq: accepted.Seq}, nil
}
func (s *Service) CaptureStatus(ctx context.Context, principal Principal, binding Binding, id string) (CaptureStatus, error) {
	if err := s.check(binding, principal, OpCapture); err != nil {
		return CaptureStatus{}, err
	}
	if s.runtime.Ingest == nil {
		return CaptureStatus{}, failure("unavailable", "ingest unavailable")
	}
	if binding.EventID == "" || !strings.HasPrefix(binding.EventID, eventPrefix(binding)) {
		return CaptureStatus{}, failure("invalid_binding", "capture event_id must belong to binding")
	}
	// The durable ingestion row, not process-local receipt state, owns status.
	status, err := s.runtime.Ingest.GetByIdentity(ctx, id, binding.SessionID, binding.EventID)
	if errors.Is(err, ingest.ErrNotFound) {
		return CaptureStatus{}, failure("not_found", "capture not found")
	}
	if err != nil {
		return CaptureStatus{}, failure("unavailable", "capture status unavailable")
	}
	return CaptureStatus(status), nil
}
