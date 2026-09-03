package wal

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Operation string

const (
	OpAdd     Operation = "add"
	OpDelete  Operation = "delete"
	OpReindex Operation = "reindex"
)

type Entry struct {
	Timestamp time.Time       `json:"timestamp"`
	Op        Operation       `json:"op"`
	DrawerID  string          `json:"drawer_id"`
	Version   int             `json:"version,omitempty"`
	Wing      string          `json:"wing,omitempty"`
	Room      string          `json:"room,omitempty"`
	Content   string          `json:"content,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

type ReplayResult struct {
	Entries     []Entry
	TornOffsets []int64
}

type WAL struct {
	dir             string
	mu              sync.Mutex
	file            *os.File
	currentSegment  string
	currentSize     int64
	segmentSeq      int
	maxSegmentBytes int64
}

type Option func(*WAL)

func WithMaxSegmentBytes(n int64) Option {
	return func(w *WAL) {
		w.maxSegmentBytes = n
	}
}

func NewWAL(palacePath string, opts ...Option) (*WAL, error) {
	walDir := filepath.Join(palacePath, "wal")
	if err := os.MkdirAll(walDir, 0755); err != nil {
		return nil, err
	}

	w := &WAL{
		dir:             walDir,
		maxSegmentBytes: 64 * 1024 * 1024,
	}

	for _, opt := range opts {
		opt(w)
	}

	if err := w.openLatestSegment(); err != nil {
		return nil, err
	}

	return w, nil
}

func (w *WAL) openLatestSegment() error {
	matches, err := filepath.Glob(filepath.Join(w.dir, "wal-*.jsonl"))
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		w.segmentSeq = 1
		return w.openSegment()
	}

	sort.Strings(matches)
	latest := matches[len(matches)-1]
	base := filepath.Base(latest)

	seqStr := strings.TrimSuffix(strings.TrimPrefix(base, "wal-"), ".jsonl")
	seq, err := strconv.Atoi(seqStr)
	if err != nil {
		w.segmentSeq = 1
	} else {
		w.segmentSeq = seq
	}

	info, err := os.Stat(latest)
	if err != nil {
		return err
	}
	w.currentSize = info.Size()

	if w.currentSize >= w.maxSegmentBytes {
		w.segmentSeq++
		return w.openSegment()
	}

	w.currentSegment = latest
	f, err := os.OpenFile(w.currentSegment, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	w.file = f

	return nil
}

func (w *WAL) openSegment() error {
	w.currentSegment = filepath.Join(w.dir, fmt.Sprintf("wal-%06d.jsonl", w.segmentSeq))
	f, err := os.OpenFile(w.currentSegment, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	w.file = f
	w.currentSize = 0
	return nil
}

func (w *WAL) Append(entry Entry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if w.currentSize+int64(len(data)) > w.maxSegmentBytes && w.currentSize > 0 {
		if err := w.file.Close(); err != nil {
			return err
		}
		w.segmentSeq++
		if err := w.openSegment(); err != nil {
			return err
		}
	}

	n, err := w.file.Write(data)
	if err != nil {
		return err
	}
	w.currentSize += int64(n)

	return w.file.Sync()
}

func (w *WAL) LogAdd(entry Entry) error {
	entry.Timestamp = time.Now()
	entry.Op = OpAdd
	return w.Append(entry)
}

func (w *WAL) LogDelete(drawerID string) error {
	entry := Entry{
		Timestamp: time.Now(),
		Op:        OpDelete,
		DrawerID:  drawerID,
	}
	return w.Append(entry)
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func (w *WAL) ReadAll() (ReplayResult, error) {
	return w.ReplayFrom(Checkpoint{})
}

func (w *WAL) ReplayFrom(cp Checkpoint) (ReplayResult, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	var result ReplayResult

	matches, err := filepath.Glob(filepath.Join(w.dir, "wal-*.jsonl"))
	if err != nil {
		return result, err
	}
	sort.Strings(matches)

	for _, match := range matches {
		base := filepath.Base(match)

		if cp.Segment != "" && base < cp.Segment {
			continue
		}

		f, err := os.Open(match)
		if err != nil {
			return result, err
		}

		if cp.Segment == base && cp.ByteOffset > 0 {
			if _, err := f.Seek(cp.ByteOffset, io.SeekStart); err != nil {
				f.Close()
				return result, err
			}
		}

		reader := bufio.NewReader(f)
		offset := int64(0)
		if cp.Segment == base {
			offset = cp.ByteOffset
		}

		for {
			line, err := reader.ReadBytes('\n')
			if len(line) > 0 {
				if line[len(line)-1] != '\n' {
					// Torn line
					result.TornOffsets = append(result.TornOffsets, offset)
				} else {
					var entry Entry
					if unmarshalErr := json.Unmarshal(bytes.TrimSpace(line), &entry); unmarshalErr == nil {
						result.Entries = append(result.Entries, entry)
					}
				}
				offset += int64(len(line))
			}
			if err != nil {
				break
			}
		}
		f.Close()
	}

	return result, nil
}
