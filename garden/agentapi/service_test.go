package agentapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProjectViVy/laputa/garden/internal/ingest"
	"github.com/ProjectViVy/laputa/garden/internal/personactx"
	"github.com/ProjectViVy/laputa/garden/internal/recall"
	"github.com/ProjectViVy/laputa/garden/internal/runtimecore"
	"github.com/ProjectViVy/laputa/laputa/evolution"
)

type frozenFixture struct{}

func (frozenFixture) Get(_ context.Context, session string) (personactx.FrozenCore, error) {
	return personactx.FrozenCore{
		SchemaVersion: evolution.FrozenCoreV2SchemaVersion,
		SessionID:     session,
		MissionStatus: evolution.MissionUnassigned,
		Sections: []personactx.FrozenSection{
			{Kind: personactx.SectionMission},
			{Kind: personactx.SectionIdentity, Content: "safe-identity"},
			{Kind: personactx.SectionRelationship},
			{Kind: personactx.SectionRedline},
			{Kind: personactx.SectionUser},
			{Kind: personactx.SectionDream},
			{Kind: personactx.SectionDark},
		},
	}, nil
}
func fixture(t *testing.T) *Service {
	t.Helper()
	ing, err := ingest.Open(filepath.Join(t.TempDir(), "state.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ing.Close() })
	return NewService(&runtimecore.Garden{ProfileID: "default", FastRecall: &recall.FastService{Frozen: frozenFixture{}}, Ingest: ing})
}
func binding() Binding {
	return Binding{ProfileID: "default", AgentID: "agent-1", Platform: "vivy", SessionID: "session-1"}
}
func assertCode(t *testing.T, err error, code string) {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || e.Code != code {
		t.Fatalf("error = %v, want %s", err, code)
	}
}

func TestBootstrapUsesFrozenSessionAndFailsClosedWithoutPrincipal(t *testing.T) {
	s := fixture(t)
	req := BootstrapRequest{Binding: binding(), SessionID: "session-1", Intent: "hello", BudgetChars: 512}
	_, err := s.Bootstrap(context.Background(), "", req)
	assertCode(t, err, "authentication_required")
	got, err := s.Bootstrap(context.Background(), PrincipalAgent, req)
	if err != nil {
		t.Fatal(err)
	}
	if got.FrozenCore.SessionID != "session-1" || !strings.Contains(got.Context, "safe-identity") || !got.Degraded {
		t.Fatalf("unexpected bootstrap: %+v", got)
	}
	req.SessionID = "other"
	_, err = s.Bootstrap(context.Background(), PrincipalAgent, req)
	assertCode(t, err, "invalid_binding")
}
func TestFastRecallBindingAndPolicy(t *testing.T) {
	s := fixture(t)
	req := FastRecallRequest{Binding: binding(), Query: "hello", BudgetChars: 512}
	got, err := s.FastRecall(context.Background(), PrincipalAgent, req)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != "fast" || got.FrozenCore.SessionID != "session-1" || !got.Degraded {
		t.Fatalf("unexpected view: %+v", got)
	}
	req.Binding.ProfileID = "other"
	_, err = s.FastRecall(context.Background(), PrincipalAgent, req)
	assertCode(t, err, "profile_mismatch")
	req.Binding = binding()
	req.Binding.AgentID = ""
	_, err = s.FastRecall(context.Background(), PrincipalAgent, req)
	assertCode(t, err, "invalid_binding")
}
func TestCaptureIsolatesAgentEventNamespace(t *testing.T) {
	s := fixture(t)
	content := "same terminal"
	hash := sha256.Sum256([]byte(content))
	req := CaptureRequest{Binding: binding(), Phase: CaptureCompleted, Content: content, ContentHash: "sha256:" + hex.EncodeToString(hash[:]), Provenance: CaptureProvenance{RunID: "run-1", EventSeq: 7}}
	first, err := s.Capture(context.Background(), PrincipalAgent, req)
	if err != nil {
		t.Fatal(err)
	}
	req.Binding.AgentID = "agent-2"
	second, err := s.Capture(context.Background(), PrincipalAgent, req)
	if err == nil && first.IngestionID == second.IngestionID {
		t.Fatal("distinct agents shared an ingestion")
	}
}

func TestCaptureStatusSurvivesIngestRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	first, err := ingest.Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(&runtimecore.Garden{ProfileID: "default", Ingest: first})
	content := "terminal result"
	h := sha256.Sum256([]byte(content))
	req := CaptureRequest{Binding: binding(), Phase: CaptureCompleted, Content: content, ContentHash: "sha256:" + hex.EncodeToString(h[:]), Provenance: CaptureProvenance{RunID: "run-1", EventSeq: 7}}
	receipt, err := s.Capture(context.Background(), PrincipalAgent, req)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := ingest.Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	s = NewService(&runtimecore.Garden{ProfileID: "default", Ingest: reopened})
	b := binding()
	b.EventID = receipt.EventID
	status, err := s.CaptureStatus(context.Background(), PrincipalAgent, b, receipt.IngestionID)
	if err != nil {
		t.Fatal(err)
	}
	if status.IngestionID != receipt.IngestionID {
		t.Fatalf("status: %+v", status)
	}
	b.EventID = eventPrefix(b) + "unknown:9"
	_, err = s.CaptureStatus(context.Background(), PrincipalAgent, b, receipt.IngestionID)
	assertCode(t, err, "not_found")
}

func TestCaptureTerminalIdempotentAndStatusAuthorized(t *testing.T) {
	s := fixture(t)
	content := "terminal result"
	hash := sha256.Sum256([]byte(content))
	req := CaptureRequest{Binding: binding(), Phase: CaptureCompleted, Content: content, ContentHash: "sha256:" + hex.EncodeToString(hash[:]), Provenance: CaptureProvenance{RunID: "run-1", EventSeq: 7}}
	_, err := s.Capture(context.Background(), PrincipalRead, req)
	assertCode(t, err, "principal_forbidden")
	req.Phase = "running"
	_, err = s.Capture(context.Background(), PrincipalAgent, req)
	assertCode(t, err, "invalid_request")
	req.Phase = CaptureCompleted
	a, err := s.Capture(context.Background(), PrincipalAgent, req)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Capture(context.Background(), PrincipalAgent, req)
	if err != nil {
		t.Fatal(err)
	}
	if a.IngestionID == "" || a.IngestionID != b.IngestionID || a.EventID != b.EventID {
		t.Fatalf("idempotency: %+v %+v", a, b)
	}
	bound := binding()
	bound.EventID = a.EventID
	_, err = s.CaptureStatus(context.Background(), "", bound, a.IngestionID)
	assertCode(t, err, "authentication_required")
	status, err := s.CaptureStatus(context.Background(), PrincipalAgent, bound, a.IngestionID)
	if err != nil {
		t.Fatal(err)
	}
	if status.IngestionID != a.IngestionID {
		t.Fatalf("status: %+v", status)
	}
	other := bound
	other.SessionID = "different"
	_, err = s.CaptureStatus(context.Background(), PrincipalAgent, other, a.IngestionID)
	assertCode(t, err, "invalid_binding")
	req.Content = "changed"
	h := sha256.Sum256([]byte(req.Content))
	req.ContentHash = "sha256:" + hex.EncodeToString(h[:])
	_, err = s.Capture(context.Background(), PrincipalAgent, req)
	assertCode(t, err, "event_conflict")
}
