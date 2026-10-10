// Package runtimecore composes Garden's in-process bootstrap, recall, and capture services.
// It creates no listener and holds no Persona or memory authority of its own.
package runtimecore

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	mentlebackends "github.com/dashimaki/garden/backends/mentle"
	"github.com/dashimaki/garden/internal/activity"
	"github.com/dashimaki/garden/internal/ingest"
	"github.com/dashimaki/garden/internal/personactx"
	"github.com/dashimaki/garden/internal/recall"
	"github.com/dashimaki/garden/memory"
	"github.com/dashimaki/laputa/actmem"
	"github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/laputa/persona"
	"github.com/dashimaki/mentle/facade"
)

// Config names every filesystem dependency. Empty, relative and home-expansion
// paths are rejected rather than inherited from the host or process cwd.
type Config struct {
	PersonaDir        string
	PalacePath        string
	ModelsDir         string
	StateDB           string
	ProfileID         string
	RequireLocalModel bool
	// Backends injects scope-bound memory backends keyed by their
	// scope/v1 encoding (contracts.md section 6). When present for a scope,
	// the injected backend wins over a Mentle adapter minted from the same
	// runtime — a revoked destination never falls back to another backend.
	Backends map[string]memory.Backend
}

// Garden is an in-process composition of the existing domain services.
// ProfileID is a binding label, not a selector for another authority.
type Garden struct {
	ProfileID      string
	Persona        *persona.Service
	Actmem         *actmem.Store
	Frozen         *personactx.Store
	FastRecall     *recall.FastService
	Ingest         *ingest.Service
	Activity       *activity.Store
	TransientSpool *activity.TransientSpool
	Trace          *recall.TraceStore
	Mentle         *facade.Service
	// LexicalOnly identifies read-only Mentle retrieval without a local model.
	LexicalOnly bool
	// Backends holds injected scope-bound backends plus lazily minted
	// Mentle adapters, keyed by scope/v1 encoding.
	Backends map[string]memory.Backend

	// Keep cache admission separate from Close's lifecycle lock: ingestion
	// drains during Close and may still need its scope-bound adapter.
	backendMu sync.Mutex
	mu        sync.Mutex
	closed    bool
}

func validate(cfg Config) error {
	for _, item := range []struct{ name, path string }{
		{"PersonaDir", cfg.PersonaDir}, {"PalacePath", cfg.PalacePath}, {"ModelsDir", cfg.ModelsDir}, {"StateDB", cfg.StateDB},
	} {
		if !filepath.IsAbs(item.path) {
			return fmt.Errorf("runtimecore: %s must be an explicit absolute path", item.name)
		}
	}
	if strings.TrimSpace(cfg.ProfileID) == "" {
		return errors.New("runtimecore: ProfileID is required")
	}
	return nil
}

// modelAvailable inspects only the configured directory; absence is the sole
// reason an optional Mentle service may be omitted. Other I/O errors fail closed.
func modelAvailable(dir string) (bool, error) {
	for _, path := range []string{filepath.Join(dir, "model.onnx"), filepath.Join(dir, "onnx", "model.onnx")} {
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return true, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return false, fmt.Errorf("inspect local model: %w", err)
		}
	}
	return false, nil
}

