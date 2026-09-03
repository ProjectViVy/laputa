package evolution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	CandidateActmemExcerptCap = 1200
	CandidateSignalCap        = 280
	CandidateRefCap           = 256
	CandidateRefCountCap      = 50
)

var (
	ErrCandidateInputInvalid   = errors.New("evolution: candidate input is invalid")
	ErrCandidateAuthorityInput = errors.New("evolution: candidate input cannot contain Persona authority material")
	ErrCandidateInputTooLarge  = errors.New("evolution: candidate input exceeds its bound")
)

// EvolutionCandidateInput is the only input shape accepted by the bounded
// candidate assembler. ACTMEM contributes an explicitly selected excerpt;
// activity, trace, and Mentle contribute opaque source references. It has no
// Persona body/history fields and no artifact or installation capability.
type EvolutionCandidateInput struct {
	Trigger            string   `json:"trigger"`
	Outcome            string   `json:"outcome"`
	ActmemExcerpt      string   `json:"actmem_excerpt"`
	ActivityRefs       []string `json:"activity_refs"`
	TraceRefs          []string `json:"trace_refs"`
	EvidenceRefs       []string `json:"evidence_refs"`
	ContentHashes      []string `json:"content_hashes"`
	PrivacyLevel       string   `json:"privacy_level"`
	AllowedScopes      []string `json:"allowed_scopes"`
	PublicationAllowed bool     `json:"publication_allowed"`
}

// Validate applies size and authority-boundary checks before any candidate is
// persisted or sent to a provider.
func (in EvolutionCandidateInput) Validate() error {
	if strings.TrimSpace(in.Trigger) == "" {
		return fmt.Errorf("%w: trigger is required", ErrCandidateInputInvalid)
	}
	if utf8.RuneCountInString(in.Trigger) > CandidateSignalCap || utf8.RuneCountInString(in.Outcome) > CandidateSignalCap {
		return ErrCandidateInputTooLarge
	}
	if utf8.RuneCountInString(in.ActmemExcerpt) > CandidateActmemExcerptCap {
		return ErrCandidateInputTooLarge
	}
	for _, ref := range append(append(append([]string{}, in.ActivityRefs...), in.TraceRefs...), in.EvidenceRefs...) {
		if err := validateCandidateRef(ref); err != nil {
			return err
		}
	}
	if len(in.ActivityRefs)+len(in.TraceRefs)+len(in.EvidenceRefs) > CandidateRefCountCap {
		return ErrCandidateInputTooLarge
	}
	for _, hash := range in.ContentHashes {
		if strings.TrimSpace(hash) == "" || utf8.RuneCountInString(hash) > CandidateRefCap {
			return ErrCandidateInputInvalid
		}
	}
	if len(in.ContentHashes) > CandidateRefCountCap {
		return ErrCandidateInputTooLarge
	}
	if containsPersonaMaterial(in.ActmemExcerpt) {
		return ErrCandidateAuthorityInput
	}
	return nil
}

func validateCandidateRef(ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" || utf8.RuneCountInString(ref) > CandidateRefCap {
		return ErrCandidateInputInvalid
	}
	if containsPersonaMaterial(ref) {
		return ErrCandidateAuthorityInput
	}
	return nil
}

func containsPersonaMaterial(value string) bool {
	lower := strings.ToLower(value)
	for _, needle := range []string{"persona/", "/persona", "persona\\", "identity.md", "relationship.md", "redline.md", "user.md", "dream.md", "dark.md", "world.md", "persona history", "/history/", "history/"} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

// BuildCandidateBundle produces an EvoMap-owned evidence bundle. Publication
// is always disabled here; a later, explicit EvoMap policy transition is the
// only route to artifact publication or installation.
func BuildCandidateBundle(in EvolutionCandidateInput) (EvolutionEvidenceBundle, error) {
	if err := in.Validate(); err != nil {
		return EvolutionEvidenceBundle{}, err
	}
	refs := make([]string, 0, len(in.ActivityRefs)+len(in.TraceRefs)+len(in.EvidenceRefs))
	refs = append(refs, in.ActivityRefs...)
	refs = append(refs, in.TraceRefs...)
	refs = append(refs, in.EvidenceRefs...)
	hashInput := strings.Join(append(append([]string{in.Trigger, in.Outcome, in.ActmemExcerpt}, refs...), in.ContentHashes...), "\x00")
	digest := sha256.Sum256([]byte(hashInput))
	privacy := strings.TrimSpace(in.PrivacyLevel)
	if privacy == "" {
		privacy = "local"
	}
	scopes := append([]string(nil), in.AllowedScopes...)
	if len(scopes) == 0 {
		scopes = []string{"activity", "trace", "evidence"}
	}
	return EvolutionEvidenceBundle{
		BundleID:       "candidate_" + hex.EncodeToString(digest[:8]),
		Trigger:        in.Trigger,
		Outcome:        in.Outcome,
		TraceRef:       first(in.TraceRefs),
		EvidenceRefs:   append([]string(nil), in.EvidenceRefs...),
		ContentHashes:  append([]string(nil), in.ContentHashes...),
		SourceRevision: "sha256:" + hex.EncodeToString(digest[:]),
		Policy:         BundlePolicy{PrivacyLevel: privacy, AllowedScopes: scopes, PublicationAllowed: false},
	}, nil
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// StartRunFromCandidateInput keeps the provider boundary explicit: input is
// assembled into a bounded run bundle, never into a Skill, HostArtifact, or
// direct installation request.
func (s *Service) StartRunFromCandidateInput(ctx context.Context, in EvolutionCandidateInput, actor string) (EvolutionRun, error) {
	bundle, err := BuildCandidateBundle(in)
	if err != nil {
		return EvolutionRun{}, err
	}
	return s.StartRun(ctx, bundle, actor)
}
