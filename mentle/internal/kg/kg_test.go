package kg

import (
	"os"
	"slices"
	"testing"
)

func TestNewKnowledgeGraphCreatesDB(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "kg_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	kg, err := New(tmpfile.Name())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer kg.Close()

	rows, err := kg.db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		t.Fatalf("query tables error = %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		tables = append(tables, name)
	}

	want := []string{"entities", "triples"}
	for _, w := range want {
		found := slices.Contains(tables, w)
		if !found {
			t.Errorf("expected table %q not found", w)
		}
	}
}

func TestAddEntity(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "kg_test_*.db")
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	kg, _ := New(tmpfile.Name())
	defer kg.Close()

	id, err := kg.AddEntity("Matthew", "person", map[string]string{"role": "developer"})
	if err != nil {
		t.Fatalf("AddEntity() error = %v", err)
	}
	if id != "matthew" {
		t.Errorf("AddEntity() id = %q, want %q", id, "matthew")
	}
}

func TestAddTriple(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "kg_test_*.db")
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	kg, _ := New(tmpfile.Name())
	defer kg.Close()

	tripleID, err := kg.AddTriple(TripleInput{
		Subject:    "Matthew",
		Predicate:  "lives in",
		Object:     "Vancouver",
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("AddTriple() error = %v", err)
	}
	if tripleID == "" {
		t.Error("AddTriple() returned empty tripleID")
	}

	var count int
	kg.db.QueryRow("SELECT COUNT(*) FROM entities WHERE id IN ('matthew', 'vancouver')").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 entities created, got %d", count)
	}
}

func TestQueryEntity(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "kg_test_*.db")
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	kg, _ := New(tmpfile.Name())
	defer kg.Close()

	kg.AddTriple(TripleInput{
		Subject:    "Matthew",
		Predicate:  "works at",
		Object:     "Acme",
		Confidence: 1.0,
	})

	results, err := kg.QueryEntity("Matthew", "", "")
	if err != nil {
		t.Fatalf("QueryEntity() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("QueryEntity() returned %d results, want 1", len(results))
	}

	r := results[0]
	if r.Predicate != "works_at" {
		t.Errorf("Predicate = %q, want %q", r.Predicate, "works_at")
	}
	if r.Object != "Acme" {
		t.Errorf("Object = %q, want %q", r.Object, "Acme")
	}
}

func TestEntityID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Matthew", "matthew"},
		{"Los Angeles", "los_angeles"},
		{"O'Brien", "obrien"},
		{"San Francisco", "san_francisco"},
	}
	for _, tt := range tests {
		got := entityID(tt.input)
		if got != tt.want {
			t.Errorf("entityID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAddTripleWithProvenance(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "kg_test_*.db")
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	kg, _ := New(tmpfile.Name())
	defer kg.Close()

	input := TripleInput{
		Subject:        "A",
		Predicate:      "B",
		Object:         "C",
		SourceCloset:   "closet1",
		SourceFile:     "file1.md",
		SourceDrawerID: "drawer1",
		AdapterName:    "adapter1",
		AdapterVersion: "v1.0",
	}

	id, err := kg.AddTriple(input)
	if err != nil {
		t.Fatalf("AddTriple failed: %v", err)
	}

	var sc, sf, sd, an, av string
	err = kg.db.QueryRow("SELECT source_closet, source_file, source_drawer_id, adapter_name, adapter_version FROM triples WHERE id = ?", id).Scan(&sc, &sf, &sd, &an, &av)
	if err != nil {
		t.Fatalf("Failed to query provenance: %v", err)
	}

	if sc != input.SourceCloset || sf != input.SourceFile || sd != input.SourceDrawerID || an != input.AdapterName || av != input.AdapterVersion {
		t.Errorf("Provenance mismatch")
	}
}

func TestQueryEntityBidirectionalDefault(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "kg_test_*.db")
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	kg, _ := New(tmpfile.Name())
	defer kg.Close()

	kg.AddTriple(TripleInput{Subject: "A", Predicate: "knows", Object: "B"})

	res, err := kg.QueryEntity("B", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].Object != "A" {
		t.Errorf("Expected to find A through bidirectional default query, got: %+v", res)
	}
}

func TestInvertedIntervalRejected(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "kg_test_*.db")
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	kg, _ := New(tmpfile.Name())
	defer kg.Close()

	_, err := kg.AddTriple(TripleInput{
		Subject:   "A",
		Predicate: "B",
		Object:    "C",
		ValidFrom: "2025-01-01T00:00:00Z",
		ValidTo:   "2020-01-01T00:00:00Z",
	})
	if err != ErrInvalidTemporalInterval {
		t.Errorf("Expected ErrInvalidTemporalInterval, got %v", err)
	}
}

func TestRFC3339Validation(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "kg_test_*.db")
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	kg, _ := New(tmpfile.Name())
	defer kg.Close()

	_, err := kg.AddTriple(TripleInput{
		Subject:   "A",
		Predicate: "B",
		Object:    "C",
		ValidFrom: "not-a-date",
	})
	if err == nil {
		t.Errorf("Expected error for invalid ValidFrom")
	}
}

func TestExistingRowsMigrated(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "kg_test_*.db")
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	kg, _ := New(tmpfile.Name())
	defer kg.Close()

	// Create table without new columns manually first
	kg.db.Exec("DROP TABLE triples")
	kg.db.Exec(`CREATE TABLE triples (
		id TEXT PRIMARY KEY, subject TEXT NOT NULL, predicate TEXT NOT NULL, object TEXT NOT NULL,
		valid_from TEXT, valid_to TEXT, confidence REAL DEFAULT 1.0, source_closet TEXT, source_file TEXT, extracted_at TEXT DEFAULT CURRENT_TIMESTAMP
	)`)
	kg.db.Exec(`INSERT INTO triples (id, subject, predicate, object) VALUES ('t1', 'A', 'B', 'C')`)

	// Run migration
	err := kg.initDB()
	if err != nil {
		t.Fatalf("initDB failed: %v", err)
	}

	_, err = kg.AddTriple(TripleInput{Subject: "X", Predicate: "Y", Object: "Z", SourceDrawerID: "d1"})
	if err != nil {
		t.Fatalf("AddTriple failed after migration: %v", err)
	}
}

func TestAsOfRangeQuery(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "kg_test_*.db")
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	kg, _ := New(tmpfile.Name())
	defer kg.Close()

	kg.AddTriple(TripleInput{
		Subject:   "A",
		Predicate: "B",
		Object:    "C",
		ValidFrom: "2026-01-01T00:00:00Z",
		ValidTo:   "2026-06-01T00:00:00Z",
	})

	// Visible
	res, _ := kg.QueryEntity("A", "2026-03-01T00:00:00Z", "")
	if len(res) != 1 {
		t.Errorf("Expected visible at 2026-03-01, got %d results", len(res))
	}

	// Invisible
	res, _ = kg.QueryEntity("A", "2026-07-01T00:00:00Z", "")
	if len(res) != 0 {
		t.Errorf("Expected invisible at 2026-07-01, got %d results", len(res))
	}
}

func TestAddTripleDeterministicID(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "kg_test_*.db")
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	kg, _ := New(tmpfile.Name())
	defer kg.Close()

	input := TripleInput{Subject: "A", Predicate: "B", Object: "C"}
	id1, _ := kg.AddTriple(input)
	id2, _ := kg.AddTriple(input)

	if id1 != id2 {
		t.Errorf("Expected deterministic IDs, got %s and %s", id1, id2)
	}
}
