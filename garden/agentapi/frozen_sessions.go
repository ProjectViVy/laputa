package agentapi

import (
	"context"
	"strings"
	"time"
)

// DiscardFrozenSession removes the admitted frozen-core snapshot of one
// session on the trusted host owner. It exists for host admission cleanup: a
// provisional capture whose owning admission lost or was never committed must
// not age into an authority row. Deletion is idempotent — a missing row is a
// success, never a not-found.
func (c *Client) DiscardFrozenSession(ctx context.Context, sessionID string) error {
	if c == nil {
		return failure("unavailable", "Garden runtime unavailable")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.runtime == nil {
		return failure("unavailable", "Garden runtime unavailable")
	}
	if c.runtime.Frozen == nil {
		return failure("unavailable", "frozen store unavailable")
	}
	if strings.TrimSpace(sessionID) == "" {
		return failure("invalid_binding", "session_id is required")
	}
	if err := c.runtime.Frozen.DiscardSession(ctx, sessionID); err != nil {
		return failure("unavailable", err.Error())
	}
	return nil
}

// ListFrozenSessions enumerates session IDs captured before capturedBefore,
// oldest first. A host startup sweep enumerates orphan candidates here and
// applies its own keep-predicate; the owner never decides which rows are
// orphans.
func (c *Client) ListFrozenSessions(ctx context.Context, capturedBefore time.Time) ([]string, error) {
	if c == nil {
		return nil, failure("unavailable", "Garden runtime unavailable")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.runtime == nil {
		return nil, failure("unavailable", "Garden runtime unavailable")
	}
	if c.runtime.Frozen == nil {
		return nil, failure("unavailable", "frozen store unavailable")
	}
	ids, err := c.runtime.Frozen.ListFrozenSessions(ctx, capturedBefore)
	if err != nil {
		return nil, failure("unavailable", err.Error())
	}
	return ids, nil
}
