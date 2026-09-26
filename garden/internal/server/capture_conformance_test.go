package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/dashimaki/garden/agentapi"
	"github.com/dashimaki/garden/internal/ingest"
	"github.com/dashimaki/garden/internal/runtimecore"
	"github.com/dashimaki/garden/internal/sqliteconn"
)

// Both entrypoints must land on the same durable ingestion, not merely return
// similarly shaped receipts from independent stores.
func TestTerminalCaptureEmbeddedAndHTTPConvergeOnOneIngestion(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "ingest.db")
	store, err := ingest.Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	srv := testServer()
	srv.Capabilities = CapabilityConfig{AgentToken: "agent-token"}
	srv.AgentAPI = agentapi.NewService(&runtimecore.Garden{ProfileID: srv.ProfileID, Ingest: store})
	// Do not set srv.Ingestions: HTTP must use the same AgentAPI domain entrypoint.
	binding := agentapi.Binding{ProfileID: srv.ProfileID, AgentID: "vivy", Platform: "desktop", SessionID: "session-1"}
	occurred := time.Date(2026, time.September, 27, 1, 2, 3, 0, time.UTC)

	httpSubmit := func(req ingest.SubmitRequest) (int, ingest.Accepted) {
		t.Helper()
		body, err := json.Marshal(req)
		if err != nil {
			t.Fatal(err)
		}
		r := localRequest(http.MethodPost, "/v2/ingest/sessions", bytes.NewBuffer(body))
		r.Header.Set("Authorization", "Bearer agent-token")
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.HTTPHandler().ServeHTTP(w, r)
		if w.Code != http.StatusAccepted {
			return w.Code, ingest.Accepted{}
		}
		var accepted ingest.Accepted
		if err := json.Unmarshal(w.Body.Bytes(), &accepted); err != nil {
			t.Fatal(err)
		}
		return w.Code, accepted
	}

	for _, tc := range []struct {
		name          string
		runID         string
		seq           uint64
		content       string
		embeddedFirst bool
	}{
		{name: "embedded then HTTP", runID: "run-1", seq: 7, content: "terminal completed one", embeddedFirst: true},
		{name: "HTTP then embedded", runID: "run-2", seq: 8, content: "terminal completed two"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hash := sha256.Sum256([]byte(tc.content))
			contentHash := fmt.Sprintf("sha256:%x", hash)
			// Independently construct the documented host namespace; the HTTP
			// request must point at the exact event Capture derives.
			key := binding.ProfileID + "\x00" + binding.AgentID + "\x00" + binding.Platform + "\x00" + binding.SessionID
			eventID := "agent/" + base64.RawURLEncoding.EncodeToString([]byte(key)) + "/" + fmt.Sprintf("%s:%d", tc.runID, tc.seq)
			capture := agentapi.CaptureRequest{Binding: binding, Phase: agentapi.CaptureCompleted, Content: tc.content, ContentHash: contentHash, OccurredAt: occurred, Provenance: agentapi.CaptureProvenance{RunID: tc.runID, EventSeq: tc.seq}}
			ordinary := ingest.SubmitRequest{SessionID: binding.SessionID, EventID: eventID, Phase: "session_end", Content: tc.content, ContentHash: contentHash, OccurredAt: occurred}

			var embedded agentapi.CaptureReceipt
			var viaHTTP ingest.Accepted
			callEmbedded := func() {
				t.Helper()
				embedded, err = srv.AgentAPI.Capture(ctx, agentapi.PrincipalAgent, capture)
				if err != nil {
					t.Fatal(err)
				}
			}
			callHTTP := func() {
				t.Helper()
				var code int
				code, viaHTTP = httpSubmit(ordinary)
				if code != http.StatusAccepted {
					t.Fatalf("HTTP capture status = %d, want 202", code)
				}
			}
			if tc.embeddedFirst {
				callEmbedded()
				callHTTP()
			} else {
				callHTTP()
				callEmbedded()
			}
			if embedded.IngestionID == "" || embedded.IngestionID != viaHTTP.IngestionID || embedded.EventID != eventID || viaHTTP.EventID != eventID || embedded.SessionID != binding.SessionID || viaHTTP.SessionID != binding.SessionID {
				t.Fatalf("capture receipts disagree: embedded=%+v HTTP=%+v expected event=%q", embedded, viaHTTP, eventID)
			}

			// Read the actual shared SQLite authority rather than trusting receipt
			// equality alone; the host phase must map to ingest session_end.
			db, err := sqliteconn.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			var count int
			var phase, savedContent, savedHash, savedTime string
			err = db.QueryRowContext(ctx, `SELECT count(*), max(phase), max(content), max(content_hash), max(occurred_at) FROM ingestions WHERE event_id=?`, eventID).Scan(&count, &phase, &savedContent, &savedHash, &savedTime)
			if err != nil {
				t.Fatal(err)
			}
			if count != 1 || phase != "session_end" || savedContent != tc.content || savedHash != contentHash || savedTime != occurred.Format(time.RFC3339Nano) {
				t.Fatalf("durable ingestion: count=%d phase=%q content=%q hash=%q occurred=%q", count, phase, savedContent, savedHash, savedTime)
			}

			// A reused host event with different content conflicts through both
			// adapters, rather than silently creating a second ingestion.
			otherContent := tc.content + " changed"
			otherHash := sha256.Sum256([]byte(otherContent))
			capture.Content, ordinary.Content = otherContent, otherContent
			capture.ContentHash = fmt.Sprintf("sha256:%x", otherHash)
			ordinary.ContentHash = capture.ContentHash
			if code, _ := httpSubmit(ordinary); code != http.StatusConflict {
				t.Fatalf("HTTP divergent event status = %d, want 409", code)
			}
			_, err = srv.AgentAPI.Capture(ctx, agentapi.PrincipalAgent, capture)
			var domainErr *agentapi.Error
			if !errors.As(err, &domainErr) || domainErr.Code != "event_conflict" {
				t.Fatalf("embedded divergent event error = %v, want event_conflict", err)
			}
		})
	}
}
