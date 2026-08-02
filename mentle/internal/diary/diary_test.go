package diary

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNewDiary(t *testing.T) {
	dir := t.TempDir()
	d, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if d == nil {
		t.Fatal("New returned nil")
	}
	if d.dir != filepath.Join(dir, "diary") {
		t.Errorf("unexpected dir: %s", d.dir)
	}
}

func TestDiaryWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	d, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	entry := Entry{
		Agent:   "test-agent",
		Wing:    "test-wing",
		Content: "test content",
	}

	if err := d.Write(entry); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	entries, err := d.Read("", "", 0, time.Time{})
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Agent != "test-agent" {
		t.Errorf("expected agent 'test-agent', got '%s'", entries[0].Agent)
	}
	if entries[0].Content != "test content" {
		t.Errorf("expected content 'test content', got '%s'", entries[0].Content)
	}
}

func TestDiaryReadWithFilters(t *testing.T) {
	dir := t.TempDir()
	d, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	entries := []Entry{
		{Agent: "agent1", Wing: "wing-a", Content: "entry 1"},
		{Agent: "agent1", Wing: "wing-b", Content: "entry 2"},
		{Agent: "agent2", Wing: "wing-a", Content: "entry 3"},
	}

	for _, e := range entries {
		if err := d.Write(e); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	// Filter by agent
	agent1Entries, err := d.Read("agent1", "", 0, time.Time{})
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(agent1Entries) != 2 {
		t.Errorf("expected 2 entries for agent1, got %d", len(agent1Entries))
	}

	// Filter by wing
	wingAEntries, err := d.Read("", "wing-a", 0, time.Time{})
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(wingAEntries) != 2 {
		t.Errorf("expected 2 entries for wing-a, got %d", len(wingAEntries))
	}
}

func TestDiaryListAgents(t *testing.T) {
	dir := t.TempDir()
	d, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	entries := []Entry{
		{Agent: "charlie", Content: "c"},
		{Agent: "alice", Content: "a"},
		{Agent: "bob", Content: "b"},
	}

	for _, e := range entries {
		if err := d.Write(e); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	agents, err := d.ListAgents()
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 3 {
		t.Errorf("expected 3 agents, got %d", len(agents))
	}
	// Should be sorted
	if agents[0] != "alice" || agents[1] != "bob" || agents[2] != "charlie" {
		t.Errorf("agents not sorted: %v", agents)
	}
}

func TestDiaryReadLimit(t *testing.T) {
	dir := t.TempDir()
	d, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	for range 10 {
		e := Entry{Agent: "test", Content: "entry"}
		if err := d.Write(e); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	entries, err := d.Read("", "", 3, time.Time{})
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries with limit, got %d", len(entries))
	}
}

func TestDiaryReadSince(t *testing.T) {
	dir := t.TempDir()
	d, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	oldEntry := Entry{Agent: "test", Timestamp: past, Content: "old"}
	newEntry := Entry{Agent: "test", Timestamp: now, Content: "new"}

	if err := d.Write(oldEntry); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := d.Write(newEntry); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	entries, err := d.Read("", "", 0, now.Add(-30*time.Minute))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry since recent time, got %d", len(entries))
	}
	if entries[0].Content != "new" {
		t.Errorf("expected 'new' entry, got '%s'", entries[0].Content)
	}

	_ = future
}

func TestDiaryReadEmptyDir(t *testing.T) {
	dir := t.TempDir()
	d, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	entries, err := d.Read("", "", 0, time.Time{})
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestDiaryWriteAutoTimestamp(t *testing.T) {
	dir := t.TempDir()
	d, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	entry := Entry{Agent: "test", Content: "content"}
	if err := d.Write(entry); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	entries, err := d.Read("", "", 0, time.Time{})
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if entries[0].Timestamp.IsZero() {
		t.Error("expected auto-generated timestamp")
	}
	if entries[0].Timestamp.After(time.Now()) {
		t.Error("timestamp is in the future")
	}
}
