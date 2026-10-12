package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	want := []string{"activity_recent", "actmem_append", "actmem_query", "actmem_read", "evidence_collections", "evidence_get", "evidence_search", "index_health", "memory_search", "persona_get", "persona_status", "record_signal"}
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

func TestMemorySearchUsesBoundedCardsWithoutRawListing(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/materials/cards" || r.URL.Query().Get("query") != "上线日期" {
			t.Errorf("raw path %s", r.URL)
			w.WriteHeader(500)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"cards": []map[string]any{{"id": "shared", "title": "周五上线", "revision": 1}}})
	}))
	defer backend.Close()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "上线日期"}
	result, err := newMCPServer(NewClient(backend.URL)).GetTool(toolMemorySearch).Handler(context.Background(), req)
	if err != nil || result.IsError {
		t.Fatal(result, err)
	}
}

func TestClientUsesPrivateHostCredential(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credential")
	if err := os.WriteFile(path, []byte("host-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GARDEN_HOST_CREDENTIAL_FILE", path)
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
	if got := <-seen; got != "Bearer host-secret" {
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
