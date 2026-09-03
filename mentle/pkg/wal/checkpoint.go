package wal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Checkpoint struct {
	Segment            string    `json:"segment"`
	ByteOffset         int64     `json:"byte_offset"`
	LastCommittedJobID string    `json:"last_committed_job_id"`
	CapturedAt         time.Time `json:"captured_at"`
}

func (w *WAL) SaveCheckpoint(cp Checkpoint) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := filepath.Join(w.dir, "checkpoint.json.tmp")
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpFile, filepath.Join(w.dir, "checkpoint.json"))
}

func (w *WAL) LoadCheckpoint() (*Checkpoint, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := os.ReadFile(filepath.Join(w.dir, "checkpoint.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, err
	}

	return &cp, nil
}
