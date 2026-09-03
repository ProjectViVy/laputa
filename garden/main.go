package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dashimaki/garden/internal/activity"
	"github.com/dashimaki/garden/internal/arbiter"
	"github.com/dashimaki/garden/internal/evolution"
	"github.com/dashimaki/garden/internal/ingest"
	"github.com/dashimaki/garden/internal/lifecycle"
	"github.com/dashimaki/garden/internal/mailbox"
	"github.com/dashimaki/garden/internal/personactx"
	"github.com/dashimaki/garden/internal/pipeline"
	"github.com/dashimaki/garden/internal/rag"
	"github.com/dashimaki/garden/internal/recall"
	"github.com/dashimaki/garden/internal/report"
	"github.com/dashimaki/garden/internal/server"
	"github.com/dashimaki/laputa/actmem"
	"github.com/dashimaki/laputa/persona"
	"github.com/dashimaki/mentle/facade"
)

func main() {
	ctx := context.Background()

	personaDir := expandHome(os.Getenv("GARDEN_PERSONA_DIR"))
	if personaDir == "" {
		personaDir = expandHome("~/.laputa")
	}
	personaService, err := persona.Open(personaDir)
	if err != nil {
		log.Fatalf("persona store: %v", err)
	}
	actmemStore := actmem.New(personaDir)

	var mem *facade.Service
	memSvc := &facade.Service{}
	if err := memSvc.Init(ctx, facade.Options{ConfigDir: expandHome(os.Getenv("GARDEN_MENTLE_CONFIG_DIR"))}); err != nil {
		log.Printf("mentle unavailable, frozen-core-only mode: %v", err)
	} else {
		mem = memSvc
		defer mem.Close()
	}
	components := map[string]string{"persona": "ok"}
	if mem == nil {
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
	} else {
		components["pipeline"] = "ok"
	}
	if os.Getenv("GARDEN_RAG_API_KEY") == "" {
		components["planner"] = "degraded"
	} else {
		components["planner"] = "ok"
	}

	stateDB := expandHome(os.Getenv("GARDEN_STATE_DB"))
	if stateDB == "" {
		stateDB = expandHome("~/.garden/garden.db")
	}
	if err := os.MkdirAll(filepath.Dir(stateDB), 0700); err != nil {
		log.Fatalf("state directory: %v", err)
	}

	frozenStore, err := personactx.OpenStore(stateDB)
	if err != nil {
		log.Fatalf("frozen core store: %v", err)
	}
	defer frozenStore.Close()
	fastRecall := &recall.FastService{Frozen: &personactx.SessionProvider{Store: frozenStore, Reader: personaService}}
	if mem != nil {
		fastRecall.Searcher = mem
	}

	checkpointStore, err := activity.OpenCheckpointStore(stateDB)
	if err != nil {
		log.Fatalf("checkpoint store: %v", err)
	}
	defer checkpointStore.Close()
	ws := activity.NewWorkingSet()
	checkpointer := &activity.Checkpointer{Store: checkpointStore, WS: ws}
	if err := checkpointer.Load(ctx, ""); err != nil {
		log.Printf("checkpoint load: %v", err)
	}
	fastRecall.WS = ws

	var memoryWriter ingest.MemoryWriter
	var memoryLister report.MemoryLister
	if mem != nil {
		memoryWriter = mem
		memoryLister = mem
	}
	ingestions, err := ingest.Open(stateDB, memoryWriter)
	if err != nil {
		log.Fatalf("ingestion store: %v", err)
	}
	defer ingestions.Close()
	reports, err := report.Open(stateDB, memoryLister, nil, nil)
	if err != nil {
		log.Fatalf("report store: %v", err)
	}
	defer reports.Close()

	activityStore, err := activity.OpenStore(stateDB)
	if err != nil {
		log.Fatalf("activity store: %v", err)
	}
	defer activityStore.Close()
	ingestions.Activity = activityStore
	spool, err := activity.OpenSpool(stateDB)
	if err != nil {
		log.Fatalf("transient spool: %v", err)
	}
	defer spool.Close()
	ingestions.Spool = spool
	if mem != nil {
		if drained, drainErr := ingestions.DrainSpool(ctx); drainErr != nil {
			log.Printf("spool drain: %v", drainErr)
		} else if drained > 0 {
			log.Printf("drained %d spooled events to mentle", drained)
		}
	}

	traceStore, err := recall.OpenTraceStore(stateDB)
	if err != nil {
		log.Fatalf("trace store: %v", err)
	}
	defer traceStore.Close()
	var graphSource recall.GraphSource
	if mem != nil {
		graphSource = mem
	}
	deepRecall := &recall.DeepService{Fast: fastRecall, Graph: graphSource, Planner: configuredPlanner(), Arbiter: arbiter.New(), Traces: traceStore}

	evoStore, err := evolution.OpenStore(stateDB)
	if err != nil {
		log.Fatalf("evolution store: %v", err)
	}
	defer evoStore.Close()
	evoEvents, err := evolution.OpenEventStore(stateDB)
	if err != nil {
		log.Fatalf("evolution events: %v", err)
	}
	defer evoEvents.Close()
	var evoProvider evolution.EvolverProvider
	components["evolution"] = "degraded"
	if hub, hubErr := evolution.OpenHubClient(evolution.HubClientOptions{BaseURL: os.Getenv("GARDEN_EVOMAP_HUB_URL"), CredsPath: expandHome(os.Getenv("GARDEN_EVOMAP_CREDS"))}); hubErr != nil {
		log.Printf("evomap unavailable: %v", hubErr)
	} else if hub.HasCredentials() {
		evoProvider = evolution.NewEvoMapProvider(hub, evoStore, evolution.DefaultProviderLimits(), os.Getenv("GARDEN_EVOMAP_HUB_PUBLISH") == "1")
		components["evolution"] = "ok"
	}
	evoService := &evolution.Service{Provider: evoProvider, Store: evoStore, Events: evoEvents, Hub: evolution.DefaultHubPolicy()}

	mailboxStore, err := mailbox.OpenStore(stateDB)
	if err != nil {
		log.Fatalf("mailbox store: %v", err)
	}
	defer mailboxStore.Close()

	addr := listenAddr()
	if !strings.HasPrefix(addr, "127.0.0.1:") && !strings.HasPrefix(addr, "localhost:") && !strings.HasPrefix(addr, "[::1]:") {
		log.Printf("HIGH RISK: Garden API is configured on non-loopback address %q; capability authentication is mandatory and loopback read exemptions do not apply", addr)
	}
	var materialsProvider server.MaterialsProvider
	if mem != nil {
		materialsProvider = mem
	}
	srv := &server.Server{Facade: mem, FastRecall: fastRecall, DeepRecall: deepRecall, TraceStore: traceStore, Evolution: evoService, Activity: activityStore, Checkpointer: checkpointer, Pipelines: manager, Ingestions: ingestions, Reports: reports, Materials: materialsProvider, Mailbox: mailboxStore, Persona: personaService, Actmem: actmemStore, Components: components, Addr: addr}
	if err := lifecycle.Run(ctx, srv); err != nil {
		log.Fatalf("lifecycle: %v", err)
	}
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
