// Package kg provides a knowledge graph for storing entities and triples.
// It implements a simple RDF-like store with temporal validity.
package kg

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dashimaki/mentle/storage/sqlite"
)

type Entity struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Properties map[string]string `json:"properties"`
	CreatedAt  time.Time         `json:"created_at"`
}

type Triple struct {
	ID           string    `json:"id"`
	Subject      string    `json:"subject"`
	Predicate    string    `json:"predicate"`
	Object       string    `json:"object"`
	ValidFrom    string    `json:"valid_from"`
	ValidTo      string    `json:"valid_to"`
	Confidence   float64   `json:"confidence"`
	SourceCloset string    `json:"source_closet"`
	SourceFile   string    `json:"source_file"`
	ExtractedAt  time.Time `json:"extracted_at"`
}

type KnowledgeGraph struct {
	db *sql.DB
}

func New(dbPath string) (*KnowledgeGraph, error) {
	db, err := sqlite.Open(dbPath)
	if err != nil {
		return nil, err
	}
	kg := &KnowledgeGraph{db: db}
	if err := kg.initDB(); err != nil {
		return nil, err
	}
	return kg, nil
}

func (kg *KnowledgeGraph) initDB() error {
	_, err := kg.db.Exec(`
		CREATE TABLE IF NOT EXISTS entities (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT DEFAULT 'unknown',
			properties TEXT DEFAULT '{}',
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS triples (
			id TEXT PRIMARY KEY,
			subject TEXT NOT NULL,
			predicate TEXT NOT NULL,
			object TEXT NOT NULL,
			valid_from TEXT,
			valid_to TEXT,
			confidence REAL DEFAULT 1.0,
			source_closet TEXT,
			source_file TEXT,
			extracted_at TEXT DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_triples_subject ON triples(subject);
		CREATE INDEX IF NOT EXISTS idx_triples_object ON triples(object);
	`)
	return err
}

func entityID(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(name, " ", "_"), "'", ""))
}

func (kg *KnowledgeGraph) AddEntity(name, entityType string, properties map[string]string) (string, error) {
	eid := entityID(name)
	props, _ := json.Marshal(properties)
	_, err := kg.db.Exec(
		"INSERT OR REPLACE INTO entities (id, name, type, properties) VALUES (?, ?, ?, ?)",
		eid, name, entityType, string(props),
	)
	return eid, err
}

