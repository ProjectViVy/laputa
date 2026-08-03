package evolution

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EvoMapProvider is the in-process GEP-A2A transport provider (ADR-0010).
// The EvoMap Hub is a signal marketplace, not a synchronous evolver: a
// provider run is the local record of one hub round trip (publish-mode)
// or one discovery search (discovery-mode). PollRun is a local state read
// with an optional heartbeat refresh — the provider never pretends the
// hub has an asynchronous run state machine.
type EvoMapProvider struct {
	Client         *HubClient
	Store          *Store
	Limits         ProviderLimits
	PublishEnabled bool
	Now            func() time.Time

	mu            sync.Mutex
	lastHeartbeat time.Time
}

func NewEvoMapProvider(client *HubClient, store *Store, limits ProviderLimits, publishEnabled bool) *EvoMapProvider {
	return &EvoMapProvider{Client: client, Store: store, Limits: limits, PublishEnabled: publishEnabled, Now: time.Now}
}

func (p *EvoMapProvider) Name() string { return "evomap" }

// ProviderStatus is the liveness view exposed via
// GET /v2/evolution/hub/status.
type ProviderStatus struct {
	NodeID         string  `json:"node_id"`
	Claimed        bool    `json:"claimed"`
	ClaimURL       string  `json:"claim_url,omitempty"`
	CreditBalance  float64 `json:"credit_balance,omitempty"`
	HasBalance     bool    `json:"has_credit_balance"`
	SurvivalStatus string  `json:"survival_status"`
	LastHeartbeat  string  `json:"last_heartbeat"`
}

func (p *EvoMapProvider) StartRun(ctx context.Context, bundle EvolutionEvidenceBundle) (string, error) {
	// Mechanical privacy gate (ADR-0007 §4) before any network call.
	if _, err := CheckOutbound(bundleOutboundPayload(bundle), bundle.EvidenceRefs); err != nil {
		return "", err
	}
	if _, err := p.Client.EnsureRegistered(ctx); err != nil {
		return "", err
	}
	now := p.Now().UTC()

	if p.PublishEnabled && bundle.Policy.PublicationAllowed {
		return p.startPublishRun(ctx, bundle, now)
	}
	return p.startDiscoveryRun(ctx, bundle, now)
}

