package evolution

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultHubURL = "https://evomap.ai"

var (
	ErrNoCredentials  = errors.New("evomap: no node credentials (run 'hello' or set GARDEN_EVOMAP_CREDS)")
	ErrUnauthorized   = errors.New("evomap: hub rejected node_secret (401)")
	ErrHubUnreachable = errors.New("evomap: hub unreachable")
)

// HubClient is a bounded GEP-A2A v1.0.0 client for the EvoMap Hub
// (ADR-0010). It is a pure HTTP protocol client: it bundles no Evolver
// code and never calls session-only endpoints (revoke/decision/asset
// management are account-page operations).
type HubClient struct {
	BaseURL   string
	CredsPath string
	HTTP      *http.Client

	creds *credentials
}

// HubClientOptions configures OpenHubClient. Zero values get defaults.
type HubClientOptions struct {
	BaseURL   string
	CredsPath string
	Timeout   time.Duration
}

type credentials struct {
	NodeID     string `json:"node_id"`
	NodeSecret string `json:"node_secret"`
	ClaimURL   string `json:"claim_url"`
	Claimed    bool   `json:"claimed"`
}

type envelope struct {
	Protocol        string         `json:"protocol"`
	ProtocolVersion string         `json:"protocol_version"`
	MessageType     string         `json:"message_type"`
	MessageID       string         `json:"message_id"`
	SenderID        string         `json:"sender_id,omitempty"`
	Timestamp       string         `json:"timestamp"`
	Payload         map[string]any `json:"payload"`
}

// OpenHubClient loads existing credentials (missing file is not an error;
// methods that need a node return ErrNoCredentials).
func OpenHubClient(opts HubClientOptions) (*HubClient, error) {
	base := opts.BaseURL
	if base == "" {
		base = defaultHubURL
	}
	credsPath := opts.CredsPath
	if credsPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		credsPath = filepath.Join(home, ".evomap", "node.json")
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	c := &HubClient{BaseURL: strings.TrimRight(base, "/"), CredsPath: credsPath, HTTP: &http.Client{Timeout: timeout}}
	if raw, err := os.ReadFile(credsPath); err == nil {
		var creds credentials
		if err := json.Unmarshal(raw, &creds); err == nil && creds.NodeID != "" && creds.NodeSecret != "" {
			c.creds = &creds
		}
	}
	return c, nil
}

// HasCredentials reports whether node credentials are loaded.
func (c *HubClient) HasCredentials() bool {
	return c != nil && c.creds != nil
}

// EnsureRegistered registers a node when no credentials exist yet and
// returns the claim URL. With credentials present it is a no-op.
func (c *HubClient) EnsureRegistered(ctx context.Context) (string, error) {
	if c.creds != nil {
		return "", nil
	}
	resp, err := c.post(ctx, "/a2a/hello", "hello", map[string]any{}, nil)
	if err != nil {
		return "", err
	}
	payload, _ := resp["payload"].(map[string]any)
	if payload == nil {
		return "", fmt.Errorf("evomap hello: response missing payload")
	}
	secret, _ := payload["node_secret"].(string)
	if secret == "" {
		return "", errors.New("evomap hello: response missing node_secret")
	}
	creds := &credentials{
		NodeID:     asString(payload["your_node_id"]),
		NodeSecret: secret,
		ClaimURL:   asString(payload["claim_url"]),
		Claimed:    payload["claimed"] == true,
	}
	if creds.NodeID == "" {
		return "", errors.New("evomap hello: response missing node id")
	}
	if err := c.saveCredentials(creds); err != nil {
		return "", err
	}
	c.creds = creds
	return creds.ClaimURL, nil
}

// HeartbeatStatus is the parsed /a2a/heartbeat payload.
type HeartbeatStatus struct {
	Status         string
	SurvivalStatus string
	NodeID         string
	CreditBalance  float64
	HasBalance     bool
	ClaimURL       string
	Claimed        bool
}

