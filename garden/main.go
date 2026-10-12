package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProjectViVy/laputa/garden/agentapi"
	"github.com/ProjectViVy/laputa/garden/internal/lifecycle"
	"github.com/ProjectViVy/laputa/garden/internal/runtimecore"
	"github.com/ProjectViVy/laputa/garden/internal/server"
	"github.com/ProjectViVy/laputa/mentle/facade"
)

func main() {
	ctx := context.Background()
	personaDir := expandHome(os.Getenv("GARDEN_PERSONA_DIR"))
	stateDB := expandHome(os.Getenv("GARDEN_STATE_DB"))
	if stateDB == "" {
		stateDB = expandHome("~/.garden/garden.db")
	}
	palacePath, modelsDir := filepath.Join(filepath.Dir(legacyAbsolutePath(stateDB)), "memory"), filepath.Join(filepath.Dir(legacyAbsolutePath(stateDB)), "models")
	var err error
	if configDir := expandHome(os.Getenv("GARDEN_MENTLE_CONFIG_DIR")); configDir != "" {
		palacePath, modelsDir, err = facade.ResolveLegacyPaths(configDir)
	}
	if err != nil {
		log.Fatalf("mentle configuration: %v", err)
	}
	profileID := os.Getenv("GARDEN_AGENT_PROFILE_ID")
	if profileID == "" {
		profileID = "default"
	}
	app, err := openApp(ctx, runtimeConfig(personaDir, palacePath, modelsDir, stateDB, profileID))
	if err != nil {
		log.Fatalf("garden: %v", err)
	}
	defer app.Close()
	if err := lifecycle.Run(ctx, app.server); err != nil {
		log.Fatalf("lifecycle: %v", err)
	}
}

// runtimeConfig makes the deployment profile an immutable binding, never a request selector.
// The monolith alone resolves legacy relative paths against its working directory;
// embedders must pass explicit absolute paths directly to agentapi.Open.
func runtimeConfig(personaDir, palacePath, modelsDir, stateDB, profileID string) runtimecore.Config {
	return runtimecore.Config{
		PersonaDir: legacyAbsolutePath(personaDir), PalacePath: legacyAbsolutePath(palacePath),
		ModelsDir: legacyAbsolutePath(modelsDir), StateDB: legacyAbsolutePath(stateDB), ProfileID: profileID,
	}
}

func legacyAbsolutePath(path string) string {
	if path == "" {
		return ""
	}
	path = expandHome(path)
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return path // runtimecore validation fails closed rather than choosing another directory.
	}
	return absolute
}

type gardenApp struct {
	owner        *runtimecore.Owner
	installation *agentapi.Installation
	server       *server.Server
}

func (a *gardenApp) Close() error {
	if a == nil || a.installation == nil {
		return nil
	}
	return a.installation.Close()
}

// The standalone process owns one installation. Legacy embedding remains in
// agentapi.Open; no app-level raw Mentle/report/autonomy source runs beside it.
func openApp(ctx context.Context, cfg runtimecore.Config) (a *gardenApp, err error) {
	home := filepath.Dir(cfg.StateDB)
	owner, err := runtimecore.OpenOwner(ctx, runtimecore.OwnerConfig{Home: home, Config: cfg})
	if err != nil {
		return nil, err
	}
	i := agentapi.NewInstallation(owner)
	a = &gardenApp{owner: owner, installation: i}
	defer func() {
		if err != nil {
			a.Close()
		}
	}()
	// Attaching an existing authority requires an explicitly configured root.
	// No default ~/.laputa discovery or captured-data migration accompanies boot.
	if cfg.PersonaDir != "" {
		if _, err = i.RegisterProfile(ctx, agentapi.ProfileRegistration{ProfileID: cfg.ProfileID, AuthorityDir: cfg.PersonaDir}); err != nil {
			return nil, err
		}
	}
	pairing, code, err := server.NewPairing()
	if err != nil {
		return nil, err
	}
	if err = writePairingCode(home, code); err != nil {
		return nil, err
	}
	a.server = &server.Server{Owner: owner, Installation: i, Pairing: pairing, Components: map[string]string{"garden": "ok", "persona": "ok"}, Addr: listenAddr()}
	return a, nil
}
func writePairingCode(home, code string) error {
	f, err := os.CreateTemp(home, ".pairing-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.WriteString(code + "\n")
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, filepath.Join(home, "pairing-code"))
}

func listenAddr() string {
	if addr := os.Getenv("GARDEN_ADDR"); addr != "" {
		return addr
	}
	return "127.0.0.1:7373"
}

func expandHome(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	return path
}
