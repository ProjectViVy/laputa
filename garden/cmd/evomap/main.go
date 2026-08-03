// Command evomap is the manual ops CLI for Garden's EvoMap Hub transport
// (GEP-A2A, ADR-0010). It is a thin wrapper over internal/evolution's
// HubClient. The server path never uses the paid fetch tier or
// session-only endpoints; this CLI is the only place where they are
// reachable by hand.
//
// The node_secret is stored only in the credentials file (0600) and is
// never printed to stdout or logs.
//
// Usage:
//
//	evomap hello       register a node and save credentials
//	evomap heartbeat   keep-alive and account status
//	evomap fetch       fetch hub assets (paid tier; explicit)
//	evomap search      free signal search
//	evomap validate    validate an asset bundle against hub rules
//	evomap publish     publish a probe Gene+Capsule pair
//	evomap report      submit a validation report for an asset
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/dashimaki/garden/internal/evolution"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	client, err := evolution.OpenHubClient(evolution.HubClientOptions{
		BaseURL:   os.Getenv("GARDEN_EVOMAP_HUB_URL"),
		CredsPath: expandHome(os.Getenv("GARDEN_EVOMAP_CREDS")),
	})
	if err != nil {
		fatal(err)
	}
	ctx := context.Background()
	switch os.Args[1] {
	case "hello":
		cmdHello(ctx, client)
	case "heartbeat":
		cmdHeartbeat(ctx, client)
	case "fetch":
		cmdFetch(ctx, client)
	case "search":
		cmdSearch(ctx, client)
	case "validate":
		cmdValidate(ctx, client)
	case "publish":
		cmdPublish(ctx, client)
	case "report":
		cmdReport(ctx, client)
	default:
		usage()
	}
}

func cmdHello(ctx context.Context, client *evolution.HubClient) {
	claimURL, err := client.EnsureRegistered(ctx)
	if err != nil {
		fatal(err)
	}
	if claimURL == "" {
		fmt.Println("node already registered")
		return
	}
	fmt.Printf("node registered; claim (24h window, binds to an EvoMap account): %s\n", claimURL)
	fmt.Printf("credentials saved: %s (0600, never commit)\n", client.CredsPath)
}

func cmdHeartbeat(ctx context.Context, client *evolution.HubClient) {
	hb, err := client.Heartbeat(ctx)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("status=%s survival=%s node=%s\n", hb.Status, hb.SurvivalStatus, hb.NodeID)
	if hb.HasBalance {
		fmt.Printf("credit_balance=%.0f\n", hb.CreditBalance)
	}
	if hb.ClaimURL != "" && !hb.Claimed {
		fmt.Printf("claim_url (unclaimed): %s\n", hb.ClaimURL)
	}
}

func cmdFetch(ctx context.Context, client *evolution.HubClient) {
	assetType := "Capsule"
	if len(os.Args) >= 3 {
		assetType = os.Args[2]
	}
	assets, err := client.Fetch(ctx, assetType, true)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("fetch results=%d\n", len(assets))
	for i, a := range assets {
		fmt.Printf("  [%d] %s %s | %s\n", i+1, a.AssetType, a.LocalID, a.Summary)
	}
}

func cmdSearch(ctx context.Context, client *evolution.HubClient) {
	query := "EVOMAP_CONNECTIVITY_PROBE"
	if len(os.Args) >= 3 {
		query = os.Args[2]
	}
	assets, err := client.Search(ctx, []string{query}, 5)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("search hits=%d\n", len(assets))
	for i, a := range assets {
		fmt.Printf("  [%d] %s %s | %s\n", i+1, a.AssetType, a.LocalID, a.Summary)
	}
}

func cmdValidate(ctx context.Context, client *evolution.HubClient) {
	summary, err := client.Validate(ctx, probeBundle())
	if err != nil {
		fatal(err)
	}
	printSummary(summary)
}

func cmdPublish(ctx context.Context, client *evolution.HubClient) {
	bundle := probeBundle()
	for _, a := range bundle {
		id, _ := evolution.CanonicalHash(a)
		fmt.Printf("publishing %s %s\n", a["type"], id)
	}
	summary, err := client.Publish(ctx, bundle)
	if err != nil {
		fatal(err)
	}
	printSummary(summary)
}

func cmdReport(ctx context.Context, client *evolution.HubClient) {
	if len(os.Args) < 3 {
		fatal(errors.New("usage: evomap report <asset_id>"))
	}
	status := "success"
	if len(os.Args) >= 4 {
		status = os.Args[3]
	}
	resp, err := client.Report(ctx, os.Args[2], status)
	if err != nil {
		fatal(err)
	}
	raw, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(raw))
}

// probeBundle builds a harmless Gene+Capsule pair (GEP-A2A requires both
// together) with content-addressed asset ids. Nothing here is real data;
// the bundle exists only for connectivity checks and hub-rule demos.
func probeBundle() []map[string]any {
	gene := map[string]any{
		"type":           "Gene",
		"schema_version": "1.5.0",
		"category":       "repair",
		"signals_match":  []string{"EVOMAP_CONNECTIVITY_PROBE"},
		"summary":        "EvoMap connectivity probe gene used by garden cmd/evomap; no-op",
		"strategy": []string{
			"Publish the probe bundle to verify GEP-A2A write path",
			"Search the probe signal to confirm hub visibility",
			"Revoke the probe bundle via the account page to clean up",
		},
		"validation": []string{`node -e "if (1 + 1 !== 2) process.exit(1)"`},
	}
	geneID, _ := evolution.CanonicalHash(gene)
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
		"env_fingerprint": map[string]string{"platform": "win32", "arch": "x64"},
		"validation":      []string{`node -e "if (1 + 1 !== 2) process.exit(1)"`},
	}
	capsuleID, _ := evolution.CanonicalHash(capsule)
	capsule["asset_id"] = capsuleID
	return []map[string]any{gene, capsule}
}

func printSummary(s evolution.PublishSummary) {
	fmt.Printf("assets total=%d accepted=%d rejected=%d\n", s.Total, s.Accepted, s.Rejected)
	for id, status := range s.Statuses {
		fmt.Printf("  %s -> %s\n", id, status)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: evomap <hello|heartbeat|fetch|search|validate|publish|report>")
	os.Exit(2)
}

func expandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	return home + path[1:]
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "evomap:", err)
	os.Exit(1)
}
