package runtimecore

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dashimaki/garden/internal/ingest"
	"github.com/dashimaki/garden/internal/recall"
	"github.com/dashimaki/laputa/persona"
)

func testConfig(t *testing.T) Config {
	t.Helper()
	root := t.TempDir()
	return Config{PersonaDir: filepath.Join(root, "persona"), PalacePath: filepath.Join(root, "palace"), ModelsDir: filepath.Join(root, "models"), StateDB: filepath.Join(root, "state", "garden.db"), ProfileID: "test"}
}

func TestOpenDegradedUsesOnlyExplicitPaths(t *testing.T) {
	cfg := testConfig(t)
	t.Setenv("HOME", filepath.Join(t.TempDir(), "forbidden"))
	t.Setenv("GARDEN_STATE_DB", filepath.Join(t.TempDir(), "ambient.db"))
	core, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("open with config %+v: %v", cfg, err)
	}
	defer core.Close()
	if core.Persona == nil || core.Actmem == nil || core.Frozen == nil || core.FastRecall == nil || core.Ingest == nil || core.Trace == nil {
		t.Fatal("missing runtime component")
	}
	if core.Mentle != nil || core.FastRecall.Searcher != nil {
		t.Fatal("missing model must degrade without substituting a searcher")
	}
	if _, err := os.Stat(cfg.StateDB); err != nil {
		t.Fatalf("state DB not at explicit path: %v", err)
	}
	if _, err := os.Stat(cfg.PalacePath); !os.IsNotExist(err) {
		t.Fatalf("unexpected palace creation: %v", err)
	}
	view, err := core.FastRecall.Recall(context.Background(), recall.FastRequest{Query: "test"})
	if err != nil || !view.Degraded || !strings.Contains(strings.Join(view.Warnings, " "), "mentle unavailable") {
		t.Fatalf("degraded recall: %+v, %v", view, err)
	}
}

