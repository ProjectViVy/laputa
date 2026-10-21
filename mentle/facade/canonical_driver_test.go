package facade

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ProjectViVy/laputa/mentle/storage/sqlite"
)

func TestCatalogReopensCanonicalFileWithoutChangingAuthorityTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "canonical.sqlite3")
	original, err := OpenCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO memories(id,kind,content,status,version,scope,tags_json,source_json,valid_from,supersedes_json,created_at,updated_at,metadata_json) VALUES('mem_existing','note','retained','active',3,'','[]','{}','2026-01-01T00:00:00Z','[]','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z','{}')`,
		`INSERT INTO index_jobs(job_id,memory_id,canonical_version,operation,content,metadata_json,created_at,updated_at) VALUES('job_existing','mem_existing',3,'upsert','retained','{}','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`,
	} {
		if _, err := original.db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := original.Close(); err != nil {
		t.Fatal(err)
	}
	legacy, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`DROP TABLE idempotency`,
		`CREATE TABLE idempotency(key TEXT PRIMARY KEY,body_hash TEXT NOT NULL,memory_id TEXT NOT NULL,created_at TEXT NOT NULL)`,
		`INSERT INTO idempotency(key,body_hash,memory_id,created_at) VALUES('existing-key','hash','mem_existing','2026-01-01T00:00:00Z')`,
		`DROP TABLE mutation_receipts`,
		`CREATE TABLE mutation_receipts(operation_id TEXT PRIMARY KEY,payload_digest TEXT NOT NULL,scope TEXT NOT NULL,destination TEXT NOT NULL,record_id TEXT NOT NULL,revision INTEGER NOT NULL,status TEXT NOT NULL,error_code TEXT NOT NULL,created_at TEXT NOT NULL)`,
		`INSERT INTO mutation_receipts(operation_id,payload_digest,scope,destination,record_id,revision,status,error_code,created_at) VALUES('legacy-op','digest','scope','dest','mem_existing',3,'applied','','2026-01-01T00:00:00Z')`,
	} {
		if _, err := legacy.Exec(statement); err != nil {
			_ = legacy.Close()
			t.Fatal(err)
		}
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	var content, key, job string
	if err := reopened.db.QueryRow(`SELECT content FROM memories WHERE id='mem_existing' AND version=3`).Scan(&content); err != nil {
		t.Fatal(err)
	}
	if err := reopened.db.QueryRow(`SELECT key FROM idempotency WHERE memory_id='mem_existing'`).Scan(&key); err != nil {
		t.Fatal(err)
	}
	if err := reopened.db.QueryRow(`SELECT job_id FROM index_jobs WHERE memory_id='mem_existing' AND canonical_version=3`).Scan(&job); err != nil {
		t.Fatal(err)
	}
	if content != "retained" || key != "existing-key" || job != "job_existing" {
		t.Fatalf("canonical state changed: %q, %q, %q", content, key, job)
	}
	var requestJSON, intentJSON []byte
	if err := reopened.db.QueryRow(`SELECT request_json FROM idempotency WHERE key='existing-key'`).Scan(&requestJSON); err != nil {
		t.Fatal(err)
	}
	if err := reopened.db.QueryRow(`SELECT intent_json FROM mutation_receipts WHERE operation_id='legacy-op'`).Scan(&intentJSON); err != nil {
		t.Fatal(err)
	}
	if len(requestJSON) != 0 || len(intentJSON) != 0 {
		t.Fatalf("legacy hash-only replay state unexpectedly has intent: request=%q mutation=%q", requestJSON, intentJSON)
	}
	var journal string
	var fk, timeout int
	if err := reopened.db.QueryRow(`PRAGMA journal_mode`).Scan(&journal); err != nil {
		t.Fatal(err)
	}
	if err := reopened.db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if err := reopened.db.QueryRow(`PRAGMA busy_timeout`).Scan(&timeout); err != nil {
		t.Fatal(err)
	}
	if journal != "wal" || fk != 1 || timeout != 5000 {
		t.Fatalf("pragmas: journal=%s fk=%d busy_timeout=%d", journal, fk, timeout)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
	service, err := OpenCatalogService(context.Background(), filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	if _, err := service.CreateMemory(context.Background(), CreateMemoryRequest{Content: "retry old key"}, "existing-key"); !errors.Is(err, ErrIdempotencyRecoveryRequired) {
		t.Fatalf("legacy idempotency retry=%v, want explicit recovery", err)
	}
}

func TestCatalogInvalidPathReturnsError(t *testing.T) {
	if c, err := OpenCatalog(filepath.Join(t.TempDir(), "missing", "canonical.sqlite3")); err == nil {
		c.Close()
		t.Fatal("expected open error")
	}
}
