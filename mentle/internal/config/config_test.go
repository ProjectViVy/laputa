package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigWithTempDir(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(configPath, []byte(`{
		"palace_path": "/test/palace",
		"collection_name": "test_collection",
		"people_map": {"alice": "/path/to/alice"},
		"topic_wings": ["test_wing"],
		"model_name": "sentence-transformers/all-MiniLM-L6-v2"
	}`), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.PalacePath != "/test/palace" {
		t.Errorf("PalacePath = %v, want /test/palace", cfg.PalacePath)
	}
	if cfg.CollectionName != "test_collection" {
		t.Errorf("CollectionName = %v, want test_collection", cfg.CollectionName)
	}
	if cfg.PeopleMap["alice"] != "/path/to/alice" {
		t.Errorf("PeopleMap[alice] = %v, want /path/to/alice", cfg.PeopleMap["alice"])
	}
	if len(cfg.TopicWings) != 1 || cfg.TopicWings[0] != "test_wing" {
		t.Errorf("TopicWings = %v, want [test_wing]", cfg.TopicWings)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.PalacePath != DefaultConfig.PalacePath {
		t.Errorf("PalacePath = %v, want %v", cfg.PalacePath, DefaultConfig.PalacePath)
	}
	if cfg.CollectionName != DefaultConfig.CollectionName {
		t.Errorf("CollectionName = %v, want %v", cfg.CollectionName, DefaultConfig.CollectionName)
	}
	if len(cfg.TopicWings) != len(DefaultConfig.TopicWings) {
		t.Errorf("TopicWings length = %v, want %v", len(cfg.TopicWings), len(DefaultConfig.TopicWings))
	}
}

func TestEnvVarOverride(t *testing.T) {
	tmpDir := t.TempDir()

	originalVal := os.Getenv("MEMPALACE_PALACE_PATH")
	defer func() {
		if originalVal != "" {
			os.Setenv("MEMPALACE_PALACE_PATH", originalVal)
		} else {
			os.Unsetenv("MEMPALACE_PALACE_PATH")
		}
	}()

	os.Setenv("MEMPALACE_PALACE_PATH", "/env/palace")

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.PalacePath != "/env/palace" {
		t.Errorf("PalacePath = %v, want /env/palace", cfg.PalacePath)
	}
}

func TestGetModelsDir(t *testing.T) {
	cfg := &Config{ModelsDir: "/custom/models"}
	if got := cfg.GetModelsDir(); got != "/custom/models" {
		t.Errorf("GetModelsDir() = %v, want /custom/models", got)
	}

	cfg = &Config{}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".mempalace", "models")
	if got := cfg.GetModelsDir(); got != expected {
		t.Errorf("GetModelsDir() = %v, want %v", got, expected)
	}
}

func TestGetIdentityPath(t *testing.T) {
	cfg := &Config{}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".mempalace", "identity.txt")
	if got, err := cfg.GetIdentityPath(); err != nil || got != expected {
		t.Errorf("GetIdentityPath() = %v, err = %v, want %v", got, err, expected)
	}
}

func TestLoadConfigWithRooms(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(configPath, []byte(`{
		"palace_path": "/test/palace",
		"collection_name": "test_collection",
		"rooms": [
			{"name": "frontend", "keywords": ["ui", "components", "views"]},
			{"name": "backend", "keywords": ["api", "server", "handlers"]},
			{"name": "general", "keywords": []}
		],
		"model_name": "sentence-transformers/all-MiniLM-L6-v2"
	}`), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.Rooms) != 3 {
		t.Fatalf("expected 3 rooms, got %d", len(cfg.Rooms))
	}
	if cfg.Rooms[0].Name != "frontend" {
		t.Errorf("Rooms[0].Name = %q, want %q", cfg.Rooms[0].Name, "frontend")
	}
	if len(cfg.Rooms[0].Keywords) != 3 {
		t.Errorf("Rooms[0].Keywords length = %d, want 3", len(cfg.Rooms[0].Keywords))
	}
}
