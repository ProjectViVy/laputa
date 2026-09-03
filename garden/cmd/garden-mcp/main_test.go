package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
