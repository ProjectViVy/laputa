package evolution

type LeakageReport struct {
	Clean       bool     `json:"clean"`
	Violations  []string `json:"violations,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
	CheckedRefs int      `json:"checked_refs"`
}

func NormalizeCandidate(candidate GeneCandidate, bundle EvolutionEvidenceBundle) (LeakageReport, error) {
	report := LeakageReport{Clean: true, CheckedRefs: len(candidate.EvidenceRefs)}

	if candidate.Kind == "capsule" && candidate.TraceRef == "" {
		report.Clean = false
		report.Violations = append(report.Violations, "capsule candidate has no trace_ref")
	}

	allowed := make(map[string]bool, len(bundle.EvidenceRefs))
	for _, ref := range bundle.EvidenceRefs {
		allowed[ref] = true
	}
	for _, ref := range candidate.EvidenceRefs {
		if !allowed[ref] {
			report.Clean = false
			report.Violations = append(report.Violations, "evidence ref outside bundle scope: "+ref)
		}
	}

	if containsProhibitedKeys(candidate.Payload) {
		report.Clean = false
		report.Violations = append(report.Violations, "payload contains prohibited keys")
	}

	if !report.Clean {
		return report, ErrLeakageDetected
	}
	return report, nil
}

func containsProhibitedKeys(payload map[string]any) bool {
	prohibited := []string{"api_key", "token", "secret", "password", "private_key"}
	for _, key := range prohibited {
		if _, ok := payload[key]; ok {
			return true
		}
	}
	return false
}
