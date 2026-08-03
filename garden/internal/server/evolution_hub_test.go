package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/dashimaki/garden/internal/evolution"
	"github.com/dashimaki/garden/internal/evolution/hubtest"
	"github.com/dashimaki/garden/internal/recall"
	"github.com/dashimaki/laputa/governance"
)

func evomapTestServer(t *testing.T, hub *hubtest.MockHub) (*Server, *evolution.Store) {
	t.Helper()
	dir := t.TempDir()
	store, err := governance.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	engine := governance.NewEngine(store)
	_ = engine.Initialize(context.Background())

	evoStore, err := evolution.OpenStore(filepath.Join(dir, "evo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { evoStore.Close() })
	evoEvents, err := evolution.OpenEventStore(filepath.Join(dir, "evo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { evoEvents.Close() })

	client, err := evolution.OpenHubClient(evolution.HubClientOptions{
		BaseURL:   hub.URL,
		CredsPath: filepath.Join(dir, "node.json"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.EnsureRegistered(context.Background()); err != nil {
		t.Fatal(err)
	}
	provider := evolution.NewEvoMapProvider(client, evoStore, evolution.DefaultProviderLimits(), false)
	evoService := &evolution.Service{Provider: provider, Store: evoStore, Events: evoEvents, Hub: evolution.DefaultHubPolicy()}
	return &Server{
		FastRecall: &recall.FastService{Gov: engine, Searcher: fakeCardSearcher{}},
		Evolution:  evoService,
		Addr:       ":0",
	}, evoStore
}

func TestEvolutionHubStatusEndpoint(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	srv, _ := evomapTestServer(t, hub)

	req := httptest.NewRequest(http.MethodGet, "/v2/evolution/hub/status", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["node_id"] != "node_test_abcd" {
		t.Fatalf("node_id=%v", resp["node_id"])
	}
	if resp["credit_balance"] != float64(100) {
		t.Fatalf("credit_balance=%v", resp["credit_balance"])
	}
	if resp["survival_status"] != "alive" {
		t.Fatalf("survival_status=%v", resp["survival_status"])
	}
}

func TestEvolutionHubStatusUnavailable(t *testing.T) {
	srv := evolutionTestServer(t) // provider is nil
	req := httptest.NewRequest(http.MethodGet, "/v2/evolution/hub/status", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestEvolutionRunFlowViaHTTP(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	srv, evoStore := evomapTestServer(t, hub)

	body := `{"trigger":"test failure in recall pipeline","outcome":"fast recall failed to rank the expected card","trace_ref":"trace_1","evidence_refs":["ref_1"],"policy":{"publication_allowed":false}}`
	req := httptest.NewRequest(http.MethodPost, "/v2/evolution/runs", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("start status=%d body=%s", rec.Code, rec.Body.String())
	}
	var run evolution.EvolutionRun
	if err := json.NewDecoder(rec.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	if run.RunID == "" || run.Provider != "evomap" {
		t.Fatalf("run=%+v", run)
	}

	req = httptest.NewRequest(http.MethodGet, "/v2/evolution/runs/"+run.RunID, nil)
	req.SetPathValue("run_id", run.RunID)
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got evolution.EvolutionRun
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "completed" {
		t.Fatalf("status=%s, want completed", got.Status)
	}

	// Proposals still flow through the standard endpoints once a
	// candidate exists in the store.
	candidate := evolution.GeneCandidate{CandidateID: "cand_http_1", RunID: run.RunID, Kind: "gene", Name: "hub_discovered", EvidenceRefs: []string{}}
	if err := evoStore.SaveCandidate(context.Background(), candidate); err != nil {
		t.Fatal(err)
	}
	proposalBody := `{"run_id":"` + run.RunID + `","candidate_id":"cand_http_1"}`
	req = httptest.NewRequest(http.MethodPost, "/v2/evolution/proposals", bytes.NewBufferString(proposalBody))
	rec = httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("proposal status=%d body=%s", rec.Code, rec.Body.String())
	}
}
