// Package config provides configuration management for mempalace-go.
// It loads settings from config files and environment variables using Viper.
package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type RoomConfig struct {
	Name     string   `mapstructure:"name"`
	Keywords []string `mapstructure:"keywords"`
}

type Config struct {
	PalacePath     string              `mapstructure:"palace_path"`
	CollectionName string              `mapstructure:"collection_name"`
	PeopleMap      map[string]string   `mapstructure:"people_map"`
	TopicWings     []string            `mapstructure:"topic_wings"`
	HallKeywords   map[string][]string `mapstructure:"hall_keywords"`
	ModelName      string              `mapstructure:"model_name"`
	ModelsDir      string              `mapstructure:"models_dir"`
	Rooms          []RoomConfig        `mapstructure:"rooms"`
}

var DefaultConfig = Config{
	PalacePath:     "~/.mempalace/palace",
	CollectionName: "mempalace_drawers",
	TopicWings:     []string{"emotions", "consciousness", "memory", "technical", "identity", "family", "creative"},
	ModelName:      "sentence-transformers/all-MiniLM-L6-v2",
}

func Load(configDir string) (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("json")

	if configDir != "" {
		v.AddConfigPath(configDir)
	} else {
		home, _ := os.UserHomeDir()
		v.AddConfigPath(filepath.Join(home, ".mempalace"))
	}

	v.SetEnvPrefix("MEMPALACE")
	v.AutomaticEnv()

	v.SetDefault("palace_path", DefaultConfig.PalacePath)
	v.SetDefault("collection_name", DefaultConfig.CollectionName)
	v.SetDefault("topic_wings", DefaultConfig.TopicWings)
	v.SetDefault("model_name", DefaultConfig.ModelName)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) GetModelsDir() string {
	if c.ModelsDir != "" {
		return c.ModelsDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mempalace", "models")
}

func (c *Config) GetIdentityPath() (string, error) {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mempalace", "identity.txt"), nil
}
