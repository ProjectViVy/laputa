package wakeup

import (
	"context"
	"strings"
	"testing"

	laputa "github.com/dashimaki/laputa/governance"
)

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

func TestSystemPromptBlock_RendersMarkdown(t *testing.T) {
	ctx := context.Background()
	engine := newTestEngine(t)
	_ = engine.SetSection(ctx, laputa.SectionDaily, map[string]any{
		"reports": []map[string]any{
			{"title": "Daily", "summary": "hello world"},
		},
	})

	provider := NewEngine(engine)
	resp, err := provider.SystemPromptBlock(ctx, "/tmp/diva")
	if err != nil {
		t.Fatalf("system prompt block: %v", err)
	}
	if resp.Status != StartupReady {
		t.Fatalf("expected ready, got %v reason=%v", resp.Status, resp.Reason)
	}
	if resp.PromptBlock == nil {
		t.Fatalf("expected prompt block")
	}
	md := resp.PromptBlock.Markdown
	if !strings.Contains(md, "## Wakeup Summary") {
		t.Errorf("expected wakeup summary in markdown: %q", md)
	}
	if !strings.Contains(md, "## Rhythm Signals") {
		t.Errorf("expected rhythm signals in markdown: %q", md)
	}
	if !strings.Contains(md, "07-daily") {
		t.Errorf("expected daily section reference: %q", md)
	}
	if !strings.Contains(md, "hello world") {
		t.Errorf("expected daily report summary in markdown: %q", md)
	}
}

func TestPrefetch_SkipWithoutIntent(t *testing.T) {
	provider := NewEngine(newTestEngine(t))
	resp, err := provider.Prefetch(context.Background(), "", nil, nil)
	if err != nil {
		t.Fatalf("prefetch: %v", err)
	}
	if resp.Status != PrefetchSkippedNoIntent {
		t.Fatalf("expected skipped, got %v", resp.Status)
	}
	if resp.PromptBlock != nil {
		t.Fatalf("expected no prompt block, got %v", *resp.PromptBlock)
	}
}

func TestPrefetch_ReadyWithIntent(t *testing.T) {
	ctx := context.Background()
	engine := newTestEngine(t)
	_ = engine.SetSection(ctx, laputa.SectionDaily, map[string]any{
		"reports": []map[string]any{
			{"title": "Daily", "summary": "milestone shipped"},
		},
	})

	provider := NewEngine(engine)
	room := "roadmap"
	resp, err := provider.Prefetch(ctx, "recall-project-status", &room, nil)
	if err != nil {
		t.Fatalf("prefetch: %v", err)
	}
	if resp.Status != PrefetchReady {
		t.Fatalf("expected ready, got %v", resp.Status)
	}
	if resp.PromptBlock == nil || !strings.Contains(*resp.PromptBlock, "milestone shipped") {
		t.Fatalf("expected daily summary in block, got %v", resp.PromptBlock)
	}
}

func TestSyncTurn_PersistsHistoryEntry(t *testing.T) {
	ctx := context.Background()
	engine := newTestEngine(t)
	provider := NewEngine(engine)

	entry := "user asked about laputa"
	resp, err := provider.SyncTurn(ctx, nil, &entry)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if resp.Status != SyncPersisted {
		t.Fatalf("expected persisted, got %v", resp.Status)
	}

	history, err := engine.GetSection(ctx, laputa.SectionHistoryMD)
	if err != nil {
		t.Fatalf("read history: %v", err)
	}
	timeline, _ := history["timeline"].([]any)
	if len(timeline) == 0 {
		t.Fatalf("expected history entries")
	}
	last, _ := timeline[len(timeline)-1].(map[string]any)
	if last["history_entry"] != entry {
		t.Errorf("expected history_entry=%v, got %v", entry, last["history_entry"])
	}
}

func TestSyncTurn_NoopWhenEmpty(t *testing.T) {
	provider := NewEngine(newTestEngine(t))
	resp, err := provider.SyncTurn(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if resp.Status != SyncNoop {
		t.Fatalf("expected noop, got %v", resp.Status)
	}
}

func TestOnSessionEnd_RecordsMarker(t *testing.T) {
	ctx := context.Background()
	engine := newTestEngine(t)
	provider := NewEngine(engine)

	sid := "session-42"
	resp, err := provider.OnSessionEnd(ctx, &sid)
	if err != nil {
		t.Fatalf("session end: %v", err)
	}
	if resp.Status != SessionTriggered {
		t.Fatalf("expected triggered, got %v", resp.Status)
	}

	history, _ := engine.GetSection(ctx, laputa.SectionHistoryMD)
	timeline, _ := history["timeline"].([]any)
	last, _ := timeline[len(timeline)-1].(map[string]any)
	if last["event"] != "session_end" {
		t.Errorf("expected event=session_end, got %v", last["event"])
	}
	if last["session_id"] != sid {
		t.Errorf("expected session_id=%v, got %v", sid, last["session_id"])
	}
}
