package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dashimaki/garden/agentapi"
	"github.com/dashimaki/garden/internal/personactx"
	"github.com/dashimaki/garden/internal/recall"
	"github.com/dashimaki/garden/internal/runtimecore"
)

type fastFrozenFixture struct{}

func (fastFrozenFixture) Get(_ context.Context, id string) (personactx.FrozenCore, error) {
	return personactx.FrozenCore{SessionID: id}, nil
}

func TestFastRecallUsesSharedServiceWithOptionalSession(t *testing.T) {
	fast := &recall.FastService{Frozen: fastFrozenFixture{}, Searcher: fakeCardSearcher{}}
	srv := testServer()
	// A route wired only to the old handler's FastRecall pointer cannot pass.
	srv.AgentAPI = agentapi.NewService(&runtimecore.Garden{ProfileID: srv.ProfileID, FastRecall: fast})
	for _, tc := range []struct {
		name, body, frozenID string
		degraded             bool
	}{
		{"session omitted", `{"query":"test","scope":"work","budget_chars":4000}`, "", true},
		{"session supplied", `{"query":"test","scope":"work","budget_chars":4000,"session_id":"session_1"}`, "session_1", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := localRequest(http.MethodPost, "/v2/recall/fast", bytes.NewBufferString(tc.body))
			rec := httptest.NewRecorder()
			srv.HTTPHandler().ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			var got agentapi.ContextView
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Scope != "work" || got.Mode != "fast" || got.BudgetChars != 4000 || got.FrozenCore.SessionID != tc.frozenID || got.Degraded != tc.degraded || len(got.Cards) != 1 || len(got.Evidence) != 1 {
				t.Fatalf("unexpected recall: %+v", got)
			}
			if tc.degraded && (len(got.Warnings) == 0 || !strings.Contains(got.Warnings[0], "session_id missing")) {
				t.Fatalf("missing degraded warning: %+v", got.Warnings)
			}
		})
	}
}

func TestFastRecallRejectsClientIdentityClaims(t *testing.T) {
	fast := &recall.FastService{Searcher: fakeCardSearcher{}}
	srv := testServer()
	srv.AgentAPI = agentapi.NewService(&runtimecore.Garden{ProfileID: srv.ProfileID, FastRecall: fast})
	for _, field := range []string{
		`"binding":{"profile_id":"other","agent_id":"attacker","platform":"external","session_id":"session_1"}`,
		`"profile_id":"other"`, `"agent_id":"attacker"`, `"actor":"operator"`,
	} {
		t.Run(field, func(t *testing.T) {
			req := localRequest(http.MethodPost, "/v2/recall/fast", bytes.NewBufferString(`{"query":"test",`+field+`}`))
			rec := httptest.NewRecorder()
			srv.HTTPHandler().ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestFastRecallRequiresSharedService(t *testing.T) {
	srv := testServer()
	srv.FastRecall = &recall.FastService{Searcher: fakeCardSearcher{}}
	req := localRequest(http.MethodPost, "/v2/recall/fast", bytes.NewBufferString(`{"query":"test"}`))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	var got struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusServiceUnavailable || got.Code != "unavailable" {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
