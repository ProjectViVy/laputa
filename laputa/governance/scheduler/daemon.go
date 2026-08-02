// Package scheduler implements Laputa's own cronjob mechanism.
//
// Laputa persists. Laputa does not rely on external schedulers (no
// system cron, no Hermes cron, no agent-diva cron). Trigger cadence is
// owned by the Laputa process itself via a time.Ticker-based daemon.
//
// Three trigger kinds are first-class:
//   - daily   : every 24h
//   - weekly  : every 7 days
//   - monthly : every 30 days
package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	laputa "github.com/dashimaki/laputa/governance"
	"github.com/dashimaki/laputa/governance/rhythm"
	"github.com/dashimaki/laputa/governance/wakeup"
)

// Kind is the trigger cadence.
type Kind string

const (
	KindDaily   Kind = "daily"
	KindWeekly  Kind = "weekly"
	KindMonthly Kind = "monthly"
)

// Config configures the daemon.
type Config struct {
	// TickEvery is the cadence at which the daemon scans for due triggers.
	// Defaults to 1 minute. Production cadence is the same; tests use shorter.
	TickEvery time.Duration
	// DryRun disables generator calls; daemon ticks still fire and log.
	DryRun bool
	// NowFn allows tests to advance time.
	NowFn func() time.Time
	// Generator is the rhythm generator to use. Defaults to MockGenerator.
	Generator rhythm.Generator
	// WorkspaceRoot is passed to wakeup.SystemPromptBlock.
	WorkspaceRoot string
	// SessionID is passed to wakeup.OnSessionEnd.
	SessionID string
	// Logger receives progress and error lines.
	Logger *log.Logger
}

// Daemon owns the time.Ticker loop.
type Daemon struct {
	engine *laputa.Engine
	cfg    Config

	mu      sync.Mutex
	lastRun map[Kind]time.Time
}

// New creates a scheduler daemon around a Laputa engine.
func New(engine *laputa.Engine, cfg Config) *Daemon {
	if cfg.TickEvery == 0 {
		cfg.TickEvery = time.Minute
	}
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	if cfg.NowFn == nil {
		cfg.NowFn = time.Now
	}
	if cfg.Generator == nil {
		cfg.Generator = rhythm.NewMockGenerator()
	}
	if cfg.WorkspaceRoot == "" {
		cfg.WorkspaceRoot = "."
	}
	return &Daemon{
		engine:  engine,
		cfg:     cfg,
		lastRun: map[Kind]time.Time{},
	}
}

// Run blocks until ctx is cancelled. On each tick it invokes due triggers.
func (d *Daemon) Run(ctx context.Context) error {
	re := rhythm.NewEngine(d.engine, d.cfg.Generator, rhythm.Config{})
	provider := wakeup.NewEngine(d.engine)

	ticker := time.NewTicker(d.cfg.TickEvery)
	defer ticker.Stop()

	d.cfg.Logger.Printf("laputa daemon: tick=%s dry-run=%v", d.cfg.TickEvery, d.cfg.DryRun)
	d.tick(ctx, re, provider)

	for {
		select {
		case <-ctx.Done():
			d.cfg.Logger.Printf("laputa daemon: shutdown (session=%s)", d.cfg.SessionID)
			if _, err := provider.OnSessionEnd(ctx, sessionPtr(d.cfg.SessionID)); err != nil {
				d.cfg.Logger.Printf("session_end error: %v", err)
			}
			return ctx.Err()
		case <-ticker.C:
			d.tick(ctx, re, provider)
		}
	}
}

func (d *Daemon) tick(ctx context.Context, re *rhythm.Engine, provider *wakeup.Engine) {
	now := d.cfg.NowFn().UTC()

	for _, kind := range []Kind{KindDaily, KindWeekly, KindMonthly} {
		if !d.isDue(kind, now) {
			continue
		}
		if d.cfg.DryRun {
			d.cfg.Logger.Printf("laputa daemon: tick %s (dry-run, skipped)", kind)
			d.markRun(kind, now)
			continue
		}
		d.cfg.Logger.Printf("laputa daemon: tick %s", kind)
		if err := re.Run(ctx, rhythm.RhythmKind(kind)); err != nil {
			d.cfg.Logger.Printf("rhythm %s error: %v", kind, err)
			continue
		}
		d.markRun(kind, now)

		// After rhythm completes, render wakeup summary and trigger session-end.
		resp, err := provider.SystemPromptBlock(ctx, d.cfg.WorkspaceRoot)
		if err != nil {
			d.cfg.Logger.Printf("wakeup error: %v", err)
		} else if resp != nil && resp.PromptBlock != nil {
			d.cfg.Logger.Printf("wakeup: status=%s bytes=%d", resp.Status, len(resp.PromptBlock.Markdown))
		}
	}
}

func (d *Daemon) isDue(kind Kind, now time.Time) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	last, ok := d.lastRun[kind]
	if !ok {
		return true
	}
	var period time.Duration
	switch kind {
	case KindDaily:
		period = 24 * time.Hour
	case KindWeekly:
		period = 7 * 24 * time.Hour
	case KindMonthly:
		period = 30 * 24 * time.Hour
	default:
		period = 24 * time.Hour
	}
	return now.Sub(last) >= period
}

func (d *Daemon) markRun(kind Kind, t time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastRun[kind] = t
}

func sessionPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// LastRun returns the last run timestamp for each kind (read-only).
func (d *Daemon) LastRun() map[Kind]time.Time {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make(map[Kind]time.Time, len(d.lastRun))
	for k, v := range d.lastRun {
		out[k] = v
	}
	return out
}

// SetLastRun overrides the last-run timestamps (used by tests).
func (d *Daemon) SetLastRun(kind Kind, t time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastRun[kind] = t
}
