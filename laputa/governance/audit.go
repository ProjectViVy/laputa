package governance

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// RotationConfig bounds audit log growth (ADR-0004 §5.2).
type RotationConfig struct {
	MaxAgeDays   int
	MaxSizeBytes int64
}

func DefaultRotationConfig() RotationConfig {
	return RotationConfig{MaxAgeDays: 90, MaxSizeBytes: 50 << 20}
}

type FileAuditLog struct {
	path string
	mu   sync.Mutex
	seq  int64
	rot  RotationConfig
	now  func() time.Time
}

func NewFileAuditLog(baseDir string) (*FileAuditLog, error) {
	return NewFileAuditLogWithRotation(baseDir, DefaultRotationConfig())
}

func NewFileAuditLogWithRotation(baseDir string, cfg RotationConfig) (*FileAuditLog, error) {
	if cfg.MaxAgeDays <= 0 {
		cfg.MaxAgeDays = 90
	}
	if cfg.MaxSizeBytes <= 0 {
		cfg.MaxSizeBytes = 50 << 20
	}
	dir := filepath.Join(baseDir, "audit")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create audit dir: %w", err)
	}
	path := filepath.Join(dir, "changelog.jsonl")
	f := &FileAuditLog{path: path, rot: cfg, now: time.Now}
	f.seq = f.lastSequence()
	return f, nil
}

func (f *FileAuditLog) Append(_ context.Context, entry AuditEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.rotateIfNeeded(); err != nil {
		return err
	}
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

// rotateIfNeeded renames the current log to changelog.{date}.jsonl.bak when
// either the size or age limit is exceeded. Age is judged from the first
// entry's timestamp, not mtime (append refreshes mtime).
func (f *FileAuditLog) rotateIfNeeded() error {
	info, err := os.Stat(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat audit log: %w", err)
	}
	if info.Size() == 0 {
		return nil
	}

	rotate := info.Size() > f.rot.MaxSizeBytes
	if !rotate {
		if oldest, ok := f.oldestTimestamp(); ok {
			rotate = f.now().Sub(oldest) > time.Duration(f.rot.MaxAgeDays)*24*time.Hour
		}
	}
	if !rotate {
		return nil
	}

	date := f.now().UTC().Format("2006-01-02")
	target := filepath.Join(filepath.Dir(f.path), "changelog."+date+".jsonl.bak")
	for i := 1; ; i++ {
		if _, err := os.Stat(target); os.IsNotExist(err) {
			break
		}
		target = filepath.Join(filepath.Dir(f.path), fmt.Sprintf("changelog.%s.jsonl.%d.bak", date, i))
		if i > 1000 {
			return fmt.Errorf("too many rotated audit logs for %s", date)
		}
	}
	if err := os.Rename(f.path, target); err != nil {
		return fmt.Errorf("rotate audit log: %w", err)
	}
	return nil
}

func (f *FileAuditLog) oldestTimestamp() (time.Time, bool) {
	file, err := os.Open(f.path)
	if err != nil {
		return time.Time{}, false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)
	if !scanner.Scan() {
		return time.Time{}, false
	}
	var e AuditEntry
	if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
		return time.Time{}, false
	}
	ts, err := time.Parse(time.RFC3339, e.Timestamp)
	if err != nil {
		return time.Time{}, false
	}
	return ts, true
}

func (f *FileAuditLog) Recent(_ context.Context, limit int) ([]AuditEntry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries := readEntries(f.path)

	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	return entries, nil
}

// lastSequence scans the current log plus every rotated backup so sequence
// numbers stay monotonic across rotation and restarts. Backup count stays
// small because rotation is rare (90 days / 50 MB).
func (f *FileAuditLog) lastSequence() int64 {
	seq := maxSequenceIn(f.path)
	for _, bak := range backups(f.path) {
		if s := maxSequenceIn(bak); s > seq {
			seq = s
		}
	}
	return seq
}

func backups(path string) []string {
	dir := filepath.Dir(path)
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range dirEntries {
		name := entry.Name()
		if strings.HasPrefix(name, "changelog.") && strings.HasSuffix(name, ".bak") {
			out = append(out, filepath.Join(dir, name))
		}
	}
	return out
}

func readEntries(path string) []AuditEntry {
	file, err := os.Open(path)
	if err != nil {
		return nil
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
	return entries
}

func maxSequenceIn(path string) int64 {
	var seq int64
	for _, e := range readEntries(path) {
		if e.Sequence > seq {
			seq = e.Sequence
		}
	}
	return seq
}
