package router

import (
	"context"
	"errors"
	"strings"
)

// Backend is the unified CRUD surface for governance and mentle.
type Backend interface {
	Write(ctx context.Context, key, value string, meta map[string]any) (string, error)
	Read(ctx context.Context, key string) (map[string]any, error)
	List(ctx context.Context, prefix string, limit int) ([]map[string]any, error)
	Forget(ctx context.Context, key string) (bool, error)
}

// Router dispatches keys to governance or mentle by prefix.
type Router struct {
	Governance Backend
	Mentle     Backend
}

// Route selects the backend for key or prefix.
func (r *Router) Route(key string) (Backend, error) {
	if strings.HasPrefix(key, "section:") {
		if r.Governance == nil {
			return nil, errors.New("governance backend not configured")
		}
		return r.Governance, nil
	}
	if strings.HasPrefix(key, "memory:") {
		if r.Mentle == nil {
			return nil, errors.New("mentle backend not configured")
		}
		return r.Mentle, nil
	}
	return nil, errors.New("unknown key prefix")
}
