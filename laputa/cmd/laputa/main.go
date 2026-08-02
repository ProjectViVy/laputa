// Deprecated: laputa.exe is replaced by garden (laputa + mentle facade).
// Kept as fallback binary for hermes plugin HTTP compatibility on :7373.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	laputa "github.com/dashimaki/laputa/governance"
	"github.com/dashimaki/laputa/governance/rhythm"
	"github.com/dashimaki/laputa/governance/scheduler"
	redisstore "github.com/dashimaki/laputa/governance/store"
	"github.com/dashimaki/laputa/governance/wakeup"
	"github.com/dashimaki/laputa/governance/web"
)

func main() {
	var (
		cmd           = flag.String("cmd", "rhythm", "laputa subcommand: rhythm|wakeup")
		dir           = flag.String("dir", ".laputa", "laputa data directory")
		storeKind     = flag.String("store", "file", "store backend: file|redis")
		redisAddr     = flag.String("redis-addr", "localhost:6379", "redis address (when store=redis)")
		redisDB       = flag.Int("redis-db", 0, "redis db number")
		redisPrefix   = flag.String("redis-prefix", "laputa:section:", "redis key prefix")
		kind          = flag.String("kind", "daily", "rhythm kind: daily|weekly|monthly")
		baseURL       = flag.String("base-url", "https://api.openai.com/v1", "LLM base URL")
		apiKey        = flag.String("api-key", os.Getenv("OPENAI_API_KEY"), "LLM API key")
		model         = flag.String("model", "gpt-4o-mini", "LLM model")
		wakeupAction  = flag.String("wakeup-action", "", "wakeup action: system-prompt|prefetch|sync-turn|session-end")
		wakeupIntent  = flag.String("wakeup-intent", "", "wakeup.prefetch intent")
		wakeupRoom    = flag.String("wakeup-room", "", "wakeup.prefetch room")
		wakeupSession = flag.String("wakeup-session", "", "wakeup.session_end session id")
		wakeupHistory = flag.String("wakeup-history", "", "wakeup.sync_turn history entry")
		daemonTick    = flag.Duration("daemon-tick", 60*time.Second, "daemon tick interval (when cmd=daemon)")
		daemonDryRun  = flag.Bool("daemon-dry-run", false, "daemon dry-run (when cmd=daemon)")
		daemonSession = flag.String("daemon-session", "laputa-daemon", "session id passed to wakeup.session_end on shutdown")
		serveAddr     = flag.String("serve-addr", "127.0.0.1:7373", "http listen address (when cmd=serve)")
	)
	flag.Parse()

	ctx := context.Background()
	store, err := openStore(ctx, *storeKind, *dir, *redisAddr, *redisDB, *redisPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "store: %v\n", err)
		os.Exit(1)
	}
	engine := laputa.NewEngine(store)
	if err := engine.Initialize(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "init: %v\n", err)
		os.Exit(1)
	}

	switch *cmd {
	case "rhythm":
		runRhythm(ctx, engine, *kind, *baseURL, *apiKey, *model)
	case "wakeup":
		runWakeup(ctx, engine, *wakeupAction, *wakeupIntent, *wakeupRoom, *wakeupHistory, *wakeupSession)
	case "daemon":
		runDaemon(ctx, engine, *daemonTick, *daemonDryRun, *daemonSession, *baseURL, *apiKey, *model)
	case "serve":
		runServe(ctx, engine, *serveAddr)
	default:
		fmt.Fprintf(os.Stderr, "unknown cmd: %s\n", *cmd)
		os.Exit(1)
	}
}

func openStore(ctx context.Context, kind, dir, addr string, db int, prefix string) (laputa.SectionStore, error) {
	switch kind {
	case "file":
		return laputa.NewFileStore(dir)
	case "redis":
		return redisstore.New(ctx, redisstore.Options{Addr: addr, DB: db, Prefix: prefix})
	default:
		return nil, fmt.Errorf("unknown store kind: %s", kind)
	}
}

func runRhythm(ctx context.Context, engine *laputa.Engine, kind, baseURL, apiKey, model string) {
	var gen rhythm.Generator = rhythm.NewMockGenerator()
	if apiKey != "" {
		var err error
		gen, err = rhythm.NewOpenAIGenerator(ctx, baseURL, apiKey, model)
		if err != nil {
			fmt.Fprintf(os.Stderr, "generator: %v\n", err)
			os.Exit(1)
		}
	}
	re := rhythm.NewEngine(engine, gen, rhythm.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
	})
	if err := re.Run(ctx, rhythm.RhythmKind(kind)); err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("rhythm report generated")
}

func runWakeup(ctx context.Context, engine *laputa.Engine, action, intent, room, history, session string) {
	provider := wakeup.NewEngine(engine)

	switch action {
	case "", "system-prompt":
		resp, err := provider.SystemPromptBlock(ctx, dirFromCwd())
		if err != nil {
			fmt.Fprintf(os.Stderr, "system_prompt_block: %v\n", err)
			os.Exit(1)
		}
		if resp.PromptBlock == nil {
			fmt.Printf("status=%s reason=%s\n", resp.Status, resp.Reason)
			return
		}
		fmt.Printf("status=%s\n%s\n", resp.Status, resp.PromptBlock.Markdown)
	case "prefetch":
		var roomPtr *string
		if room != "" {
			r := room
			roomPtr = &r
		}
		resp, err := provider.Prefetch(ctx, intent, roomPtr, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "prefetch: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("status=%s\n", resp.Status)
		if resp.PromptBlock != nil {
			fmt.Println(*resp.PromptBlock)
		}
	case "sync-turn":
		var histPtr *string
		if history != "" {
			h := history
			histPtr = &h
		}
		resp, err := provider.SyncTurn(ctx, nil, histPtr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sync_turn: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("status=%s\n", resp.Status)
	case "session-end":
		var sessPtr *string
		if session != "" {
			s := session
			sessPtr = &s
		}
		resp, err := provider.OnSessionEnd(ctx, sessPtr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "session_end: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("status=%s\n", resp.Status)
	default:
		fmt.Fprintf(os.Stderr, "unknown wakeup action: %s\n", action)
		os.Exit(1)
	}
}

func dirFromCwd() string {
	if d, err := os.Getwd(); err == nil {
		return d
	}
	return "."
}

func runDaemon(ctx context.Context, engine *laputa.Engine, tick time.Duration, dryRun bool, sessionID, baseURL, apiKey, model string) {
	var gen rhythm.Generator = rhythm.NewMockGenerator()
	if !dryRun && apiKey != "" {
		var err error
		gen, err = rhythm.NewOpenAIGenerator(ctx, baseURL, apiKey, model)
		if err != nil {
			fmt.Fprintf(os.Stderr, "generator: %v\n", err)
			os.Exit(1)
		}
	}
	d := scheduler.New(engine, scheduler.Config{
		TickEvery:     tick,
		DryRun:        dryRun,
		Generator:     gen,
		WorkspaceRoot: dirFromCwd(),
		SessionID:     sessionID,
		Logger:        log.New(os.Stderr, "[laputa] ", log.LstdFlags),
	})
	if err := d.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "daemon: %v\n", err)
	}
}

func runServe(ctx context.Context, engine *laputa.Engine, addr string) {
	srv, err := web.New(engine, addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "[laputa] web dashboard listening on http://%s\n", addr)
	if err := srv.ListenAndServe(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
	}
}
