package governance

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type FileAuditLog struct {
	path string
	mu   sync.Mutex
	seq  int64
}

func NewFileAuditLog(baseDir string) (*FileAuditLog, error) {
	dir := filepath.Join(baseDir, "audit")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create audit dir: %w", err)
	}
	path := filepath.Join(dir, "changelog.jsonl")
	f := &FileAuditLog{path: path}
	f.seq = f.lastSequence()
	return f, nil
}

func (f *FileAuditLog) Append(_ context.Context, entry AuditEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	entry.Sequence = f.seq

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal audit entry: %w", err)
	}

	file, err := os.OpenFile(f.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open audit log: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("write audit entry: %w", err)
	}
	return nil
}

func (f *FileAuditLog) Recent(_ context.Context, limit int) ([]AuditEntry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	file, err := os.Open(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open audit log: %w", err)
	}
	defer file.Close()

	var entries []AuditEntry
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)
	for scanner.Scan() {
		var e AuditEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan audit log: %w", err)
	}

	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	return entries, nil
}

func (f *FileAuditLog) lastSequence() int64 {
	file, err := os.Open(f.path)
	if err != nil {
		return 0
	}
	defer file.Close()

	var seq int64
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)
	for scanner.Scan() {
		var e AuditEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		if e.Sequence > seq {
			seq = e.Sequence
		}
	}
	return seq
}
