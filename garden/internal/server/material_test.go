package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dashimaki/garden/agentapi"
	"github.com/dashimaki/garden/internal/runtimecore"
	"github.com/dashimaki/garden/memory"
	"github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/mentle/facade"
)

func materialsTestServer(t *testing.T) *Server {
	t.Helper()
	srv, _ := materialsTestServerMentle(t)
	return srv
}

func materialsTestServerMentle(t *testing.T) (*Server, *facade.Service) {
	t.Helper()
	dir := t.TempDir()
	m, err := facade.OpenCatalogService(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Close() })
	return &Server{ProfileID: "profile_1", AgentAPI: agentapi.NewService(&runtimecore.Garden{ProfileID: "profile_1", Mentle: m})}, m
}

func materialsSeedCard(t *testing.T, m *facade.Service, body string) facade.Memory {
	t.Helper()
	rec, err := m.CreateMemory(context.Background(), facade.CreateMemoryRequest{
		Content: body, Kind: "note",
		Scope:  memory.EncodeScope(evolution.Scope{SubjectID: "profile_1", Kind: evolution.ScopePersonal}),
		Source: facade.MemorySource{Type: "agent"},
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func materialRequest(s *Server, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	r.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()
	s.HTTPHandler().ServeHTTP(rec, r)
	return rec
}

func TestMaterialsCardsPreserveWireKeys(t *testing.T) {
	rec := materialRequest(materialsTestServer(t), "/v2/materials/cards?query=signal")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"cards", "next_cursor", "source"} {
		if _, ok := result[key]; !ok {
			t.Errorf("missing %s", key)
		}
	}
	if result["source"] != "live" || len(result) != 3 {
		t.Fatalf("unexpected response %v", result)
	}
}

func TestMaterialsEvidencePreserveWireKeys(t *testing.T) {
	srv, m := materialsTestServerMentle(t)
	rec0 := materialsSeedCard(t, m, "evidence body")
	rec := materialRequest(srv, "/v2/materials/cards/"+rec0.ID+"/evidence?expected_revision=1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"card_id", "fragments", "source"} {
		if _, ok := result[key]; !ok {
			t.Errorf("missing %s", key)
		}
	}
	if result["card_id"] != rec0.ID || result["source"] != "live" || len(result) != 3 {
		t.Fatalf("unexpected response %v", result)
	}
}

func TestMaterialsCardsQueryValidation(t *testing.T) {
	srv := materialsTestServer(t)
	for _, path := range []string{"/v2/materials/cards", "/v2/materials/cards?query=x&limit=0", "/v2/materials/cards?query=x&limit=101", "/v2/materials/cards?query=x&limit=abc"} {
		rec := materialRequest(srv, path)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
	for _, path := range []string{"/v2/materials/cards?query=x&limit=1", "/v2/materials/cards?query=x&limit=100", "/v2/materials/cards?query=x&limit=", "/v2/materials/cards?query=+++"} {
		rec := materialRequest(srv, path)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestMaterialsEvidenceBudgetFallback(t *testing.T) {
	srv, m := materialsTestServerMentle(t)
	rec0 := materialsSeedCard(t, m, "budget body")
	for _, path := range []string{"/v2/materials/cards/" + rec0.ID + "/evidence?expected_revision=1&per_item_budget=bad&total_budget=0", "/v2/materials/cards/" + rec0.ID + "/evidence?expected_revision=1&per_item_budget=4001&total_budget=16001", "/v2/materials/cards/" + rec0.ID + "/evidence?expected_revision=1&per_item_budget=1&total_budget=1"} {
		rec := materialRequest(srv, path)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
	// A card id that does not exist reports not_found, never a 2xx.
	rec := materialRequest(srv, "/v2/materials/cards/%20/evidence?expected_revision=1")
	if rec.Code != http.StatusNotFound {
		t.Errorf("nonexistent card: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMaterialsCollectionsUnavailableWhenMentleGraphMissing(t *testing.T) {
	rec := materialRequest(materialsTestServer(t), "/v2/materials/collections")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMaterialsMissingAgentAPIIsUnavailable(t *testing.T) {
	s := materialsTestServer(t)
	s.AgentAPI = nil
	for _, path := range []string{"/v2/materials/cards?query=signal", "/v2/materials/cards/card_1/evidence?expected_revision=1", "/v2/materials/collections"} {
		rec := materialRequest(s, path)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s: status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestMaterialsGETLoopbackReadPolicy(t *testing.T) {
	s := materialsTestServer(t)
	for _, path := range []string{"/v2/materials/cards?query=signal", "/v2/materials/cards/card_1/evidence?expected_revision=1", "/v2/materials/collections"} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		r.RemoteAddr = "192.0.2.1:1234"
		rec := httptest.NewRecorder()
		s.HTTPHandler().ServeHTTP(rec, r)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s remote status=%d", path, rec.Code)
		}
	}
}

func TestMaterialsEndpointsAreReadOnly(t *testing.T) {
	srv := materialsTestServer(t)
	for _, path := range []string{"/v2/materials/cards?query=x", "/v2/materials/cards/card_1/evidence?expected_revision=1", "/v2/materials/collections"} {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
			req := httptest.NewRequest(method, path, nil)
			rec := httptest.NewRecorder()
			srv.HTTPHandler().ServeHTTP(rec, req)
			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s %s: status=%d", method, path, rec.Code)
			}
		}
	}
}
