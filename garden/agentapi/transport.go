package agentapi

import (
	"context"
	"github.com/ProjectViVy/laputa/garden/memory"
)

// AdmittedService adapts a trusted client into an HTTP handler while keeping
// its host identity fixed. Each service operation rechecks current authority.
func (c *Client) AdmittedService(sessionID string) (*Service, Binding, error) {
	b, err := c.BindSession(sessionID)
	if err != nil {
		return nil, Binding{}, err
	}
	return c.newService(), b.binding, nil
}
func (b *BoundClient) AdmittedService() (*Service, Binding) { return b.client.newService(), b.binding }
func (s *Service) Check(ctx context.Context, p Principal, b Binding, op Operation) error {
	return s.check(b, p, op)
}

func (s *Service) ListMemory(ctx context.Context, p Principal, b Binding, q memory.ListRequest) (memory.RecordPage, error) {
	if err := s.check(b, p, OpSearch); err != nil {
		return memory.RecordPage{}, err
	}
	if s.installation != nil {
		return s.accessBackend(b, p, MemoryTarget{}).List(ctx, q)
	}
	backend, scopes, err := s.memoryBackend(b)
	if err != nil {
		return memory.RecordPage{}, err
	}
	m, ok := backend.(memory.ManagementBackend)
	if !ok {
		return memory.RecordPage{}, failure("unavailable", "management unavailable")
	}
	q.Scopes, err = requestedScopes(q.Scopes, scopes)
	if err != nil {
		return memory.RecordPage{}, err
	}
	out, err := m.List(ctx, q)
	if p != PrincipalUser {
		for j := range out.Items {
			out.Items[j].Body = ""
		}
	}
	return out, materialErrorOrNil(err)
}
func (s *Service) GetMemory(ctx context.Context, p Principal, b Binding, id string) (memory.Record, error) {
	if err := s.check(b, p, OpSearch); err != nil {
		return memory.Record{}, err
	}
	if s.installation != nil {
		return s.accessBackend(b, p, MemoryTarget{}).Get(ctx, id)
	}
	backend, _, err := s.memoryBackend(b)
	if err != nil {
		return memory.Record{}, err
	}
	m, ok := backend.(memory.ManagementBackend)
	if !ok {
		return memory.Record{}, failure("unavailable", "management unavailable")
	}
	out, err := m.Get(ctx, id)
	if p != PrincipalUser {
		out.Body = ""
	}
	return out, materialErrorOrNil(err)
}
func (s *Service) MutateMemory(ctx context.Context, p Principal, b Binding, target MemoryTarget, m memory.AuthorizedMutation) (memory.MutationReceipt, error) {
	return s.mutateMemory(ctx, p, b, target, m)
}
