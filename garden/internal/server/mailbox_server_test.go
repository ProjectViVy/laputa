package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ProjectViVy/laputa/garden/internal/mailbox"
)

func newMailboxTestServer(t *testing.T) (*Server, *mailbox.Store) {
	t.Helper()
	store, err := mailbox.OpenStore(filepath.Join(t.TempDir(), "mailbox.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return &Server{Mailbox: store, Capabilities: CapabilityConfig{UserToken: "user-secret"}}, store
}

func TestMailboxHandlersReturn503WhenUnavailable(t *testing.T) {
	srv := &Server{}
	for _, path := range []string{"/v2/mailbox/inbox", "/v2/mailbox/outbox", "/v2/mailbox/dead-letter"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.HTTPHandler().ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s code=%d", path, rec.Code)
		}
	}
}

func TestMailboxInboxListAndReview(t *testing.T) {
	srv, store := newMailboxTestServer(t)
	ctx := context.Background()
	item, err := store.AddInbox(ctx, map[string]any{"kind": "proposal"}, []string{"mentle:cards/abc"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v2/mailbox/inbox", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list code=%d", rec.Code)
	}
	var listed struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil || listed.Count != 1 {
		t.Fatalf("listed=%+v err=%v", listed, err)
	}

	req = httptest.NewRequest(http.MethodPost, "/v2/mailbox/items/"+item.ID+"/approve", bytes.NewBufferString(`{"reason":"checked"}`))
	req.SetPathValue("id", item.ID)
	req.Header.Set("Authorization", "Bearer user-secret")
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve code=%d body=%s", rec.Code, rec.Body.String())
	}
	var approved mailbox.Item
	if err := json.Unmarshal(rec.Body.Bytes(), &approved); err != nil || approved.State != mailbox.StateApproved {
		t.Fatalf("approved=%+v err=%v", approved, err)
	}

	req = httptest.NewRequest(http.MethodPost, "/v2/mailbox/items/"+item.ID+"/reject", bytes.NewBufferString(`{}`))
	req.SetPathValue("id", item.ID)
	req.Header.Set("Authorization", "Bearer user-secret")
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("second review code=%d", rec.Code)
	}
}

func TestMailboxApproveUnknownItemIs404(t *testing.T) {
	srv, _ := newMailboxTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/v2/mailbox/items/mbx_missing/approve", bytes.NewBufferString(`{}`))
	req.SetPathValue("id", "mbx_missing")
	req.Header.Set("Authorization", "Bearer user-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}
