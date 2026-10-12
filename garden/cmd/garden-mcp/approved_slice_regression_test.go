package main

import (
	"context"
	"encoding/json"
	"github.com/mark3labs/mcp-go/mcp"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestApprovedSliceMCPACTMEMRendersScopedEntries(t *testing.T) {
	// The exact agent shape returned by server.handleActmemRead.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/actmem" {
			t.Error(r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"revision": 2, "updated_at": "", "markdown": "", "truncated": false, "entries": []map[string]any{{"id": "allowed-entry", "body": "PERMITTED_ACTMEM_BODY"}}})
	}))
	defer backend.Close()
	result, err := newMCPServer(NewClient(backend.URL)).GetTool(toolActmemRead).Handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(result)
	if !strings.Contains(string(raw), "PERMITTED_ACTMEM_BODY") {
		t.Fatalf("successful tool result silently omitted permitted entries: %s", raw)
	}
}
