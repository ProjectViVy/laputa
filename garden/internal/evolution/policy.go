package evolution

import (
	"errors"
	"strings"
)

var (
	ErrHubDisabled       = errors.New("evolution: hub publish is disabled by default")
	ErrAutoInstallDenied = errors.New("evolution: automatic installation is not permitted")
	ErrEvidenceScope     = errors.New("evolution: evidence refs exceed allowed scope")
	ErrLeakageDetected   = errors.New("evolution: leakage detected in candidate output")
	ErrInvalidDecision   = errors.New("evolution: decision must be approved or rejected")
)

const maxEvidenceRefs = 50

type HubPolicy struct {
	LocalEvolutionAllowed      bool
	HostExportRequiresApproval bool
	HubPublishEnabled          bool
	HubFetchRequiresAudit      bool
}

func DefaultHubPolicy() HubPolicy {
	return HubPolicy{
		LocalEvolutionAllowed:      true,
		HostExportRequiresApproval: true,
		HubPublishEnabled:          false,
		HubFetchRequiresAudit:      true,
	}
}

func (p HubPolicy) CanPublishHub(explicitApproval bool) bool {
	return p.HubPublishEnabled && explicitApproval
}

func (p HubPolicy) CanInstallArtifact(explicitApproval bool) bool {
	return explicitApproval
}

func ValidateBundleInput(bundle EvolutionEvidenceBundle) error {
	if len(bundle.EvidenceRefs) > maxEvidenceRefs {
		return ErrEvidenceScope
	}
	for _, ref := range bundle.EvidenceRefs {
		if isProhibitedRef(ref) {
			return ErrEvidenceScope
		}
	}
	return nil
}

// CheckOutbound runs the mechanical privacy gate over an outbound payload
// and its evidence refs (ADR-0007 §4).
func CheckOutbound(payload map[string]any, evidenceRefs []string) (LeakageReport, error) {
	report := LeakageReport{Clean: true, CheckedRefs: len(evidenceRefs)}
	if err := ValidateBundleInput(EvolutionEvidenceBundle{EvidenceRefs: evidenceRefs}); err != nil {
		report.Clean = false
		report.Violations = append(report.Violations, err.Error())
	}
	if containsProhibitedKeys(payload) {
		report.Clean = false
		report.Violations = append(report.Violations, "payload contains prohibited keys")
	}
	if !report.Clean {
		return report, ErrLeakageDetected
	}
	return report, nil
}

func isProhibitedRef(ref string) bool {
	lower := strings.ToLower(ref)
	prohibited := []string{".env", "token", "secret", "personality", ".laputa/sections/01"}
	for _, p := range prohibited {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}
