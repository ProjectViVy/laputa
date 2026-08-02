package recall

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestTraceStoreSaveAndGet(t *testing.T) {
	store, err := OpenTraceStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	trace := RecallTrace{
		TraceID:       "deep_123",
		Query:         "test query",
		Scope:         "project",
		TriggerReason: "verification",
		SourceSet:     []string{"cards", "kg"},
		Steps:         []TraceStep{{Step: "fast_seed", Status: "ok", DurationMS: 5}},
		StartedAt:     time.Now().UTC().Format(time.RFC3339Nano),
		DurationMS:    42,
	}
	if err := store.Save(context.Background(), trace); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := store.Get(context.Background(), "deep_123")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.TraceID != "deep_123" || got.Query != "test query" || got.TriggerReason != "verification" {
		t.Fatalf("got=%+v", got)
	}
	if len(got.Steps) != 1 || got.Steps[0].Step != "fast_seed" {
		t.Fatalf("steps=%+v", got.Steps)
	}
	if got.DurationMS != 42 {
		t.Fatalf("duration=%d", got.DurationMS)
	}
}

func TestTraceStoreGetNotFound(t *testing.T) {
	store, err := OpenTraceStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, err = store.Get(context.Background(), "nonexistent")
	if err != ErrTraceNotFound {
		t.Fatalf("err=%v, want ErrTraceNotFound", err)
	}
}

func TestTraceBuilderAccumulatesSteps(t *testing.T) {
	tb := newTraceBuilder("query", "scope", "reason", 6000)
	tb.step("fast_seed", "ok", 10*time.Millisecond, "")
	tb.step("kg_expansion", "error", 5*time.Millisecond, "kg_timeout")
	tb.trace.CandidateIDs = append(tb.trace.CandidateIDs, "mem_1", "mem_2")

	trace := tb.finish(true, "kg_timeout")

	if trace.TraceID == "" {
		t.Fatal("trace ID is empty")
	}
	if trace.Query != "query" || trace.Scope != "scope" || trace.TriggerReason != "reason" {
		t.Fatalf("trace=%+v", trace)
	}
	if len(trace.Steps) != 2 {
		t.Fatalf("steps=%d, want 2", len(trace.Steps))
	}
	if trace.Steps[1].ErrorCode != "kg_timeout" {
		t.Fatalf("step[1]=%+v", trace.Steps[1])
	}
	if !trace.Degraded || trace.FailureState != "kg_timeout" {
		t.Fatalf("degraded=%v failure=%q", trace.Degraded, trace.FailureState)
	}
	if trace.Budget.BudgetChars != 6000 {
		t.Fatalf("budget=%d", trace.Budget.BudgetChars)
	}
	if len(trace.CandidateIDs) != 2 {
		t.Fatalf("candidates=%v", trace.CandidateIDs)
	}
}
