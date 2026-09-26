package facade

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dashimaki/mentle/storage/sqlite"
)

func TestLexicalOnlyOpensExistingCanonicalWithoutModelOrDerivedWrites(t *testing.T) {
	dir := t.TempDir()
	catalog, err := OpenCatalog(filepath.Join(dir, "canonical.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := insertTestMemory(catalog, Memory{ID: "mem_offline", Kind: "note", Content: "uniqueofflinekeyword evidence", Status: "active", Version: 1, Scope: "global", Tags: []string{}, Source: MemorySource{Type: "user"}, ValidFrom: now, CreatedAt: now, UpdatedAt: now, Metadata: map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Close(); err != nil {
		t.Fatal(err)
	}
	var svc Service
	if err := svc.Init(context.Background(), Options{PalacePath: dir, ModelsDir: filepath.Join(dir, "missing-model"), LexicalOnly: true}); err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	if svc.Embedder != nil || svc.Searcher != nil || svc.Hybrid == nil {
		t.Fatalf("unexpected components: embedder=%v searcher=%v hybrid=%v", svc.Embedder, svc.Searcher, svc.Hybrid)
	}
	page, err := svc.SearchCards(context.Background(), CardQuery{Text: "uniqueofflinekeyword"})
	if err != nil || len(page.Cards) != 1 || page.Cards[0].ID != "mem_offline" {
		t.Fatalf("cards=%+v err=%v", page, err)
	}
	for _, name := range []string{"vectors.db", "knowledge_graph.sqlite3"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s unexpectedly exists: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "missing-model")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("model directory unexpectedly exists: %v", err)
	}
}

func insertTestMemory(c *Catalog, m Memory) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	if err := insertMemory(context.Background(), tx, m); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func TestLexicalOnlyRejectsMutationsWithoutChangingCanonical(t *testing.T) {
	dir := t.TempDir()
	catalog, err := OpenCatalog(filepath.Join(dir, "canonical.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := insertTestMemory(catalog, Memory{ID: "mem_keep", Kind: "note", Content: "original", Status: "active", Version: 1, Tags: []string{}, Source: MemorySource{Type: "user"}, ValidFrom: now, CreatedAt: now, UpdatedAt: now, Metadata: map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	catalog.Close()
	var svc Service
	if err := svc.Init(context.Background(), Options{PalacePath: dir, LexicalOnly: true}); err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	ctx := context.Background()
	if _, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "new"}, "", ""); !errors.Is(err, ErrReadOnly) {
		t.Errorf("create: %v", err)
	}
	updated := "changed"
	version := 1
	if _, err := svc.UpdateMemory(ctx, "mem_keep", UpdateMemoryRequest{Content: &updated, ExpectedVersion: &version}); !errors.Is(err, ErrReadOnly) {
		t.Errorf("update: %v", err)
	}
	if _, err := svc.DeleteMemory(ctx, "mem_keep", 1, "", ""); !errors.Is(err, ErrReadOnly) {
		t.Errorf("delete: %v", err)
	}
	if err := svc.SaveSessionCursor("session", now.Format(time.RFC3339Nano)); !errors.Is(err, ErrReadOnly) {
		t.Errorf("cursor: %v", err)
	}
	if _, err := svc.AcquireSessionLease(ctx, "session", "owner", time.Minute); !errors.Is(err, ErrReadOnly) {
		t.Errorf("lease: %v", err)
	}
	m, err := svc.GetMemory(ctx, "mem_keep")
	if err != nil || m.Content != "original" || m.Version != 1 {
		t.Fatalf("canonical changed: %+v %v", m, err)
	}
}

