package governance

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestAuditLog(t *testing.T, cfg RotationConfig, now time.Time) *FileAuditLog {
	t.Helper()
	f, err := NewFileAuditLogWithRotation(t.TempDir(), cfg)
	if err != nil {
		t.Fatalf("NewFileAuditLogWithRotation: %v", err)
	}
	f.now = func() time.Time { return now }
	return f
}

func appendEntry(t *testing.T, f *FileAuditLog, ts time.Time) AuditEntry {
	t.Helper()
	entry := AuditEntry{Section: "01-identity", Action: "write", Actor: "user", Timestamp: ts.UTC().Format(time.RFC3339)}
	if err := f.Append(context.Background(), entry); err != nil {
		t.Fatalf("Append: %v", err)
	}
	return entry
}

func TestAuditRotationBySize(t *testing.T) {
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	f := newTestAuditLog(t, RotationConfig{MaxAgeDays: 90, MaxSizeBytes: 100}, now)

	appendEntry(t, f, now)
	appendEntry(t, f, now) // ~150-byte log exceeds 100-byte limit, rotates before write

	backups := backupFiles(t, f)
	if len(backups) != 1 {
		t.Fatalf("got %d backups, want 1", len(backups))
	}
	if !strings.Contains(backups[0], "changelog.2026-08-03.jsonl") || !strings.HasSuffix(backups[0], ".bak") {
		t.Errorf("unexpected backup name: %s", backups[0])
	}

	entries, err := f.Recent(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("current log has %d entries, want 1", len(entries))
	}
}

func TestAuditRotationByAge(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	f := newTestAuditLog(t, RotationConfig{MaxAgeDays: 90, MaxSizeBytes: 50 << 20}, start)

	appendEntry(t, f, start)

	// Advance past the age limit; next append must rotate.
	f.now = func() time.Time { return start.AddDate(0, 0, 91) }
	appendEntry(t, f, start.AddDate(0, 0, 91))

	backups := backupFiles(t, f)
	if len(backups) != 1 {
		t.Fatalf("got %d backups, want 1", len(backups))
	}
	entries, err := f.Recent(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("current log has %d entries, want 1", len(entries))
	}
}

func TestAuditNoRotationWithinLimits(t *testing.T) {
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	f := newTestAuditLog(t, DefaultRotationConfig(), now)

	for i := 0; i < 5; i++ {
		appendEntry(t, f, now)
	}
	if got := len(backupFiles(t, f)); got != 0 {
		t.Fatalf("got %d backups, want 0", got)
	}
	entries, err := f.Recent(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 5 {
		t.Errorf("got %d entries, want 5", len(entries))
	}
}

func TestAuditSequenceMonotonicAcrossRotationAndRestart(t *testing.T) {
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	f, err := NewFileAuditLogWithRotation(dir, RotationConfig{MaxAgeDays: 90, MaxSizeBytes: 100})
	if err != nil {
		t.Fatal(err)
	}
	f.now = func() time.Time { return now }

	appendEntry(t, f, now) // seq 1
	appendEntry(t, f, now) // rotates: backup=[seq1], current=[seq2]
	appendEntry(t, f, now) // rotates: backup=[seq2], current=[seq3]

	// Simulate the current log being lost after rotation; a fresh
	// instance must recover the highest sequence from the backups.
	if err := os.Remove(f.path); err != nil {
		t.Fatal(err)
	}
	f2, err := NewFileAuditLogWithRotation(dir, RotationConfig{MaxAgeDays: 90, MaxSizeBytes: 1 << 30})
	if err != nil {
		t.Fatal(err)
	}
	if f2.seq != 2 {
		t.Fatalf("recovered seq = %d, want 2 (max across backups)", f2.seq)
	}
	appendEntry(t, f2, now)

	entries, err := f2.Recent(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Sequence != 3 {
		t.Errorf("after restart: entries=%v, want single entry with seq 3", entries)
	}
}

func TestAuditRotationSameDaySuffix(t *testing.T) {
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	f := newTestAuditLog(t, RotationConfig{MaxAgeDays: 90, MaxSizeBytes: 100}, now)

	appendEntry(t, f, now)
	appendEntry(t, f, now) // rotate #1
	appendEntry(t, f, now) // rotate #2 (same day)

	names := backupFiles(t, f)
	if len(names) != 2 {
		t.Fatalf("got %d backups, want 2", len(names))
	}
	seen := map[string]bool{}
	for _, name := range names {
		base := filepath.Base(name)
		if seen[base] {
			t.Errorf("duplicate backup name: %s", base)
		}
		seen[base] = true
		if !strings.HasSuffix(base, ".bak") {
			t.Errorf("backup %s missing .bak suffix", base)
		}
	}
}

func backupFiles(t *testing.T, f *FileAuditLog) []string {
	t.Helper()
	return backups(f.path)
}

func TestDefaultRotationConfig(t *testing.T) {
	cfg := DefaultRotationConfig()
	if cfg.MaxAgeDays != 90 || cfg.MaxSizeBytes != 50<<20 {
		t.Errorf("DefaultRotationConfig = %+v", cfg)
	}
}
