package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ProjectViVy/laputa/garden/agentapi"
	"github.com/ProjectViVy/laputa/garden/internal/runtimecore"
)

// A configured AgentAPI with no Mentle must fail closed on all three GET routes.
func TestMaterialsGETUsesAgentAPIReadDomain(t *testing.T) {
	s := materialsTestServer(t)
	s.ProfileID = "profile_1"
	s.AgentAPI = agentapi.NewService(&runtimecore.Garden{ProfileID: s.ProfileID})
	for _, path := range []string{"/v2/materials/cards?query=signal", "/v2/materials/cards/card_1/evidence?expected_revision=1", "/v2/materials/collections"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.RemoteAddr = "127.0.0.1:12345"
			s.HTTPHandler().ServeHTTP(rec, req)
			if rec.Code != http.StatusServiceUnavailable {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}
