// Package diary provides AAAK-encoded diary storage for specialist agents.
// Each agent has its own wing and personal diary for domain-specific expertise.
package diary

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Entry represents a single diary entry for a specialist agent.
type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Agent     string    `json:"agent"`
	Wing      string    `json:"wing"`
	Content   string    `json:"content"`
}

// Diary manages agent-specific diary entries stored as JSONL files.
type Diary struct {
	dir string
}

// New creates a new Diary instance, ensuring the directory exists.
func New(palacePath string) (*Diary, error) {
	dir := filepath.Join(palacePath, "diary")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create diary dir: %w", err)
	}
	return &Diary{dir: dir}, nil
}

// Write appends a new diary entry.
func (d *Diary) Write(entry Entry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	filename := filepath.Join(d.dir, fmt.Sprintf("%s.jsonl", entry.Agent))
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open diary file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write entry: %w", err)
	}
	return nil
}

// Read retrieves recent diary entries with optional filters.
func (d *Diary) Read(agent string, wing string, limit int, since time.Time) ([]Entry, error) {
	if limit == 0 {
		limit = 50
	}

	entries, err := d.readAll()
	if err != nil {
		return nil, err
	}

	// Apply filters
	var filtered []Entry
	for _, e := range entries {
		if agent != "" && e.Agent != agent {
			continue
		}
		if wing != "" && e.Wing != wing {
			continue
		}
		if !since.IsZero() && e.Timestamp.Before(since) {
			continue
		}
		filtered = append(filtered, e)
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})

	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

// ListAgents returns all agents with diary entries.
func (d *Diary) ListAgents() ([]string, error) {
	entries, err := os.ReadDir(d.dir)
	if err != nil {
		return nil, err
	}

	var agents []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			agents = append(agents, strings.TrimSuffix(e.Name(), ".jsonl"))
		}
	}
	sort.Strings(agents)
	return agents, nil
}

// readAll reads all diary entries from all agent files.
func (d *Diary) readAll() ([]Entry, error) {
	files, err := filepath.Glob(filepath.Join(d.dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}

	var allEntries []Entry
	for _, file := range files {
		entries, err := d.readFile(file)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", file, err)
		}
		allEntries = append(allEntries, entries...)
	}

	// Sort by timestamp ascending
	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].Timestamp.Before(allEntries[j].Timestamp)
	})
	return allEntries, nil
}

// readFile reads entries from a single JSONL file.
func (d *Diary) readFile(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, scanner.Err()
}
