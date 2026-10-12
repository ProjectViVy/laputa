// Package hosts owns consented host-file projections; it never owns Persona.
package hosts

import (
	"context"
	"errors"
	"github.com/ProjectViVy/laputa/garden/internal/registry"
	"github.com/ProjectViVy/laputa/laputa/persona"
	"os"
	"path/filepath"
	"sync"
)

type Service struct {
	mu        sync.Mutex
	closed    bool
	registry  *registry.Store
	backupDir string
	persona   func(context.Context, string) (*persona.Service, error)
	// Qualified adapters may supply proof of an exact authenticated session
	// loading this projection. No adapters are qualified by this common core.
	LoadedEvidence func(context.Context, registry.HostBinding, string, string) error
}

func New(r *registry.Store, backupDir string, reader func(context.Context, string) (*persona.Service, error)) *Service {
	return &Service{registry: r, backupDir: backupDir, persona: reader}
}

type Candidate struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	State  string `json:"state"`
}

func (s *Service) Discover(paths []string) ([]Candidate, error) {
	out := []Candidate{}
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			return nil, errors.New("absolute approved discovery path required")
		}
		info, err := os.Lstat(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if info != nil && !info.Mode().IsRegular() {
			return nil, errors.New("host path must be a regular file")
		}
		out = append(out, Candidate{Path: path, Exists: err == nil, State: "inactive"})
	}
	return out, nil
}
func (s *Service) binding(ctx context.Context, id string, syncRequired bool) (registry.HostBinding, error) {
	if s.closed {
		return registry.HostBinding{}, errors.New("host service closed")
	}
	b, err := s.registry.Binding(ctx, id)
	if err != nil {
		return b, err
	}
	if b.Revoked || b.Worker {
		return b, errors.New("host binding revoked or persona-free")
	}
	if syncRequired && !b.Controls.PersonaSync {
		return b, errors.New("persona sync disabled")
	}
	return b, nil
}

func (s *Service) Close() { s.mu.Lock(); defer s.mu.Unlock(); s.closed = true }
