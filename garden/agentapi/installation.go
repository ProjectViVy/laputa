package agentapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/ProjectViVy/laputa/garden/internal/registry"
	"github.com/ProjectViVy/laputa/garden/internal/runtimecore"
)

type InstallationConfig struct {
	Home, StateDB, PalacePath, ModelsDir, BackendID, DestinationID string
	RequireLocalModel                                              bool
}
type Profile = registry.Profile
type ProfileRegistration = registry.ProfileRegistration
type ClientConfig struct {
	AgentID, Platform, WorkspaceID string
	Principal                      Principal
}
type Installation struct {
	owner                    *runtimecore.Owner
	backendID, destinationID string
}

func OpenInstallation(ctx context.Context, cfg InstallationConfig) (*Installation, error) {
	if cfg.BackendID != "" && cfg.BackendID != BackendMentle {
		return nil, failure("backend_unavailable", "configured backend not qualified")
	}
	if cfg.DestinationID != "" && cfg.DestinationID != BackendMentle {
		return nil, failure("invalid_binding", "Mentle destination must match configured writer")
	}
	owner, err := runtimecore.OpenOwner(ctx, runtimecore.OwnerConfig{Home: cfg.Home, Config: runtimecore.Config{PalacePath: cfg.PalacePath, ModelsDir: cfg.ModelsDir, StateDB: cfg.StateDB, RequireLocalModel: cfg.RequireLocalModel}})
	if err != nil {
		return nil, err
	}
	return NewInstallation(owner), nil
}

// NewInstallation adapts the application's already-open owner; it never opens
// another backend or exports a storage handle through the domain API.
func NewInstallation(owner *runtimecore.Owner) *Installation {
	return &Installation{owner: owner, backendID: BackendMentle, destinationID: BackendMentle}
}
func (i *Installation) Close() error {
	if i == nil || i.owner == nil {
		return nil
	}
	return i.owner.Close()
}
func (i *Installation) RegisterProfile(ctx context.Context, req ProfileRegistration) (Profile, error) {
	return i.owner.RegisterProfile(ctx, req)
}
func (i *Installation) ListProfiles(ctx context.Context) ([]Profile, error) {
	unlock, err := i.owner.Enter()
	if err != nil {
		return nil, failure("unavailable", "installation closed")
	}
	defer unlock()
	return i.owner.Registry.ListProfiles(ctx)
}

func (i *Installation) BindProfile(profileID string, cfg ClientConfig) (*Client, error) {
	if cfg.Principal == "" {
		return nil, failure("authentication_required", "trusted host principal required")
	}
	if strings.TrimSpace(cfg.AgentID) == "" || strings.TrimSpace(cfg.Platform) == "" {
		return nil, failure("invalid_binding", "trusted host identity required")
	}
	if err := Authorize(profileID, Binding{ProfileID: profileID}, cfg.Principal, OpBootstrap); err != nil {
		return nil, err
	}
	g, err := i.owner.Subject(context.Background(), profileID)
	if err != nil {
		return nil, err
	}
	rb, err := i.owner.Registry.TrustedBinding(context.Background(), registry.HostBinding{ProfileID: profileID, AgentID: cfg.AgentID, Platform: cfg.Platform, WorkspaceID: cfg.WorkspaceID, Controls: HostControls{PersonaSync: true, Recall: true, Capture: true}, Operations: trustedOperations})
	if err != nil {
		return nil, err
	}
	return &Client{bindingID: rb.BindingID, runtime: g, principal: cfg.Principal, identity: Binding{ProfileID: profileID, AgentID: cfg.AgentID, Platform: cfg.Platform, WorkspaceID: cfg.WorkspaceID}, backendID: i.backendID, destinationID: i.destinationID, installation: i}, nil
}

func bindingKey(b Binding) string {
	return fmt.Sprintf("%d:%s%d:%s%d:%s", len(b.AgentID), b.AgentID, len(b.Platform), b.Platform, len(b.WorkspaceID), b.WorkspaceID)
}

type boundFrozen struct {
	runtime   *runtimecore.Garden
	binding   Binding
	bindingID string
}

func (p *boundFrozen) Get(ctx context.Context, sessionID string) (FrozenCore, error) {
	key, err := p.runtime.SessionKey(p.bindingID, sessionID)
	if err != nil {
		return FrozenCore{}, err
	}
	return p.runtime.Frozen.CaptureKey(ctx, key, sessionID, p.runtime.Persona)
}
