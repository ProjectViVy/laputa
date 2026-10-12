package agentapi

import (
	"context"
	"errors"
	"time"

	"github.com/ProjectViVy/laputa/garden/internal/ingest"
)

// SessionSubmitRequest is the ordinary Garden session-ingest payload. It is
// distinct from external terminal Capture, which requires bound provenance.
type SessionSubmitRequest struct {
	SessionID   string    `json:"session_id"`
	EventID     string    `json:"event_id"`
	Phase       string    `json:"phase"`
	Content     string    `json:"content"`
	ContentHash string    `json:"content_hash"`
	Workspace   string    `json:"workspace,omitempty"`
	OccurredAt  time.Time `json:"occurred_at,omitempty"`
}

type SessionAccepted struct {
	IngestionID string `json:"ingestion_id"`
	SessionID   string `json:"session_id"`
	EventID     string `json:"event_id"`
	Status      string `json:"status"`
}

// SubmitSession preserves the ordinary Garden session-ingest contract. Unlike
// external terminal Capture, its event ID and precompact phase come from the
// existing Garden caller; the trusted principal still passes shared policy.
func (s *Service) SubmitSession(ctx context.Context, principal Principal, req SessionSubmitRequest) (SessionAccepted, error) {
	if s == nil || s.runtime == nil {
		return SessionAccepted{}, failure("unavailable", "Garden runtime unavailable")
	}
	if err := Authorize(s.runtime.ProfileID, Binding{ProfileID: s.runtime.ProfileID}, principal, OpCapture); err != nil {
		return SessionAccepted{}, err
	}
	if s.runtime.Ingest == nil {
		return SessionAccepted{}, failure("unavailable", "ingest unavailable")
	}
	accepted, err := s.runtime.Ingest.Submit(ctx, ingest.SubmitRequest{
		SessionID: req.SessionID, EventID: req.EventID, Phase: req.Phase,
		Content: req.Content, ContentHash: req.ContentHash, Workspace: req.Workspace,
		OccurredAt: req.OccurredAt,
	})
	if errors.Is(err, ingest.ErrEventConflict) {
		return SessionAccepted{}, failure("event_conflict", err.Error())
	}
	if err != nil {
		return SessionAccepted{}, failure("invalid_request", err.Error())
	}
	return SessionAccepted{IngestionID: accepted.IngestionID, SessionID: accepted.SessionID, EventID: accepted.EventID, Status: accepted.Status}, nil
}

// SubmitBoundSession accepts lifecycle content for one admitted host session.
// The host workspace is fixed; explicit authored sharing uses MutateMemory.
func (s *Service) SubmitBoundSession(ctx context.Context, p Principal, b Binding, req SessionSubmitRequest) (SessionAccepted, error) {
	if err := s.check(b, p, OpCapture); err != nil {
		return SessionAccepted{}, err
	}
	if req.Workspace != "" && req.Workspace != b.WorkspaceID {
		return SessionAccepted{}, failure("invalid_binding", "workspace conflicts with host binding")
	}
	event := s.eventPrefix(b) + req.EventID
	session, identity, err := s.captureIdentity(ctx, b, event)
	if err != nil {
		return SessionAccepted{}, err
	}
	accepted, err := s.runtime.Ingest.Submit(ctx, ingest.SubmitRequest{Identity: identity, SessionID: session, EventID: event, Phase: req.Phase, Content: req.Content, ContentHash: req.ContentHash, Workspace: b.WorkspaceID, OccurredAt: req.OccurredAt})
	if errors.Is(err, ingest.ErrEventConflict) {
		return SessionAccepted{}, failure("event_conflict", err.Error())
	}
	if err != nil {
		return SessionAccepted{}, failure("invalid_request", err.Error())
	}
	return SessionAccepted{IngestionID: accepted.IngestionID, SessionID: b.SessionID, EventID: accepted.EventID, Status: accepted.Status}, nil
}
func (s *Service) SessionStatus(ctx context.Context, p Principal, b Binding, id, event string) (CaptureStatus, error) {
	b.EventID = event
	return s.CaptureStatus(ctx, p, b, id)
}
