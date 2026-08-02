// Package wal implements a Write-Ahead Log for durable drawer operations.
// It provides crash recovery and audit trail capabilities.
package wal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Operation string

const (
	OpAdd    Operation = "add"
	OpDelete Operation = "delete"
)

type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Op        Operation `json:"op"`
	DrawerID  string    `json:"drawer_id"`
	Wing      string    `json:"wing"`
	Room      string    `json:"room"`
	Content   string    `json:"content,omitempty"`
	Metadata  any       `json:"metadata,omitempty"`
}

type WAL struct {
	dir string
}

func NewWAL(palacePath string) (*WAL, error) {
	walDir := filepath.Join(palacePath, "wal")
	if err := os.MkdirAll(walDir, 0755); err != nil {
		return nil, err
	}
	return &WAL{dir: walDir}, nil
}

func (w *WAL) LogAdd(entry Entry) error {
	entry.Timestamp = time.Now()
	entry.Op = OpAdd
	return w.write(entry)
}

func (w *WAL) LogDelete(drawerID string) error {
	entry := Entry{
		Timestamp: time.Now(),
		Op:        OpDelete,
		DrawerID:  drawerID,
	}
	return w.write(entry)
}

func (w *WAL) write(entry Entry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	filename := filepath.Join(w.dir, fmt.Sprintf("write_log_%s.jsonl", time.Now().Format("2006-01-02")))
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(data, '\n'))
	return err
}

func (w *WAL) ReadAll() ([]Entry, error) {
	var entries []Entry
	pattern := filepath.Join(w.dir, "write_log_*.jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	for _, match := range matches {
		entries = append(entries, w.readFile(match)...)
	}
	return entries, nil
}

func (w *WAL) readFile(path string) []Entry {
	var entries []Entry
	data, err := os.ReadFile(path)
	if err != nil {
		return entries
	}
	lines := splitLines(string(data))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
