package report

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dashimaki/mentle/facade"
)

type fakeLister struct{ items []facade.Memory }

func (f fakeLister) ListMemories(context.Context, facade.ListMemoryOptions) (facade.MemoryPage, error) {
	return facade.MemoryPage{Items: f.items}, nil
}

func TestGenerateReportIsSourceIdempotent(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	svc, err := Open(filepath.Join(t.TempDir(), "garden.db"), fakeLister{items: []facade.Memory{{ID: "mem_1", Content: "API accepted", UpdatedAt: now}}})
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
