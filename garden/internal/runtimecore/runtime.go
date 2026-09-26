// Package runtimecore composes Garden's in-process bootstrap, recall, and capture services.
// It creates no listener and holds no Persona or memory authority of its own.
package runtimecore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dashimaki/garden/internal/activity"
	"github.com/dashimaki/garden/internal/ingest"
	"github.com/dashimaki/garden/internal/personactx"
	"github.com/dashimaki/garden/internal/recall"
	"github.com/dashimaki/laputa/actmem"
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

	mu     sync.Mutex
	closed bool
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
// starting an HTTP listener. A missing optional local model yields frozen-only
// recall and spooled ingestion; all other initialization failures propagate.
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
	g := &Garden{ProfileID: cfg.ProfileID}
	defer func() {
		if err != nil {
			_ = g.Close()
		}
	}()
	if g.Persona, err = persona.Open(cfg.PersonaDir); err != nil {
		return nil, fmt.Errorf("persona: %w", err)
	}
	g.Actmem = actmem.New(cfg.PersonaDir)
	if available {
		m := &facade.Service{}
		if err = m.Init(ctx, facade.Options{PalacePath: cfg.PalacePath, ModelsDir: cfg.ModelsDir, RequireLocalModel: true}); err != nil {
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
	g.FastRecall = &recall.FastService{Frozen: &personactx.SessionProvider{Store: g.Frozen, Reader: g.Persona}}
	var writer ingest.MemoryWriter
	if g.Mentle != nil {
		g.FastRecall.Searcher = g.Mentle
		writer = g.Mentle
	}
	if g.Ingest, err = ingest.Open(cfg.StateDB, writer); err != nil {
		return nil, fmt.Errorf("ingest: %w", err)
	}
	g.Ingest.Activity = g.Activity
	g.Ingest.Spool = g.TransientSpool
	if g.Trace, err = recall.OpenTraceStore(cfg.StateDB); err != nil {
		return nil, fmt.Errorf("trace: %w", err)
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