func TestCloseReopen(t *testing.T) {
	cfg := testConfig(t)
	core, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := core.Close(); err != nil {
		t.Fatal(err)
	}
	if err := core.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
	again, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if err := again.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBootstrapFrozenAcrossReopenExcludesToolOnlyAuthority(t *testing.T) {
	cfg := testConfig(t)
	core, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := core.Persona.Initialize(persona.Initialization{Identity: "identity sentinel", Relationship: "relationship", Redline: "redline", User: "user", World: "world private sentinel"}, "user", persona.SourceInit, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := core.Actmem.AppendPulse("session-1", "activity private sentinel"); err != nil {
		t.Fatal(err)
	}
	first, err := core.Frozen.Capture(context.Background(), "session-1", core.Persona)
	if err != nil {
		t.Fatal(err)
	}
	if err := core.Close(); err != nil {
		t.Fatal(err)
	}
	core, err = Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer core.Close()
	second, err := core.Frozen.Capture(context.Background(), "session-1", core.Persona)
	if err != nil {
		t.Fatal(err)
	}
	if first.Content(0) != second.Content(0) {
		t.Fatal("frozen session drifted after reopen")
	}
	view, err := core.FastRecall.Recall(context.Background(), recall.FastRequest{Query: "identity", SessionID: "session-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(view.Context, "identity sentinel") || strings.Contains(view.Context, "world private sentinel") || strings.Contains(view.Context, "activity private sentinel") {
		t.Fatalf("context boundary violated: %q", view.Context)
	}
}

func TestDegradedCaptureReusesDurableIngestion(t *testing.T) {
	cfg := testConfig(t)
	core, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	content := "terminal event content"
	digest := sha256.Sum256([]byte(content))
	req := ingest.SubmitRequest{SessionID: "session-1", EventID: "terminal-1", Phase: "session_end", Content: content, ContentHash: fmt.Sprintf("sha256:%x", digest)}
	first, err := core.Ingest.Submit(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if err := core.Close(); err != nil {
		t.Fatal(err)
	}
	core, err = Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer core.Close()
	again, err := core.Ingest.Submit(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if first.IngestionID != again.IngestionID {
		t.Fatalf("duplicate terminal capture: %q vs %q", first.IngestionID, again.IngestionID)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		status, err := core.Ingest.Get(context.Background(), first.IngestionID)
		if err != nil {
			t.Fatal(err)
		}
		if status.Status == "spooled" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("capture did not degrade to spooled: %+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestDegradedCapturePersistsActivityAndTransientSpoolAcrossReopen(t *testing.T) {
	cfg := testConfig(t)
	core, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if core.Activity == nil || core.TransientSpool == nil {
		_ = core.Close()
		t.Fatal("activity and transient spool must be composed")
	}
	if core.Ingest.Activity != core.Activity || core.Ingest.Spool != core.TransientSpool {
		_ = core.Close()
		t.Fatal("ingest must use composed activity and spool")
	}
	content := "durable degraded capture"
	digest := sha256.Sum256([]byte(content))
	req := ingest.SubmitRequest{SessionID: "session-1", EventID: "terminal-1", Phase: "session_end", Content: content, ContentHash: fmt.Sprintf("sha256:%x", digest)}
	accepted, err := core.Ingest.Submit(context.Background(), req)
	if err != nil {
		_ = core.Close()
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		pending, err := core.TransientSpool.Pending(context.Background())
		if err != nil {
			_ = core.Close()
			t.Fatal(err)
		}
		if len(pending) == 1 {
			if pending[0].EventID != req.EventID || pending[0].Content != content || pending[0].ContentHash != req.ContentHash {
				_ = core.Close()
				t.Fatalf("incorrect spooled entry: %+v", pending[0])
			}
			break
		}
		if time.Now().After(deadline) {
			_ = core.Close()
			t.Fatalf("capture not spooled: %+v", pending)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := core.Close(); err != nil {
		t.Fatal(err)
	}
	core, err = Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer core.Close()
	events, err := core.Activity.SessionEvents(context.Background(), req.SessionID, 10)
	if err != nil || len(events) != 1 || events[0].ID != req.EventID {
		t.Fatalf("activity after reopen: %+v, %v", events, err)
	}
	pending, err := core.TransientSpool.Pending(context.Background())
	if err != nil || len(pending) != 1 || pending[0].EventID != req.EventID {
		t.Fatalf("spool after reopen: %+v, %v", pending, err)
	}
	deadline = time.Now().Add(3 * time.Second)
	for {
		status, err := core.Ingest.Get(context.Background(), accepted.IngestionID)
		if err != nil {
			t.Fatal(err)
		}
		if status.Status == "spooled" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("ingestion after reopen: %+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRequireLocalModelRejectsMissing(t *testing.T) {
	cfg := testConfig(t)
	cfg.RequireLocalModel = true
	core, err := Open(context.Background(), cfg)
	if err == nil || core != nil {
		t.Fatalf("required model unexpectedly opened: %v", err)
	}
	if _, err := os.Stat(cfg.StateDB); !os.IsNotExist(err) {
		t.Fatalf("failed open left state: %v", err)
	}
}

func TestCorruptExplicitModelDoesNotDownloadOrFallBack(t *testing.T) {
	cfg := testConfig(t)
	if err := os.MkdirAll(cfg.ModelsDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.ModelsDir, "model.onnx"), []byte("not a model"), 0600); err != nil {
		t.Fatal(err)
	}
	core, err := Open(context.Background(), cfg)
	if err == nil || core != nil {
		if core != nil {
			_ = core.Close()
		}
		t.Fatalf("corrupt explicit model accepted: %v", err)
	}
	if _, err := os.Stat(cfg.PalacePath); !os.IsNotExist(err) {
		t.Fatalf("failed model created palace: %v", err)
	}
}

func TestLiteralSpecialCharactersInExplicitPaths(t *testing.T) {
	cfg := testConfig(t)
	root := filepath.Join(t.TempDir(), "literal%$#")
	cfg.PersonaDir = filepath.Join(root, "persona")
	cfg.PalacePath = filepath.Join(root, "palace")
	cfg.ModelsDir = filepath.Join(root, "models")
	cfg.StateDB = filepath.Join(root, "state", "garden.db")
	core, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("literal path characters must be accepted: %v", err)
	}
	if err := core.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cfg.StateDB); err != nil {
		t.Fatalf("state DB missing at explicit literal path: %v", err)
	}
}

func TestRejectsImplicitAndRelativePaths(t *testing.T) {
	cfg := testConfig(t)
	cfg.StateDB = "relative.db"
	if core, err := Open(context.Background(), cfg); err == nil || core != nil {
		t.Fatalf("relative state accepted: %v", err)
	}
	cfg = testConfig(t)
	cfg.ModelsDir = ""
	if core, err := Open(context.Background(), cfg); err == nil || core != nil {
		t.Fatalf("implicit models accepted: %v", err)
	}
}
