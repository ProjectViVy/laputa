package router

import (
	"context"

	"github.com/dashimaki/mentle/facade"
)

// MentleAdapter wraps facade.Service as a CRUD backend.
type MentleAdapter struct {
	Service *facade.Service
}

// NewMentleAdapter wraps a mentle facade service as a CRUD backend.
func NewMentleAdapter(service *facade.Service) *MentleAdapter {
	return &MentleAdapter{Service: service}
}

func (a *MentleAdapter) Write(ctx context.Context, key, value string, meta map[string]any) (string, error) {
	return a.Service.Write(ctx, key, value, meta)
}

func (a *MentleAdapter) Read(ctx context.Context, key string) (map[string]any, error) {
	return a.Service.Read(ctx, key)
}

func (a *MentleAdapter) List(ctx context.Context, prefix string, limit int) ([]map[string]any, error) {
	return a.Service.List(ctx, prefix, limit)
}

func (a *MentleAdapter) Forget(ctx context.Context, key string) (bool, error) {
	return a.Service.Forget(ctx, key)
}
