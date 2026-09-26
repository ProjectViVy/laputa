package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestAdvertisedToolsAreOnlyTheExistingAdapterSurface(t *testing.T) {
	tools := newMCPServer(NewClient("http://127.0.0.1:7373")).ListTools()
	got := make([]string, 0, len(tools))
	for name := range tools {
		got = append(got, name)
	}
	slices.Sort(got)
	want := []string{"activity_recent", "actmem_append", "actmem_query", "actmem_read", "evidence_collections", "evidence_get", "evidence_search", "index_health", "memory_search", "persona_get", "persona_p16", "persona_propose", "persona_status", "record_signal"}
	if !slices.Equal(got, want) {
		t.Fatalf("advertised tools=%v, want %v", got, want)
	}
}

func TestBackendErrorKeepsGardenEnvelopeInMCPResult(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":"revision_conflict","message":"stale revision","retryable":false,"request_id":"req-42","details":{"expected":3}}`))
	}))
	defer backend.Close()
	tool := newMCPServer(NewClient(backend.URL)).GetTool(toolPersonaStatus)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{})
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	payload, ok := result.StructuredContent.(map[string]any)
	if !ok || payload["code"] != "revision_conflict" || payload["message"] != "stale revision" || payload["retryable"] != false || payload["request_id"] != "req-42" || payload["status"] != 409 {
		t.Fatalf("structured error=%+v", result.StructuredContent)
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok || strings.Contains(text.Text, `{"code"`) || !strings.Contains(text.Text, "stale revision") {
		t.Fatalf("fallback text=%+v", result.Content)
	}
}

func TestBackendUnreachableIsRetryableMCPToolError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	client := NewClient(backend.URL)
	backend.Close()
	result, err := newMCPServer(client).GetTool(toolPersonaStatus).Handler(context.Background(), mcp.CallToolRequest{})
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	payload, ok := result.StructuredContent.(map[string]any)
	if !ok || payload["code"] != "backend_unavailable" || payload["retryable"] != true {
		t.Fatalf("structured error=%+v", result.StructuredContent)
	}
}

func TestBackendErrorDoesNotExposeRawBody(t *testing.T) {
	secret := "private-backend-stack-and-token"
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(secret))
	}))
	defer backend.Close()
	result, err := newMCPServer(NewClient(backend.URL)).GetTool(toolPersonaStatus).Handler(context.Background(), mcp.CallToolRequest{})
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	wire, err := json.Marshal(result)
	if err != nil || strings.Contains(string(wire), secret) || !strings.Contains(string(wire), `"retryable":true`) {
		t.Fatalf("wire=%s err=%v", wire, err)
	}
	if strings.Contains((&HTTPError{Status: 503, Body: secret}).Error(), secret) {
		t.Fatal("HTTPError.Error leaks backend body through degraded output")
	}
}

func TestMemorySearchBothBackendErrorsKeepSeparateStructuredEnvelopes(t *testing.T) {
	secret := "private-backend-stack-and-token"
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/recall/fast":
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"code":"recall_throttled","message":"recall busy","retryable":true,"request_id":"recall-42","private":"` + secret + `"}`))
		case "/v2/memories":
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"code":"memory_conflict","message":"memory conflict","retryable":false,"request_id":"memory-43","private":"` + secret + `"}`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer backend.Close()
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{"query": "recovery"}
	result, err := newMCPServer(NewClient(backend.URL)).GetTool(toolMemorySearch).Handler(context.Background(), request)
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	payload, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structured error=%+v", result.StructuredContent)
	}
	if payload["code"] != "backend_error" || payload["retryable"] != true {
		t.Fatalf("composite error=%+v", payload)
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok || !strings.Contains(text.Text, "error [backend_error]") {
		t.Fatalf("fallback text=%+v", result.Content)
	}
	details, ok := payload["details"].(map[string]any)
	if !ok {
		t.Fatalf("missing backend details: %+v", payload)
	}
	for _, check := range []struct {
		name, code, message, requestID string
		status                         int
		retryable                      bool
	}{
		{"recall", "recall_throttled", "recall busy", "recall-42", 429, true},
		{"memories", "memory_conflict", "memory conflict", "memory-43", 409, false},
	} {
		got, ok := details[check.name].(map[string]any)
		if !ok || got["code"] != check.code || got["message"] != check.message || got["retryable"] != check.retryable || got["request_id"] != check.requestID || got["status"] != check.status {
			t.Errorf("%s error=%+v", check.name, details[check.name])
		}
	}
	wire, err := json.Marshal(result)
	if err != nil || strings.Contains(string(wire), secret) {
		t.Fatalf("wire=%s err=%v", wire, err)
	}
}

func TestFilterMemoryItemsOnlyReturnsLiteralMatches(t *testing.T) {
	items := []memoryItem{
		{ID: "match", Kind: "decision", Content: "The index recovery is canonical."},
		{ID: "miss", Kind: "note", Content: "A completely unrelated note."},
	}
	matched := filterMemoryItems(items, "RECOVERY")
	if len(matched) != 1 || matched[0].ID != "match" {
		t.Fatalf("matched=%+v", matched)
	}
	text := formatMemorySearch("recovery", 5, nil, nil, &memoriesResponse{Items: items}, nil)
	if !strings.Contains(text, "Canonical memories matching query") || !strings.Contains(text, "- [match]") || strings.Contains(text, "- [miss]") {
		t.Fatalf("search output includes an unmatching memory: %s", text)
	}
}

func TestFormatMemorySearchDistinguishesEmptyAndDegraded(t *testing.T) {
	empty := formatMemorySearch("missing", 5, nil, nil, &memoriesResponse{}, nil)
	if !strings.Contains(empty, "status=empty") || strings.Contains(empty, "status=degraded") {
		t.Fatalf("empty output=%s", empty)
	}
	degraded := formatMemorySearch("missing", 5, nil, contextCanceledError{}, &memoriesResponse{}, nil)
	if !strings.Contains(degraded, "status=degraded") || !strings.Contains(degraded, "recall unavailable") {
		t.Fatalf("degraded output=%s", degraded)
	}
}

type contextCanceledError struct{}

func (contextCanceledError) Error() string { return "recall unavailable" }

func TestClientUsesAgentCapabilityToken(t *testing.T) {
	t.Setenv("GARDEN_CAPABILITY_AGENT_TOKEN", "agent-secret")
	seen := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	var response map[string]any
	if err := client.get(context.Background(), "/health", &response); err != nil {
		t.Fatal(err)
	}
	if got := <-seen; got != "Bearer agent-secret" {
		t.Fatalf("authorization=%q", got)
	}
}

func TestOptionalIntegerRejectsFractionsAndOutOfRange(t *testing.T) {
	if _, err := optionalInteger(map[string]any{"limit": 1.5}, "limit", 5, 1, 10); err == nil {
		t.Fatal("fractional limit should fail")
	}
	if _, err := optionalInteger(map[string]any{"limit": 11}, "limit", 5, 1, 10); err == nil {
		t.Fatal("out-of-range limit should fail")
	}
	if got, err := optionalInteger(nil, "limit", 5, 1, 10); err != nil || got != 5 {
		t.Fatalf("default=%d err=%v", got, err)
	}
}
