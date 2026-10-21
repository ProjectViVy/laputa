package agentapi

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProjectViVy/laputa/garden/internal/ingest"
	"github.com/ProjectViVy/laputa/garden/internal/personactx"
	"github.com/ProjectViVy/laputa/garden/internal/recall"
	"github.com/ProjectViVy/laputa/garden/internal/runtimecore"
	"github.com/ProjectViVy/laputa/garden/internal/sqliteconn"
	"github.com/ProjectViVy/laputa/laputa/actmem"
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
	req := CaptureRequest{Binding: binding(), Phase: CaptureCompleted, Content: content, Provenance: CaptureProvenance{RunID: "run-1", EventSeq: 7}}
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
	req := CaptureRequest{Binding: binding(), Phase: CaptureCompleted, Content: content, Provenance: CaptureProvenance{RunID: "run-1", EventSeq: 7}}
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
	req := CaptureRequest{Binding: binding(), Phase: CaptureCompleted, Content: content, Provenance: CaptureProvenance{RunID: "run-1", EventSeq: 7}}
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
	_, err = s.Capture(context.Background(), PrincipalAgent, req)
	assertCode(t, err, "event_conflict")
}

func TestCaptureUnverifiableLegacyReplayPointsToReceiptRecovery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.db")
	ing, err := ingest.OpenPaused(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ing.Close()
	ing.ProfileID = "default"
	ing.Actmem = actmem.New(filepath.Join(dir, "persona"))
	s := NewService(&runtimecore.Garden{ProfileID: "default", Ingest: ing})
	req := CaptureRequest{
		Binding: binding(), Phase: CaptureCompleted, Content: "terminal result",
		Provenance: CaptureProvenance{RunID: "legacy-run", EventSeq: 9},
		Activity:   &CaptureActivity{UserText: "admitted source"},
	}
	receipt, err := s.Capture(context.Background(), PrincipalAgent, req)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := json.Marshal(map[string]any{
		"schema": "garden.capture-activity/v1",
		"source": map[string]string{"phase": "completed", "user_text": ""},
		"status": "applied", "revision": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	db, err := sqliteconn.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`UPDATE ingestions SET activity_json=? WHERE event_id=?`, string(legacy), receipt.EventID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Capture(context.Background(), PrincipalAgent, req); err == nil {
		t.Fatal("legacy replay unexpectedly returned a receipt")
	} else {
		assertCode(t, err, "event_replay_unverifiable")
		if !strings.Contains(err.Error(), "receipt lookup") {
			t.Fatalf("recovery guidance missing from error: %v", err)
		}
	}
	got, found, err := s.LookupCapture(context.Background(), PrincipalAgent, req.Binding, req.Provenance)
	if err != nil || !found || got.IngestionID != receipt.IngestionID || got.Seq != receipt.Seq {
		t.Fatalf("receipt lookup recovery: %+v found=%v err=%v", got, found, err)
	}
}

func TestCaptureReceiptLookupUsesTrustedEventIdentity(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	content := "legacy acknowledgement"
	req := CaptureRequest{Binding: binding(), Phase: CaptureCompleted, Content: content, Provenance: CaptureProvenance{RunID: "old-run", EventSeq: 7}}
	receipt, err := s.Capture(ctx, PrincipalAgent, req)
	if err != nil {
		t.Fatal(err)
	}
	lookup, ok := any(s).(interface {
		LookupCapture(context.Context, Principal, Binding, CaptureProvenance) (CaptureReceipt, bool, error)
	})
	if !ok {
		t.Fatal("capture receipt lookup unavailable for accepted event recovery")
	}
	got, found, err := lookup.LookupCapture(ctx, PrincipalAgent, req.Binding, req.Provenance)
	if err != nil || !found || got.IngestionID != receipt.IngestionID || got.Seq != receipt.Seq {
		t.Fatalf("receipt changed: %+v found=%v err=%v", got, found, err)
	}
	for _, b := range []Binding{func() Binding { b := binding(); b.SessionID = "other-session"; return b }(), func() Binding { b := binding(); b.AgentID = "other-agent"; return b }()} {
		_, found, err := lookup.LookupCapture(ctx, PrincipalAgent, b, req.Provenance)
		if err != nil || found {
			t.Fatalf("foreign binding read receipt: found=%v err=%v", found, err)
		}
	}
	_, _, err = lookup.LookupCapture(ctx, "", req.Binding, req.Provenance)
	assertCode(t, err, "authentication_required")
}
