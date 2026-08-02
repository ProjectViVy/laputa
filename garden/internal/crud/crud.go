package crud

import (
	"context"

	"github.com/dashimaki/garden/internal/router"
	"github.com/dashimaki/laputa/governance"
	"github.com/dashimaki/mentle/facade"
)

// Handler dispatches CRUD operations through the key router.
type Handler struct {
	Gov    *governance.Engine
	Facade *facade.Service
	Router *router.Router
}

// NewHandler wires governance and mentle facade through the key router.
// fac may be nil when mentle is unavailable; memory: keys will then fail at route time.
func NewHandler(gov *governance.Engine, fac *facade.Service) *Handler {
	var mentle router.Backend
	if fac != nil {
		mentle = router.NewMentleAdapter(fac)
	}
	return &Handler{
		Gov:    gov,
		Facade: fac,
		Router: &router.Router{
			Governance: router.NewGovernanceBackend(gov),
			Mentle:     mentle,
		},
	}
}

// Write stores a record at key.
func (h *Handler) Write(ctx context.Context, key, value string, meta map[string]any) (string, error) {
	backend, err := h.Router.Route(key)
	if err != nil {
		return "", err
	}
	return backend.Write(ctx, key, value, meta)
}

// Read retrieves a record by key.
func (h *Handler) Read(ctx context.Context, key string) (map[string]any, error) {
	backend, err := h.Router.Route(key)
	if err != nil {
		return nil, err
	}
	return backend.Read(ctx, key)
}

// List returns records matching prefix.
func (h *Handler) List(ctx context.Context, prefix string, limit int) ([]map[string]any, error) {
	backend, err := h.Router.Route(prefix)
	if err != nil {
		return nil, err
	}
	return backend.List(ctx, prefix, limit)
}

// Forget removes a record by key.
func (h *Handler) Forget(ctx context.Context, key string) (bool, error) {
	backend, err := h.Router.Route(key)
	if err != nil {
		return false, err
	}
	return backend.Forget(ctx, key)
}

func (h *Handler) CreateMemory(ctx context.Context, req facade.CreateMemoryRequest, idempotencyKey, bodyHash string) (facade.Memory, error) {
	if h.Facade == nil {
		return facade.Memory{}, facade.ErrUnavailable
	}
	return h.Facade.CreateMemory(ctx, req, idempotencyKey, bodyHash)
}

func (h *Handler) GetMemory(ctx context.Context, id string) (facade.Memory, error) {
	if h.Facade == nil {
		return facade.Memory{}, facade.ErrUnavailable
	}
	return h.Facade.GetMemory(ctx, id)
}

func (h *Handler) UpdateMemory(ctx context.Context, id string, req facade.UpdateMemoryRequest) (facade.Memory, error) {
	if h.Facade == nil {
		return facade.Memory{}, facade.ErrUnavailable
	}
	return h.Facade.UpdateMemory(ctx, id, req)
}

func (h *Handler) DeleteMemory(ctx context.Context, id, actor, requestID string) (facade.DeleteResult, error) {
	if h.Facade == nil {
		return facade.DeleteResult{}, facade.ErrUnavailable
	}
	return h.Facade.DeleteMemory(ctx, id, actor, requestID)
}

func (h *Handler) ListMemories(ctx context.Context, opts facade.ListMemoryOptions) (facade.MemoryPage, error) {
	if h.Facade == nil {
		return facade.MemoryPage{}, facade.ErrUnavailable
	}
	return h.Facade.ListMemories(ctx, opts)
}
