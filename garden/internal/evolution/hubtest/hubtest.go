// Package hubtest provides a hermetic mock of the EvoMap GEP-A2A v1.0.0
// surface for automated tests (ADR-0010 §6). It deliberately does not
// import internal/evolution so any package's tests can use it. The live
// hub is exercised only manually through cmd/evomap.
package hubtest

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

type MockHub struct {
	*httptest.Server

	mu sync.Mutex

	NodeSecret        string
	Claimed           bool
	CreditBalance     float64
	HeartbeatStatus   string
	SurvivalStatus    string
	SearchAssets      []map[string]any
	PublishStatus     string
	RejectPublish     bool
	ValidateFailures  []string
	LeakSecretOnError bool

	Messages  []string // message_type of every authenticated POST
	Published [][]map[string]any
}

// New starts a mock hub with sane defaults.
func New() *MockHub {
	m := &MockHub{
		NodeSecret:      "test_secret_" + randHex(8),
		CreditBalance:   100,
		HeartbeatStatus: "ok",
		SurvivalStatus:  "alive",
		PublishStatus:   "quarantine",
	}
	m.Server = httptest.NewServer(http.HandlerFunc(m.handle))
	return m
}

// Asset returns a hub search-result item shaped like the live API.
func Asset(assetType, localID, summary string, gdi float64) map[string]any {
	return map[string]any{
		"asset_type": assetType,
		"local_id":   localID,
		"asset_id":   "sha256:" + randHex(16),
		"title":      summary,
		"rank":       0.9,
		"payload": map[string]any{
			"summary":   summary,
			"gdi_score": gdi,
		},
	}
}

func (m *MockHub) handle(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !strings.HasPrefix(r.URL.Path, "/a2a/") {
		http.NotFound(w, r)
		return
	}
	if r.URL.Path == "/a2a/hello" {
		m.handleHello(w, r)
		return
	}
	if r.URL.Path == "/a2a/leak" {
		// Test-only endpoint that echoes the bearer secret in a 500 body.
		writeErr(w, http.StatusInternalServerError, "leak: "+r.Header.Get("Authorization"))
		return
	}
	if !m.authorized(r) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	switch r.URL.Path {
	case "/a2a/heartbeat":
		m.handleHeartbeat(w, r)
	case "/a2a/assets/search":
		m.handleSearch(w, r)
	case "/a2a/fetch":
		m.handleFetch(w, r)
	case "/a2a/validate":
		m.handleValidate(w, r)
	case "/a2a/publish":
		m.handlePublish(w, r)
	case "/a2a/report":
		writeJSON(w, http.StatusOK, map[string]any{"payload": map[string]any{"status": "accepted"}})
	case "/a2a/revoke":
		// Session-only endpoint: a node_secret A2A client can never
		// authenticate this call (verified against the live hub).
		writeErr(w, http.StatusUnauthorized, "session_only")
	default:
		http.NotFound(w, r)
	}
}

func (m *MockHub) handleHello(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"payload": map[string]any{
		"your_node_id":          "node_test_abcd",
		"node_secret":           m.NodeSecret,
		"claim_url":             m.Server.URL + "/claim/test",
		"claimed":               m.Claimed,
		"heartbeat_interval_ms": 60000,
	}})
}

func (m *MockHub) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"payload": map[string]any{
		"status":          m.HeartbeatStatus,
		"survival_status": m.SurvivalStatus,
		"your_node_id":    "node_test_abcd",
		"credit_balance":  m.CreditBalance,
		"claim_url":       m.Server.URL + "/claim/test",
		"claimed":         m.Claimed,
	}})
}

func (m *MockHub) handleSearch(w http.ResponseWriter, r *http.Request) {
	if len(m.SearchAssets) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"assets": []any{}})
		return
	}
	out := make([]any, len(m.SearchAssets))
	for i, a := range m.SearchAssets {
		out[i] = a
	}
	writeJSON(w, http.StatusOK, map[string]any{"assets": out})
}

func (m *MockHub) handleFetch(w http.ResponseWriter, r *http.Request) {
	out := make([]any, len(m.SearchAssets))
	for i, a := range m.SearchAssets {
		out[i] = a
	}
	writeJSON(w, http.StatusOK, map[string]any{"payload": map[string]any{
		"results": out,
		"mode":    "full_content",
		"credit_cost": map[string]any{
			"kind":                 "fetch_full_content",
			"balance_insufficient": false,
		},
	}})
}

