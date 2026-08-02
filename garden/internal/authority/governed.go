package authority

import (
	"context"

	"github.com/dashimaki/laputa/governance"
)

type GovernedWriter struct {
	Gov *governance.GovernedService
}

func (g *GovernedWriter) ApplyMutation(ctx context.Context, req governance.MutationRequest) error {
	return g.Gov.Mutate(ctx, req)
}

func (g *GovernedWriter) RecentAudit(ctx context.Context, limit int) ([]governance.AuditEntry, error) {
	return g.Gov.RecentAudit(ctx, limit)
}
