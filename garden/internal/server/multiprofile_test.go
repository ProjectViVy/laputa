package server

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/ProjectViVy/laputa/garden/agentapi"
	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMultiProfileRouteMatrix(t *testing.T) {
	s, code := installationServer(t)
	ctx := context.Background()
	cookie := pairOwner(t, s, code)
	records := map[string]string{}
	for _, id := range []string{"A", "B"} {
		g, _ := s.Owner.Subject(ctx, id)
		if _, e := g.Actmem.AppendEntry(evolution.Entry{Section: evolution.SectionPulse, Scope: evolution.Scope{SubjectID: id, Kind: evolution.ScopePersonal}, SessionID: "chat", Body: id + "_ACTMEM_PRIVATE"}); e != nil {
			t.Fatal(e)
		}
		c, err := s.Installation.BindProfile(id, agentapi.ClientConfig{AgentID: "seed", Platform: "test", Principal: agentapi.PrincipalUser})
		if err != nil {
			t.Fatal(err)
		}
		b, _ := c.BindSession("seed")
		r, err := b.MutateMemory(ctx, agentapi.MemoryTarget{}, memory.AuthorizedMutation{OperationID: "seed-" + id, Operation: evolution.MutationCreate, ExpectedAbsent: true, Body: id + "_PRIVATE_SENTINEL", Inference: evolution.InferenceObserved})
		if err != nil {
			t.Fatal(err)
		}
		records[id] = r.TargetRef
	}
	host, token, err := s.Installation.CreateHostBinding(ctx, agentapi.HostBinding{ProfileID: "A", AgentID: "host", Platform: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Installation.SetHostControls(ctx, host.BindingID, agentapi.HostControls{Recall: true, PersonaSync: true, Capture: true}); err != nil {
		t.Fatal(err)
	}
	routes := []struct {
		method, path, body string
		status             int
	}{
		{"GET", "/v2/memories", "", 200},
		{"GET", "/v2/memories/" + records["B"], "", 404},
		{"PATCH", "/v2/memories/" + records["B"], `{"content":"attempt","expected_version":1,"reason":"correction"}`, 404},
		{"DELETE", "/v2/memories/" + records["B"], `{"expected_version":1,"reason":"cleanup"}`, 404},
		{"POST", "/v2/memories", `{"content":"own HTTP fact","kind":"fact","tags":["scope-check"]}`, 201},
		{"GET", "/v2/persona/documents/identity", "", 200},
		{"GET", "/v2/actmem", "", 200},
		{"POST", "/v2/actmem/query", `{"query":"PRIVATE"}`, 200},
		{"GET", "/v2/materials/cards?query=SENTINEL", "", 200},
		{"GET", "/v2/materials/collections", "", 200},
		{"GET", "/v2/materials/cards/" + records["B"] + "/evidence?expected_revision=1", "", 404},
		{"POST", "/v2/recall/fast", `{"query":"SENTINEL","session_id":"chat"}`, 200},
		{"POST", "/v2/recall/bootstrap", `{"session_id":"chat","intent":"SENTINEL"}`, 200},
		{"GET", "/v2/reports", "", 503}, {"GET", "/v2/pipelines", "", 503}, {"GET", "/v2/admin/spool", "", 503}, {"GET", "/v2/recall/traces/foreign", "", 503},
	}
	for _, owner := range []bool{false, true} {
		for _, row := range routes {
			r := httptest.NewRequest(row.method, "http://localhost"+row.path, bytes.NewBufferString(row.body))
			r.Header.Set("X-Garden-Profile", "A")
			r.Header.Set("Origin", "http://localhost")
			r.Header.Set("Content-Type", "application/json")
			if owner {
				r.AddCookie(cookie)
			} else {
				r.Header.Set("Authorization", "Bearer "+token)
			}
			w := httptest.NewRecorder()
			s.HTTPHandler().ServeHTTP(w, r)
			if w.Code != row.status || strings.Contains(w.Body.String(), "B_PRIVATE_SENTINEL") || strings.Contains(w.Body.String(), "B_ACTMEM_PRIVATE") {
				t.Errorf("owner=%t %s: %d %s", owner, row.path, w.Code, w.Body.String())
			}
			if row.path == "/v2/memories" && w.Code == 200 {
				var body map[string]any
				if err = json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	r := httptest.NewRequest("GET", "http://localhost/v2/memories", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("X-Garden-Profile", "B")
	w := httptest.NewRecorder()
	s.HTTPHandler().ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("host profile spoof", w.Code, w.Body.String())
	}
}
