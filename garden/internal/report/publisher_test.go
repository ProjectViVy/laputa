package report

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ProjectViVy/laputa/mentle/facade"
)

type rejectingPublisher struct{}

func (rejectingPublisher) Publish(context.Context, Report) error {
	return errors.New("external publication is not report authority")
}

func TestReportAuthorityRemainsGardenSQLite(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	svc, err := Open(filepath.Join(t.TempDir(), "garden.db"), fakeLister{items: []facade.Memory{{ID: "mem_1", Content: "work", UpdatedAt: now}}}, rejectingPublisher{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	if _, err := svc.Generate(context.Background(), "daily", now); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Latest(context.Background(), "daily"); err != nil {
		t.Fatal(err)
	}
}
