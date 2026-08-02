package evolution

import (
	"context"
	"path/filepath"
	"testing"
)

type mockProvider struct {
	startErr   error
	pollStatus RunStatus
	candidates []GeneCandidate
}

func (m *mockProvider) Name() string { return "mock" }

func (m *mockProvider) StartRun(_ context.Context, _ EvolutionEvidenceBundle) (string, error) {
	if m.startErr != nil {
		return "", m.startErr
	}
	return "run_test_1", nil
}

func (m *mockProvider) PollRun(_ context.Context, _ string) (RunStatus, error) {
	return m.pollStatus, nil
}

func (m *mockProvider) Candidates(_ context.Context, _ string) ([]GeneCandidate, error) {
	return m.candidates, nil
}

func testService(t *testing.T, provider EvolverProvider) *Service {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	events, err := OpenEventStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { events.Close() })
	return &Service{Provider: provider, Store: store, Events: events, Hub: DefaultHubPolicy()}
}

func TestEvolverCannotReadOutsideEvidenceRefs(t *testing.T) {
	candidate := GeneCandidate{
		Kind:         "gene",
		EvidenceRefs: []string{"ref_a", "ref_outside"},
	}
	bundle := EvolutionEvidenceBundle{EvidenceRefs: []string{"ref_a", "ref_b"}}
	report, err := NormalizeCandidate(candidate, bundle)
	if err != ErrLeakageDetected {
		t.Fatalf("err=%v, want ErrLeakageDetected", err)
	}
	if report.Clean {
		t.Fatal("report should not be clean")
	}
	if len(report.Violations) == 0 {
		t.Fatal("expected violations")
	}
}

func TestNoTraceNoCapsuleClaim(t *testing.T) {
	candidate := GeneCandidate{Kind: "capsule", TraceRef: "", EvidenceRefs: []string{}}
	bundle := EvolutionEvidenceBundle{EvidenceRefs: []string{}}
	report, err := NormalizeCandidate(candidate, bundle)
	if err != ErrLeakageDetected {
		t.Fatalf("err=%v, want ErrLeakageDetected", err)
	}
	if report.Clean {
		t.Fatal("capsule without trace must not be clean")
	}
}

func TestHubPublishImpossibleWithoutPolicyAndApproval(t *testing.T) {
	policy := DefaultHubPolicy()
	if policy.CanPublishHub(false) {
		t.Fatal("publish without approval must be false")
	}
	if policy.CanPublishHub(true) {
		t.Fatal("publish with approval but disabled must be false")
	}
}

func TestHostArtifactCannotInstallWithoutApproval(t *testing.T) {
	policy := DefaultHubPolicy()
	if policy.CanInstallArtifact(false) {
		t.Fatal("install without approval must be false")
	}
	if !policy.CanInstallArtifact(true) {
		t.Fatal("install with explicit approval should be true")
	}
}

func TestEvolverUnavailableDegradesOnlyEvolution(t *testing.T) {
	svc := testService(t, nil)
	_, err := svc.StartRun(context.Background(), EvolutionEvidenceBundle{Trigger: "test"}, "agent")
	if err != ErrProviderUnavailable {
		t.Fatalf("err=%v, want ErrProviderUnavailable", err)
	}
}

func TestBundleValidationRejectsProhibitedRefs(t *testing.T) {
	bundle := EvolutionEvidenceBundle{
		EvidenceRefs: []string{"src/main.go", "config/.env.production"},
	}
	if err := ValidateBundleInput(bundle); err != ErrEvidenceScope {
		t.Fatalf("err=%v, want ErrEvidenceScope", err)
	}
}

func TestRunLifecycle(t *testing.T) {
	provider := &mockProvider{pollStatus: RunStatus{RunID: "run_test_1", Status: "completed"}}
	svc := testService(t, provider)

	run, err := svc.StartRun(context.Background(), EvolutionEvidenceBundle{
		Trigger:      "test failure",
		EvidenceRefs: []string{"trace_001"},
	}, "agent")
	if err != nil {
		t.Fatal(err)
	}
	if run.RunID != "run_test_1" || run.Status != "running" {
		t.Fatalf("run=%+v", run)
	}

	got, err := svc.GetRun(context.Background(), "run_test_1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "completed" {
		t.Fatalf("status=%s, want completed", got.Status)
	}
}

func TestProposalReviewCreatesEvent(t *testing.T) {
	provider := &mockProvider{}
	svc := testService(t, provider)
	ctx := context.Background()

	run, _ := svc.StartRun(ctx, EvolutionEvidenceBundle{Trigger: "test", EvidenceRefs: []string{"ref_1"}}, "agent")
	candidate := GeneCandidate{CandidateID: "cand_1", RunID: run.RunID, Kind: "gene", Name: "test_gene", EvidenceRefs: []string{}}
	_ = svc.Store.SaveCandidate(ctx, candidate)

	proposal, err := svc.CreateProposal(ctx, run.RunID, "cand_1", "agent")
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Status != "pending" {
		t.Fatalf("status=%s", proposal.Status)
	}

	reviewed, err := svc.ReviewProposal(ctx, proposal.ProposalID, "approved", "user", "looks good")
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.Status != "approved" || reviewed.Reviewer != "user" {
		t.Fatalf("reviewed=%+v", reviewed)
	}

	events, err := svc.Events.ByRun(ctx, run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 2 {
		t.Fatalf("events=%d, want >= 2 (run_started + proposal_created + approved)", len(events))
	}
}

func TestNormalizeCleanCandidate(t *testing.T) {
	candidate := GeneCandidate{
		Kind:         "gene",
		EvidenceRefs: []string{"ref_a"},
		Payload:      map[string]any{"code": "fmt.Println()"},
	}
	bundle := EvolutionEvidenceBundle{EvidenceRefs: []string{"ref_a", "ref_b"}}
	report, err := NormalizeCandidate(candidate, bundle)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !report.Clean {
		t.Fatalf("report=%+v", report)
	}
}