func (kg *KnowledgeGraph) AddTriple(subject, predicate, obj, validFrom, validTo string, confidence float64) (string, error) {
	subID := entityID(subject)
	objID := entityID(obj)
	pred := strings.ToLower(strings.ReplaceAll(predicate, " ", "_"))

	hash := sha256.Sum256([]byte(time.Now().String()))
	hashStr := fmt.Sprintf("%x", hash)[:12]
	tripleID := fmt.Sprintf("t_%s_%s_%s_%s", subID, pred, objID, hashStr)

	_, err := kg.db.Exec(`
		INSERT OR IGNORE INTO entities (id, name) VALUES (?, ?), (?, ?)
	`, subID, subject, objID, obj)

	if err != nil {
		return "", err
	}

	_, err = kg.db.Exec(`
		INSERT INTO triples (id, subject, predicate, object, valid_from, valid_to, confidence)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, tripleID, subID, pred, objID, validFrom, validTo, confidence)

	return tripleID, err
}

type QueryResult struct {
	Predicate    string  `json:"predicate"`
	Object       string  `json:"object"`
	ValidFrom    string  `json:"valid_from"`
	ValidTo      string  `json:"valid_to"`
	Confidence   float64 `json:"confidence"`
	SourceCloset string  `json:"source_closet"`
}

func (kg *KnowledgeGraph) QueryEntity(name string, asOf string, direction string) ([]QueryResult, error) {
	eid := entityID(name)

	var query string
	var args []any

	switch direction {
	case "outgoing":
		query = `
			SELECT t.predicate, t.object, t.valid_from, t.valid_to, t.confidence, t.source_closet,
			       e.name as obj_name
			FROM triples t JOIN entities e ON t.object = e.id
			WHERE t.subject = ?`
		args = []any{eid}
	case "incoming":
		query = `
			SELECT t.predicate, t.subject, t.valid_from, t.valid_to, t.confidence, t.source_closet,
			       e.name as subj_name
			FROM triples t JOIN entities e ON t.subject = e.id
			WHERE t.object = ?`
		args = []any{eid}
	default:
		query = `
			SELECT t.predicate, t.object, t.valid_from, t.valid_to, t.confidence, t.source_closet,
			       e.name as other_name
			FROM triples t JOIN entities e ON t.object = e.id
			WHERE t.subject = ?`
		args = []any{eid}
	}

	if asOf != "" {
		query += " AND (t.valid_from IS NULL OR t.valid_from <= ?) AND (t.valid_to IS NULL OR t.valid_to >= ?)"
		args = append(args, asOf, asOf)
	}

	rows, err := kg.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []QueryResult
	for rows.Next() {
		var pred, otherName, validFrom, validTo string
		var confidence float64
		var sourceCloset sql.NullString
		var otherNameCol string
		if err := rows.Scan(&pred, &otherName, &validFrom, &validTo, &confidence, &sourceCloset, &otherNameCol); err != nil {
			return nil, err
		}
		result := QueryResult{
			Predicate:  pred,
			Object:     otherNameCol,
			ValidFrom:  validFrom,
			ValidTo:    validTo,
			Confidence: confidence,
		}
		if sourceCloset.Valid {
			result.SourceCloset = sourceCloset.String
		}
		results = append(results, result)
	}
	return results, nil
}

func (kg *KnowledgeGraph) Close() error {
	return kg.db.Close()
}

type KGStats struct {
	EntityCount       int            `json:"entity_count"`
	TripleCount       int            `json:"triple_count"`
	RelationshipTypes map[string]int `json:"relationship_types"`
}

func (kg *KnowledgeGraph) Stats() (*KGStats, error) {
	var entityCount, tripleCount int

	err := kg.db.QueryRow("SELECT COUNT(*) FROM entities").Scan(&entityCount)
	if err != nil {
		return nil, err
	}

	err = kg.db.QueryRow("SELECT COUNT(*) FROM triples").Scan(&tripleCount)
	if err != nil {
		return nil, err
	}

	rows, err := kg.db.Query("SELECT predicate, COUNT(*) as cnt FROM triples GROUP BY predicate ORDER BY cnt DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	relTypes := make(map[string]int)
	for rows.Next() {
		var predicate string
		var count int
		if err := rows.Scan(&predicate, &count); err != nil {
			return nil, err
		}
		relTypes[predicate] = count
	}

	return &KGStats{
		EntityCount:       entityCount,
		TripleCount:       tripleCount,
		RelationshipTypes: relTypes,
	}, nil
}

type TimelineEntry struct {
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
	ValidFrom string `json:"valid_from"`
	ValidTo   string `json:"valid_to"`
}

func (kg *KnowledgeGraph) Timeline(name string) ([]TimelineEntry, error) {
	eid := entityID(name)

	query := `
		SELECT t.predicate, e2.name as obj_name, t.valid_from, t.valid_to
		FROM triples t
		JOIN entities e1 ON t.subject = e1.id
		JOIN entities e2 ON t.object = e2.id
		WHERE t.subject = ? OR t.object = ?
		ORDER BY t.valid_from ASC, t.valid_to ASC`

	rows, err := kg.db.Query(query, eid, eid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []TimelineEntry
	for rows.Next() {
		var entry TimelineEntry
		var objName sql.NullString
		if err := rows.Scan(&entry.Predicate, &objName, &entry.ValidFrom, &entry.ValidTo); err != nil {
			return nil, err
		}
		if objName.Valid {
			entry.Object = objName.String
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (kg *KnowledgeGraph) Invalidate(subject, predicate, obj string, validTo string) error {
	subID := entityID(subject)
	objID := entityID(obj)
	pred := strings.ToLower(strings.ReplaceAll(predicate, " ", "_"))

	query := `
		UPDATE triples
		SET valid_to = ?
		WHERE subject = ? AND predicate = ? AND object = ? AND (valid_to IS NULL OR valid_to >= ?)`

	_, err := kg.db.Exec(query, validTo, subID, pred, objID, validTo)
	return err
}