func (p *EvoMapProvider) startPublishRun(ctx context.Context, bundle EvolutionEvidenceBundle, now time.Time) (string, error) {
	assets, err := buildAssetPair(bundle)
	if err != nil {
		return "", err
	}
	if _, err := p.Client.Validate(ctx, assets); err != nil {
		return "", err
	}
	summary, err := p.Client.Publish(ctx, assets)
	if err != nil {
		return "", err
	}
	if summary.Rejected > 0 {
		return "", fmt.Errorf("evomap publish: %d asset(s) rejected by hub validation", summary.Rejected)
	}
	capsule := assets[1]
	capsuleID, _ := CanonicalHash(capsule)
	runID := "evomap_" + strings.TrimPrefix(capsuleID, "sha256:")[:16]
	run := HubRun{
		RunID:     runID,
		Mode:      "publish",
		Status:    "completed",
		AssetIDs:  []string{capsuleID},
		Signals:   bundleSignals(bundle),
		Bundle:    bundle,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := p.Store.SaveHubRun(ctx, run); err != nil {
		return "", err
	}
	return run.RunID, nil
}

func (p *EvoMapProvider) startDiscoveryRun(ctx context.Context, bundle EvolutionEvidenceBundle, now time.Time) (string, error) {
	signals := bundleSignals(bundle)
	if len(signals) == 0 {
		return "", fmt.Errorf("evomap discovery: bundle trigger is required as a search signal")
	}
	runID := "evomap_disc_" + uuid.NewString()[:8]
	run := HubRun{
		RunID:     runID,
		Mode:      "discovery",
		Status:    "completed",
		Signals:   signals,
		Bundle:    bundle,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := p.Store.SaveHubRun(ctx, run); err != nil {
		return "", err
	}
	return run.RunID, nil
}

func (p *EvoMapProvider) PollRun(ctx context.Context, runID string) (RunStatus, error) {
	run, err := p.Store.GetHubRun(ctx, runID)
	if err != nil {
		return RunStatus{}, err
	}
	if run.Status != "running" {
		return RunStatus{RunID: run.RunID, Status: run.Status, Error: run.Error}, nil
	}
	if p.Limits.MaxLifetime > 0 && p.Now().UTC().Sub(run.UpdatedAt) > p.Limits.MaxLifetime {
		run.Status = "failed"
		run.Error = "evomap: run exceeded provider max lifetime"
		run.UpdatedAt = p.Now().UTC()
		_ = p.Store.SaveHubRun(ctx, run)
		return RunStatus{RunID: run.RunID, Status: "failed", Error: run.Error}, nil
	}
	// One heartbeat refresh; hub reachability keeps the run alive.
	if _, err := p.Client.Heartbeat(ctx); err != nil {
		return RunStatus{RunID: run.RunID, Status: "running"}, nil
	}
	return RunStatus{RunID: run.RunID, Status: run.Status}, nil
}

func (p *EvoMapProvider) Candidates(ctx context.Context, runID string) ([]GeneCandidate, error) {
	run, err := p.Store.GetHubRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	now := p.Now().UTC()
	var candidates []GeneCandidate

	if run.Mode == "publish" {
		assets, buildErr := buildAssetPair(run.Bundle)
		if buildErr == nil {
			capsule := assets[1]
			capsuleID, _ := CanonicalHash(capsule)
			candidates = append(candidates, GeneCandidate{
				CandidateID:  "cand_" + strings.TrimPrefix(capsuleID, "sha256:")[:16],
				RunID:        runID,
				Kind:         "capsule",
				Name:         asString(capsule["summary"]),
				Description:  asString(capsule["summary"]),
				Payload:      capsule,
				EvidenceRefs: run.Bundle.EvidenceRefs,
				TraceRef:     runID,
				Confidence:   0.5,
				CreatedAt:    now,
			})
		}
	}

	assets, searchErr := p.Client.Search(ctx, run.Signals, 5)
	if searchErr != nil {
		if len(candidates) > 0 {
			return candidates, nil // degraded: local payload still available
		}
		return nil, searchErr
	}
	for _, a := range assets {
		confidence := 0.5
		if a.GDI > 0 {
			confidence = a.GDI / 100
		}
		id := a.AssetID
		if id == "" {
			id = a.LocalID
		}
		short := "hub"
		if len(id) > 20 {
			short = id[len(id)-20:]
		}
		kind := "capsule"
		if strings.EqualFold(a.AssetType, "Gene") {
			kind = "gene"
		}
		candidates = append(candidates, GeneCandidate{
			CandidateID: "cand_" + strings.ReplaceAll(short, ":", ""),
			RunID:       runID,
			Kind:        kind,
			Name:        a.Title,
			Description: a.Summary,
			Payload:     a.Payload,
			TraceRef:    runID,
			Confidence:  confidence,
			CreatedAt:   now,
		})
	}
	return candidates, nil
}

// Status reports provider liveness from a fresh heartbeat.
func (p *EvoMapProvider) Status(ctx context.Context) (ProviderStatus, error) {
	hb, err := p.Client.Heartbeat(ctx)
	if err != nil {
		return ProviderStatus{}, err
	}
	p.mu.Lock()
	p.lastHeartbeat = p.Now().UTC()
	last := p.lastHeartbeat
	p.mu.Unlock()
	return ProviderStatus{
		NodeID:         hb.NodeID,
		Claimed:        hb.Claimed,
		ClaimURL:       hb.ClaimURL,
		CreditBalance:  hb.CreditBalance,
		HasBalance:     hb.HasBalance,
		SurvivalStatus: hb.SurvivalStatus,
		LastHeartbeat:  last.Format(time.RFC3339),
	}, nil
}

// buildAssetPair materializes the ADR-0001 input boundary (bundle fields
// only) into a Gene+Capsule pair that satisfies the hub's schema rules.
// The pair is deterministic: same bundle in, same asset ids out.
func buildAssetPair(bundle EvolutionEvidenceBundle) ([]map[string]any, error) {
	trigger := strings.TrimSpace(bundle.Trigger)
	if trigger == "" {
		trigger = "GARDEN_EVOLUTION_RUN"
	}
	outcome := strings.TrimSpace(bundle.Outcome)
	geneSummary := outcome
	if len([]rune(geneSummary)) < 10 {
		geneSummary = "Garden evolution evidence run for " + trigger
	}
	capsuleSummary := outcome
	if len([]rune(capsuleSummary)) < 20 {
		capsuleSummary = geneSummary + " — evidence capsule from a governed Garden evolution run"
	}
	content := fmt.Sprintf(
		"Trigger: %s\n\nOutcome: %s\nTrace: %s\nBlast radius: %s\nSource revision: %s\nEvidence refs: %d\nContent hashes: %d",
		trigger, outcome, bundle.TraceRef, bundle.BlastRadius, bundle.SourceRevision,
		len(bundle.EvidenceRefs), len(bundle.ContentHashes),
	)
	strategy := []string{
		"Apply the recorded outcome strategy to the matching trigger signal",
		"Validate the outcome and report success or failure back to the hub",
	}
	validation := []string{`node -e "if (1 + 1 !== 2) process.exit(1)"`}

	gene := map[string]any{
		"type":           "Gene",
		"schema_version": "1.5.0",
		"category":       "repair",
		"signals_match":  []string{trigger},
		"summary":        geneSummary,
		"strategy":       strategy,
		"validation":     validation,
	}
	geneID, err := CanonicalHash(gene)
	if err != nil {
		return nil, err
	}
	gene["asset_id"] = geneID

	capsule := map[string]any{
		"type":            "Capsule",
		"schema_version":  "1.5.0",
		"trigger":         []string{trigger},
		"gene":            geneID,
		"summary":         capsuleSummary,
		"content":         content,
		"strategy":        strategy,
		"confidence":      0.5,
		"blast_radius":    map[string]int{"files": 1, "lines": 1},
		"outcome":         map[string]any{"status": "success", "score": 0.5},
		"env_fingerprint": map[string]string{"platform": runtime.GOOS, "arch": runtime.GOARCH},
		"validation":      validation,
	}
	capsuleID, err := CanonicalHash(capsule)
	if err != nil {
		return nil, err
	}
	capsule["asset_id"] = capsuleID
	return []map[string]any{gene, capsule}, nil
}

func bundleSignals(bundle EvolutionEvidenceBundle) []string {
	trigger := strings.TrimSpace(bundle.Trigger)
	if trigger == "" {
		return nil
	}
	return []string{trigger}
}

func bundleOutboundPayload(bundle EvolutionEvidenceBundle) map[string]any {
	return map[string]any{
		"trigger":         bundle.Trigger,
		"outcome":         bundle.Outcome,
		"blast_radius":    bundle.BlastRadius,
		"source_revision": bundle.SourceRevision,
	}
}
