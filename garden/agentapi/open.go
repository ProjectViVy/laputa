package agentapi

import (
	"context"
	"strings"
	"sync"

	"github.com/dashimaki/garden/internal/runtimecore"
)

// Config contains explicit host-owned paths and trusted identity. Callers can
// bind a session later, but cannot replace the configured principal or actor.
type Config struct {
	PersonaDir        string
	PalacePath        string
	ModelsDir         string
	StateDB           string
	ProfileID         string
	AgentID           string
	Platform          string
	RequireLocalModel bool
	Principal         Principal
	// WorkspaceID is the host-issued trusted workspace binding; empty is the
	// implicit personal workspace.
	WorkspaceID string
}

// Client owns a domain runtime without opening a listener or exporting storage handles.
type Client struct {
	mu        sync.RWMutex
	runtime   *runtimecore.Garden
	principal Principal
	identity  Binding
}

// Open composes an isolated single-profile domain runtime for a trusted Go host.
func Open(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Principal == "" {
		return nil, failure("authentication_required", "trusted host principal required")
	}
	if strings.TrimSpace(cfg.AgentID) == "" || strings.TrimSpace(cfg.Platform) == "" {
		return nil, failure("invalid_binding", "trusted host agent_id and platform required")
	}
	if err := Authorize(cfg.ProfileID, Binding{ProfileID: cfg.ProfileID}, cfg.Principal, OpBootstrap); err != nil {
		return nil, err
	}
	core, err := runtimecore.Open(ctx, runtimecore.Config{
		PersonaDir: cfg.PersonaDir, PalacePath: cfg.PalacePath, ModelsDir: cfg.ModelsDir,
		StateDB: cfg.StateDB, ProfileID: cfg.ProfileID, RequireLocalModel: cfg.RequireLocalModel,
	})
	if err != nil {
		return nil, err
	}
	return &Client{runtime: core, principal: cfg.Principal, identity: Binding{ProfileID: cfg.ProfileID, AgentID: cfg.AgentID, Platform: cfg.Platform, WorkspaceID: cfg.WorkspaceID}}, nil
}

// Close releases owned resources; a bound handle cannot continue after closure.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runtime == nil {
		return nil
	}
	err := c.runtime.Close()
	c.runtime = nil
	return err
}

// BoundClient fixes a host Session to the identity configured at Open; binding
// creates no session, run, journal, or database record.
type BoundClient struct {
	client  *Client
	binding Binding
}

// BindSession fixes only the host session; trusted identity comes from Open.
func (c *Client) BindSession(sessionID string) (*BoundClient, error) {
	if c == nil {
		return nil, failure("unavailable", "Garden runtime unavailable")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.runtime == nil {
		return nil, failure("unavailable", "Garden runtime unavailable")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, failure("invalid_binding", "session_id is required")
	}
	binding := c.identity
	binding.SessionID = sessionID
	return &BoundClient{client: c, binding: binding}, nil
}

func (b *BoundClient) service() (*Service, func(), error) {
	if b == nil || b.client == nil {
		return nil, nil, failure("unavailable", "Garden runtime unavailable")
	}
	b.client.mu.RLock()
	if b.client.runtime == nil {
		b.client.mu.RUnlock()
		return nil, nil, failure("unavailable", "Garden runtime unavailable")
	}
	return NewService(b.client.runtime), b.client.mu.RUnlock, nil
}

// Requests may omit Binding; an explicit conflicting binding fails instead of
// quietly changing the host-bound identity.
func (b *BoundClient) requestBinding(supplied Binding) (Binding, error) {
	if supplied != (Binding{}) && supplied != b.binding {
		return Binding{}, failure("invalid_binding", "request binding does not match host binding")
	}
	return b.binding, nil
}

func (b *BoundClient) Bootstrap(ctx context.Context, req BootstrapRequest) (BootstrapResponse, error) {
	s, unlock, err := b.service()
	if err != nil {
		return BootstrapResponse{}, err
	}
	defer unlock()
	req.Binding, err = b.requestBinding(req.Binding)
	if err != nil {
		return BootstrapResponse{}, err
	}
	return s.Bootstrap(ctx, b.client.principal, req)
}

func (b *BoundClient) FastRecall(ctx context.Context, req FastRecallRequest) (ContextView, error) {
	s, unlock, err := b.service()
	if err != nil {
		return ContextView{}, err
	}
	defer unlock()
	req.Binding, err = b.requestBinding(req.Binding)
	if err != nil {
		return ContextView{}, err
	}
	return s.FastRecall(ctx, b.client.principal, req)
}

func (b *BoundClient) Capture(ctx context.Context, req CaptureRequest) (CaptureReceipt, error) {
	s, unlock, err := b.service()
	if err != nil {
		return CaptureReceipt{}, err
	}
	defer unlock()
	req.Binding, err = b.requestBinding(req.Binding)
	if err != nil {
		return CaptureReceipt{}, err
	}
	return s.Capture(ctx, b.client.principal, req)
}

// CaptureStatus uses the durable receipt identity, scoped to this bound host session.
func (b *BoundClient) CaptureStatus(ctx context.Context, ingestionID, eventID string) (CaptureStatus, error) {
	s, unlock, err := b.service()
	if err != nil {
		return CaptureStatus{}, err
	}
	defer unlock()
	bound := b.binding
	bound.EventID = eventID
	return s.CaptureStatus(ctx, b.client.principal, bound, ingestionID)
}