// Open composes existing services without reading environment configuration or
// starting an HTTP listener. A missing optional local model uses a read-only
// lexical projection when an existing canonical catalog is present; otherwise
// recall is frozen-only and ingestion spools. Other initialization failures propagate.
func Open(ctx context.Context, cfg Config) (_ *Garden, err error) {
	if err = validate(cfg); err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	available, err := modelAvailable(cfg.ModelsDir)
	if err != nil {
		return nil, err
	}
	if !available && cfg.RequireLocalModel {
		return nil, fmt.Errorf("runtimecore: required local model missing under %q", cfg.ModelsDir)
	}
	lexicalOnly := false
	if !available {
		canonical := filepath.Join(cfg.PalacePath, "canonical.sqlite3")
		info, statErr := os.Stat(canonical)
		switch {
		case statErr == nil:
			if info.IsDir() {
				return nil, fmt.Errorf("runtimecore: canonical catalog is a directory: %q", canonical)
			}
			lexicalOnly = true
		case os.IsNotExist(statErr):
			// Without canonical authority, do not create one just for recall.
		default:
			return nil, fmt.Errorf("runtimecore: inspect canonical catalog: %w", statErr)
		}
	}
	g := &Garden{ProfileID: cfg.ProfileID, LexicalOnly: lexicalOnly}
	defer func() {
		if err != nil {
			_ = g.Close()
		}
	}()
	if g.Persona, err = persona.Open(cfg.PersonaDir); err != nil {
		return nil, fmt.Errorf("persona: %w", err)
	}
	g.Actmem = actmem.New(cfg.PersonaDir)
	if available || lexicalOnly {
		m := &facade.Service{}
		if err = m.Init(ctx, facade.Options{PalacePath: cfg.PalacePath, ModelsDir: cfg.ModelsDir, RequireLocalModel: !lexicalOnly, LexicalOnly: lexicalOnly}); err != nil {
			_ = m.Close()
			return nil, fmt.Errorf("mentle: %w", err)
		}
		g.Mentle = m
	}
	if err = os.MkdirAll(filepath.Dir(cfg.StateDB), 0700); err != nil {
		return nil, fmt.Errorf("state directory: %w", err)
	}
	if g.Frozen, err = personactx.OpenStore(cfg.StateDB); err != nil {
		return nil, fmt.Errorf("frozen: %w", err)
	}
	if g.Activity, err = activity.OpenStore(cfg.StateDB); err != nil {
		return nil, fmt.Errorf("activity: %w", err)
	}
	if g.TransientSpool, err = activity.OpenSpool(cfg.StateDB); err != nil {
		return nil, fmt.Errorf("transient spool: %w", err)
	}
	g.FastRecall = &recall.FastService{Frozen: &personactx.SessionProvider{Store: g.Frozen, Reader: g.Persona}, LexicalOnly: lexicalOnly}
	g.Backends = map[string]memory.Backend{}
	for key, backend := range cfg.Backends {
		g.Backends[key] = backend
	}
	var writer ingest.MemoryWriter
	if g.Mentle != nil {
		g.FastRecall.Searcher = g.Mentle
	}
	if !lexicalOnly {
		writer = &scopedMemoryWriter{g: g}
	}
	if g.Ingest, err = ingest.OpenPaused(cfg.StateDB, writer); err != nil {
		return nil, fmt.Errorf("ingest: %w", err)
	}
	g.Ingest.Activity = g.Activity
	g.Ingest.Actmem = g.Actmem
	g.Ingest.Spool = g.TransientSpool
	g.Ingest.ProfileID = cfg.ProfileID
	if !lexicalOnly && g.Mentle != nil {
		if drained, drainErr := g.Ingest.DrainSpool(ctx); drainErr != nil {
			return nil, fmt.Errorf("spool drain: %w", drainErr)
		} else if drained > 0 {
			log.Printf("spool drain: recovered %d entries", drained)
		}
	}
	if g.Trace, err = recall.OpenTraceStore(cfg.StateDB); err != nil {
		return nil, fmt.Errorf("trace: %w", err)
	}
	if err = g.Ingest.Start(); err != nil {
		return nil, fmt.Errorf("ingest worker: %w", err)
	}
	return g, nil
}

// Close stops workers and closes owned stores in reverse initialization order.
// Repeated calls are harmless, including after a partial Open failure.
func (g *Garden) Close() error {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return nil
	}
	g.closed = true
	var errs []error
	if g.Trace != nil {
		errs = append(errs, g.Trace.Close())
		g.Trace = nil
	}
	if g.Ingest != nil {
		errs = append(errs, g.Ingest.Close())
		g.Ingest = nil
	}
	g.FastRecall = nil
	if g.TransientSpool != nil {
		errs = append(errs, g.TransientSpool.Close())
		g.TransientSpool = nil
	}
	if g.Activity != nil {
		errs = append(errs, g.Activity.Close())
		g.Activity = nil
	}
	if g.Frozen != nil {
		errs = append(errs, g.Frozen.Close())
		g.Frozen = nil
	}
	if g.Mentle != nil {
		errs = append(errs, g.Mentle.Close())
		g.Mentle = nil
	}
	g.Actmem = nil
	g.Persona = nil
	return errors.Join(errs...)
}

