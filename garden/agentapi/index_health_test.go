package agentapi

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/ProjectViVy/laputa/garden/internal/runtimecore"
	"github.com/ProjectViVy/laputa/mentle/facade"
)

func TestIndexHealthRequiresTrustedPrincipalAndFixedBinding(t *testing.T) {
	s := NewService(&runtimecore.Garden{ProfileID: "default"})
	ctx := context.Background()
	_, err := s.IndexHealth(ctx, "", binding())
	assertCode(t, err, "authentication_required")
	b := binding()
	b.ProfileID = "other"
	_, err = s.IndexHealth(ctx, PrincipalAgent, b)
	assertCode(t, err, "profile_mismatch")
	b = binding()
	b.SessionID = ""
	_, err = s.IndexHealth(ctx, PrincipalAgent, b)
	assertCode(t, err, "index_health_unavailable")
	_, err = s.IndexHealth(ctx, Principal("untrusted"), binding())
	assertCode(t, err, "principal_forbidden")
}

func TestIndexHealthUnavailableDoesNotReportHealthyFallback(t *testing.T) {
	ctx := context.Background()
	for _, mentle := range []*facade.Service{nil, {}} {
		s := NewService(&runtimecore.Garden{ProfileID: "default", Mentle: mentle})
		health, err := s.IndexHealth(ctx, PrincipalAgent, binding())
		if health.Status != "unavailable" || !reflect.DeepEqual(health.Reasons, []string{"canonical_probe_failed"}) {
			t.Fatalf("health = %+v", health)
		}
		if mentle != nil && health.ObservedAt.IsZero() {
			t.Fatal("discarded Mentle live probe timestamp")
		}
		assertCode(t, err, "index_health_unavailable")
		var apiErr *Error
		if !errors.As(err, &apiErr) || !reflect.DeepEqual(apiErr.Details["reasons"], health.Reasons) {
			t.Fatalf("error does not expose public reason codes: %+v", apiErr)
		}
	}
}