func TestLexicalOnlyIndexHealthDoesNotCrashOnAbsentVectorStore(t *testing.T) {
	dir := t.TempDir()
	catalog, err := OpenCatalog(filepath.Join(dir, "canonical.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	catalog.Close()
	var svc Service
	if err := svc.Init(context.Background(), Options{PalacePath: dir, LexicalOnly: true}); err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	health, err := svc.IndexHealth(context.Background())
	if !errors.Is(err, ErrIndexHealthUnavailable) || health.Status != "unavailable" {
		t.Fatalf("health=%+v err=%v", health, err)
	}
}

func TestLexicalOnlyCloseMakesSearchUnavailable(t *testing.T) {
	dir := t.TempDir()
	catalog, err := OpenCatalog(filepath.Join(dir, "canonical.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.Close(); err != nil {
		t.Fatal(err)
	}
	var svc Service
	if err := svc.Init(context.Background(), Options{PalacePath: dir, LexicalOnly: true}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Close(); err != nil {
		t.Fatal(err)
	}
	if svc.Hybrid != nil || svc.lexicalOnly {
		t.Fatalf("closed service retains lexical state: hybrid=%v lexicalOnly=%v", svc.Hybrid, svc.lexicalOnly)
	}
	if _, err := svc.SearchCards(context.Background(), CardQuery{Text: "anything"}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("search after close: %v", err)
	}
}

func TestLexicalOnlyDoesNotMigrateOldSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "canonical.sqlite3")
	catalog, err := OpenCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	// Prior index_jobs schema lacks the canonical_version column migrated by OpenCatalog.
	if _, err := catalog.db.Exec(`DROP TABLE index_jobs; CREATE TABLE index_jobs (job_id TEXT NOT NULL, memory_id TEXT PRIMARY KEY, operation TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Close(); err != nil {
		t.Fatal(err)
	}
	var svc Service
	if err := svc.Init(context.Background(), Options{PalacePath: dir, LexicalOnly: true}); err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	if _, err := svc.Catalog.db.Exec(`CREATE TABLE forbidden (id INTEGER)`); err == nil {
		t.Fatal("lexical catalog accepted DDL")
	}
	if _, err := svc.Catalog.db.Exec(`INSERT INTO index_jobs(job_id, memory_id, operation) VALUES ('j', 'm', 'upsert')`); err == nil {
		t.Fatal("lexical catalog accepted DML")
	}
	var migrated int
	rows, err := svc.Catalog.db.Query(`PRAGMA table_info(index_jobs)`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		if name == "canonical_version" {
			migrated++
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	rows.Close()
	if migrated != 0 {
		t.Fatal("lexical-only migrated old index_jobs schema")
	}
	// Verify the on-disk schema from a separate connection, not just the read-only handle.
	check, err := sqlite.OpenReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer check.Close()
	if err := check.QueryRow(`SELECT count(*) FROM pragma_table_info('index_jobs') WHERE name = 'canonical_version'`).Scan(&migrated); err != nil {
		t.Fatal(err)
	}
	if migrated != 0 {
		t.Fatal("on-disk schema migrated")
	}
}

func TestLexicalOnlyRejectsRequireLocalModelBeforeOpeningPaths(t *testing.T) {
	dir := t.TempDir()
	catalog, err := OpenCatalog(filepath.Join(dir, "canonical.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.Close(); err != nil {
		t.Fatal(err)
	}
	var svc Service
	err = svc.Init(context.Background(), Options{PalacePath: dir, LexicalOnly: true, RequireLocalModel: true})
	if err == nil || !strings.Contains(err.Error(), "RequireLocalModel") {
		t.Fatalf("contradictory options: %v", err)
	}
	if svc.Catalog != nil || svc.Hybrid != nil {
		t.Fatalf("contradictory options retained resources: catalog=%p hybrid=%p", svc.Catalog, svc.Hybrid)
	}
}

func TestLexicalOnlyRequiresExistingCanonical(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "absent")
	var svc Service
	if err := svc.Init(context.Background(), Options{PalacePath: dir, LexicalOnly: true}); err == nil || !strings.Contains(err.Error(), "canonical") {
		t.Fatalf("expected missing canonical error, got %v", err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("created directory: %v", err)
	}
}
