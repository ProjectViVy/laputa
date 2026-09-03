package evolution

import (
	"errors"
	"strings"
	"testing"
)

func TestBuildCandidateBundleKeepsInputsBoundedAndDisablesPublication(t *testing.T) {
	bundle, err := BuildCandidateBundle(EvolutionCandidateInput{
		Trigger:       "repeated export failures",
		Outcome:       "propose a bounded diagnostic capability",
		ActmemExcerpt: "## Pulse\n- session=sess_1: export failed",
		ActivityRefs:  []string{"activity:evt_1"},
		TraceRefs:     []string{"trace:tr_1"},
		EvidenceRefs:  []string{"card:mem_1#fragment_1"},
		ContentHashes: []string{"sha256:abc"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.BundleID == "" || bundle.TraceRef != "trace:tr_1" {
		t.Fatalf("unexpected bundle identity: %+v", bundle)
	}
	if len(bundle.EvidenceRefs) != 1 || bundle.EvidenceRefs[0] != "card:mem_1#fragment_1" {
		t.Fatalf("evidence refs escaped bundle: %+v", bundle.EvidenceRefs)
	}
	if bundle.Policy.PublicationAllowed {
		t.Fatal("candidate input must not enable publication")
	}
}

func TestCandidateInputRejectsPersonaBodiesAndHistory(t *testing.T) {
	cases := []EvolutionCandidateInput{
		{Trigger: "x", ActmemExcerpt: "IDENTITY.MD\nprivate persona body"},
		{Trigger: "x", EvidenceRefs: []string{"persona/history/identity/1.md"}},
		{Trigger: "x", TraceRefs: []string{"/persona/requests/request.json"}},
	}
	for _, input := range cases {
		if !errors.Is(input.Validate(), ErrCandidateAuthorityInput) {
			t.Fatalf("input should be rejected at authority boundary: %+v", input)
		}
	}
}

func TestCandidateInputRejectsUnboundedExcerptAndReferences(t *testing.T) {
	if !errors.Is((EvolutionCandidateInput{Trigger: "x", ActmemExcerpt: strings.Repeat("x", CandidateActmemExcerptCap+1)}).Validate(), ErrCandidateInputTooLarge) {
		t.Fatal("unbounded ACTMEM excerpt accepted")
	}
	refs := make([]string, CandidateRefCountCap+1)
	for i := range refs {
		refs[i] = "activity:event"
	}
	if !errors.Is((EvolutionCandidateInput{Trigger: "x", ActivityRefs: refs}).Validate(), ErrCandidateInputTooLarge) {
		t.Fatal("too many references accepted")
	}
}
