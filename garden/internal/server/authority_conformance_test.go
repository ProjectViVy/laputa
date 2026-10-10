package server

import (
	"net/http"
	"testing"

	"github.com/ProjectViVy/laputa/garden/agentapi"
)

// Exercise the real HTTP mux and capability-token lookup, rather than calling
// the HTTP authorization helper directly. For allowed writes, deliberately
// invalid payloads prove that the request passed authorization without changing
// the fixture's Persona or ACTMEM state.
func TestAuthorityHTTPTokenPrincipalConformsToAgentPolicy(t *testing.T) {
	f := newConformanceFixture(t)
	f.srv.Capabilities.AutodreamToken = "conformance-autodream"

	principals := []struct {
		name      string
		principal agentapi.Principal
		token     string
	}{
		{"read", agentapi.PrincipalRead, "conformance-read"},
		{"user", agentapi.PrincipalUser, "conformance-user"},
		{"agent", agentapi.PrincipalAgent, "conformance-agent"},
		{"autodream", agentapi.PrincipalAutodream, "conformance-autodream"},
		{"operator", agentapi.PrincipalOperator, "conformance-operator"},
	}

	// allowedStatus is a status *after* the authorization gate. This
	// distinguishes a genuine grant from a route that rejects everyone.
	routes := []struct {
		name, method, path string
		operation          agentapi.Operation
		body               any
		allowedStatus      int
	}{
		{"persona metadata", http.MethodGet, "/v2/persona/documents", agentapi.OpPersonaGet, nil, http.StatusOK},
		{"persona explicit document", http.MethodGet, "/v2/persona/documents/world", agentapi.OpPersonaGet, nil, http.StatusOK},
		{"persona history", http.MethodGet, "/v2/persona/history/identity", agentapi.OpPersonaGet, nil, http.StatusOK},
		{"persona history revision", http.MethodGet, "/v2/persona/history/identity/1", agentapi.OpPersonaGet, nil, http.StatusOK},
		{"persona review list", http.MethodGet, "/v2/persona/reviews", agentapi.OpPersonaGet, nil, http.StatusOK},
		{"persona review read", http.MethodGet, "/v2/persona/reviews/missing", agentapi.OpPersonaGet, nil, http.StatusNotFound},
		{"persona proposal", http.MethodPost, "/v2/persona/reviews", agentapi.OpPersonaPropose, map[string]any{}, http.StatusBadRequest},
		{"persona document write", http.MethodPut, "/v2/persona/documents/identity", agentapi.OpPersonaP16, map[string]any{}, http.StatusBadRequest},
		{"persona review approve", http.MethodPost, "/v2/persona/reviews/missing/approve", agentapi.OpPersonaReview, nil, http.StatusNotFound},
		{"persona review reject", http.MethodPost, "/v2/persona/reviews/missing/reject", agentapi.OpPersonaReview, nil, http.StatusNotFound},
		{"persona repair", http.MethodPost, "/v2/persona/repair", agentapi.OpPersonaRepair, map[string]any{}, http.StatusBadRequest},
		{"ACTMEM explicit read", http.MethodGet, "/v2/actmem", agentapi.OpActmemRead, nil, http.StatusOK},
		{"ACTMEM query", http.MethodPost, "/v2/actmem/query", agentapi.OpActmemQuery, map[string]any{"query": "pulse"}, http.StatusOK},
		{"ACTMEM capsules", http.MethodGet, "/v2/actmem/capsules", agentapi.OpActmemRead, nil, http.StatusOK},
		{"ACTMEM capsule read", http.MethodGet, "/v2/actmem/capsules/missing", agentapi.OpActmemRead, nil, http.StatusNotFound},
		{"ACTMEM owner save", http.MethodPut, "/v2/actmem", agentapi.OpActmemSave, map[string]any{}, http.StatusBadRequest},
		{"ACTMEM system append", http.MethodPost, "/v2/actmem/maintenance", agentapi.OpActmemMaintain, map[string]any{"operation": "system_append"}, http.StatusBadRequest},
		{"ACTMEM work patch", http.MethodPost, "/v2/actmem/maintenance", agentapi.OpActmemMaintain, map[string]any{"operation": "work_patch"}, http.StatusBadRequest},
		{"ACTMEM fold session", http.MethodPost, "/v2/actmem/maintenance", agentapi.OpActmemMaintain, map[string]any{"operation": "fold_session"}, http.StatusBadRequest},
		{"ACTMEM capsule delete", http.MethodDelete, "/v2/actmem/capsules/missing", agentapi.OpActmemSave, nil, http.StatusNotFound},
	}
	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			for _, identity := range principals {
				t.Run(identity.name, func(t *testing.T) {
					policy := agentapi.Authorize(f.binding.ProfileID, f.binding, identity.principal, route.operation)
					status, raw := f.wire(t, route.method, route.path, identity.token, route.body, "user")
					if policy != nil {
						if policy.Code != "principal_forbidden" || status != http.StatusForbidden || conformanceWireCode(t, raw) != policy.Code {
							t.Fatalf("policy=%v HTTP status=%d body=%s", policy, status, raw)
						}
						return
					}
					if status != route.allowedStatus {
						t.Fatalf("policy permits %s, HTTP status=%d want=%d body=%s", identity.principal, status, route.allowedStatus, raw)
					}
				})
			}
		})
	}
}
