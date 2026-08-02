package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dashimaki/mentle/facade"
)

type mockMaterials struct {
	cards       facade.CardPage
	evidence    []facade.EvidenceFragment
	collections []facade.CollectionInfo
	err         error
}

func (m *mockMaterials) SearchCards(_ context.Context, _ facade.CardQuery) (facade.CardPage, error) {
	if m.err != nil {
		return facade.CardPage{}, m.err
	}
	return m.cards, nil
}

func (m *mockMaterials) ReadEvidence(_ context.Context, _ facade.EvidenceQuery) ([]facade.EvidenceFragment, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.evidence, nil
}

func (m *mockMaterials) ListCollections(_ context.Context) ([]facade.CollectionInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.collections, nil
}

func materialsTestServer(mat MaterialsProvider) *Server {
	return &Server{Materials: mat, Components: map[string]string{}, Addr: ":0"}
}

func TestMaterialsCardsReturnsResults(t *testing.T) {
	mock := &mockMaterials{cards: facade.CardPage{
		Cards: []facade.MemoryCard{{ID: "mem_1", Title: "Test Card", Summary: "A summary", Status: "active"}},
	}}
	srv := materialsTestServer(mock)
	req := httptest.NewRequest(http.MethodGet, "/v2/materials/cards?query=test", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["source"] != "live" {
		t.Errorf("source=%v, want live", resp["source"])
	}
	cards, ok := resp["cards"].([]any)
	if !ok || len(cards) != 1 {
		t.Fatalf("cards=%v, want 1 item", resp["cards"])
	}
	card := cards[0].(map[string]any)
	if card["id"] != "mem_1" {
		t.Errorf("id=%v", card["id"])
	}
	if _, hasContent := card["content"]; hasContent {
		t.Error("card must not contain content field")
	}
}

func TestMaterialsCardsRequiresQuery(t *testing.T) {
	srv := materialsTestServer(&mockMaterials{})
	req := httptest.NewRequest(http.MethodGet, "/v2/materials/cards", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
}

func TestMaterialsCardsLimitValidation(t *testing.T) {
	srv := materialsTestServer(&mockMaterials{})
	for _, limit := range []string{"0", "200", "abc"} {
		req := httptest.NewRequest(http.MethodGet, "/v2/materials/cards?query=x&limit="+limit, nil)
		rec := httptest.NewRecorder()
		srv.HTTPHandler().ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("limit=%s: status=%d, want 400", limit, rec.Code)
		}
	}
}

func TestMaterialsCardsNilProvider(t *testing.T) {
	srv := materialsTestServer(nil)
	req := httptest.NewRequest(http.MethodGet, "/v2/materials/cards?query=test", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want 503", rec.Code)
	}
}

func TestMaterialsEvidenceReturnsFragments(t *testing.T) {
	mock := &mockMaterials{evidence: []facade.EvidenceFragment{
		{CardID: "mem_1", Excerpt: "bounded text", ContentHash: "abc123", Validity: "active"},
	}}
	srv := materialsTestServer(mock)
	req := httptest.NewRequest(http.MethodGet, "/v2/materials/cards/mem_1/evidence", nil)
	req.SetPathValue("id", "mem_1")
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["source"] != "live" {
		t.Errorf("source=%v, want live", resp["source"])
	}
	if resp["card_id"] != "mem_1" {
		t.Errorf("card_id=%v", resp["card_id"])
	}
	fragments, ok := resp["fragments"].([]any)
	if !ok || len(fragments) != 1 {
		t.Fatalf("fragments=%v", resp["fragments"])
	}
}

func TestMaterialsEvidenceNilProvider(t *testing.T) {
	srv := materialsTestServer(nil)
	req := httptest.NewRequest(http.MethodGet, "/v2/materials/cards/mem_1/evidence", nil)
	req.SetPathValue("id", "mem_1")
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want 503", rec.Code)
	}
}

func TestMaterialsCollectionsReturnsList(t *testing.T) {
	mock := &mockMaterials{collections: []facade.CollectionInfo{
		{Name: "architecture", Count: 5},
		{Name: "debugging", Count: 3},
	}}
	srv := materialsTestServer(mock)
	req := httptest.NewRequest(http.MethodGet, "/v2/materials/collections", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["source"] != "live" {
		t.Errorf("source=%v, want live", resp["source"])
	}
	collections, ok := resp["collections"].([]any)
	if !ok || len(collections) != 2 {
		t.Fatalf("collections=%v", resp["collections"])
	}
}

func TestMaterialsCollectionsNilProvider(t *testing.T) {
	srv := materialsTestServer(nil)
	req := httptest.NewRequest(http.MethodGet, "/v2/materials/collections", nil)
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want 503", rec.Code)
	}
}

func TestMaterialsEndpointsAreReadOnly(t *testing.T) {
	srv := materialsTestServer(&mockMaterials{})
	paths := []string{
		"/v2/materials/cards?query=x",
		"/v2/materials/collections",
	}
	for _, path := range paths {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
			req := httptest.NewRequest(method, path, nil)
			rec := httptest.NewRecorder()
			srv.HTTPHandler().ServeHTTP(rec, req)
			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s %s: status=%d, want 405", method, path, rec.Code)
			}
		}
	}
}
