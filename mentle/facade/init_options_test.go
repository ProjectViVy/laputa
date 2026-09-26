package facade

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dashimaki/mentle/internal/config"
	"github.com/dashimaki/mentle/internal/embedder"
)

func TestInitRequireLocalModelRejectsMissingDespiteBundledCWD(t *testing.T) {
	_ = bundledModelsDir(t)
	palace := filepath.Join(t.TempDir(), "palace")
	missing := filepath.Join(t.TempDir(), "no-model")
	var svc Service
	err := svc.Init(context.Background(), Options{PalacePath: palace, ModelsDir: missing, RequireLocalModel: true})
	if !errors.Is(err, embedder.ErrLocalModelMissing) {
		t.Fatalf("strict init: got %v, want ErrLocalModelMissing", err)
	}
	if svc.Embedder != nil || svc.Catalog != nil {
		t.Fatalf("failed init retained resources: embedder=%p catalog=%p", svc.Embedder, svc.Catalog)
	}
	if _, statErr := os.Stat(palace); !os.IsNotExist(statErr) {
		t.Fatalf("strict init created palace before model validation: %v", statErr)
	}
}

func TestInitRequireLocalModelRejectsEmptyDirectory(t *testing.T) {
	var svc Service
	err := svc.Init(context.Background(), Options{PalacePath: filepath.Join(t.TempDir(), "palace"), RequireLocalModel: true})
	if !errors.Is(err, embedder.ErrLocalModelMissing) {
		t.Fatalf("strict init without explicit modelsDir: %v", err)
	}
}

func TestInitExplicitPalaceReopenCanonicalLifecycle(t *testing.T) {
	modelsDir := bundledModelsDir(t)
	palace := filepath.Join(t.TempDir(), "palace")
	opts := Options{PalacePath: palace, ModelsDir: modelsDir, RequireLocalModel: true}
	ctx := context.Background()
	var first Service
	if err := first.Init(ctx, opts); err != nil {
		t.Fatalf("first init: %v", err)
	}
	created, err := first.CreateMemory(ctx, CreateMemoryRequest{Content: "m1 original decision", Kind: "decision"}, "m1-create", "m1-create-hash")
	if err != nil {
		first.Close()
		t.Fatalf("create: %v", err)
	}
	updatedContent := "m1 revised decision"
	expectedVersion := created.Version
	updated, err := first.UpdateMemory(ctx, created.ID, UpdateMemoryRequest{Content: &updatedContent, ExpectedVersion: &expectedVersion})
	if err != nil {
		first.Close()
		t.Fatalf("update: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first: %v", err)
	}

	var reopened Service
	if err := reopened.Init(ctx, opts); err != nil {
		t.Fatalf("reopen: %v", err)
	}
	persisted, err := reopened.GetMemory(ctx, created.ID)
	if err != nil || persisted.Content != updatedContent || persisted.Version != updated.Version {
		reopened.Close()
		t.Fatalf("reopened canonical memory=%+v err=%v", persisted, err)
	}
	page, err := reopened.ListMemories(ctx, ListMemoryOptions{Limit: 10})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != created.ID {
		reopened.Close()
		t.Fatalf("reopened canonical list=%+v err=%v", page, err)
	}
	hits, err := reopened.Retrieve(ctx, RetrievalQuery{Text: updatedContent, Limit: 5})
	if err != nil || len(hits) != 1 || hits[0].ID != created.ID || hits[0].Content != updatedContent {
		reopened.Close()
		t.Fatalf("reopened retrieval=%+v err=%v", hits, err)
	}
	deleted, err := reopened.DeleteMemory(ctx, created.ID, persisted.Version, "user_request", "m1-delete")
	if err != nil || !deleted.Deleted {
		reopened.Close()
		t.Fatalf("delete after reopen=%+v err=%v", deleted, err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatalf("close reopened: %v", err)
	}

	var final Service
	if err := final.Init(ctx, opts); err != nil {
		t.Fatalf("final reopen: %v", err)
	}
	defer final.Close()
	if _, err := final.GetMemory(ctx, created.ID); !errors.Is(err, ErrMemoryNotFound) {
		t.Fatalf("tombstone not persisted across reopen: %v", err)
	}
	page, err = final.ListMemories(ctx, ListMemoryOptions{Limit: 10})
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("deleted memory returned after reopen: %+v err=%v", page, err)
	}
}

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
