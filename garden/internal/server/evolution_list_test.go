package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dashimaki/garden/internal/evolution"
)

func TestEvolutionListRunsEmpty(t *testing.T) {
	srv := evolutionTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/v2/evolution/runs", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Items []evolution.EvolutionRun `json:"items"`
		Count int                      `json:"count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Items == nil || len(resp.Items) != 0 || resp.Count != 0 {
		t.Fatalf("resp=%+v, want empty items", resp)
	}
}

func TestEvolutionListRunsReturnsSavedRun(t *testing.T) {
	srv := evolutionTestServer(t)
	run := evolution.EvolutionRun{
		RunID:     "run_listed_1",
		Status:    "completed",
		Provider:  "evomap",
		StartedAt: time.Now().UTC(),
	}
	if err := srv.Evolution.Store.SaveRun(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/v2/evolution/runs", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Items []evolution.EvolutionRun `json:"items"`
		Count int                      `json:"count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Count != 1 || resp.Items[0].RunID != "run_listed_1" {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestEvolutionListRunsCoexistsWithByID(t *testing.T) {
	srv := evolutionTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/v2/evolution/runs/nonexistent", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("by-id route status=%d, want 404", rec.Code)
	}
}

func TestEvolutionListRunsLimitValidation(t *testing.T) {
	srv := evolutionTestServer(t)
	for _, limit := range []string{"0", "101", "abc"} {
		req := httptest.NewRequest(http.MethodGet, "/v2/evolution/runs?limit="+limit, nil)
		rec := httptest.NewRecorder()
		srv.HTTPHandler().ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("limit=%s: status=%d, want 400", limit, rec.Code)
		}
	}
}

func TestEvolutionListProposalsEmpty(t *testing.T) {
	srv := evolutionTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/v2/evolution/proposals", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Items []evolution.EvolutionProposal `json:"items"`
		Count int                           `json:"count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Items == nil || len(resp.Items) != 0 || resp.Count != 0 {
		t.Fatalf("resp=%+v, want empty items", resp)
	}
}

func TestEvolutionListProposalsReturnsSaved(t *testing.T) {
	srv := evolutionTestServer(t)
	proposal := evolution.EvolutionProposal{
		ProposalID: "prop_listed_1",
		RunID:      "run_x",
		Status:     "pending",
		CreatedAt:  time.Now().UTC(),
	}
	if err := srv.Evolution.Store.SaveProposal(t.Context(), proposal); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/v2/evolution/proposals", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Items []evolution.EvolutionProposal `json:"items"`
		Count int                           `json:"count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Count != 1 || resp.Items[0].ProposalID != "prop_listed_1" {
		t.Fatalf("resp=%+v", resp)
	}
}
