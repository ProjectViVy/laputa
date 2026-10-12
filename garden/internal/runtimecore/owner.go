package runtimecore

import (
	"context"
	"errors"
	"path/filepath"
	"sync"

	mentlebackend "github.com/ProjectViVy/laputa/garden/backends/mentle"
	"github.com/ProjectViVy/laputa/garden/internal/personactx"
	"github.com/ProjectViVy/laputa/garden/internal/recall"
	"github.com/ProjectViVy/laputa/garden/internal/registry"
	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/actmem"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/laputa/persona"
	"github.com/google/uuid"
)

type OwnerConfig struct {
	Config
	Home string
}

// Owner opens installation resources once. mu is also the bounded lifecycle
// gate: admitted API calls hold a read lock until they finish; Close drains them.
type Owner struct {
	mu           sync.RWMutex
	base         *Garden
	Registry     *registry.Store
	home         string
	closed       bool
	locks        *ownership
	subjectLocks map[string]*ownership
	subjects     map[string]*Garden
}

func OpenOwner(ctx context.Context, cfg OwnerConfig) (_ *Owner, err error) {
	if !filepath.IsAbs(cfg.Home) {
		return nil, errors.New("Home must be an explicit absolute path")
	}
	cfg.Config.ProfileID = "installation"
	cfg.Config.PersonaDir = cfg.Home
	if err = validate(cfg.Config); err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	locks, err := acquireOwnership(cfg.Home, cfg.StateDB, cfg.PalacePath)
	if err != nil {
		return nil, err
	}
	o := &Owner{home: cfg.Home, locks: locks, subjectLocks: map[string]*ownership{}, subjects: map[string]*Garden{}}
	defer func() {
		if err != nil {
			_ = o.Close()
		}
	}()
	if o.Registry, err = registry.Open(cfg.StateDB, cfg.Home); err != nil {
		return nil, err
	}
	if o.base, err = open(ctx, cfg.Config, false); err != nil {
		return nil, err
	}
	return o, nil
}

func (o *Owner) Enter() (func(), error) {
	o.mu.RLock()
	if o.closed {
		o.mu.RUnlock()
		return nil, errors.New("installation closed")
	}
	return o.mu.RUnlock, nil
}

func (o *Owner) RegisterProfile(ctx context.Context, req registry.ProfileRegistration) (registry.Profile, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return registry.Profile{}, errors.New("installation closed")
	}
	if req.ProfileID == "" && req.Create {
		req.ProfileID = uuid.NewString()
	}
	if err := registry.ValidateProfileID(req.ProfileID); err != nil {
		return registry.Profile{}, err
	}
	if req.AuthorityDir == "" && req.Create {
		req.AuthorityDir = filepath.Join(o.home, "subjects", req.ProfileID)
	}
	path, err := registry.CanonicalPath(req.AuthorityDir)
	if err != nil {
		return registry.Profile{}, err
	}
	lock := o.subjectLocks[path]
	newLock := lock == nil
	if newLock {
		lock, err = acquireOwnership(path)
		if err != nil {
			return registry.Profile{}, err
		}
	}
	p, err := o.Registry.RegisterProfile(ctx, req)
	if err != nil {
		if newLock {
			_ = lock.Close()
		}
		return registry.Profile{}, err
	}
	o.subjectLocks[path] = lock
	return p, nil
}

func (o *Owner) Subject(ctx context.Context, profileID string) (*Garden, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return nil, errors.New("installation closed")
	}
	if g := o.subjects[profileID]; g != nil {
		return g, nil
	}
	p, err := o.Registry.Profile(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if o.subjectLocks[p.AuthorityDir] == nil {
		lock, err := acquireOwnership(p.AuthorityDir)
		if err != nil {
			return nil, err
		}
		o.subjectLocks[p.AuthorityDir] = lock
	}
	svc, err := persona.Open(p.AuthorityDir)
	if err != nil {
		return nil, err
	}
	b := o.base
	g := &Garden{ProfileID: profileID, Persona: svc, Actmem: actmem.New(p.AuthorityDir), Frozen: b.Frozen, Ingest: b.Ingest, Activity: b.Activity, TransientSpool: b.TransientSpool, Trace: b.Trace, Mentle: b.Mentle, LexicalOnly: b.LexicalOnly, owner: o}
	g.FastRecall = &recall.FastService{Frozen: &personactx.SessionProvider{Store: g.Frozen, Reader: svc}, LexicalOnly: g.LexicalOnly}
	o.subjects[profileID] = g
	return g, nil
}

// BackendFor is used only after trusted Garden resolution. Resource subjects
// such as shared:<id> are storage keys, never registry Persona profiles.
func (o *Owner) BackendFor(ctx context.Context, scope evolution.Scope, destinationID string) (memory.Backend, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	b := o.base
	b.backendMu.Lock()
	defer b.backendMu.Unlock()
	key := memory.EncodeScope(scope)
	if backend := b.Backends[key]; backend != nil {
		return backend, nil
	}
	if b.Mentle == nil {
		return nil, &evolution.ContractError{Code: evolution.ErrBackendUnavailable, Message: "selected memory backend unavailable"}
	}
	admitted := []evolution.Scope{{SubjectID: scope.SubjectID, Kind: evolution.ScopePersonal}}
	if scope.Kind == evolution.ScopeWorkspace {
		admitted = append(admitted, scope)
	}
	backend, err := mentlebackend.New(b.Mentle, scope, destinationID, admitted)
	if err != nil {
		return nil, err
	}
	b.Backends[key] = backend
	return backend, nil
}

func (o *Owner) Close() error {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return nil
	}
	o.closed = true
	var errs []error
	if o.base != nil {
		errs = append(errs, o.base.Close())
	}
	if o.Registry != nil {
		errs = append(errs, o.Registry.Close())
	}
	for _, lock := range o.subjectLocks {
		errs = append(errs, lock.Close())
	}
	if o.locks != nil {
		errs = append(errs, o.locks.Close())
	}
	return errors.Join(errs...)
}

func (g *Garden) SessionKey(bindingID, sessionID string) (string, error) {
	if g.owner == nil {
		return sessionID, nil
	}
	return registry.SessionKey(g.ProfileID, bindingID, sessionID)
}

func (g *Garden) Enter() (func(), error) {
	if g.owner == nil {
		return func() {}, nil
	}
	return g.owner.Enter()
}

func (g *Garden) InstallationOwned() bool { return g.owner != nil }
