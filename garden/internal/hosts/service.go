// Package hosts owns consented host-file projections; it never owns Persona.
package hosts

import (
	"context"
	"errors"
	"github.com/ProjectViVy/laputa/garden/internal/registry"
	"github.com/ProjectViVy/laputa/laputa/persona"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Service struct {
	mu        sync.Mutex
	previews  map[string]*nativePreview
	now       func() time.Time
	closed    bool
	registry  *registry.Store
	backupDir string
	protected []string
	persona   func(context.Context, string) (*persona.Service, error)
	// Qualified adapters may supply proof of an exact authenticated session
	// loading this projection. No adapters are qualified by this common core.
	LoadedEvidence func(context.Context, registry.HostBinding, string, string) error
}

func New(r *registry.Store, backupDir string, reader func(context.Context, string) (*persona.Service, error), protected ...string) *Service {
	return &Service{previews: map[string]*nativePreview{}, now: time.Now, registry: r, backupDir: backupDir, persona: reader, protected: append([]string{filepath.Dir(backupDir)}, protected...)}
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
		return b, failure("principal_forbidden", "host binding revoked or persona-free")
	}
	if syncRequired && !b.Controls.PersonaSync {
		return b, failure("principal_forbidden", "persona sync disabled")
	}
	return b, nil
}

func (s *Service) Close() { s.mu.Lock(); defer s.mu.Unlock(); s.closed = true }

func failure(code, message string) error { return &registry.AccessError{Code: code, Message: message} }

// Host projections are derivatives, never a writer into owned control,
// memory or Persona authority. This is a filesystem ownership check.
func (s *Service) validateTarget(ctx context.Context, path string) error {
	target, err := registry.CanonicalPath(path)
	if err != nil {
		return failure("invalid_request", "invalid host target")
	}
	profiles, err := s.registry.ListProfiles(ctx)
	if err != nil {
		return err
	}
	roots := append([]string{}, s.protected...)
	for _, p := range profiles {
		roots = append(roots, p.AuthorityDir)
	}
	for _, root := range roots {
		canonical, err := registry.CanonicalPath(root)
		if err != nil {
			return err
		}
		if target == registry.OwnershipLockPath(canonical) {
			return failure("invalid_request", "host target overlaps Garden ownership lock")
		}
		relative, err := filepath.Rel(canonical, target)
		if err == nil && (relative == "." || (!strings.HasPrefix(relative, ".."+string(filepath.Separator)) && relative != "..")) {
			return failure("invalid_request", "host target overlaps Garden-owned data")
		}
	}
	return nil
}
