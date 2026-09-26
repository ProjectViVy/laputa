package agentapi

import (
	"context"

	"github.com/dashimaki/mentle/facade"
)

// IndexHealth reports live Mentle canonical and derived-index health. A
// degraded index is a valid report; a missing or failed probe is unavailable.
// Binding and principal are checked before any Mentle access.
func (s *Service) IndexHealth(ctx context.Context, principal Principal, binding Binding) (facade.IndexHealth, error) {
	if err := s.check(binding, principal, OpIndexHealth); err != nil {
		return facade.IndexHealth{}, err
	}
	if s.runtime.Mentle == nil {
		return unavailableIndexHealth([]string{"canonical_probe_failed"})
	}
	health, err := s.runtime.Mentle.IndexHealth(ctx)
	if err != nil {
		apiErr := failure("index_health_unavailable", "live index probes unavailable")
		apiErr.Details["reasons"] = health.Reasons
		return health, apiErr
	}
	return health, nil
}

func unavailableIndexHealth(reasons []string) (facade.IndexHealth, error) {
	if len(reasons) == 0 {
		reasons = []string{"canonical_probe_failed"}
	}
	health := facade.IndexHealth{Status: "unavailable", Reasons: reasons}
	err := failure("index_health_unavailable", "live index probes unavailable")
	err.Details["reasons"] = reasons
	return health, err
}
