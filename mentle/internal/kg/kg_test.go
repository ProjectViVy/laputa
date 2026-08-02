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

	tripleID, err := kg.AddTriple("Matthew", "lives in", "Vancouver", "", "", 1.0)
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

	kg.AddTriple("Matthew", "works at", "Acme", "", "", 1.0)

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
