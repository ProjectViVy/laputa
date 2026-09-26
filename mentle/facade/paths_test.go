package facade

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLegacyPathsReadsConfigDirectory(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	if err := os.Mkdir(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	palace := filepath.Join(root, "palace")
	models := filepath.Join(root, "models")
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"palace_path":`+quotedPath(palace)+`,"models_dir":`+quotedPath(models)+`}`), 0600); err != nil {
		t.Fatal(err)
	}

	gotPalace, gotModels, err := ResolveLegacyPaths(configDir)
	if err != nil || gotPalace != palace || gotModels != models {
		t.Fatalf("ResolveLegacyPaths() = (%q, %q, %v), want (%q, %q, nil)", gotPalace, gotModels, err, palace, models)
	}
	if _, err := os.Stat(palace); !os.IsNotExist(err) {
		t.Fatalf("resolver created palace directory: %v", err)
	}
}

func TestResolveLegacyPathsHonorsEnvironmentAndExpandsPalace(t *testing.T) {
	root := t.TempDir()
	configDir := writeInitConfig(t, filepath.Join(root, "configured-palace"), filepath.Join(root, "configured-models"))
	t.Setenv("LEGACY_TEST_ROOT", root)
	t.Setenv("MEMPALACE_PALACE_PATH", "$LEGACY_TEST_ROOT/from-env")
	t.Setenv("MEMPALACE_MODELS_DIR", filepath.Join(root, "environment-models"))

	palace, models, err := ResolveLegacyPaths(configDir)
	if err != nil || filepath.Clean(palace) != filepath.Join(root, "from-env") || models != filepath.Join(root, "environment-models") {
		t.Fatalf("environment paths = (%q, %q, %v)", palace, models, err)
	}
}

func TestResolveLegacyPathsUsesDefaultModelsDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("MEMPALACE_PALACE_PATH", "")
	t.Setenv("MEMPALACE_MODELS_DIR", "")

	palace, models, err := ResolveLegacyPaths("")
	if err != nil || filepath.Clean(palace) != filepath.Join(home, ".mempalace", "palace") || models != filepath.Join(home, ".mempalace", "models") {
		t.Fatalf("default paths = (%q, %q, %v)", palace, models, err)
	}
}

func TestResolveLegacyPathsReportsMalformedConfig(t *testing.T) {
	configDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"palace_path":`), 0600); err != nil {
		t.Fatal(err)
	}
	palace, models, err := ResolveLegacyPaths(configDir)
	if err == nil || palace != "" || models != "" {
		t.Fatalf("malformed config = (%q, %q, %v), want empty paths and error", palace, models, err)
	}
}
