package agentapi

import (
	"context"
	"errors"
	"slices"

	"github.com/ProjectViVy/laputa/garden/internal/registry"
)

type HostControls = registry.HostControls
type SharedSpace = registry.SharedSpace
type SharedMember = registry.SharedMember
type HostBinding struct {
	BindingID     string       `json:"binding_id"`
	ProfileID     string       `json:"profile_id"`
	AgentID       string       `json:"agent_id"`
	Platform      string       `json:"platform"`
	WorkspaceID   string       `json:"workspace_id"`
	Controls      HostControls `json:"controls"`
	Operations    []Operation  `json:"operations"`
	Worker        bool         `json:"worker"`
	Revoked       bool         `json:"revoked"`
	SelectedCards []string     `json:"selected_cards,omitempty"`
}

func accessError(err error) error {
	if err == nil {
		return nil
	}
	var ae *registry.AccessError
	if errors.As(err, &ae) {
		return failure(ae.Code, ae.Message)
	}
	return err
}
func (i *Installation) enter() (func(), error) {
	if i == nil || i.owner == nil {
		return nil, failure("unavailable", "installation unavailable")
	}
	u, err := i.owner.Enter()
	return u, accessError(err)
}

func (i *Installation) CreateHostBinding(ctx context.Context, b HostBinding) (HostBinding, string, error) {
	u, err := i.enter()
	if err != nil {
		return HostBinding{}, "", err
	}
	defer u()
	rb, err := convert[registry.HostBinding](b)
	if err != nil {
		return HostBinding{}, "", err
	}
	rb, token, err := i.owner.Registry.CreateHostBinding(ctx, rb)
	if err != nil {
		return HostBinding{}, "", accessError(err)
	}
	b, err = convert[HostBinding](rb)
	return b, token, err
}
func (i *Installation) AuthenticateHost(ctx context.Context, credential string) (HostBinding, error) {
	u, err := i.enter()
	if err != nil {
		return HostBinding{}, err
	}
	defer u()
	b, err := i.owner.Registry.Authenticate(ctx, credential)
	if err != nil {
		return HostBinding{}, accessError(err)
	}
	return convert[HostBinding](b)
}
func (i *Installation) HostBinding(ctx context.Context, id string) (HostBinding, error) {
	u, err := i.enter()
	if err != nil {
		return HostBinding{}, err
	}
	defer u()
	b, err := i.owner.Registry.Binding(ctx, id)
	if err != nil {
		return HostBinding{}, accessError(err)
	}
	return convert[HostBinding](b)
}
func (i *Installation) ListHostBindings(ctx context.Context) ([]HostBinding, error) {
	u, err := i.enter()
	if err != nil {
		return nil, err
	}
	defer u()
	b, err := i.owner.Registry.ListHostBindings(ctx)
	if err != nil {
		return nil, err
	}
	return convert[[]HostBinding](b)
}
func (i *Installation) SetHostControls(ctx context.Context, id string, c HostControls) error {
	u, err := i.enter()
	if err != nil {
		return err
	}
	defer u()
	return accessError(i.owner.Registry.SetHostControls(ctx, id, c))
}
func (i *Installation) RevokeHostBinding(ctx context.Context, id string) error {
	_, err := i.RevokeHostProjection(ctx, id)
	return err
}
func (i *Installation) CreateSharedSpace(ctx context.Context, name string) (SharedSpace, error) {
	u, err := i.enter()
	if err != nil {
		return SharedSpace{}, err
	}
	defer u()
	return i.owner.Registry.CreateSharedSpace(ctx, name)
}
func (i *Installation) SetSharedMember(ctx context.Context, spaceID, profileID, access string) error {
	u, err := i.enter()
	if err != nil {
		return err
	}
	defer u()
	return i.owner.Registry.SetSharedMember(ctx, spaceID, profileID, access)
}
func (i *Installation) RemoveSharedMember(ctx context.Context, spaceID, profileID string) error {
	u, err := i.enter()
	if err != nil {
		return err
	}
	defer u()
	return i.owner.Registry.RemoveSharedMember(ctx, spaceID, profileID)
}
func (i *Installation) ListSharedSpaces(ctx context.Context) ([]SharedSpace, error) {
	u, err := i.enter()
	if err != nil {
		return nil, err
	}
	defer u()
	return i.owner.Registry.ListSharedSpaces(ctx)
}
func (i *Installation) ListSharedMembers(ctx context.Context, spaceID string) ([]SharedMember, error) {
	u, err := i.enter()
	if err != nil {
		return nil, err
	}
	defer u()
	return i.owner.Registry.ListSharedMembers(ctx, spaceID)
}

func (i *Installation) BindHostSession(bindingID, sessionID string) (*BoundClient, error) {
	b, err := i.HostBinding(context.Background(), bindingID)
	if err != nil {
		return nil, err
	}
	if b.Revoked {
		return nil, failure("binding_revoked", "binding revoked")
	}
	g, err := i.owner.Subject(context.Background(), b.ProfileID)
	if err != nil {
		return nil, err
	}
	c := &Client{runtime: g, installation: i, bindingID: b.BindingID, principal: PrincipalAgent, identity: Binding{ProfileID: b.ProfileID, AgentID: b.AgentID, Platform: b.Platform, WorkspaceID: b.WorkspaceID}, backendID: i.backendID, destinationID: i.destinationID}
	return c.BindSession(sessionID)
}

var trustedOperations = []string{"bootstrap", "search", "expand", "index_health", "capture", "remember", "persona_get", "persona_propose", "persona_p16", "persona_review", "persona_repair", "actmem_read", "actmem_query", "actmem_write", "actmem_maintain", "actmem_save"}

func (s *Service) resolve(ctx context.Context, binding Binding, op Operation, spaceID string) (registry.ResolvedAccess, error) {
	if s.installation == nil {
		return registry.ResolvedAccess{}, nil
	}
	a, err := s.installation.owner.Registry.Resolve(ctx, registry.AccessRequest{BindingID: s.bindingID, SessionID: binding.SessionID, Operation: string(op), SharedSpaceID: spaceID})
	if err != nil {
		return registry.ResolvedAccess{}, accessError(err)
	}
	b, err := s.installation.owner.Registry.Binding(ctx, s.bindingID)
	if err != nil {
		return registry.ResolvedAccess{}, accessError(err)
	}
	if b.ProfileID != binding.ProfileID || b.AgentID != binding.AgentID || b.Platform != binding.Platform || b.WorkspaceID != binding.WorkspaceID {
		return registry.ResolvedAccess{}, failure("invalid_binding", "binding does not match admitted caller")
	}
	return a, nil
}

func (s *Service) checkSelectedCards(ctx context.Context, b Binding, q EvidenceRead) error {
	if s.installation == nil {
		return nil
	}
	a, err := s.resolve(ctx, b, OpExpand, "")
	if err != nil {
		return err
	}
	if !a.Worker {
		return nil
	}
	for _, item := range q.Items {
		if !slices.Contains(a.SelectedCards, item.CardID) {
			return failure("principal_forbidden", "worker source was not selected")
		}
	}
	return nil
}

func (c *Client) newService() *Service {
	return &Service{runtime: c.runtime, installation: c.installation, bindingID: c.bindingID, principal: c.principal}
}
