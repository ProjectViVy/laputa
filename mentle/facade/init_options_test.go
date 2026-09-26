package facade

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dashimaki/mentle/internal/config"
)

func bundledModelsDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "models", "onnx"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "model.onnx")); err != nil {
		t.Skipf("local ONNX model unavailable: %v", err)
	}
	return dir
}

func writeInitConfig(t *testing.T, palacePath, modelsDir string) string {
	t.Helper()
	dir := t.TempDir()
	config := `{"palace_path":` + quotedPath(palacePath) + `,"models_dir":` + quotedPath(modelsDir) + `}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func quotedPath(path string) string {
	// JSON escaping is important for native Windows paths.
	b, _ := json.Marshal(path)
	return string(b)
}

func TestInitExplicitPathsIgnoreAmbientConfigAndEnv(t *testing.T) {
	modelsDir := bundledModelsDir(t)
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(filepath.Join(home, ".mempalace"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".mempalace", "config.json"), []byte(`{"broken":`), 0600); err != nil {
		t.Fatal(err)
	}
	// Go's os.UserHomeDir uses USERPROFILE rather than HOME on Windows.
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("MEMPALACE_COLLECTION_NAME", "ambient-collection")
	t.Setenv("MEMPALACE_MODEL_NAME", "ambient-model")
	palace := filepath.Join(root, "embedded")
	var svc Service
	if err := svc.Init(context.Background(), Options{PalacePath: palace, ModelsDir: modelsDir}); err != nil {
		t.Fatalf("init with explicit paths despite malformed ambient config: %v", err)
	}
	defer svc.Close()
	if svc.PalacePath != palace || svc.Cfg.PalacePath != palace || svc.Cfg.ModelsDir != modelsDir {
		t.Fatalf("effective paths: palace=%q config=%+v", svc.PalacePath, svc.Cfg)
	}
	if svc.Cfg.CollectionName != config.DefaultConfig.CollectionName || svc.Cfg.ModelName != config.DefaultConfig.ModelName || !reflect.DeepEqual(svc.Cfg.TopicWings, config.DefaultConfig.TopicWings) {
		t.Fatalf("ambient environment leaked or defaults missing: config=%+v", svc.Cfg)
	}
	if _, err := os.Stat(filepath.Join(palace, "canonical.sqlite3")); err != nil {
		t.Fatalf("canonical store not under explicit palace: %v", err)
	}
}

func TestInitExplicitPathsOverrideLoadedConfig(t *testing.T) {
	modelDir := bundledModelsDir(t)
	root := t.TempDir()
	legacyPalace := filepath.Join(root, "legacy")
	palace := filepath.Join(root, "embedded")
	// A local invalid ONNX file makes ignoring ModelsDir fail without downloading.
	legacyModels := filepath.Join(root, "invalid-model")
	if err := os.MkdirAll(legacyModels, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyModels, "model.onnx"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	configDir := writeInitConfig(t, legacyPalace, legacyModels)
	var svc Service
	if err := svc.Init(context.Background(), Options{ConfigDir: configDir, PalacePath: palace, ModelsDir: modelDir}); err != nil {
		t.Fatalf("init with explicit overrides: %v", err)
	}
	defer svc.Close()
	if svc.PalacePath != palace || svc.Cfg.PalacePath != palace || svc.Cfg.ModelsDir != modelDir {
		t.Fatalf("effective paths: palace=%q config=%+v", svc.PalacePath, svc.Cfg)
	}
	if _, err := os.Stat(filepath.Join(palace, "canonical.sqlite3")); err != nil {
		t.Fatalf("canonical store not under explicit palace: %v", err)
	}
	if _, err := os.Stat(legacyPalace); !os.IsNotExist(err) {
		t.Fatalf("legacy palace unexpectedly used: %v", err)
	}
}

func TestInitPartialOverridesKeepConfiguredValues(t *testing.T) {
	modelsDir := bundledModelsDir(t)
	root := t.TempDir()
	configuredPalace := filepath.Join(root, "configured")
	configuredModels := filepath.Join(root, "missing-models")
	for _, tc := range []struct {
		name       string
		opts       Options
		wantPalace string
		wantModels string
	}{
		{"palace only", Options{ConfigDir: writeInitConfig(t, configuredPalace, modelsDir), PalacePath: filepath.Join(root, "override")}, filepath.Join(root, "override"), modelsDir},
		{"models only", Options{ConfigDir: writeInitConfig(t, configuredPalace, configuredModels), ModelsDir: modelsDir}, configuredPalace, modelsDir},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var svc Service
			if err := svc.Init(context.Background(), tc.opts); err != nil {
				t.Fatalf("init with partial override: %v", err)
			}
			defer svc.Close()
			if svc.PalacePath != tc.wantPalace || svc.Cfg.ModelsDir != tc.wantModels {
				t.Fatalf("effective paths: palace=%q models=%q", svc.PalacePath, svc.Cfg.ModelsDir)
			}
		})
	}
}

func TestInitConfigDirStillSuppliesPathsWithoutOverrides(t *testing.T) {
	modelsDir := bundledModelsDir(t)
	palace := filepath.Join(t.TempDir(), "configured")
	configDir := writeInitConfig(t, palace, modelsDir)
	var svc Service
	if err := svc.Init(context.Background(), Options{ConfigDir: configDir}); err != nil {
		t.Fatalf("init with legacy config: %v", err)
	}
	defer svc.Close()
	if svc.PalacePath != palace || svc.Cfg.ModelsDir != modelsDir {
		t.Fatalf("legacy paths: palace=%q models=%q", svc.PalacePath, svc.Cfg.ModelsDir)
	}
}
