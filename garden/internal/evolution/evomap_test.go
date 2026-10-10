package evolution

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ProjectViVy/laputa/garden/internal/evolution/hubtest"
)

func newTestProvider(t *testing.T, hub *hubtest.MockHub, publish bool) (*EvoMapProvider, *Store) {
	t.Helper()
	client, err := OpenHubClient(HubClientOptions{
		BaseURL:   hub.URL,
		CredsPath: filepath.Join(t.TempDir(), "node.json"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.EnsureRegistered(context.Background()); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return NewEvoMapProvider(client, store, DefaultProviderLimits(), publish), store
}

func testBundle() EvolutionEvidenceBundle {
	return EvolutionEvidenceBundle{
		Trigger:        "test failure in recall pipeline",
		Outcome:        "the fast recall pipeline failed to rank the expected card",
		TraceRef:       "trace_recall_001",
		BlastRadius:    "recall only, no writes",
		EvidenceRefs:   []string{"trace_recall_001", "evidence_card_003"},
		SourceRevision: "abc123",
		ContentHashes:  []string{"sha256:hash1"},
		Policy:         BundlePolicy{PrivacyLevel: "scoped", PublicationAllowed: true},
	}
}

func TestProviderStartRunPublishMode(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	provider, _ := newTestProvider(t, hub, true)

	runID, err := provider.StartRun(context.Background(), testBundle())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(runID, "evomap_") || strings.HasPrefix(runID, "evomap_disc_") {
		t.Fatalf("runID=%q, want publish-mode id", runID)
	}
	if len(hub.Published) != 1 {
		t.Fatalf("publish calls=%d, want 1", len(hub.Published))
	}
	if len(hub.Published[0]) != 2 {
		t.Fatalf("published assets=%d, want Gene+Capsule pair", len(hub.Published[0]))
	}
	run, err := provider.Store.GetHubRun(context.Background(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Mode != "publish" || run.Status != "completed" {
		t.Fatalf("hub run=%+v", run)
	}
	if len(run.AssetIDs) != 1 {
		t.Fatalf("asset ids=%v", run.AssetIDs)
	}

	status, err := provider.PollRun(context.Background(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "completed" {
		t.Fatalf("poll status=%+v", status)
	}
}

func TestProviderStartRunDiscoveryMode(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	provider, _ := newTestProvider(t, hub, false)

	runID, err := provider.StartRun(context.Background(), testBundle())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(runID, "evomap_disc_") {
		t.Fatalf("runID=%q, want discovery-mode id", runID)
	}
	if len(hub.Published) != 0 {
		t.Fatalf("publish calls=%d, want 0 (publish disabled)", len(hub.Published))
	}
	run, err := provider.Store.GetHubRun(context.Background(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Mode != "discovery" || len(run.Signals) != 1 {
		t.Fatalf("hub run=%+v", run)
	}
}

func TestProviderStartRunRespectsBundlePolicy(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	provider, _ := newTestProvider(t, hub, true)

	bundle := testBundle()
	bundle.Policy.PublicationAllowed = false
	runID, err := provider.StartRun(context.Background(), bundle)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(runID, "evomap_disc_") {
		t.Fatalf("runID=%q, want discovery-mode despite publish enabled", runID)
	}
	if len(hub.Published) != 0 {
		t.Fatal("bundle without publication permission must not publish")
	}
}

func TestProviderStartRunLeakageGate(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	provider, _ := newTestProvider(t, hub, true)

	bundle := testBundle()
	bundle.EvidenceRefs = []string{"config/.env.production"}
	_, err := provider.StartRun(context.Background(), bundle)
	if !errors.Is(err, ErrLeakageDetected) {
		t.Fatalf("err=%v, want ErrLeakageDetected", err)
	}
	if len(hub.Messages) != 0 {
		t.Fatalf("network calls=%v, want zero before the privacy gate", hub.Messages)
	}
}

func TestProviderStartRunDiscoveryRequiresTrigger(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	provider, _ := newTestProvider(t, hub, false)

	bundle := testBundle()
	bundle.Trigger = ""
	_, err := provider.StartRun(context.Background(), bundle)
	if err == nil || !strings.Contains(err.Error(), "trigger") {
		t.Fatalf("err=%v, want trigger-required error", err)
	}
}

func TestProviderCandidatesDiscovery(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	hub.SearchAssets = []map[string]any{
		hubtest.Asset("Capsule", "cap_1", "discovered capsule", 80),
		hubtest.Asset("Gene", "gene_1", "discovered gene", 0),
	}
	provider, _ := newTestProvider(t, hub, false)
	ctx := context.Background()

	runID, err := provider.StartRun(ctx, testBundle())
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := provider.Candidates(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidates=%d, want 2", len(candidates))
	}
	if candidates[0].Kind != "capsule" || candidates[0].Name != "discovered capsule" {
		t.Fatalf("candidate[0]=%+v", candidates[0])
	}
	if candidates[0].Confidence != 0.8 {
		t.Fatalf("confidence=%v, want 0.8 (gdi 80/100)", candidates[0].Confidence)
	}
	if candidates[1].Kind != "gene" || candidates[1].Confidence != 0.5 {
		t.Fatalf("candidate[1]=%+v", candidates[1])
	}
	if candidates[0].TraceRef != runID {
		t.Fatalf("trace_ref=%q, want run id", candidates[0].TraceRef)
	}
}

func TestProviderCandidatesPublishIncludesOwnCapsule(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	hub.SearchAssets = []map[string]any{hubtest.Asset("Capsule", "cap_ext", "external capsule", 60)}
	provider, _ := newTestProvider(t, hub, true)
	ctx := context.Background()

	runID, err := provider.StartRun(ctx, testBundle())
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := provider.Candidates(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidates=%d, want own capsule + 1 discovered", len(candidates))
	}
	own := candidates[0]
	if own.Kind != "capsule" || own.TraceRef != runID {
		t.Fatalf("own candidate=%+v", own)
	}
	if len(own.EvidenceRefs) != 2 {
		t.Fatalf("own candidate evidence refs=%v, want bundle refs", own.EvidenceRefs)
	}
	if own.Payload["trigger"] == nil {
		t.Fatal("own candidate payload should carry the capsule asset")
	}
}

func TestProviderCandidatesDegradedFallback(t *testing.T) {
	hub := hubtest.New()
	provider, _ := newTestProvider(t, hub, true)
	ctx := context.Background()

	runID, err := provider.StartRun(ctx, testBundle())
	if err != nil {
		t.Fatal(err)
	}
	hub.Close() // hub goes away after the run was recorded

	candidates, err := provider.Candidates(ctx, runID)
	if err != nil {
		t.Fatalf("err=%v, want local fallback without error", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates=%d, want own capsule only (degraded)", len(candidates))
	}
	if candidates[0].CandidateID == "" {
		t.Fatal("fallback candidate missing id")
	}
}

func TestProviderPollRunMaxLifetimeTimeout(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	provider, _ := newTestProvider(t, hub, false)
	ctx := context.Background()
	fixed := time.Now().UTC()

	provider.Limits.MaxLifetime = time.Minute
	provider.Now = func() time.Time { return fixed.Add(2 * time.Minute) }
	run := HubRun{
		RunID:     "evomap_disc_stale",
		Mode:      "discovery",
		Status:    "running",
		Signals:   []string{"sig"},
		CreatedAt: fixed,
		UpdatedAt: fixed,
	}
	if err := provider.Store.SaveHubRun(ctx, run); err != nil {
		t.Fatal(err)
	}

	status, err := provider.PollRun(ctx, run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "failed" || !strings.Contains(status.Error, "max lifetime") {
		t.Fatalf("status=%+v, want failed(timeout)", status)
	}
}

func TestProviderStatus(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	provider, _ := newTestProvider(t, hub, false)

	status, err := provider.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.NodeID != "node_test_abcd" || !status.HasBalance || status.CreditBalance != 100 {
		t.Fatalf("status=%+v", status)
	}
	if status.SurvivalStatus != "alive" || status.LastHeartbeat == "" {
		t.Fatalf("status=%+v", status)
	}
}

func TestServiceWithEvoMapProvider(t *testing.T) {
	hub := hubtest.New()
	defer hub.Close()
	provider, _ := newTestProvider(t, hub, false)
	svc := testService(t, provider)
	ctx := context.Background()

	run, err := svc.StartRun(ctx, testBundle(), "agent")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "running" {
		t.Fatalf("run=%+v", run)
	}
	got, err := svc.GetRun(ctx, run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "completed" {
		t.Fatalf("status=%s, want completed after hub round trip", got.Status)
	}

	candidate := GeneCandidate{CandidateID: "cand_1", RunID: run.RunID, Kind: "gene", Name: "test_gene", EvidenceRefs: []string{}}
	_ = svc.Store.SaveCandidate(ctx, candidate)
	proposal, err := svc.CreateProposal(ctx, run.RunID, "cand_1", "agent")
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Status != "pending" {
		t.Fatalf("proposal=%+v", proposal)
	}
}

func TestServiceHubStatusUnavailableWithoutProvider(t *testing.T) {
	svc := testService(t, nil)
	_, err := svc.HubStatus(context.Background())
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("err=%v, want ErrProviderUnavailable", err)
	}
}
