package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/ProjectViVy/laputa/garden/agentapi"
	"github.com/ProjectViVy/laputa/garden/internal/runtimecore"
	"github.com/ProjectViVy/laputa/mentle/facade"
)

func healthRequest(t *testing.T, srv *Server, path string, status int) map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(rec, localRequest(http.MethodGet, path, nil))
	if rec.Code != status {
		t.Fatalf("%s: status=%d body=%s", path, rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestAdminIndexHealthRequiresSharedServiceNotLegacyFacade(t *testing.T) {
	srv := testServer()
	srv.Facade = &facade.Service{}
	body := healthRequest(t, srv, "/v2/admin/index-health", http.StatusServiceUnavailable)
	if body["code"] != "index_health_unavailable" {
		t.Fatalf("envelope=%v", body)
	}
	details, ok := body["details"].(map[string]any)
	if !ok || !slices.Equal(details["reasons"].([]any), []any{"canonical_probe_failed"}) {
		t.Fatalf("details=%v", details)
	}
}

func TestAdminIndexHealthAndAggregatesShareLiveUnavailableProbe(t *testing.T) {
	srv := testServer()
	// A nil legacy facade proves that every response comes from the shared service.
	srv.AgentAPI = agentapi.NewService(&runtimecore.Garden{ProfileID: srv.ProfileID, Mentle: &facade.Service{}})
	body := healthRequest(t, srv, "/v2/admin/index-health", http.StatusServiceUnavailable)
	if body["code"] != "index_health_unavailable" {
		t.Fatalf("health=%v", body)
	}
	details := body["details"].(map[string]any)
	reasons, ok := details["reasons"].([]any)
	if !ok || !slices.Contains(reasons, "canonical_probe_failed") {
		t.Fatalf("health=%v", body)
	}
	for _, route := range []string{"/v2/admin/overview", "/v2/admin/components", "/health"} {
		aggregate := healthRequest(t, srv, route, http.StatusOK)
		health, ok := aggregate["index_health"].(map[string]any)
		if !ok || health["status"] != "unavailable" || health["observed_at"] == nil {
			t.Fatalf("%s: %v", route, aggregate)
		}
		if route == "/v2/admin/overview" || route == "/health" {
			if aggregate["status"] != "degraded" {
				t.Fatalf("%s: %v", route, aggregate)
			}
		} else {
			found := false
			for _, entry := range aggregate["components"].([]any) {
				c := entry.(map[string]any)
				if c["name"] == "mentle" && c["status"] == "unavailable" {
					found = true
				}
			}
			if !found {
				t.Fatalf("%s: %v", route, aggregate)
			}
		}
	}
}
