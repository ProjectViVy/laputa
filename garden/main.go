package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dashimaki/garden/agentapi"
	"github.com/dashimaki/garden/internal/activity"
	"github.com/dashimaki/garden/internal/arbiter"
	"github.com/dashimaki/garden/internal/evolution"
	"github.com/dashimaki/garden/internal/lifecycle"
	"github.com/dashimaki/garden/internal/mailbox"
	"github.com/dashimaki/garden/internal/pipeline"
	"github.com/dashimaki/garden/internal/rag"
	"github.com/dashimaki/garden/internal/recall"
	"github.com/dashimaki/garden/internal/report"
	"github.com/dashimaki/garden/internal/runtimecore"
	"github.com/dashimaki/garden/internal/server"
	"github.com/dashimaki/mentle/facade"
)

func main() {
	ctx := context.Background()
	personaDir := expandHome(os.Getenv("GARDEN_PERSONA_DIR"))
	if personaDir == "" {
		personaDir = expandHome("~/.laputa")
	}
	stateDB := expandHome(os.Getenv("GARDEN_STATE_DB"))
	if stateDB == "" {
		stateDB = expandHome("~/.garden/garden.db")
	}
	palacePath, modelsDir, err := facade.ResolveLegacyPaths(expandHome(os.Getenv("GARDEN_MENTLE_CONFIG_DIR")))
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
	runtime    *runtimecore.Garden
	server     *server.Server
	closeExtra []func() error
}

func (a *gardenApp) Close() error {
	if a == nil {
		return nil
	}
	var first error
	for i := len(a.closeExtra) - 1; i >= 0; i-- {
		if err := a.closeExtra[i](); err != nil && first == nil {
			first = err
		}
	}
	a.closeExtra = nil
	if a.runtime != nil {
		if err := a.runtime.Close(); err != nil && first == nil {
			first = err
		}
		a.runtime = nil
	}
	return first
}

// openApp opens the domain once; the HTTP and agent adapters share its owned services.
// Pipeline, report, checkpoint, EvoMap and mailbox remain app-level management.
func openApp(ctx context.Context, cfg runtimecore.Config) (_ *gardenApp, err error) {
	core, err := runtimecore.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}
	a := &gardenApp{runtime: core}
	defer func() {
		if err != nil {
			_ = a.Close()
		}
	}()
	components := map[string]string{"persona": "ok"}
	if core.Mentle == nil || core.LexicalOnly {
		components["mentle"] = "degraded"
	} else {
		components["mentle"] = "ok"
	}
	pipelinePath := expandHome(os.Getenv("GARDEN_PIPELINE_CONFIG"))
	if pipelinePath == "" {
		pipelinePath = expandHome("~/.garden/pipelines.yaml")
	}
	var manager *pipeline.Manager
	if pipelineCfg, revision, cfgErr := pipeline.LoadConfig(pipelinePath); cfgErr != nil {
		log.Printf("pipeline unavailable: %v", cfgErr)
		components["pipeline"] = "degraded"
	} else if manager, err = pipeline.NewManager(pipelineCfg.Pipelines, revision); err != nil {
		log.Printf("pipeline unavailable: %v", err)
		components["pipeline"] = "degraded"
		err = nil
	} else {
		components["pipeline"] = "ok"
	}
	if os.Getenv("GARDEN_RAG_API_KEY") == "" {
		components["planner"] = "degraded"
	} else {
		components["planner"] = "ok"
	}
	checkpointStore, err := activity.OpenCheckpointStore(cfg.StateDB)
	if err != nil {
		return nil, fmt.Errorf("checkpoint store: %w", err)
	}
	a.closeExtra = append(a.closeExtra, checkpointStore.Close)
	ws := activity.NewWorkingSet()
	checkpointer := &activity.Checkpointer{Store: checkpointStore, WS: ws}
	if err := checkpointer.Load(ctx, ""); err != nil {
		log.Printf("checkpoint load: %v", err)
	}
	core.FastRecall.WS = ws
	var memoryLister report.MemoryLister
	if core.Mentle != nil {
		memoryLister = core.Mentle
	}
	reports, err := report.Open(cfg.StateDB, memoryLister, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("report store: %w", err)
	}
	a.closeExtra = append(a.closeExtra, reports.Close)
	var graphSource recall.GraphSource
	if core.Mentle != nil && !core.LexicalOnly {
		graphSource = core.Mentle
	}
	deepRecall := &recall.DeepService{Fast: core.FastRecall, Graph: graphSource, Planner: configuredPlanner(), Arbiter: arbiter.New(), Traces: core.Trace}
	evoStore, err := evolution.OpenStore(cfg.StateDB)
	if err != nil {
		return nil, fmt.Errorf("evolution store: %w", err)
	}
	a.closeExtra = append(a.closeExtra, evoStore.Close)
	evoEvents, err := evolution.OpenEventStore(cfg.StateDB)
	if err != nil {
		return nil, fmt.Errorf("evolution events: %w", err)
	}
	a.closeExtra = append(a.closeExtra, evoEvents.Close)
	var evoProvider evolution.EvolverProvider
	components["evolution"] = "degraded"
	if hub, hubErr := evolution.OpenHubClient(evolution.HubClientOptions{BaseURL: os.Getenv("GARDEN_EVOMAP_HUB_URL"), CredsPath: expandHome(os.Getenv("GARDEN_EVOMAP_CREDS"))}); hubErr != nil {
		log.Printf("evomap unavailable: %v", hubErr)
	} else if hub.HasCredentials() {
		evoProvider = evolution.NewEvoMapProvider(hub, evoStore, evolution.DefaultProviderLimits(), os.Getenv("GARDEN_EVOMAP_HUB_PUBLISH") == "1")
		components["evolution"] = "ok"
	}
	evoService := &evolution.Service{Provider: evoProvider, Store: evoStore, Events: evoEvents, Hub: evolution.DefaultHubPolicy()}
	mailboxStore, err := mailbox.OpenStore(cfg.StateDB)
	if err != nil {
		return nil, fmt.Errorf("mailbox store: %w", err)
	}
	a.closeExtra = append(a.closeExtra, mailboxStore.Close)
	addr := listenAddr()
	if !strings.HasPrefix(addr, "127.0.0.1:") && !strings.HasPrefix(addr, "localhost:") && !strings.HasPrefix(addr, "[::1]:") {
		log.Printf("HIGH RISK: Garden API is configured on non-loopback address %q; capability authentication is mandatory and loopback read exemptions do not apply", addr)
	}
	a.server = &server.Server{ProfileID: core.ProfileID, AgentAPI: agentapi.NewService(core), Facade: core.Mentle, FastRecall: core.FastRecall, DeepRecall: deepRecall, TraceStore: core.Trace, Evolution: evoService, Activity: core.Activity, Checkpointer: checkpointer, Pipelines: manager, Ingestions: core.Ingest, Reports: reports, Mailbox: mailboxStore, Persona: core.Persona, Actmem: core.Actmem, Components: components, Addr: addr}
	return a, nil
}

func configuredPlanner() rag.Planner {
	baseURL := os.Getenv("GARDEN_RAG_BASE_URL")
	apiKey := os.Getenv("GARDEN_RAG_API_KEY")
	model := os.Getenv("GARDEN_RAG_MODEL")
	if baseURL == "" || apiKey == "" || model == "" {
		return rag.RulePlanner{}
	}
	return rag.FallbackPlanner{Primary: &rag.OpenAIPlanner{BaseURL: baseURL, APIKey: apiKey, Model: model}, Fallback: rag.RulePlanner{}}
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