// BackendFor returns the backend bound to scope. Injected backends win; a
// Mentle adapter is minted lazily over the shared facade for any scope of
// this profile's subject. A foreign subject is denied, an absent backend
// reports unavailable — memory is never silently re-routed.
func (g *Garden) BackendFor(scope evolution.Scope) (memory.Backend, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	key := memory.EncodeScope(scope)
	g.backendMu.Lock()
	defer g.backendMu.Unlock()
	if g.Backends == nil {
		g.Backends = map[string]memory.Backend{}
	}
	if backend, ok := g.Backends[key]; ok {
		return backend, nil
	}
	if scope.SubjectID != g.ProfileID {
		return nil, &evolution.ContractError{Code: evolution.ErrAuthorityDenied, Message: "scope subject is not this profile"}
	}
	if g.Mentle == nil {
		return nil, &evolution.ContractError{Code: evolution.ErrBackendUnavailable, Message: "no memory backend for scope"}
	}
	backend, err := mentlebackends.New(g.Mentle, scope, "mentle", g.admittedScopes(scope))
	if err != nil {
		return nil, err
	}
	g.Backends[key] = backend
	return backend, nil
}

// admittedScopes is the read union for one bound scope: the subject's
// personal scope plus the bound scope itself.
func (g *Garden) admittedScopes(scope evolution.Scope) []evolution.Scope {
	personal := evolution.Scope{SubjectID: scope.SubjectID, Kind: evolution.ScopePersonal}
	if scope.Equal(personal) {
		return []evolution.Scope{personal}
	}
	return []evolution.Scope{personal, scope}
}

// scopedMemoryWriter routes ingest writes through the backend bound to
// the scope encoded on the request; writes whose scope has no admitted
// backend fail rather than falling back.
type scopedMemoryWriter struct{ g *Garden }

func (w *scopedMemoryWriter) CreateMemory(ctx context.Context, req facade.CreateMemoryRequest, operationID, payloadDigest string) (facade.Memory, error) {
	scope, err := memory.DecodeScope(req.Scope)
	if err != nil {
		return facade.Memory{}, err
	}
	backend, err := w.g.BackendFor(scope)
	if err != nil {
		return facade.Memory{}, err
	}
	creator, ok := backend.(interface {
		CreateMemory(context.Context, facade.CreateMemoryRequest, string, string) (facade.Memory, error)
	})
	if !ok {
		return facade.Memory{}, &evolution.ContractError{Code: evolution.ErrCapabilityUnavailable, Message: "bound backend cannot accept ingest writes"}
	}
	return creator.CreateMemory(ctx, req, operationID, payloadDigest)
}

// SessionStateStore delegation: session leases and cursors are per-session
// canonical state, so they go to the shared facade rather than a scope
// binding. Returns ErrUnavailable semantics when Mentle is absent.
func (w *scopedMemoryWriter) AcquireSessionLease(ctx context.Context, sessionID, owner string, ttl time.Duration) (bool, error) {
	if w.g.Mentle == nil {
		return false, facade.ErrUnavailable
	}
	return w.g.Mentle.AcquireSessionLease(ctx, sessionID, owner, ttl)
}

func (w *scopedMemoryWriter) ReleaseSessionLease(ctx context.Context, sessionID, owner string) error {
	if w.g.Mentle == nil {
		return facade.ErrUnavailable
	}
	return w.g.Mentle.ReleaseSessionLease(ctx, sessionID, owner)
}

func (w *scopedMemoryWriter) SaveSessionCursor(sessionID, timestamp string) error {
	if w.g.Mentle == nil {
		return facade.ErrUnavailable
	}
	return w.g.Mentle.SaveSessionCursor(sessionID, timestamp)
}
