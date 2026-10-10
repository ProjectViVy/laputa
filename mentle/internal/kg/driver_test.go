package kg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ProjectViVy/laputa/mentle/storage/sqlite"
)

func TestLegacyKGFileReopensAndPreservesRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "knowledge_graph.sqlite3")
	db, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE entities(id TEXT PRIMARY KEY, name TEXT NOT NULL, type TEXT DEFAULT 'unknown', properties TEXT DEFAULT '{}', created_at TEXT DEFAULT CURRENT_TIMESTAMP); CREATE TABLE triples(id TEXT PRIMARY KEY, subject TEXT NOT NULL, predicate TEXT NOT NULL, object TEXT NOT NULL, valid_from TEXT, valid_to TEXT, confidence REAL DEFAULT 1.0, source_closet TEXT, source_file TEXT, extracted_at TEXT DEFAULT CURRENT_TIMESTAMP); INSERT INTO entities(id,name) VALUES('legacy','Legacy'); INSERT INTO triples(id,subject,predicate,object) VALUES('t_legacy','legacy','related','legacy')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	graph, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	stats, err := graph.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.EntityCount != 1 || stats.TripleCount != 1 {
		t.Fatalf("legacy stats: %+v", stats)
	}
	if err := graph.Close(); err != nil {
		t.Fatal(err)
	}
	graph, err = New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer graph.Close()
	stats, err = graph.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.EntityCount != 1 || stats.TripleCount != 1 {
		t.Fatalf("reopened stats: %+v", stats)
	}
}

func TestNewClosesDatabaseAfterSchemaFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad_kg.sqlite3")
	db, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE triples (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if graph, err := New(path); err == nil {
		graph.Close()
		t.Fatal("expected schema error")
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("database handle remains open after schema failure: %v", err)
	}
}
