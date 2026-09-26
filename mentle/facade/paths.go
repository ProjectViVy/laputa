package facade

import (
	"fmt"

	"github.com/dashimaki/mentle/internal/config"
)

// ResolveLegacyPaths resolves the palace and model paths using the legacy
// config.json and MEMPALACE environment settings. It does not initialize a service.
func ResolveLegacyPaths(configDir string) (palacePath, modelsDir string, err error) {
	cfg, err := config.Load(configDir)
	if err != nil {
		return "", "", fmt.Errorf("config: %w", err)
	}
	return expandPalacePath(cfg.PalacePath), cfg.GetModelsDir(), nil
}
