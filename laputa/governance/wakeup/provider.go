// Package wakeup provides the AutoDream / recall boundary for Laputa.
//
// It mirrors the agent-diva-core MemoryProvider contract in Go:
//   - SystemPromptBlock (startup wakeup)
//   - Prefetch          (intent-aware mid-turn recall)
//   - SyncTurn          (post-turn durable write)
//   - OnSessionEnd      (shutdown / session-end rhythm)
//
// The implementation is intentionally backend-agnostic: it consumes the
// Laputa Engine snapshot and returns prompt-ready markdown blocks.
package wakeup

import (
	"context"
	"fmt"
	"strings"
	"time"

	laputa "github.com/dashimaki/laputa/governance"
)

// InjectionShape describes how the consumer wants the wakeup block formatted.
type InjectionShape string

const (
	// CompactRenderedMarkdown is a prompt-ready markdown block.
	CompactRenderedMarkdown InjectionShape = "compact_rendered_markdown"
)

// SystemPromptBlock is the startup block spliced into the agent system prompt.
type SystemPromptBlock struct {
	Shape    InjectionShape
	Markdown string
}

// StartupStatus indicates whether a fresh wakeup was produced.
type StartupStatus string

const (
	StartupReady    StartupStatus = "ready"
	StartupDegraded StartupStatus = "degraded"
)

// SystemPromptResponse is the result of startup wakeup generation.
type SystemPromptResponse struct {
	Status      StartupStatus
	Reason      string
	PromptBlock *SystemPromptBlock
}

// PrefetchStatus indicates the outcome of a mid-turn recall.
type PrefetchStatus string

const (
	PrefetchSkippedNoIntent PrefetchStatus = "skipped_no_intent"
	PrefetchReady           PrefetchStatus = "ready"
	PrefetchFailed          PrefetchStatus = "failed"
)

// PrefetchResponse carries optional memory material for a live turn.
type PrefetchResponse struct {
	Status      PrefetchStatus
	PromptBlock *string
}

// SyncTurnStatus indicates the result of post-turn synchronization.
type SyncTurnStatus string

const (
	SyncPersisted SyncTurnStatus = "persisted"
	SyncNoop      SyncTurnStatus = "noop"
	SyncFailed    SyncTurnStatus = "failed"
)

// SyncTurnResponse is the durable-write result.
type SyncTurnResponse struct {
	Status SyncTurnStatus
}

// SessionEndStatus indicates the result of the shutdown hook.
type SessionEndStatus string

const (
	SessionTriggered      SessionEndStatus = "triggered"
	SessionNoop           SessionEndStatus = "noop"
	SessionAlreadyHandled SessionEndStatus = "already_handled"
	SessionFailed         SessionEndStatus = "failed"
)

// SessionEndResponse is the shutdown result.
type SessionEndResponse struct {
	Status SessionEndStatus
}

// WakeupPackSummary is a structured wakeup summary.
type WakeupPackSummary struct {
	Identity          string
	RecentState       string
	LatestCapsule     *string
	KeyRelations      []string
	UnresolvedThreads []string
	GeneratedAt       *string
}

// RhythmTrigger is a named rhythm signal relevant at startup.
type RhythmTrigger struct {
	Name   string
	Reason *string
}

// StartupContextSnapshot holds the raw material for startup wakeup rendering.
type StartupContextSnapshot struct {
	StateRoot      *string
	SoulMarkdown   *string
	WakeupMarkdown *string
	WakeupPack     *WakeupPackSummary
	RhythmTriggers []RhythmTrigger
	MemoryMarkdown *string
}

// Provider is the long-memory boundary consumed by agent runtimes.
type Provider interface {
	SystemPromptBlock(ctx context.Context, workspaceRoot string) (*SystemPromptResponse, error)
	Prefetch(ctx context.Context, intent string, currentRoom *string, userMessage *string) (*PrefetchResponse, error)
	SyncTurn(ctx context.Context, memoryUpdateMarkdown *string, historyEntry *string) (*SyncTurnResponse, error)
	OnSessionEnd(ctx context.Context, sessionID *string) (*SessionEndResponse, error)
}

// Engine implements Provider by reading from a Laputa Engine.
type Engine struct {
	laputa *laputa.Engine
}

// NewEngine creates a wakeup provider backed by a Laputa engine.
func NewEngine(laputaEngine *laputa.Engine) *Engine {
	return &Engine{laputa: laputaEngine}
}