// handleValidate mirrors the live hub: payload.valid + computed_assets on
// success, HTTP 400 validation_error with details when any asset fails.
func (m *MockHub) handleValidate(w http.ResponseWriter, r *http.Request) {
	env := decodeEnvelope(w, r)
	if env == nil {
		return
	}
	rawAssets := assetsOf(env)
	if problem, id := m.assetProblem(rawAssets); problem != "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "validation_error",
			"message": "asset failed validation",
			"details": []any{map[string]any{"path": []string{"payload", "assets"}, "message": problem, "code": "invalid_asset", "asset_id": id}},
		})
		return
	}
	computed := make([]any, 0, len(rawAssets))
	for _, asset := range rawAssets {
		computed = append(computed, map[string]any{"type": asset["type"], "asset_id": asset["asset_id"]})
	}
	writeJSON(w, http.StatusOK, map[string]any{"payload": map[string]any{
		"valid":              true,
		"dry_run":            true,
		"computed_assets":    computed,
		"computed_bundle_id": "bundle_test",
		"estimated_fee":      0,
	}})
}

// handlePublish mirrors the live hub: an immediate decision plus the
// content-addressed asset ids.
func (m *MockHub) handlePublish(w http.ResponseWriter, r *http.Request) {
	env := decodeEnvelope(w, r)
	if env == nil {
		return
	}
	rawAssets := assetsOf(env)
	if problem, id := m.assetProblem(rawAssets); problem != "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "validation_error",
			"message": "asset failed validation",
			"details": []any{map[string]any{"path": []string{"payload", "assets"}, "message": problem, "code": "invalid_asset", "asset_id": id}},
		})
		return
	}
	decision := m.PublishStatus
	if m.RejectPublish {
		decision = "rejected"
	}
	ids := make([]any, 0, len(rawAssets))
	for _, asset := range rawAssets {
		ids = append(ids, asset["asset_id"])
	}
	published := make([]map[string]any, 0, len(rawAssets))
	for _, asset := range rawAssets {
		published = append(published, asset)
	}
	m.Published = append(m.Published, published)
	writeJSON(w, http.StatusOK, map[string]any{"payload": map[string]any{
		"decision":  decision,
		"reason":    "safety_candidate",
		"bundle_id": "bundle_test",
		"asset_ids": ids,
		"hint":      "evolution_event_recommended",
	}})
}

func assetsOf(env map[string]any) []map[string]any {
	payload, _ := env["payload"].(map[string]any)
	raw, _ := payload["assets"].([]any)
	assets := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if asset, ok := item.(map[string]any); ok {
			assets = append(assets, asset)
		}
	}
	return assets
}

// assetProblem returns a validation problem for the first failing asset,
// mirroring the live hub's rule set subset (canonical id, required pair).
func (m *MockHub) assetProblem(assets []map[string]any) (string, string) {
	if len(assets) < 2 {
		return "expected array to have >=2 items", ""
	}
	for _, asset := range assets {
		id, _ := asset["asset_id"].(string)
		if !canonicalOK(asset) {
			return "asset_id does not match canonical content hash", id
		}
		for _, fail := range m.ValidateFailures {
			if fail == id {
				return "asset rejected by test configuration", id
			}
		}
	}
	return "", ""
}

func (m *MockHub) authorized(r *http.Request) bool {
	secret := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return secret == m.NodeSecret
}

func decodeEnvelope(w http.ResponseWriter, r *http.Request) map[string]any {
	var env map[string]any
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		writeErr(w, http.StatusBadRequest, "bad envelope: "+err.Error())
		return nil
	}
	return env
}

func canonicalOK(asset map[string]any) bool {
	clone := make(map[string]any, len(asset))
	for k, v := range asset {
		clone[k] = v
	}
	delete(clone, "asset_id")
	raw, err := json.Marshal(clone)
	if err != nil {
		return false
	}
	sum := sha256.Sum256(raw)
	return asset["asset_id"] == "sha256:"+hex.EncodeToString(sum[:])
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": "error", "message": msg}})
}

func randHex(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
