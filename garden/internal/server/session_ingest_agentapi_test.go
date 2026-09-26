package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/dashimaki/garden/agentapi"
	"github.com/dashimaki/garden/internal/ingest"
	"github.com/dashimaki/garden/internal/runtimecore"
)

func TestSessionIngestHTTPUsesSharedServiceAndLegacyPayload(t *testing.T) {
	ing, err := ingest.Open(filepath.Join(t.TempDir(), "ingest.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ing.Close() })
	srv := testServer()
	srv.AgentAPI = agentapi.NewService(&runtimecore.Garden{ProfileID: srv.ProfileID, Ingest: ing})
	// Ingestions is deliberately nil: a direct handler call to the legacy field fails.
	content := "legacy precompact"
	sum := sha256.Sum256([]byte(content))
	payload, err := json.Marshal(ingest.SubmitRequest{SessionID: "session-1", EventID: "event-1", Phase: "precompact", Content: content, ContentHash: "sha256:" + hex.EncodeToString(sum[:]), Workspace: "workspace"})
	if err != nil {
		t.Fatal(err)
	}
	var firstID string
	for _, token := range []string{"user", "agent", "autodream"} {
		req := localRequest(http.MethodPost, "/v2/ingest/sessions", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		srv.Capabilities = CapabilityConfig{UserToken: "user", AgentToken: "agent", AutodreamToken: "autodream"}
		rec := httptest.NewRecorder()
		srv.HTTPHandler().ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("%s: status=%d body=%s", token, rec.Code, rec.Body.String())
		}
		var accepted ingest.Accepted
		if err := json.Unmarshal(rec.Body.Bytes(), &accepted); err != nil {
			t.Fatal(err)
		}
		if accepted.IngestionID == "" || accepted.EventID != "event-1" || accepted.SessionID != "session-1" {
			t.Fatalf("%s: %+v", token, accepted)
		}
		if firstID == "" {
			firstID = accepted.IngestionID
		}
	}
	badPayload, err := json.Marshal(ingest.SubmitRequest{SessionID: "session-1", EventID: "bad-event", Phase: "running", Content: content, ContentHash: "sha256:" + hex.EncodeToString(sum[:])})
	if err != nil {
		t.Fatal(err)
	}
	badReq := localRequest(http.MethodPost, "/v2/ingest/sessions", bytes.NewBuffer(badPayload))
	badReq.Header.Set("Content-Type", "application/json")
	badReq.Header.Set("Authorization", "Bearer agent")
	badRec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid phase: status=%d body=%s", badRec.Code, badRec.Body.String())
	}
	// The existing GET receipt path keeps its separate direct ingest wiring.
	srv.Ingestions = ing
	statusRec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(statusRec, localRequest(http.MethodGet, "/v2/ingestions/"+firstID, nil))
	if statusRec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", statusRec.Code, statusRec.Body.String())
	}
}
