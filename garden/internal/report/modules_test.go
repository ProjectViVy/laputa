package report

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dashimaki/mentle/facade"
)

func openTestService(t *testing.T, items []facade.Memory) *Service {
	t.Helper()
	svc, err := Open(filepath.Join(t.TempDir(), "garden.db"), fakeLister{items: items}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { svc.Close() })
	return svc
}

func insertModule(t *testing.T, svc *Service, id, kind, content, status, created string) {
	t.Helper()
	if _, err := svc.db.Exec(`INSERT INTO human_modules(id,kind,content,status,created_at,updated_at) VALUES(?,?,?,?,?,?)`, id, kind, content, status, created, created); err != nil {
		t.Fatal(err)
	}
}

func TestModuleCRUD(t *testing.T) {
	svc := openTestService(t, nil)
	ctx := context.Background()

	amb, err := svc.CreateModule(ctx, ModuleKindAmbition, "become the best gardener")
	if err != nil {
		t.Fatal(err)
	}
	if amb.ID == "" || amb.Kind != ModuleKindAmbition || amb.Status != ModuleStatusActive {
		t.Fatalf("created=%+v", amb)
	}
	sug, err := svc.CreateModule(ctx, ModuleKindSuggestion, "add a dark theme")
	if err != nil {
		t.Fatal(err)
	}

	active, err := svc.ListModules(ctx, ModuleKindAmbition, ModuleStatusActive)
	if err != nil || len(active) != 1 || active[0].ID != amb.ID {
		t.Fatalf("active=%v err=%v", active, err)
	}
	all, err := svc.ListModules(ctx, ModuleKindSuggestion, "all")
	if err != nil || len(all) != 1 || all[0].ID != sug.ID {
		t.Fatalf("all=%v err=%v", all, err)
	}

	updated, err := svc.UpdateModule(ctx, amb.ID, "become the best gardener in the world", "")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Content != "become the best gardener in the world" {
		t.Fatalf("updated=%+v", updated)
	}
	got, err := svc.ListModules(ctx, ModuleKindAmbition, "all")
	if err != nil || len(got) != 1 || got[0].Content != "become the best gardener in the world" {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestModuleValidation(t *testing.T) {
	svc := openTestService(t, nil)
	ctx := context.Background()

	if _, err := svc.CreateModule(ctx, "wish", "x"); !errors.Is(err, ErrInvalidModuleKind) {
		t.Fatalf("bad kind err=%v", err)
	}
	if _, err := svc.CreateModule(ctx, ModuleKindAmbition, ""); !errors.Is(err, ErrEmptyModuleContent) {
		t.Fatalf("empty content err=%v", err)
	}
	if _, err := svc.CreateModule(ctx, ModuleKindAmbition, "   "); !errors.Is(err, ErrEmptyModuleContent) {
		t.Fatalf("blank content err=%v", err)
	}
	long := strings.Repeat("x", MaxModuleContentRunes+1)
	if _, err := svc.CreateModule(ctx, ModuleKindAmbition, long); !errors.Is(err, ErrModuleContentTooLong) {
		t.Fatalf("long content err=%v", err)
	}

	m, err := svc.CreateModule(ctx, ModuleKindAmbition, "ok")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateModule(ctx, m.ID, "", "bogus"); !errors.Is(err, ErrInvalidModuleStatus) {
		t.Fatalf("bad status err=%v", err)
	}
	if _, err := svc.UpdateModule(ctx, m.ID, "", ""); !errors.Is(err, ErrNoModuleChange) {
		t.Fatalf("no change err=%v", err)
	}
	if _, err := svc.UpdateModule(ctx, m.ID, "   ", ""); !errors.Is(err, ErrEmptyModuleContent) {
		t.Fatalf("blank edit err=%v", err)
	}
	if _, err := svc.ListModules(ctx, ModuleKindAmbition, "weird"); !errors.Is(err, ErrInvalidModuleStatus) {
		t.Fatalf("bad list status err=%v", err)
	}
	if _, err := svc.ListModules(ctx, "nope", ModuleStatusActive); !errors.Is(err, ErrInvalidModuleKind) {
		t.Fatalf("bad list kind err=%v", err)
	}
	if _, err := svc.UpdateModule(ctx, "mod_missing", "x", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing id err=%v", err)
	}
}

func TestModuleStateMachine(t *testing.T) {
	svc := openTestService(t, nil)
	ctx := context.Background()

	m, err := svc.CreateModule(ctx, ModuleKindSuggestion, "keep reports deterministic")
	if err != nil {
		t.Fatal(err)
	}
	dismissed, err := svc.UpdateModule(ctx, m.ID, "", ModuleStatusDismissed)
	if err != nil || dismissed.Status != ModuleStatusDismissed {
		t.Fatalf("dismissed=%+v err=%v", dismissed, err)
	}
	active, err := svc.ListModules(ctx, ModuleKindSuggestion, ModuleStatusActive)
	if err != nil || len(active) != 0 {
		t.Fatalf("active should be empty: %v err=%v", active, err)
	}
	dismissedList, err := svc.ListModules(ctx, ModuleKindSuggestion, ModuleStatusDismissed)
	if err != nil || len(dismissedList) != 1 {
		t.Fatalf("dismissed list=%v err=%v", dismissedList, err)
	}
	restored, err := svc.UpdateModule(ctx, m.ID, "", ModuleStatusActive)
	if err != nil || restored.Status != ModuleStatusActive {
		t.Fatalf("restored=%+v err=%v", restored, err)
	}
}

func TestMonthlyReportAttachesModuleNames(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	// Open with an empty lister so the startup loop generates nothing;
	// the lister is populated after module inserts to keep the test
	// deterministic (Generate is idempotent per source_hash).
	lister := &fakeLister{}
	svc, err := Open(filepath.Join(t.TempDir(), "garden.db"), lister, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { svc.Close() })
	ctx := context.Background()

	insertModule(t, svc, "mod_amb", ModuleKindAmbition, "ship obsidian adapter", ModuleStatusActive, "2026-07-05T10:00:00Z")
	insertModule(t, svc, "mod_sug", ModuleKindSuggestion, "bundle the console", ModuleStatusActive, "2026-07-06T10:00:00Z")
	insertModule(t, svc, "mod_old", ModuleKindAmbition, "out of window", ModuleStatusActive, "2026-06-20T10:00:00Z")
	insertModule(t, svc, "mod_done", ModuleKindSuggestion, "dismissed in window", ModuleStatusDismissed, "2026-07-07T10:00:00Z")
	lister.items = []facade.Memory{{ID: "mem_1", Content: "work", UpdatedAt: now}}

	monthly, err := svc.Generate(ctx, "monthly", now)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"AMBITION", "USER SUGGESTIONS"}
	if len(monthly.Modules) != 2 || monthly.Modules[0] != want[0] || monthly.Modules[1] != want[1] {
		t.Fatalf("monthly modules=%v", monthly.Modules)
	}
	saved, err := svc.Latest(ctx, "monthly")
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Modules) != 2 || saved.Modules[0] != want[0] || saved.Modules[1] != want[1] {
		t.Fatalf("saved modules=%v", saved.Modules)
	}

	daily, err := svc.Generate(ctx, "daily", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(daily.Modules) != 0 {
		t.Fatalf("daily modules=%v", daily.Modules)
	}
	weekly, err := svc.Generate(ctx, "weekly", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(weekly.Modules) != 0 {
		t.Fatalf("weekly modules=%v", weekly.Modules)
	}
}

func TestMonthlyReportEmptyModulesWhenNoneInWindow(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	lister := &fakeLister{}
	svc, err := Open(filepath.Join(t.TempDir(), "garden.db"), lister, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { svc.Close() })
	ctx := context.Background()

	insertModule(t, svc, "mod_amb", ModuleKindAmbition, "last month only", ModuleStatusActive, "2026-06-20T10:00:00Z")
	lister.items = []facade.Memory{{ID: "mem_1", Content: "work", UpdatedAt: now}}

	monthly, err := svc.Generate(ctx, "monthly", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(monthly.Modules) != 0 {
		t.Fatalf("modules=%v", monthly.Modules)
	}
}

func TestLegacyReportRowReadsBackEmptyModules(t *testing.T) {
	path := filepath.Join(t.TempDir(), "garden.db")
	svc, err := Open(path, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	if _, err := svc.db.Exec(`INSERT INTO reports(cadence,window_start,window_end,source_ids,source_hash,title,summary,highlights,open_questions,generated_at,artifact) VALUES('monthly','2026-06-01T00:00:00Z','2026-07-01T00:00:00Z','["mem_1"]','sha256:abc','Old','old','[]','[]','2026-06-20T12:00:00Z','{}')`); err != nil {
		t.Fatal(err)
	}
	latest, err := svc.Latest(context.Background(), "monthly")
	if err != nil {
		t.Fatal(err)
	}
	if latest.Modules == nil || len(latest.Modules) != 0 {
		t.Fatalf("modules=%v (want empty, non-nil)", latest.Modules)
	}
}
