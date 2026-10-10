package runtimecore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ProjectViVy/laputa/garden/internal/ingest"
	"github.com/ProjectViVy/laputa/garden/internal/personactx"
	"github.com/ProjectViVy/laputa/garden/internal/recall"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/laputa/persona"
	"github.com/ProjectViVy/laputa/mentle/facade"
	_ "modernc.org/sqlite"
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
	if core.Mentle != nil || core.FastRecall.Searcher != nil || core.LexicalOnly {
		t.Fatal("missing model and canonical must degrade without substituting a searcher")
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

func TestOpenMissingModelReadsExistingCanonicalLexically(t *testing.T) {
	cfg := testConfig(t)
	if err := os.MkdirAll(cfg.PalacePath, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfg.PalacePath, "canonical.sqlite3")
	catalog, err := facade.OpenCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = db.Exec(`INSERT INTO memories(id,kind,content,status,version,scope,tags_json,source_json,valid_from,supersedes_json,created_at,updated_at,metadata_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, "mem_lexical", "note", "distinctivelexicaltoken evidence", "active", 1, "global", "[]", `{"type":"user"}`, now, "[]", now, now, "{}")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	core, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer core.Close()
	if core.Mentle == nil || core.FastRecall.Searcher == nil || !core.LexicalOnly {
		t.Fatalf("lexical wiring: mentle=%v searcher=%v lexical=%v", core.Mentle, core.FastRecall.Searcher, core.LexicalOnly)
	}
	content := "offline capture cannot mutate canonical"
	digest := sha256.Sum256([]byte(content))
	accepted, err := core.Ingest.Submit(context.Background(), ingest.SubmitRequest{SessionID: "lexical", EventID: "event-lexical", Phase: "session_end", Content: content, ContentHash: fmt.Sprintf("sha256:%x", digest)})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		status, err := core.Ingest.Get(context.Background(), accepted.IngestionID)
		if err != nil {
			t.Fatal(err)
		}
		if status.Status == "spooled" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("lexical ingestion did not spool: %+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
	view, err := core.FastRecall.Recall(context.Background(), recall.FastRequest{Query: "distinctivelexicaltoken", BudgetChars: 6000})
	if err != nil || len(view.Cards) != 1 || view.Cards[0].ID != "mem_lexical" || !view.Degraded || !strings.Contains(strings.Join(view.Warnings, " "), "lexical") {
		t.Fatalf("lexical recall: %+v %v", view, err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("lexical fallback mutated canonical SQLite")
	}
	for _, name := range []string{"vectors.db", "knowledge_graph.sqlite3"} {
		if _, err := os.Stat(filepath.Join(cfg.PalacePath, name)); !os.IsNotExist(err) {
			t.Fatalf("derived index %s created: %v", name, err)
		}
	}
}

func TestOpenFullModelPersistsAndRetrievesCardAcrossReopen(t *testing.T) {
	cfg := testConfig(t)
	var err error
	cfg.ModelsDir, err = filepath.Abs(filepath.Join("..", "..", "..", "mentle", "models"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cfg.ModelsDir, "onnx", "model.onnx")); err != nil {
		t.Fatalf("model fixture required: %v", err)
	}
	core, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if core.Mentle == nil || core.LexicalOnly {
		t.Fatal("full model did not open in writable mode")
	}
	if _, err := core.Persona.Initialize(persona.Initialization{Identity: "identity", Relationship: "relationship", Redline: "redline", User: "user", World: "world"}, "user", persona.SourceInit, "init"); err != nil {
		t.Fatal(err)
	}
	memory, err := core.Mentle.CreateMemory(context.Background(), facade.CreateMemoryRequest{Content: "uniquefullmodeltoken evidence", Scope: "global"}, "", "")
	if err != nil {
		core.Close()
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
	view, err := core.FastRecall.Recall(context.Background(), recall.FastRequest{Query: "uniquefullmodeltoken", SessionID: "session-full", BudgetChars: 6000})
	if err != nil || view.Degraded || len(view.Cards) == 0 {
		t.Fatalf("full model recall: %+v %v", view, err)
	}
	found := false
	for _, card := range view.Cards {
		if card.ID == memory.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("persisted card %q missing: %+v", memory.ID, view.Cards)
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
	if _, err := core.Actmem.AppendEntry(evolution.Entry{Section: evolution.SectionPulse, Scope: evolution.Scope{SubjectID: "test", Kind: evolution.ScopePersonal}, SessionID: "session-1", Body: "activity private sentinel"}); err != nil {
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
	if first.Content(personactx.SectionIdentity) != second.Content(personactx.SectionIdentity) {
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

func TestOpenLexicalOnlyLeavesExistingSpoolPending(t *testing.T) {
	cfg := testConfig(t)
	ctx := context.Background()
	core, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	content := "lexical spool must remain pending"
	digest := sha256.Sum256([]byte(content))
	accepted, err := core.Ingest.Submit(ctx, ingest.SubmitRequest{SessionID: "lexical-session", EventID: "lexical-pending", Phase: "session_end", Content: content, ContentHash: fmt.Sprintf("sha256:%x", digest)})
	if err != nil {
		_ = core.Close()
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		status, getErr := core.Ingest.Get(ctx, accepted.IngestionID)
		if getErr != nil {
			_ = core.Close()
			t.Fatal(getErr)
		}
		if status.Status == "spooled" {
			break
		}
		if time.Now().After(deadline) {
			_ = core.Close()
			t.Fatalf("capture did not spool: %+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := core.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cfg.PalacePath, 0700); err != nil {
		t.Fatal(err)
	}
	catalog, err := facade.OpenCatalog(filepath.Join(cfg.PalacePath, "canonical.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.Close(); err != nil {
		t.Fatal(err)
	}
	core, err = Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer core.Close()
	if !core.LexicalOnly {
		t.Fatal("expected lexical-only runtime")
	}
	pending, err := core.TransientSpool.Pending(ctx)
	if err != nil || len(pending) != 1 || pending[0].EventID != "lexical-pending" {
		t.Fatalf("lexical runtime drained spool: %+v %v", pending, err)
	}
	deadline = time.Now().Add(3 * time.Second)
	for {
		status, getErr := core.Ingest.Get(ctx, accepted.IngestionID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		if status.Status == "spooled" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("lexical runtime changed ingestion: %+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestOpenRecoversSpooledCaptureWhenModelReturns(t *testing.T) {
	cfg := testConfig(t)
	ctx := context.Background()
	core, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	content := "recovered durable capture unique recovery token"
	digest := sha256.Sum256([]byte(content))
	req := ingest.SubmitRequest{SessionID: "recover-session", EventID: "recover-event", Phase: "session_end", Content: content, ContentHash: fmt.Sprintf("sha256:%x", digest)}
	accepted, err := core.Ingest.Submit(ctx, req)
	if err != nil {
		_ = core.Close()
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		status, getErr := core.Ingest.Get(ctx, accepted.IngestionID)
		if getErr != nil {
			_ = core.Close()
			t.Fatal(getErr)
		}
		if status.Status == "spooled" {
			break
		}
		if time.Now().After(deadline) {
			_ = core.Close()
			t.Fatalf("capture did not spool: %+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := core.Close(); err != nil {
		t.Fatal(err)
	}
	cfg.ModelsDir, err = filepath.Abs(filepath.Join("..", "..", "..", "mentle", "models"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cfg.ModelsDir, "onnx", "model.onnx")); err != nil {
		t.Fatalf("model fixture required: %v", err)
	}
	core, err = Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	status, err := core.Ingest.Get(ctx, accepted.IngestionID)
	if err != nil || status.Status != "completed" {
		_ = core.Close()
		t.Fatalf("shared runtime did not recover ingestion: %+v %v", status, err)
	}
	pending, err := core.TransientSpool.Pending(ctx)
	if err != nil || len(pending) != 0 {
		_ = core.Close()
		t.Fatalf("recovered spool still pending: %+v %v", pending, err)
	}
	if err := core.Close(); err != nil {
		t.Fatal(err)
	}
	core, err = Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer core.Close()
	status, err = core.Ingest.Get(ctx, accepted.IngestionID)
	if err != nil || status.Status != "completed" {
		t.Fatalf("reopen changed recovery status: %+v %v", status, err)
	}
	pending, err = core.TransientSpool.Pending(ctx)
	if err != nil || len(pending) != 0 {
		t.Fatalf("reopen duplicated pending spool: %+v %v", pending, err)
	}
}

func TestOpenReportsSpoolDrainFailureInsteadOfHidingIt(t *testing.T) {
	cfg := testConfig(t)
	ctx := context.Background()
	core, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	content := "recovery failure should remain retryable"
	digest := sha256.Sum256([]byte(content))
	accepted, err := core.Ingest.Submit(ctx, ingest.SubmitRequest{
		SessionID: "failed-recovery-session", EventID: "failed-recovery-event",
		Phase: "session_end", Content: content, ContentHash: fmt.Sprintf("sha256:%x", digest),
	})
	if err != nil {
		_ = core.Close()
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		status, getErr := core.Ingest.Get(ctx, accepted.IngestionID)
		if getErr != nil {
			_ = core.Close()
			t.Fatal(getErr)
		}
		if status.Status == "spooled" {
			break
		}
		if time.Now().After(deadline) {
			_ = core.Close()
			t.Fatalf("not spooled: %+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := core.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", cfg.StateDB)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TRIGGER reject_runtime_drain BEFORE UPDATE ON ingestions WHEN NEW.status='completed' BEGIN SELECT RAISE(ABORT, 'runtime drain blocked'); END`); err != nil {
		t.Fatal(err)
	}
	cfg.ModelsDir, err = filepath.Abs(filepath.Join("..", "..", "..", "mentle", "models"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cfg.ModelsDir, "onnx", "model.onnx")); err != nil {
		t.Fatalf("model fixture required: %v", err)
	}
	core, err = Open(ctx, cfg)
	if core != nil {
		_ = core.Close()
		t.Fatal("Open returned a runtime despite failed spool recovery")
	}
	if err == nil || !strings.Contains(err.Error(), "runtime drain blocked") {
		t.Fatalf("Open did not report drain failure: %v", err)
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM ingestions WHERE ingestion_id=?`, accepted.IngestionID).Scan(&status); err != nil || status != "spooled" {
		t.Fatalf("ingestion status after failed drain: %q, %v", status, err)
	}
	var pending int
	if err := db.QueryRow(`SELECT COUNT(*) FROM transient_spool WHERE event_id=? AND status='pending_mentle'`, accepted.EventID).Scan(&pending); err != nil || pending != 1 {
		t.Fatalf("pending spool after failed drain: %d, %v", pending, err)
	}
}

func TestRequireLocalModelRejectsMissingEvenWithCanonical(t *testing.T) {
	cfg := testConfig(t)
	if err := os.MkdirAll(cfg.PalacePath, 0700); err != nil {
		t.Fatal(err)
	}
	catalog, err := facade.OpenCatalog(filepath.Join(cfg.PalacePath, "canonical.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.Close(); err != nil {
		t.Fatal(err)
	}
	cfg.RequireLocalModel = true
	core, err := Open(context.Background(), cfg)
	if err == nil || core != nil {
		if core != nil {
			core.Close()
		}
		t.Fatalf("required model unexpectedly fell back to lexical: %v", err)
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
