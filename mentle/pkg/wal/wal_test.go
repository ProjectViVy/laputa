package wal

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewWALCreatesDirectory(t *testing.T) {
	tmp := t.TempDir()
	wal, err := NewWAL(tmp)
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	expected := filepath.Join(tmp, "wal")
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Errorf("WAL directory was not created at %s", expected)
	}
	if wal.dir != expected {
		t.Errorf("WAL.dir = %s, want %s", wal.dir, expected)
	}
}

func TestLogAddWritesEntry(t *testing.T) {
	tmp := t.TempDir()
	wal, err := NewWAL(tmp)
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	entry := Entry{
		DrawerID: "drawer-1",
		Wing:     "north",
		Room:     "101",
		Content:  "test content",
	}
	if err := wal.LogAdd(entry); err != nil {
		t.Fatalf("LogAdd failed: %v", err)
	}
	entries, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ReadAll returned %d entries, want 1", len(entries))
	}
	if entries[0].Op != OpAdd {
		t.Errorf("entries[0].Op = %s, want %s", entries[0].Op, OpAdd)
	}
	if entries[0].DrawerID != "drawer-1" {
		t.Errorf("entries[0].DrawerID = %s, want drawer-1", entries[0].DrawerID)
	}
	if entries[0].Wing != "north" {
		t.Errorf("entries[0].Wing = %s, want north", entries[0].Wing)
	}
	if entries[0].Room != "101" {
		t.Errorf("entries[0].Room = %s, want 101", entries[0].Room)
	}
	if entries[0].Content != "test content" {
		t.Errorf("entries[0].Content = %s, want 'test content'", entries[0].Content)
	}
}

func TestLogDeleteWritesEntry(t *testing.T) {
	tmp := t.TempDir()
	wal, err := NewWAL(tmp)
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	if err := wal.LogDelete("drawer-42"); err != nil {
		t.Fatalf("LogDelete failed: %v", err)
	}
	entries, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ReadAll returned %d entries, want 1", len(entries))
	}
	if entries[0].Op != OpDelete {
		t.Errorf("entries[0].Op = %s, want %s", entries[0].Op, OpDelete)
	}
	if entries[0].DrawerID != "drawer-42" {
		t.Errorf("entries[0].DrawerID = %s, want drawer-42", entries[0].DrawerID)
	}
}

func TestReadAllReadsAllEntries(t *testing.T) {
	tmp := t.TempDir()
	wal, err := NewWAL(tmp)
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	if err := wal.LogAdd(Entry{DrawerID: "d1", Wing: "W", Room: "R"}); err != nil {
		t.Fatalf("LogAdd failed: %v", err)
	}
	if err := wal.LogAdd(Entry{DrawerID: "d2", Wing: "W", Room: "R"}); err != nil {
		t.Fatalf("LogAdd failed: %v", err)
	}
	if err := wal.LogDelete("d1"); err != nil {
		t.Fatalf("LogDelete failed: %v", err)
	}
	entries, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("ReadAll returned %d entries, want 3", len(entries))
	}
}

func TestWALWorksAcrossMultipleFiles(t *testing.T) {
	tmp := t.TempDir()
	wal, err := NewWAL(tmp)
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	f1 := filepath.Join(wal.dir, "write_log_"+today+".jsonl")
	f2 := filepath.Join(wal.dir, "write_log_"+tomorrow+".jsonl")
	if err := os.WriteFile(f1, []byte(`{"timestamp":"2024-01-01T00:00:00Z","op":"add","drawer_id":"d1","wing":"W1","room":"R1","content":"c1"}
`), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(f2, []byte(`{"timestamp":"2024-01-02T00:00:00Z","op":"add","drawer_id":"d2","wing":"W2","room":"R2","content":"c2"}
`), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	entries, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("ReadAll returned %d entries, want 2", len(entries))
	}
	if entries[0].DrawerID != "d1" {
		t.Errorf("entries[0].DrawerID = %s, want d1", entries[0].DrawerID)
	}
	if entries[1].DrawerID != "d2" {
		t.Errorf("entries[1].DrawerID = %s, want d2", entries[1].DrawerID)
	}
}
