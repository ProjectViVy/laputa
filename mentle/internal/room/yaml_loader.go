package room

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

type yamlConfig struct {
	ProjectName string       `yaml:"project_name"`
	Rooms       []RoomConfig `yaml:"rooms"`
}

type RoomConfig struct {
	Name     string   `yaml:"name" json:"name"`
	Keywords []string `yaml:"keywords" json:"keywords,omitempty"`
}

func LoadRoomsFromYAML(path string) ([]RoomConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg yamlConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config %s: %w", path, err)
	}

	return cfg.Rooms, nil
}

func FindConfigFile(projectDir string) (string, bool) {
	candidates := []string{
		filepath.Join(projectDir, "mempalace.yaml"),
		filepath.Join(projectDir, "mempal.yaml"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}

	return "", false
}

func SaveRoomsToYAML(path string, projectName string, rooms []RoomConfig) error {
	cfg := yamlConfig{
		ProjectName: projectName,
		Rooms:       rooms,
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", path, err)
	}

	return nil
}
