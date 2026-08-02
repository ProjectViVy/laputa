package pipeline

import (
	"crypto/sha256"
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	SchemaVersion string       `yaml:"schema_version" json:"schema_version"`
	Pipelines     []Definition `yaml:"pipelines" json:"pipelines"`
}

func DefaultConfig() Config {
	return Config{SchemaVersion: "1", Pipelines: []Definition{{
		Name: "agentic_recall_v1", Version: "1.0.0",
		Capabilities: []string{"governance", "llm", "hybrid", "kg", "timeline"}, MaxSteps: 20, MaxVisitsPerStep: 2,
		Steps: []StepConfig{
			{ID: "load_governance", TimeoutMS: 3000},
			{ID: "resolve_policy", TimeoutMS: 1000},
			{ID: "build_plan", TimeoutMS: 10000, OnError: "skip"},
			{ID: "retrieve_candidates", TimeoutMS: 5000, OnError: "skip"},
			{ID: "expand_graph", TimeoutMS: 5000, OnError: "skip"},
			{ID: "judge_sufficiency", TimeoutMS: 10000, OnError: "skip", Next: []string{"retrieve_candidates"}},
			{ID: "deduplicate_and_rerank", TimeoutMS: 1000},
			{ID: "governance_filter", TimeoutMS: 1000},
			{ID: "assemble_context", TimeoutMS: 10000, OnError: "skip"},
		},
	}}}
}

func LoadConfig(path string) (Config, string, error) {
	if path == "" {
		cfg := DefaultConfig()
		raw, _ := yaml.Marshal(cfg)
		return cfg, revision(raw), nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			encoded, _ := yaml.Marshal(cfg)
			return cfg, revision(encoded), nil
		}
		return Config{}, "", fmt.Errorf("read pipeline config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, "", fmt.Errorf("parse pipeline config: %w", err)
	}
	if cfg.SchemaVersion != "1" {
		return Config{}, "", fmt.Errorf("unsupported pipeline schema version %q", cfg.SchemaVersion)
	}
	return cfg, revision(raw), nil
}

func revision(raw []byte) string { sum := sha256.Sum256(raw); return fmt.Sprintf("%x", sum[:6]) }
