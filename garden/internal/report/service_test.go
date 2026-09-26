package report

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/dashimaki/mentle/facade"
	_ "modernc.org/sqlite"
)

type fakeLister struct{ items []facade.Memory }

func (f fakeLister) ListMemories(context.Context, facade.ListMemoryOptions) (facade.MemoryPage, error) {
	return facade.MemoryPage{Items: f.items}, nil
}

func TestGenerateReportIsSourceIdempotent(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	svc, err := Open(filepath.Join(t.TempDir(), "garden.db"), fakeLister{items: []facade.Memory{{ID: "mem_1", Content: "API accepted", UpdatedAt: now}}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	first, err := svc.Generate(context.Background(), "daily", now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Generate(context.Background(), "daily", now)
	if err != nil {
		t.Fatal(err)
	}
	if first.SourceHash != second.SourceHash {
		t.Fatal("source hash changed")
	}
	latest, err := svc.Latest(context.Background(), "daily")
	if err != nil || latest.SourceHash != first.SourceHash {
		t.Fatalf("latest=%+v err=%v", latest, err)
	}
	var count int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM reports WHERE cadence='daily'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

func TestGenerateArtifactDerivation(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	svc, err := Open(filepath.Join(t.TempDir(), "garden.db"), fakeLister{items: []facade.Memory{
		{ID: "mem_1", Kind: "fact", Content: "implemented recall", UpdatedAt: now},
		{ID: "mem_2", Kind: "decision", Content: "adopted dual-write storage", UpdatedAt: now},
	}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	if _, err := svc.Generate(context.Background(), "daily", now); err != nil {
		t.Fatal(err)
	}
	latest, err := svc.Latest(context.Background(), "daily")
	if err != nil {
		t.Fatal(err)
	}
	if latest.Scope != "mentle_active" {
		t.Fatalf("scope=%q", latest.Scope)
	}
	if latest.Generator != GeneratorDeterministic {
		t.Fatalf("generator=%q", latest.Generator)
	}
	if len(latest.Decisions) != 1 || latest.Decisions[0] != "adopted dual-write storage" {
		t.Fatalf("decisions=%v", latest.Decisions)
	}
	if len(latest.Completed) != len(latest.Highlights) {
		t.Fatalf("completed=%v highlights=%v", latest.Completed, latest.Highlights)
	}
	if latest.Goals == nil || latest.OpenLoops == nil || len(latest.Goals) != 0 || len(latest.OpenLoops) != 0 {
		t.Fatalf("goals=%v openLoops=%v", latest.Goals, latest.OpenLoops)
	}
	if len(latest.SourceRefs) != 2 || latest.SourceRefs[0] != "mem_1" {
		t.Fatalf("sourceRefs=%v", latest.SourceRefs)
	}
	if latest.Revision != 1 {
		t.Fatalf("revision=%d", latest.Revision)
	}
}

func TestGenerateRevisionIncrements(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	lister := &fakeLister{items: []facade.Memory{{ID: "mem_1", Content: "first", UpdatedAt: now}}}
	svc, err := Open(filepath.Join(t.TempDir(), "garden.db"), lister, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	first, err := svc.Generate(context.Background(), "daily", now)
	if err != nil {
		t.Fatal(err)
	}
	if first.Revision != 1 {
		t.Fatalf("first revision=%d", first.Revision)
	}
	lister.items = append(lister.items, facade.Memory{ID: "mem_2", Content: "second", UpdatedAt: now})
	second, err := svc.Generate(context.Background(), "daily", now)
	if err != nil {
		t.Fatal(err)
	}
	if second.Revision != 2 {
		t.Fatalf("second revision=%d", second.Revision)
	}
	latest, err := svc.Latest(context.Background(), "daily")
	if err != nil {
		t.Fatal(err)
	}
	if latest.Revision != 2 {
		t.Fatalf("latest revision=%d", latest.Revision)
	}
}

func TestOpenMigratesLegacySchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "garden.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE reports(cadence TEXT NOT NULL,window_start TEXT NOT NULL,window_end TEXT NOT NULL,source_ids TEXT NOT NULL,source_hash TEXT NOT NULL,title TEXT NOT NULL,summary TEXT NOT NULL,highlights TEXT NOT NULL,open_questions TEXT NOT NULL,generated_at TEXT NOT NULL,PRIMARY KEY(cadence,window_start,source_hash));CREATE INDEX reports_latest ON reports(cadence,generated_at DESC);`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO reports VALUES('daily','2026-07-15T00:00:00Z','2026-07-16T00:00:00Z','["mem_1"]','sha256:abc','Legacy','legacy summary','["legacy highlight"]','[]','2026-07-15T12:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	svc, err := Open(path, fakeLister{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	latest, err := svc.Latest(context.Background(), "daily")
	if err != nil {
		t.Fatal(err)
	}
	if latest.Title != "Legacy" || latest.Summary != "legacy summary" {
		t.Fatalf("legacy row lost: %+v", latest)
	}
	if latest.Generator != GeneratorDeterministic || latest.Revision != 0 {
		t.Fatalf("artifact defaults=%q rev=%d", latest.Generator, latest.Revision)
	}
}
