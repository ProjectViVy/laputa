package wal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewWALCreatesDirectory(t *testing.T) {
	tmp := t.TempDir()
	wal, err := NewWAL(tmp)
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	defer wal.Close()

	expected := filepath.Join(tmp, "wal")
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Errorf("WAL directory was not created at %s", expected)
	}
	if wal.dir != expected {
		t.Errorf("WAL.dir = %s, want %s", wal.dir, expected)
	}
}

func TestAppendSyncsToFile(t *testing.T) {
	tmp := t.TempDir()
	wal, err := NewWAL(tmp)
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	defer wal.Close()

	entry := Entry{
		Timestamp: time.Now(),
		Op:        OpAdd,
		DrawerID:  "drawer-1",
		Version:   1,
		Wing:      "north",
		Room:      "101",
		Content:   "test sync",
	}

	if err := wal.Append(entry); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Verify file on disk has the data
	files, err := filepath.Glob(filepath.Join(wal.dir, "wal-*.jsonl"))
	if err != nil || len(files) == 0 {
		t.Fatalf("Failed to find wal segment files")
	}

	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("Failed to read wal file: %v", err)
	}
	if !strings.Contains(string(data), "test sync") {
		t.Errorf("Expected 'test sync' in file, got: %s", string(data))
	}
}

func TestSegmentRotation(t *testing.T) {
	tmp := t.TempDir()
	// Set very small max bytes to trigger rotation immediately
	wal, err := NewWAL(tmp, WithMaxSegmentBytes(100))
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	defer wal.Close()

	// Write entries that will exceed 100 bytes
	for i := 0; i < 3; i++ {
		entry := Entry{
			Timestamp: time.Now(),
			Op:        OpAdd,
			DrawerID:  "drawer-long",
			Content:   strings.Repeat("X", 60), // Each will definitely exceed 100 bytes
		}
		if err := wal.Append(entry); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	files, err := filepath.Glob(filepath.Join(wal.dir, "wal-*.jsonl"))
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}
	if len(files) < 2 {
		t.Errorf("Expected multiple segment files, got %d", len(files))
	}
}

func TestReplayFromCheckpoint(t *testing.T) {
	tmp := t.TempDir()
	wal, err := NewWAL(tmp)
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	defer wal.Close()

	for i := 1; i <= 10; i++ {
		wal.Append(Entry{Op: OpAdd, DrawerID: "d" + string(rune(i))})
	}

	// Find offset to simulate checkpoint at entry 5
	files, _ := filepath.Glob(filepath.Join(wal.dir, "wal-000001.jsonl"))
	data, _ := os.ReadFile(files[0])
	lines := strings.Split(string(data), "\n")

	offset := int64(0)
	for i := 0; i < 5; i++ {
		offset += int64(len(lines[i])) + 1 // +1 for newline
	}

	cp := Checkpoint{
		Segment:    filepath.Base(files[0]),
		ByteOffset: offset,
	}

	res, err := wal.ReplayFrom(cp)
	if err != nil {
		t.Fatalf("ReplayFrom failed: %v", err)
	}

	if len(res.Entries) != 5 {
		t.Errorf("Expected 5 entries, got %d", len(res.Entries))
	}
}

func TestTornLineRecovery(t *testing.T) {
	tmp := t.TempDir()
	walDir := filepath.Join(tmp, "wal")
	os.MkdirAll(walDir, 0755)

	segment := filepath.Join(walDir, "wal-000001.jsonl")

	validEntry := `{"op":"add","drawer_id":"valid"}` + "\n"
	tornEntry := `{"op":"add","drawer_id":"incomple` // No newline

	os.WriteFile(segment, []byte(validEntry+tornEntry), 0644)

	wal, err := NewWAL(tmp)
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	defer wal.Close()

	res, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(res.Entries) != 1 {
		t.Errorf("Expected 1 valid entry, got %d", len(res.Entries))
	}

	if len(res.TornOffsets) != 1 {
		t.Fatalf("Expected 1 torn offset, got %d", len(res.TornOffsets))
	}

	if res.TornOffsets[0] != int64(len(validEntry)) {
		t.Errorf("Expected torn offset at %d, got %d", len(validEntry), res.TornOffsets[0])
	}
}

func TestReplayIdempotence(t *testing.T) {
	tmp := t.TempDir()
	wal, err := NewWAL(tmp)
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	defer wal.Close()

	wal.Append(Entry{Op: OpAdd, DrawerID: "idempotent"})

	res1, _ := wal.ReadAll()
	res2, _ := wal.ReadAll()

	if len(res1.Entries) != len(res2.Entries) {
		t.Errorf("ReadAll not idempotent, res1 len %d != res2 len %d", len(res1.Entries), len(res2.Entries))
	}
}

func TestCheckpointPersistence(t *testing.T) {
	tmp := t.TempDir()
	wal, err := NewWAL(tmp)
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	defer wal.Close()

	cp := Checkpoint{
		Segment:            "wal-000002.jsonl",
		ByteOffset:         1024,
		LastCommittedJobID: "job-123",
		CapturedAt:         time.Now().UTC(),
	}

	if err := wal.SaveCheckpoint(cp); err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	loaded, err := wal.LoadCheckpoint()
	if err != nil {
		t.Fatalf("LoadCheckpoint failed: %v", err)
	}
	if loaded == nil {
		t.Fatalf("LoadCheckpoint returned nil")
	}

	if loaded.Segment != cp.Segment || loaded.ByteOffset != cp.ByteOffset || loaded.LastCommittedJobID != cp.LastCommittedJobID {
		t.Errorf("Loaded checkpoint does not match saved")
	}
}

func TestLogAddBackwardCompat(t *testing.T) {
	tmp := t.TempDir()
	wal, err := NewWAL(tmp)
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	defer wal.Close()

	if err := wal.LogAdd(Entry{DrawerID: "test"}); err != nil {
		t.Fatalf("LogAdd failed: %v", err)
	}

	res, _ := wal.ReadAll()
	if len(res.Entries) != 1 || res.Entries[0].Op != OpAdd {
		t.Errorf("LogAdd did not append correctly")
	}
}

func TestLogDeleteBackwardCompat(t *testing.T) {
	tmp := t.TempDir()
	wal, err := NewWAL(tmp)
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	defer wal.Close()

	if err := wal.LogDelete("test"); err != nil {
		t.Fatalf("LogDelete failed: %v", err)
	}

	res, _ := wal.ReadAll()
	if len(res.Entries) != 1 || res.Entries[0].Op != OpDelete {
		t.Errorf("LogDelete did not append correctly")
	}
}

func TestReadAllEmptyDir(t *testing.T) {
	tmp := t.TempDir()
	wal, err := NewWAL(tmp)
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	defer wal.Close()

	res, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if len(res.Entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(res.Entries))
	}
}