func (c *HubClient) Heartbeat(ctx context.Context) (HeartbeatStatus, error) {
	creds, err := c.requireCredentials()
	if err != nil {
		return HeartbeatStatus{}, err
	}
	resp, err := c.post(ctx, "/a2a/heartbeat", "heartbeat", map[string]any{"node_id": creds.NodeID}, creds)
	if err != nil {
		return HeartbeatStatus{}, err
	}
	payload, _ := resp["payload"].(map[string]any)
	if payload == nil {
		payload = resp
	}
	return HeartbeatStatus{
		Status:         asString(payload["status"]),
		SurvivalStatus: asString(payload["survival_status"]),
		NodeID:         asString(payload["your_node_id"]),
		CreditBalance:  asFloat(payload["credit_balance"]),
		HasBalance:     payload["credit_balance"] != nil,
		ClaimURL:       asString(payload["claim_url"]),
		Claimed:        payload["claimed"] == true,
	}, nil
}

// HubAsset is one search/fetch result item.
type HubAsset struct {
	AssetType string
	LocalID   string
	AssetID   string
	Title     string
	Summary   string
	GDI       float64
	Rank      float64
	Payload   map[string]any
}

// Search queries the hub's free discovery endpoint by signal.
func (c *HubClient) Search(ctx context.Context, signals []string, limit int) ([]HubAsset, error) {
	creds, err := c.requireCredentials()
	if err != nil {
		return nil, err
	}
	if len(signals) == 0 {
		return nil, errors.New("evomap search: at least one signal is required")
	}
	if limit <= 0 {
		limit = 5
	}
	endpoint := "/a2a/assets/search?signals=" + url.QueryEscape(strings.Join(signals, ",")) + "&limit=" + fmt.Sprintf("%d", limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+creds.NodeSecret)
	req.Header.Set("x-correlation-id", newMessageID())
	raw, err := c.do(req)
	if err != nil {
		return nil, err
	}
	var out struct {
		Assets []map[string]any `json:"assets"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	assets := make([]HubAsset, 0, len(out.Assets))
	for _, item := range out.Assets {
		payload, _ := item["payload"].(map[string]any)
		assets = append(assets, HubAsset{
			AssetType: asString(item["asset_type"]),
			LocalID:   asString(item["local_id"]),
			AssetID:   asString(item["asset_id"]),
			Title:     asString(item["title"]),
			Summary:   asString(payload["summary"]),
			GDI:       asFloat(payload["gdi_score"]),
			Rank:      asFloat(item["rank"]),
			Payload:   payload,
		})
	}
	return assets, nil
}

// Fetch retrieves assets from the hub. The full-content tier consumes
// credits; it is intended for explicit CLI use only (ADR-0010 §2.2).
func (c *HubClient) Fetch(ctx context.Context, assetType string, includeTasks bool) ([]HubAsset, error) {
	creds, err := c.requireCredentials()
	if err != nil {
		return nil, err
	}
	resp, err := c.post(ctx, "/a2a/fetch", "fetch", map[string]any{
		"asset_type":    assetType,
		"include_tasks": includeTasks,
	}, creds)
	if err != nil {
		return nil, err
	}
	payload, _ := resp["payload"].(map[string]any)
	if payload == nil {
		payload = resp
	}
	results, _ := payload["results"].([]any)
	assets := make([]HubAsset, 0, len(results))
	for _, r := range results {
		item, _ := r.(map[string]any)
		inner, _ := item["payload"].(map[string]any)
		assets = append(assets, HubAsset{
			AssetType: asString(item["asset_type"]),
			LocalID:   asString(item["local_id"]),
			AssetID:   asString(item["asset_id"]),
			Title:     asString(item["title"]),
			Summary:   asString(inner["summary"]),
			Payload:   inner,
		})
	}
	return assets, nil
}

// PublishSummary is the per-asset outcome of validate/publish.
type PublishSummary struct {
	Total    int
	Accepted int
	Rejected int
	Statuses map[string]string // asset_id -> quarantine|candidate|rejected|...
}

// Validate checks an asset bundle against hub schema rules (free).
// The live hub answers with payload.valid + computed_assets, or HTTP 400
// with a validation_error body when any asset fails its rules.
func (c *HubClient) Validate(ctx context.Context, assets []map[string]any) (PublishSummary, error) {
	return c.assetRoundTrip(ctx, "/a2a/validate", "validate", assets)
}

// Publish submits an asset bundle (Gene+Capsule pairs) to the hub.
// The live hub answers with an immediate decision (quarantine/candidate/
// accepted/rejected) plus the content-addressed asset ids.
func (c *HubClient) Publish(ctx context.Context, assets []map[string]any) (PublishSummary, error) {
	return c.assetRoundTrip(ctx, "/a2a/publish", "publish", assets)
}

func (c *HubClient) assetRoundTrip(ctx context.Context, endpoint, messageType string, assets []map[string]any) (PublishSummary, error) {
	creds, err := c.requireCredentials()
	if err != nil {
		return PublishSummary{}, err
	}
	status, raw, err := c.postRaw(ctx, endpoint, messageType, map[string]any{"assets": assets}, creds)
	if err != nil {
		return PublishSummary{}, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return PublishSummary{}, err
	}
	if status != http.StatusOK {
		return PublishSummary{}, hubError(endpoint, out, creds)
	}
	payload, _ := out["payload"].(map[string]any)
	if payload == nil {
		return PublishSummary{}, fmt.Errorf("evomap %s: response missing payload", endpoint)
	}
	if endpoint == "/a2a/publish" {
		return parsePublishDecision(payload, assets)
	}
	return parseValidateResult(payload, assets)
}

// parseValidateResult handles the live validate payload: valid + dry_run +
// computed_assets (no per-asset statuses). Strict parse: an unrecognized
// shape is an error, not a silent empty summary.
func parseValidateResult(payload map[string]any, assets []map[string]any) (PublishSummary, error) {
	valid, _ := payload["valid"].(bool)
	computed, _ := payload["computed_assets"].([]any)
	ids := make([]string, 0, len(computed))
	for _, c := range computed {
		item, _ := c.(map[string]any)
		if id := asString(item["asset_id"]); id != "" {
			ids = append(ids, id)
		}
	}
	summary := PublishSummary{Total: len(assets), Statuses: map[string]string{}}
	if len(ids) == 0 {
		return summary, fmt.Errorf("evomap validate: unexpected response shape (missing computed_assets)")
	}
	for _, id := range ids {
		summary.Statuses[id] = "valid"
	}
	if !valid {
		summary.Rejected = summary.Total
		summary.Accepted = 0
		return summary, nil
	}
	summary.Accepted = summary.Total
	return summary, nil
}

// parsePublishDecision handles the live publish payload: decision +
// bundle_id + asset_ids.
func parsePublishDecision(payload map[string]any, assets []map[string]any) (PublishSummary, error) {
	decision := asString(payload["decision"])
	ids := assetIDStrings(payload["asset_ids"])
	summary := PublishSummary{Total: len(ids), Statuses: map[string]string{}}
	if len(ids) == 0 {
		return summary, fmt.Errorf("evomap publish: unexpected response shape (missing asset_ids)")
	}
	for _, id := range ids {
		summary.Statuses[id] = decision
	}
	if decision == "rejected" {
		summary.Rejected = len(ids)
	} else {
		summary.Accepted = len(ids)
	}
	return summary, nil
}

// hubError converts a non-200 hub body ({error, message, details}) into a
// descriptive error; the node_secret is redacted from every string.
func hubError(endpoint string, out map[string]any, creds *credentials) error {
	code := asString(out["error"])
	message := asString(out["message"])
	if inner, ok := out["error"].(map[string]any); ok {
		code = asString(inner["code"])
		if message == "" {
			message = asString(inner["message"])
		}
	}
	if code == "" && message == "" {
		return fmt.Errorf("evomap %s: HTTP error response", endpoint)
	}
	detail := ""
	if details, _ := out["details"].([]any); len(details) > 0 {
		if first, _ := details[0].(map[string]any); first != nil {
			detail = ": " + asString(first["message"])
		}
	}
	return fmt.Errorf("evomap %s: %s: %s%s", endpoint, code, redactSecret(creds, message), redactSecret(creds, detail))
}

func assetIDStrings(v any) []string {
	raw, _ := v.([]any)
	ids := make([]string, 0, len(raw))
	for _, item := range raw {
		if id := asString(item); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// Report submits a validation result for a published asset.
func (c *HubClient) Report(ctx context.Context, assetID, status string) (map[string]any, error) {
	creds, err := c.requireCredentials()
	if err != nil {
		return nil, err
	}
	if status == "" {
		status = "success"
	}
	return c.post(ctx, "/a2a/report", "report", map[string]any{
		"asset_id":          assetID,
		"validation_report": map[string]any{"status": status},
	}, creds)
}

// CanonicalHash implements the hub's content-addressed asset id:
// sha256 of the canonical JSON (sorted keys at all levels, no asset_id
// field). encoding/json sorts map keys, which matches the canonical form.
func CanonicalHash(asset map[string]any) (string, error) {
	raw, err := json.Marshal(asset)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (c *HubClient) requireCredentials() (*credentials, error) {
	if c == nil || c.creds == nil {
		return nil, ErrNoCredentials
	}
	return c.creds, nil
}

func (c *HubClient) post(ctx context.Context, endpoint, messageType string, payload map[string]any, creds *credentials) (map[string]any, error) {
	status, raw, err := c.postRaw(ctx, endpoint, messageType, payload, creds)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, hubError(endpoint, out, creds)
	}
	return out, nil
}

func (c *HubClient) postRaw(ctx context.Context, endpoint, messageType string, payload map[string]any, creds *credentials) (int, []byte, error) {
	env := envelope{
		Protocol:        "gep-a2a",
		ProtocolVersion: "1.0.0",
		MessageType:     messageType,
		MessageID:       newMessageID(),
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Payload:         payload,
	}
	if creds != nil {
		env.SenderID = creds.NodeID
	}
	body, err := json.Marshal(env)
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-correlation-id", env.MessageID)
	if creds != nil {
		req.Header.Set("Authorization", "Bearer "+creds.NodeSecret)
	}
	status, raw, err := c.doRaw(req)
	if err != nil {
		return 0, nil, err
	}
	if status == http.StatusUnauthorized {
		return 0, nil, ErrUnauthorized
	}
	return status, raw, nil
}

func (c *HubClient) do(req *http.Request) ([]byte, error) {
	status, raw, err := c.doRaw(req)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("evomap %s: HTTP %d: %s", req.URL.Path, status, redactSecret(c.creds, truncate(string(raw), 500)))
	}
	return raw, nil
}

func (c *HubClient) doRaw(req *http.Request) (int, []byte, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", ErrHubUnreachable, redactSecret(c.creds, err.Error()))
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", ErrHubUnreachable, redactSecret(c.creds, err.Error()))
	}
	return resp.StatusCode, raw, nil
}

func (c *HubClient) saveCredentials(creds *credentials) error {
	if err := os.MkdirAll(filepath.Dir(c.CredsPath), 0700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.CredsPath, raw, 0600)
}

func newMessageID() string {
	buf := make([]byte, 4)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("msg_%d_%s", time.Now().UnixMilli(), hex.EncodeToString(buf))
}

// redactSecret strips node_secret occurrences from any string (errors,
// bodies, logs) so the credential can never leak through the client.
func redactSecret(creds *credentials, s string) string {
	if creds == nil || creds.NodeSecret == "" {
		return s
	}
	return strings.ReplaceAll(s, creds.NodeSecret, "[REDACTED]")
}

func asString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case float64:
		return fmt.Sprintf("%.0f", s)
	case bool:
		return fmt.Sprintf("%v", s)
	}
	return ""
}

func asFloat(v any) float64 {
	f, _ := v.(float64)
	return f
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