// SystemPromptBlock builds the startup memory block from the current Laputa snapshot.
func (e *Engine) SystemPromptBlock(ctx context.Context, workspaceRoot string) (*SystemPromptResponse, error) {
	snapshot, err := e.laputa.Snapshot(ctx)
	if err != nil {
		return degradedResponse(fmt.Sprintf("snapshot failed: %v", err)), nil
	}
	if snapshot == nil {
		return degradedResponse("empty laputa snapshot"), nil
	}

	pack := snapshotToWakeupPack(snapshot)
	snap := &StartupContextSnapshot{
		WakeupPack:     pack,
		RhythmTriggers: detectRhythmTriggers(snapshot),
	}
	root := workspaceRoot + "/.laputa"
	snap.StateRoot = &root

	markdown := renderStartupContextSnapshot(snap)
	if strings.TrimSpace(markdown) == "" {
		return degradedResponse("rendered startup context was empty"), nil
	}

	return &SystemPromptResponse{
		Status: StartupReady,
		PromptBlock: &SystemPromptBlock{
			Shape:    CompactRenderedMarkdown,
			Markdown: markdown,
		},
	}, nil
}

// Prefetch performs intent-aware recall. Returns skipped when no intent is
// provided, or a compact summary of the latest daily report otherwise.
func (e *Engine) Prefetch(ctx context.Context, intent string, currentRoom *string, userMessage *string) (*PrefetchResponse, error) {
	_ = userMessage
	intent = strings.TrimSpace(intent)
	if intent == "" {
		return &PrefetchResponse{Status: PrefetchSkippedNoIntent}, nil
	}

	section, err := e.laputa.GetSection(ctx, laputa.SectionDaily)
	if err != nil {
		return &PrefetchResponse{Status: PrefetchFailed}, fmt.Errorf("read daily section: %w", err)
	}
	reports, _ := section["reports"].([]any)
	latest := "No recent daily report available."
	if len(reports) > 0 {
		last, _ := reports[len(reports)-1].(map[string]any)
		if last != nil && last["summary"] != nil {
			latest = fmt.Sprintf("Latest daily report: %v", last["summary"])
		}
	}

	block := fmt.Sprintf("Intent: %s\nContext: %s\n\n%s", intent, roomOrNone(currentRoom), latest)
	return &PrefetchResponse{Status: PrefetchReady, PromptBlock: &block}, nil
}

// SyncTurn persists a memory update and/or history entry into the history section.
func (e *Engine) SyncTurn(ctx context.Context, memoryUpdateMarkdown *string, historyEntry *string) (*SyncTurnResponse, error) {
	if memoryUpdateMarkdown == nil && historyEntry == nil {
		return &SyncTurnResponse{Status: SyncNoop}, nil
	}

	entry := map[string]any{
		"at": time.Now().UTC().Format(time.RFC3339),
	}
	if memoryUpdateMarkdown != nil {
		entry["memory_update"] = *memoryUpdateMarkdown
	}
	if historyEntry != nil {
		entry["history_entry"] = *historyEntry
	}

	section, err := e.laputa.GetSection(ctx, laputa.SectionHistoryMD)
	if err != nil {
		return &SyncTurnResponse{Status: SyncFailed}, fmt.Errorf("read history section: %w", err)
	}
	timeline, _ := section["timeline"].([]any)
	section["timeline"] = append(timeline, entry)

	if err := e.laputa.SetSection(ctx, laputa.SectionHistoryMD, section); err != nil {
		return &SyncTurnResponse{Status: SyncFailed}, fmt.Errorf("write history section: %w", err)
	}
	return &SyncTurnResponse{Status: SyncPersisted}, nil
}

// OnSessionEnd triggers session-end rhythm work by recording a marker.
func (e *Engine) OnSessionEnd(ctx context.Context, sessionID *string) (*SessionEndResponse, error) {
	marker := map[string]any{
		"at":     time.Now().UTC().Format(time.RFC3339),
		"event":  "session_end",
		"source": "wakeup",
	}
	if sessionID != nil {
		marker["session_id"] = *sessionID
	}

	section, err := e.laputa.GetSection(ctx, laputa.SectionHistoryMD)
	if err != nil {
		return &SessionEndResponse{Status: SessionFailed}, fmt.Errorf("read history section: %w", err)
	}
	timeline, _ := section["timeline"].([]any)
	section["timeline"] = append(timeline, marker)

	if err := e.laputa.SetSection(ctx, laputa.SectionHistoryMD, section); err != nil {
		return &SessionEndResponse{Status: SessionFailed}, fmt.Errorf("write history section: %w", err)
	}
	return &SessionEndResponse{Status: SessionTriggered}, nil
}

// ---- helpers ----

func degradedResponse(reason string) *SystemPromptResponse {
	return &SystemPromptResponse{
		Status: StartupDegraded,
		Reason: reason,
		PromptBlock: &SystemPromptBlock{
			Shape: CompactRenderedMarkdown,
			Markdown: fmt.Sprintf("## Memory Startup Status\n- status: degraded\n- reason: %s\n", reason),
		},
	}
}

