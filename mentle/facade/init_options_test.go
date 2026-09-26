package facade

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
