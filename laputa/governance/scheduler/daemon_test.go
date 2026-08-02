package scheduler

import (
	"bytes"
	"context"
	"log"
	"testing"
	"time"

	laputa "github.com/dashimaki/laputa/governance"
	"github.com/dashimaki/laputa/governance/rhythm"
	"github.com/dashimaki/laputa/governance/wakeup"
)

func newRhythmEngine(e *laputa.Engine) *rhythm.Engine {
	return rhythm.NewEngine(e, rhythm.NewMockGenerator(), rhythm.Config{})
}

func newWakeupEngine(e *laputa.Engine) *wakeup.Engine {
	return wakeup.NewEngine(e)
}

func newTestEngine(t *testing.T) *laputa.Engine {
	t.Helper()
	store, err := laputa.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	engine := laputa.NewEngine(store)
	if err := engine.Initialize(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}
	return engine
}

func TestDaemon_RunsAllKindsOnFirstTick(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	buf := &bytes.Buffer{}
	engine := newTestEngine(t)
	d := New(engine, Config{
		TickEvery:     10 * time.Millisecond,
		DryRun:        false,
		Logger:        log.New(buf, "", 0),
		WorkspaceRoot: "/tmp/test",
	})

	// Pre-populate lastRun so only 'daily' is fresh on first tick.
	now := time.Now().UTC()
	d.SetLastRun(KindWeekly, now.Add(-time.Hour))
	d.SetLastRun(KindMonthly, now.Add(-time.Hour))

	// Manually drive one tick.
	re := newRhythmEngine(engine)
	provider := newWakeupEngine(engine)
	d.tick(ctx, re, provider)

	last := d.LastRun()
	if _, ok := last[KindDaily]; !ok {
		t.Errorf("expected daily lastRun to be set")
	}
	if !last[KindWeekly].Equal(now.Add(-time.Hour)) {
		t.Errorf("expected weekly lastRun untouched (hysteresis)")
	}
	if !last[KindMonthly].Equal(now.Add(-time.Hour)) {
		t.Errorf("expected monthly lastRun untouched (hysteresis)")
	}

	// Verify daily section was written
	daily, _ := engine.GetSection(ctx, laputa.SectionDaily)
	reports, _ := daily["reports"].([]any)
	if len(reports) == 0 {
		t.Errorf("expected daily section to contain a report")
	}
}

func TestDaemon_Hysteresis_BlocksEarlyRetrigger(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	engine := newTestEngine(t)
	d := New(engine, Config{
		TickEvery: 10 * time.Millisecond,
		Logger:    log.New(&bytes.Buffer{}, "", 0),
	})

	now := time.Now().UTC()
	d.SetLastRun(KindDaily, now) // just ran

	re := newRhythmEngine(engine)
	provider := newWakeupEngine(engine)
	d.tick(ctx, re, provider)

	// Should not produce a new report
	daily, _ := engine.GetSection(ctx, laputa.SectionDaily)
	reports, _ := daily["reports"].([]any)
	if len(reports) != 0 {
		t.Errorf("expected no new daily report due to hysteresis")
	}
}

func TestDaemon_DryRunSkipsGenerator(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	engine := newTestEngine(t)
	d := New(engine, Config{
		TickEvery: 10 * time.Millisecond,
		DryRun:    true,
		Logger:    log.New(&bytes.Buffer{}, "", 0),
	})

	re := newRhythmEngine(engine)
	provider := newWakeupEngine(engine)
	d.tick(ctx, re, provider)

	daily, _ := engine.GetSection(ctx, laputa.SectionDaily)
	reports, _ := daily["reports"].([]any)
	if len(reports) != 0 {
		t.Errorf("expected dry-run to skip generator")
	}
}

func TestDaemon_ShutdownTriggersSessionEnd(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	engine := newTestEngine(t)
	d := New(engine, Config{
		TickEvery:     10 * time.Millisecond,
		DryRun:        true,
		Logger:        log.New(&bytes.Buffer{}, "", 0),
		WorkspaceRoot: "/tmp/test",
		SessionID:     "session-daemon",
	})

	done := make(chan error, 1)
	go func() { done <- d.Run(ctx) }()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// expected
	case <-time.After(2 * time.Second):
		t.Fatalf("daemon did not shut down")
	}

	history, _ := engine.GetSection(ctx, laputa.SectionHistoryMD)
	timeline, _ := history["timeline"].([]any)
	if len(timeline) == 0 {
		t.Fatalf("expected history timeline entry from session-end")
	}
	last, _ := timeline[len(timeline)-1].(map[string]any)
	if last["event"] != "session_end" {
		t.Errorf("expected session_end event, got %v", last["event"])
	}
	if last["session_id"] != "session-daemon" {
		t.Errorf("expected session_id=session-daemon, got %v", last["session_id"])
	}
}
