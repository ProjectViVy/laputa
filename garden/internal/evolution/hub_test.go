package evolution

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dashimaki/garden/internal/evolution/hubtest"
)

func newTestClient(t *testing.T, hub *hubtest.MockHub) *HubClient {
	t.Helper()
	client, err := OpenHubClient(HubClientOptions{
		BaseURL:   hub.URL,
		CredsPath: filepath.Join(t.TempDir(), "node.json"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestHelloRegistersAndSavesCredentials(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	client := newTestClient(t, hub)

	claimURL, err := client.EnsureRegistered(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if claimURL == "" {
		t.Fatal("expected claim URL after registration")
	}
	raw, err := os.ReadFile(client.CredsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), hub.NodeSecret) {
		t.Fatal("credentials file missing node_secret")
	}
	if !client.HasCredentials() {
		t.Fatal("client should have credentials after hello")
	}

	// Second registration is a no-op (no new hello).
	hub.Messages = nil
	if _, err := client.EnsureRegistered(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(hub.Messages) != 0 {
		t.Fatalf("hello called again: %v", hub.Messages)
	}
}

func TestHeartbeatParsesPayload(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	client := newTestClient(t, hub)
	ctx := context.Background()
	if _, err := client.EnsureRegistered(ctx); err != nil {
		t.Fatal(err)
	}

	hb, err := client.Heartbeat(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if hb.Status != "ok" || hb.SurvivalStatus != "alive" || hb.NodeID != "node_test_abcd" {
		t.Fatalf("heartbeat=%+v", hb)
	}
	if !hb.HasBalance || hb.CreditBalance != 100 {
		t.Fatalf("balance not parsed: %+v", hb)
	}
}

func TestSearchMapsAssets(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	hub.SearchAssets = []map[string]any{
		hubtest.Asset("Capsule", "cap_1", "first capsule", 76.5),
		hubtest.Asset("Gene", "gene_1", "first gene", 0),
	}
	client := newTestClient(t, hub)
	ctx := context.Background()
	if _, err := client.EnsureRegistered(ctx); err != nil {
		t.Fatal(err)
	}

	assets, err := client.Search(ctx, []string{"some signal"}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 2 {
		t.Fatalf("assets=%d, want 2", len(assets))
	}
	if assets[0].AssetType != "Capsule" || assets[0].Summary != "first capsule" || assets[0].GDI != 76.5 {
		t.Fatalf("asset[0]=%+v", assets[0])
	}
}

func TestValidateAndPublishRoundTrip(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	client := newTestClient(t, hub)
	ctx := context.Background()
	if _, err := client.EnsureRegistered(ctx); err != nil {
		t.Fatal(err)
	}
	gene, capsule := pair(t)

	vs, err := client.Validate(ctx, []map[string]any{gene, capsule})
	if err != nil {
		t.Fatal(err)
	}
	if vs.Accepted != 2 || vs.Rejected != 0 {
		t.Fatalf("validate summary=%+v", vs)
	}
	if len(vs.Statuses) != 2 {
		t.Fatalf("statuses=%v, want both asset ids", vs.Statuses)
	}

	ps, err := client.Publish(ctx, []map[string]any{gene, capsule})
	if err != nil {
		t.Fatal(err)
	}
	if ps.Total != 2 || ps.Rejected != 0 {
		t.Fatalf("publish summary=%+v", ps)
	}
	for id, status := range ps.Statuses {
		if status != "quarantine" {
			t.Fatalf("asset %s status=%q, want quarantine", id, status)
		}
	}
	if len(hub.Published) != 1 {
		t.Fatalf("publish calls=%d, want 1", len(hub.Published))
	}
}

func TestValidateRejectsInvalidAssetID(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	client := newTestClient(t, hub)
	ctx := context.Background()
	if _, err := client.EnsureRegistered(ctx); err != nil {
		t.Fatal(err)
	}
	gene, capsule := pair(t)
	capsule["asset_id"] = "sha256:deadbeef"

	_, err := client.Validate(ctx, []map[string]any{gene, capsule})
	if err == nil {
		t.Fatal("expected validation error for corrupted asset_id")
	}
	if !strings.Contains(err.Error(), "canonical content hash") {
		t.Fatalf("err=%v, want canonical-hash detail", err)
	}
}

func TestPublishRejectedDecision(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	hub.RejectPublish = true
	client := newTestClient(t, hub)
	ctx := context.Background()
	if _, err := client.EnsureRegistered(ctx); err != nil {
		t.Fatal(err)
	}
	gene, capsule := pair(t)

	ps, err := client.Publish(ctx, []map[string]any{gene, capsule})
	if err != nil {
		t.Fatal(err)
	}
	if ps.Rejected != 2 || ps.Accepted != 0 {
		t.Fatalf("publish summary=%+v, want all rejected", ps)
	}
	for id, status := range ps.Statuses {
		if status != "rejected" {
			t.Fatalf("asset %s status=%q, want rejected", id, status)
		}
	}
}

func TestReportAccepted(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	client := newTestClient(t, hub)
	ctx := context.Background()
	if _, err := client.EnsureRegistered(ctx); err != nil {
		t.Fatal(err)
	}
	resp, err := client.Report(ctx, "sha256:abc", "success")
	if err != nil {
		t.Fatal(err)
	}
	if resp["payload"] == nil {
		t.Fatalf("report response missing payload: %v", resp)
	}
}

func TestRevokeIsSessionOnly(t *testing.T) {
	// The A2A client intentionally does not implement revoke/decision:
	// the live hub rejects them with 401 for node_secret clients. The
	// mock reproduces that boundary so it stays visible in tests.
	hub := hubtest.New()
	defer hub.Close()
	client := newTestClient(t, hub)
	ctx := context.Background()
	if _, err := client.EnsureRegistered(ctx); err != nil {
		t.Fatal(err)
	}
	_, err := client.post(ctx, "/a2a/revoke", "revoke", map[string]any{"asset_id": "sha256:abc"}, client.creds)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err=%v, want ErrUnauthorized (session-only endpoint)", err)
	}
}

func TestUnauthorizedMapsToErrUnauthorized(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	client := newTestClient(t, hub)
	if _, err := client.EnsureRegistered(context.Background()); err != nil {
		t.Fatal(err)
	}
	hub.NodeSecret = "different_secret" // invalidate the client's credentials

	_, err := client.Heartbeat(context.Background())
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err=%v, want ErrUnauthorized", err)
	}
}

func TestNoCredentials(t *testing.T) {
	client, err := OpenHubClient(HubClientOptions{CredsPath: filepath.Join(t.TempDir(), "missing.json")})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Heartbeat(context.Background())
	if !errors.Is(err, ErrNoCredentials) {
		t.Fatalf("err=%v, want ErrNoCredentials", err)
	}
}

func TestHubUnreachable(t *testing.T) {
	hub := hubtest.New()
	url := hub.URL
	hub.Close()
	client, err := OpenHubClient(HubClientOptions{BaseURL: url, CredsPath: filepath.Join(t.TempDir(), "node.json")})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.EnsureRegistered(context.Background())
	if !errors.Is(err, ErrHubUnreachable) {
		t.Fatalf("err=%v, want ErrHubUnreachable", err)
	}
}

func TestErrorStringsRedactSecret(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	client := newTestClient(t, hub)
	if _, err := client.EnsureRegistered(context.Background()); err != nil {
		t.Fatal(err)
	}

	_, err := client.post(context.Background(), "/a2a/leak", "leak", map[string]any{}, client.creds)
	if err == nil {
		t.Fatal("expected error from leak endpoint")
	}
	if strings.Contains(err.Error(), hub.NodeSecret) {
		t.Fatalf("error leaks node_secret: %v", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("error should contain redaction marker: %v", err)
	}
}

func TestCanonicalHashDeterministic(t *testing.T) {
	a := map[string]any{"type": "Gene", "summary": "hello", "tags": []string{"x", "y"}}
	b := map[string]any{"type": "Gene", "summary": "hello", "tags": []string{"x", "y"}}
	c := map[string]any{"type": "Gene", "summary": "different"}
	ha, err := CanonicalHash(a)
	if err != nil {
		t.Fatal(err)
	}
	hb, err := CanonicalHash(b)
	if err != nil {
		t.Fatal(err)
	}
	hc, err := CanonicalHash(c)
	if err != nil {
		t.Fatal(err)
	}
	if ha != hb {
		t.Fatalf("same content produced different ids: %s vs %s", ha, hb)
	}
	if ha == hc {
		t.Fatal("different content produced the same id")
	}
	if !strings.HasPrefix(ha, "sha256:") || len(ha) != len("sha256:")+64 {
		t.Fatalf("unexpected id shape: %q", ha)
	}
}

func pair(t *testing.T) (map[string]any, map[string]any) {
	t.Helper()
	gene := map[string]any{
		"type":           "Gene",
		"schema_version": "1.5.0",
		"category":       "repair",
		"signals_match":  []string{"EVOMAP_CONNECTIVITY_PROBE"},
		"summary":        "EvoMap connectivity probe gene used by garden tests; no-op",
		"strategy": []string{
			"Publish the probe bundle to verify GEP-A2A write path",
			"Search the probe signal to confirm hub visibility",
		},
		"validation": []string{`node -e "if (1 + 1 !== 2) process.exit(1)"`},
	}
	geneID, err := CanonicalHash(gene)
	if err != nil {
		t.Fatal(err)
	}
	gene["asset_id"] = geneID

	capsule := map[string]any{
		"type":            "Capsule",
		"schema_version":  "1.5.0",
		"trigger":         []string{"EVOMAP_CONNECTIVITY_PROBE"},
		"gene":            geneID,
		"summary":         "EvoMap connectivity probe capsule paired with the probe gene; no-op",
		"content":         "Intent: verify GEP-A2A publish/fetch/revoke round trip.\n\nStrategy:\n1. Publish probe bundle\n2. Search by signal\n3. Revoke\n\nScope: 1 file(s), 1 line(s)\n\nOutcome score: 0.5",
		"strategy":        []string{"Publish probe bundle", "Search by signal", "Revoke"},
		"confidence":      0.5,
		"blast_radius":    map[string]int{"files": 1, "lines": 1},
		"outcome":         map[string]any{"status": "success", "score": 0.5},
		"env_fingerprint": map[string]string{"platform": "test", "arch": "test"},
		"validation":      []string{`node -e "if (1 + 1 !== 2) process.exit(1)"`},
	}
	capsuleID, err := CanonicalHash(capsule)
	if err != nil {
		t.Fatal(err)
	}
	capsule["asset_id"] = capsuleID
	return gene, capsule
}
