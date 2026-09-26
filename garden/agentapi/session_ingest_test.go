package agentapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/dashimaki/garden/internal/runtimecore"
)

func TestSubmitSessionPreservesLegacyPayloadAndRoles(t *testing.T) {
	s := fixture(t)
	content := "session notes"
	hash := sha256.Sum256([]byte(content))
	req := SessionSubmitRequest{SessionID: "session-1", EventID: "event-1", Phase: "precompact", Content: content, ContentHash: "sha256:" + hex.EncodeToString(hash[:]), Workspace: "project"}
	for _, principal := range []Principal{PrincipalUser, PrincipalAgent, PrincipalAutodream} {
		req.EventID = "event-" + string(principal)
		receipt, err := s.SubmitSession(context.Background(), principal, req)
		if err != nil {
			t.Fatalf("%s: %v", principal, err)
		}
		if receipt.SessionID != req.SessionID || receipt.EventID != "event-user" || receipt.IngestionID == "" {
			t.Fatalf("receipt: %+v", receipt)
		}
	}
	_, err := s.SubmitSession(context.Background(), PrincipalRead, req)
	assertCode(t, err, "principal_forbidden")
	_, err = s.SubmitSession(context.Background(), "", req)
	assertCode(t, err, "authentication_required")
}

func TestSubmitSessionRequiresRuntimeIngestAndValidatesThroughDomain(t *testing.T) {
	s := fixture(t)
	_, err := s.SubmitSession(context.Background(), PrincipalUser, SessionSubmitRequest{SessionID: "session-1"})
	assertCode(t, err, "invalid_request")
	s = NewService(nil)
	_, err = s.SubmitSession(context.Background(), PrincipalUser, SessionSubmitRequest{SessionID: "session-1"})
	assertCode(t, err, "unavailable")
	s = NewService(&runtimecore.Garden{ProfileID: "default"})
	_, err = s.SubmitSession(context.Background(), PrincipalUser, SessionSubmitRequest{SessionID: "session-1"})
	assertCode(t, err, "unavailable")
}