func snapshotToWakeupPack(snapshot map[string]any) *WakeupPackSummary {
	sections, _ := snapshot["sections"].(map[string]any)
	if sections == nil {
		return &WakeupPackSummary{}
	}

	identity := ""
	if s, ok := sections[string(laputa.SectionIdentity)]; ok {
		if data, ok := s.(map[string]any)["data"].(map[string]any); ok {
			identity = fmt.Sprintf("role: %v; capabilities: %v", data["role"], data["capabilities"])
		}
	}

	var recent []string
	if s, ok := sections[string(laputa.SectionDaily)]; ok {
		if data, ok := s.(map[string]any)["data"].(map[string]any); ok {
			if reports, ok := data["reports"].([]any); ok && len(reports) > 0 {
				last, _ := reports[len(reports)-1].(map[string]any)
				if last != nil {
					recent = append(recent, fmt.Sprintf("%v: %v", last["title"], last["summary"]))
				}
			}
		}
	}

	return &WakeupPackSummary{
		Identity:          identity,
		RecentState:       strings.Join(recent, "\n"),
		KeyRelations:      []string{},
		UnresolvedThreads: []string{},
	}
}

func detectRhythmTriggers(snapshot map[string]any) []RhythmTrigger {
	var triggers []RhythmTrigger
	sections, _ := snapshot["sections"].(map[string]any)
	if sections == nil {
		return triggers
	}
	for _, name := range []laputa.SectionName{laputa.SectionDaily, laputa.SectionWeekly, laputa.SectionMonthly} {
		if s, ok := sections[string(name)]; ok {
			if data, ok := s.(map[string]any)["data"].(map[string]any); ok {
				if reports, ok := data["reports"].([]any); ok && len(reports) > 0 {
					reason := "latest report available"
					triggers = append(triggers, RhythmTrigger{Name: string(name), Reason: &reason})
				}
			}
		}
	}
	return triggers
}

func renderStartupContextSnapshot(s *StartupContextSnapshot) string {
	var sections []string

	if s.MemoryMarkdown != nil && strings.TrimSpace(*s.MemoryMarkdown) != "" {
		sections = append(sections, strings.TrimSpace(*s.MemoryMarkdown))
	}
	if s.SoulMarkdown != nil && strings.TrimSpace(*s.SoulMarkdown) != "" {
		sections = append(sections, fmt.Sprintf("## Soul Projection\n%s", strings.TrimSpace(*s.SoulMarkdown)))
	}
	if s.WakeupMarkdown != nil && strings.TrimSpace(*s.WakeupMarkdown) != "" {
		sections = append(sections, fmt.Sprintf("## Wakeup Projection\n%s", strings.TrimSpace(*s.WakeupMarkdown)))
	} else if s.WakeupPack != nil {
		sections = append(sections, renderWakeupPack(s.WakeupPack))
	}
	if len(s.RhythmTriggers) > 0 {
		var lines []string
		for _, t := range s.RhythmTriggers {
			if t.Reason != nil && strings.TrimSpace(*t.Reason) != "" {
				lines = append(lines, fmt.Sprintf("- %s — %s", t.Name, strings.TrimSpace(*t.Reason)))
			} else {
				lines = append(lines, fmt.Sprintf("- %s", t.Name))
			}
		}
		sections = append(sections, fmt.Sprintf("## Rhythm Signals\n%s", strings.Join(lines, "\n")))
	}
	return strings.Join(sections, "\n\n")
}

func renderWakeupPack(pack *WakeupPackSummary) string {
	latest := "None"
	if pack.LatestCapsule != nil {
		latest = strings.TrimSpace(*pack.LatestCapsule)
	}
	relations := "- None"
	if len(pack.KeyRelations) > 0 {
		relations = bulletList(pack.KeyRelations)
	}
	threads := "- None"
	if len(pack.UnresolvedThreads) > 0 {
		threads = bulletList(pack.UnresolvedThreads)
	}

	rendered := "## Wakeup Summary"
	if pack.GeneratedAt != nil {
		rendered += fmt.Sprintf("\nGenerated: %s", strings.TrimSpace(*pack.GeneratedAt))
	}
	rendered += fmt.Sprintf("\n\n### Identity\n%s\n\n### Recent State\n%s\n\n### Latest Capsule\n%s\n\n### Key Relations\n%s\n\n### Unresolved Threads\n%s",
		strings.TrimSpace(pack.Identity),
		strings.TrimSpace(pack.RecentState),
		latest,
		relations,
		threads,
	)
	return rendered
}

func bulletList(items []string) string {
	var lines []string
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("- %s", strings.TrimSpace(item)))
	}
	return strings.Join(lines, "\n")
}

func roomOrNone(room *string) string {
	if room == nil {
		return "none"
	}
	return strings.TrimSpace(*room)
}
