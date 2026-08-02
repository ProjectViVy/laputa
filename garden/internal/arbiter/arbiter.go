package arbiter

import (
	"fmt"
	"math"
)

type Assertion struct {
	Subject      string   `json:"subject"`
	Predicate    string   `json:"predicate"`
	Object       string   `json:"object"`
	ValidFrom    string   `json:"valid_from,omitempty"`
	ValidTo      string   `json:"valid_to,omitempty"`
	Confidence   float64  `json:"confidence"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Status       string   `json:"status"`
}

type ConflictType string

const (
	ConflictReplace ConflictType = "replace"
	ConflictRefine  ConflictType = "refine"
	ConflictCoexist ConflictType = "coexist"
	ConflictRetract ConflictType = "retract"
)

type ValidityChange struct {
	AssertionKey string `json:"assertion_key"`
	OldValidTo   string `json:"old_valid_to,omitempty"`
	NewValidTo   string `json:"new_valid_to,omitempty"`
}

type SupersedesEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type ConflictProposal struct {
	ConflictType       ConflictType     `json:"conflict_type"`
	AffectedAssertions []string         `json:"affected_assertions"`
	ValidityChanges    []ValidityChange `json:"validity_changes,omitempty"`
	SupersedesEdges    []SupersedesEdge `json:"supersedes_edges,omitempty"`
	EvidenceRefs       []string         `json:"evidence_refs"`
	Confidence         float64          `json:"confidence"`
	Rationale          string           `json:"rationale"`
}

type Arbiter struct{}

func New() *Arbiter {
	return &Arbiter{}
}

func AssertionKey(a Assertion) string {
	return fmt.Sprintf("%s/%s/%s", a.Subject, a.Predicate, a.Object)
}

func (a *Arbiter) DetectConflicts(assertions []Assertion) []ConflictProposal {
	type groupKey struct{ Subject, Predicate string }
	groups := map[groupKey][]Assertion{}
	for _, as := range assertions {
		k := groupKey{as.Subject, as.Predicate}
		groups[k] = append(groups[k], as)
	}

	var proposals []ConflictProposal
	for _, group := range groups {
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				if p := comparePair(group[i], group[j]); p != nil {
					proposals = append(proposals, *p)
				}
			}
		}
	}
	return proposals
}

func comparePair(a, b Assertion) *ConflictProposal {
	keyA := AssertionKey(a)
	keyB := AssertionKey(b)

	if a.Object == b.Object {
		if !periodsOverlap(a, b) {
			return &ConflictProposal{
				ConflictType:       ConflictCoexist,
				AffectedAssertions: []string{keyA, keyB},
				EvidenceRefs:       mergeRefs(a.EvidenceRefs, b.EvidenceRefs),
				Confidence:         min(a.Confidence, b.Confidence),
				Rationale:          "same object, non-overlapping validity periods",
			}
		}
		return nil
	}

	if !periodsOverlap(a, b) {
		return &ConflictProposal{
			ConflictType:       ConflictCoexist,
			AffectedAssertions: []string{keyA, keyB},
			EvidenceRefs:       mergeRefs(a.EvidenceRefs, b.EvidenceRefs),
			Confidence:         min(a.Confidence, b.Confidence),
			Rationale:          "different objects, non-overlapping validity periods",
		}
	}

	if math.Abs(a.Confidence-b.Confidence) < 0.1 {
		return &ConflictProposal{
			ConflictType:       ConflictRefine,
			AffectedAssertions: []string{keyA, keyB},
			EvidenceRefs:       mergeRefs(a.EvidenceRefs, b.EvidenceRefs),
			Confidence:         max(a.Confidence, b.Confidence),
			Rationale:          "overlapping assertions with similar confidence; insufficient evidence to replace",
		}
	}

	winner, loser := a, b
	if b.Confidence > a.Confidence || (b.Confidence == a.Confidence && b.ValidFrom > a.ValidFrom) {
		winner, loser = b, a
	}
	winnerKey := AssertionKey(winner)
	loserKey := AssertionKey(loser)

	return &ConflictProposal{
		ConflictType:       ConflictReplace,
		AffectedAssertions: []string{winnerKey, loserKey},
		ValidityChanges: []ValidityChange{{
			AssertionKey: loserKey,
			OldValidTo:   loser.ValidTo,
			NewValidTo:   winner.ValidFrom,
		}},
		SupersedesEdges: []SupersedesEdge{{From: winnerKey, To: loserKey}},
		EvidenceRefs:    mergeRefs(winner.EvidenceRefs, loser.EvidenceRefs),
		Confidence:      winner.Confidence,
		Rationale:       fmt.Sprintf("%s supersedes %s (confidence %.2f > %.2f)", winnerKey, loserKey, winner.Confidence, loser.Confidence),
	}
}

func periodsOverlap(a, b Assertion) bool {
	if a.ValidTo == "" && b.ValidTo == "" {
		return true
	}
	if a.ValidTo == "" {
		return a.ValidFrom <= b.ValidTo
	}
	if b.ValidTo == "" {
		return b.ValidFrom <= a.ValidTo
	}
	return a.ValidFrom <= b.ValidTo && b.ValidFrom <= a.ValidTo
}

func mergeRefs(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range append(a, b...) {
		if !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	return out
}
