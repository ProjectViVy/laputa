package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dashimaki/garden/internal/activity"
	"github.com/dashimaki/garden/internal/arbiter"
	"github.com/dashimaki/garden/internal/authority"
	gardencog "github.com/dashimaki/garden/internal/cognitive"
	"github.com/dashimaki/garden/internal/crud"
	"github.com/dashimaki/garden/internal/evolution"
	"github.com/dashimaki/garden/internal/ingest"
	"github.com/dashimaki/garden/internal/lifecycle"
	"github.com/dashimaki/garden/internal/pipeline"
	"github.com/dashimaki/garden/internal/rag"
	"github.com/dashimaki/garden/internal/recall"
	"github.com/dashimaki/garden/internal/report"
	"github.com/dashimaki/garden/internal/router"
	"github.com/dashimaki/garden/internal/server"
	"github.com/dashimaki/laputa/governance"
	"github.com/dashimaki/laputa/governance/cognitive"
	"github.com/dashimaki/laputa/governance/rhythm"
	"github.com/dashimaki/mentle/facade"
)

func main() {
	ctx := context.Background()

	storeDir := expandHome(os.Getenv("GARDEN_GOVERNANCE_DIR"))
	if storeDir == "" {
		storeDir = expandHome("~/.laputa/sections")
	}
	store, err := governance.NewFileStore(storeDir)
	if err != nil {
		log.Fatalf("governance store: %v", err)
	}

	gov := governance.NewEngine(store)
	if err := gov.Initialize(ctx); err != nil {
		log.Fatalf("governance init: %v", err)
	}

	auditLog, err := governance.NewFileAuditLogWithRotation(storeDir, auditRotationConfig())
	if err != nil {
		log.Fatalf("audit log: %v", err)
	}
	governed := governance.NewGovernedService(gov, auditLog)

	var mem *facade.Service
	memOpts := facade.Options{ConfigDir: expandHome(os.Getenv("GARDEN_MENTLE_CONFIG_DIR"))}
	memSvc := &facade.Service{}
	if err := memSvc.Init(ctx, memOpts); err != nil {
		log.Printf("mentle unavailable, governance-only mode: %v", err)
	} else {
		mem = memSvc
		defer mem.Close()
	}

	h := crud.NewHandler(gov, mem)
	h.Router.Governance.(*router.GovernanceBackend).Governed = governed
	components := map[string]string{}
	if mem == nil {
		components["mentle"] = "degraded"
	} else {
		components["mentle"] = "ok"
	}

	cognitiveDir := expandHome(os.Getenv("GARDEN_COGNITIVE_DIR"))
	if cognitiveDir == "" {
		cognitiveDir = expandHome("~/.laputa/cognitive")
	}
	components["cognitive"] = "ok"
	if err := cognitive.InitializeDir(cognitiveDir); err != nil {
		log.Printf("cognitive init: %v", err)
		components["cognitive"] = "degraded"
	}
	worldPath := filepath.Join(cognitiveDir, cognitive.WorldFileName)
	worldStore, worldErr := cognitive.LoadWorld(worldPath)
	if worldErr != nil {
		log.Printf("world load: %v", worldErr)
		worldStore = &cognitive.WorldStore{Path: worldPath}
		components["cognitive"] = "degraded"
	}
	rules, rulesErr := gardencog.Load(filepath.Join(cognitiveDir, cognitive.MemRulesFileName))
	if rulesErr != nil {
		log.Printf("memrules load: %v", rulesErr)
		components["cognitive"] = "degraded"
	} else if !rules.MatchesDefault() {
		_ = auditLog.Append(ctx, governance.AuditEntry{
			Section:     "memrules",
			Action:      "manual_edit",
			Actor:       "user",
			Reason:      "MEMRULES manual edit detected at reload",
			RollbackRef: rules.HashPrefix(),
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		})
	}

	fastRecall := &recall.FastService{Gov: gov, World: worldStore}
	if mem != nil {
		fastRecall.Searcher = mem
	}

	var manager *pipeline.Manager
	var resolver rag.Resolver
	pipelinePath := expandHome(os.Getenv("GARDEN_PIPELINE_CONFIG"))
	if pipelinePath == "" {
		pipelinePath = expandHome("~/.garden/pipelines.yaml")
	}
	pipelineCfg, revision, cfgErr := pipeline.LoadConfig(pipelinePath)
	if cfgErr != nil {
		log.Printf("pipeline unavailable: %v", cfgErr)
		components["pipeline"] = "degraded"
	} else {
		manager, err = pipeline.NewManager(pipelineCfg.Pipelines, revision)
		if err != nil {
			log.Printf("pipeline unavailable: %v", err)
			components["pipeline"] = "degraded"
		} else {
			components["pipeline"] = "ok"
			planner := configuredPlanner()
			if os.Getenv("GARDEN_RAG_API_KEY") == "" {
				components["planner"] = "degraded"
			} else {
				components["planner"] = "ok"
			}
			var retriever rag.Retriever
			if mem != nil {
				retriever = mem
			}
			ragService, ragErr := rag.NewService(manager, rag.PolicyResolver{Governance: gov}, retriever, planner)
			if ragErr != nil {
				log.Printf("agentic RAG unavailable: %v", ragErr)
				components["pipeline"] = "degraded"
			} else {
				resolver = ragService
			}
		}
	}
	stateDB := expandHome(os.Getenv("GARDEN_STATE_DB"))
	if stateDB == "" {
		stateDB = expandHome("~/.garden/garden.db")
	}
	if err := os.MkdirAll(filepath.Dir(stateDB), 0700); err != nil {
		log.Fatalf("state directory: %v", err)
	}
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
	var enricher report.Enricher
	if baseURL, apiKey, model := os.Getenv("GARDEN_RAG_BASE_URL"), os.Getenv("GARDEN_RAG_API_KEY"), os.Getenv("GARDEN_RAG_MODEL"); baseURL != "" && apiKey != "" && model != "" {
		if gen, genErr := rhythm.NewOpenAIGenerator(ctx, baseURL, apiKey, model); genErr != nil {
			log.Printf("report enricher unavailable: %v", genErr)
		} else {
			enricher = &report.LLMEnricher{Gen: gen}
		}
	}
	reports, err := report.Open(stateDB, memoryLister, &report.GovernedPublisher{Gov: governed}, enricher)
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

	ws := activity.NewWorkingSet()
	checkpointer := &activity.Checkpointer{Gov: gov, WS: ws}
	if err := checkpointer.Load(ctx, ""); err != nil {
		log.Printf("checkpoint load: %v", err)
	}
	fastRecall.WS = ws

	traceStore, err := recall.OpenTraceStore(stateDB)
	if err != nil {
		log.Fatalf("trace store: %v", err)
	}
	defer traceStore.Close()

	var graphSource recall.GraphSource
	if mem != nil {
		graphSource = mem
	}
	deepRecall := &recall.DeepService{
		Fast:    fastRecall,
		Graph:   graphSource,
		Planner: configuredPlanner(),
		Arbiter: arbiter.New(),
		Traces:  traceStore,
	}

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
	evoService := &evolution.Service{Provider: evoProvider, Store: evoStore, Events: evoEvents, Hub: evolution.DefaultHubPolicy()}

	addr := listenAddr()
	if !strings.HasPrefix(addr, "127.0.0.1:") && !strings.HasPrefix(addr, "localhost:") && !strings.HasPrefix(addr, "[::1]:") {
		log.Printf("HIGH RISK: unauthenticated Garden API is configured on non-loopback address %q", addr)
	}
	var materialsProvider server.MaterialsProvider
	if mem != nil {
		materialsProvider = mem
	}
	srv := &server.Server{Handler: h, Resolver: resolver, FastRecall: fastRecall, DeepRecall: deepRecall, TraceStore: traceStore, Evolution: evoService, Activity: activityStore, Checkpointer: checkpointer, Pipelines: manager, Ingestions: ingestions, Reports: reports, Governed: governed, GovernedWriter: &authority.GovernedWriter{Gov: governed}, Materials: materialsProvider, Cognitive: worldStore, Components: components, Addr: addr}
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

func auditRotationConfig() governance.RotationConfig {
	cfg := governance.DefaultRotationConfig()
	if v := os.Getenv("GARDEN_AUDIT_MAX_AGE_DAYS"); v != "" {
		if days, err := strconv.Atoi(v); err == nil && days > 0 {
			cfg.MaxAgeDays = days
		} else {
			log.Printf("ignoring invalid GARDEN_AUDIT_MAX_AGE_DAYS %q", v)
		}
	}
	if v := os.Getenv("GARDEN_AUDIT_MAX_SIZE_MB"); v != "" {
		if mb, err := strconv.Atoi(v); err == nil && mb > 0 {
			cfg.MaxSizeBytes = int64(mb) << 20
		} else {
			log.Printf("ignoring invalid GARDEN_AUDIT_MAX_SIZE_MB %q", v)
		}
	}
	return cfg
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
