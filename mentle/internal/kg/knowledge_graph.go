// Package kg provides a knowledge graph for storing entities and triples.
// It implements a simple RDF-like store with temporal validity.
package kg

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dashimaki/mentle/storage/sqlite"
)

var ErrInvalidTemporalInterval = errors.New("kg: inverted temporal interval (valid_to < valid_from)")

type Entity struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Properties map[string]string `json:"properties"`
	CreatedAt  time.Time         `json:"created_at"`
}

type Triple struct {
	ID             string    `json:"id"`
	Subject        string    `json:"subject"`
	Predicate      string    `json:"predicate"`
	Object         string    `json:"object"`
	ValidFrom      string    `json:"valid_from"`
	ValidTo        string    `json:"valid_to"`
	Confidence     float64   `json:"confidence"`
	SourceCloset   string    `json:"source_closet"`
	SourceFile     string    `json:"source_file"`
	SourceDrawerID string    `json:"source_drawer_id"`
	AdapterName    string    `json:"adapter_name"`
	AdapterVersion string    `json:"adapter_version"`
	ExtractedAt    time.Time `json:"extracted_at"`
}

type TripleInput struct {
	Subject        string
	Predicate      string
	Object         string
	ValidFrom      string
	ValidTo        string
	Confidence     float64
	SourceCloset   string
	SourceFile     string
	SourceDrawerID string
	AdapterName    string
	AdapterVersion string
	ExtractedAt    time.Time
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
		return nil, errors.Join(err, db.Close())
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
	if err != nil {
		return err
	}

	// Schema migrations
	cols := []string{
		"ALTER TABLE triples ADD COLUMN source_drawer_id TEXT;",
		"ALTER TABLE triples ADD COLUMN adapter_name TEXT;",
		"ALTER TABLE triples ADD COLUMN adapter_version TEXT;",
	}
	for _, query := range cols {
		kg.db.Exec(query)
	}

	_, err = kg.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_triples_source_drawer ON triples(source_drawer_id);
		CREATE INDEX IF NOT EXISTS idx_triples_adapter ON triples(adapter_name);
		CREATE INDEX IF NOT EXISTS idx_triples_extracted ON triples(extracted_at);
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

func (kg *KnowledgeGraph) AddTriple(input TripleInput) (string, error) {
	if input.ValidFrom != "" {
		if _, err := time.Parse(time.RFC3339, input.ValidFrom); err != nil {
			return "", fmt.Errorf("invalid ValidFrom: %w", err)
		}
	}
	if input.ValidTo != "" {
		if _, err := time.Parse(time.RFC3339, input.ValidTo); err != nil {
			return "", fmt.Errorf("invalid ValidTo: %w", err)
		}
	}
	if input.ValidFrom != "" && input.ValidTo != "" {
		if input.ValidTo < input.ValidFrom {
			return "", ErrInvalidTemporalInterval
		}
	}

	if input.ExtractedAt.IsZero() {
		input.ExtractedAt = time.Now().UTC()
	}

	subID := entityID(input.Subject)
	objID := entityID(input.Object)
	pred := strings.ToLower(strings.ReplaceAll(input.Predicate, " ", "_"))

	hashData := fmt.Sprintf("%s%s%s%s", input.Subject, input.Predicate, input.Object, input.ValidFrom)
	hash := sha256.Sum256([]byte(hashData))
	hashStr := fmt.Sprintf("%x", hash)[:12]
	tripleID := fmt.Sprintf("t_%s_%s_%s_%s", subID, pred, objID, hashStr)

	_, err := kg.db.Exec(`
		INSERT OR IGNORE INTO entities (id, name) VALUES (?, ?), (?, ?)
	`, subID, input.Subject, objID, input.Object)

	if err != nil {
		return "", err
	}

	_, err = kg.db.Exec(`
		INSERT INTO triples (id, subject, predicate, object, valid_from, valid_to, confidence, source_closet, source_file, source_drawer_id, adapter_name, adapter_version, extracted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tripleID, subID, pred, objID, input.ValidFrom, input.ValidTo, input.Confidence, input.SourceCloset, input.SourceFile, input.SourceDrawerID, input.AdapterName, input.AdapterVersion, input.ExtractedAt.Format(time.RFC3339))

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
	if asOf != "" {
		if _, err := time.Parse(time.RFC3339, asOf); err != nil {
			return nil, fmt.Errorf("invalid asOf: %w", err)
		}
	}

	eid := entityID(name)

	var query string
	var args []any

	baseOutgoing := `
		SELECT t.predicate, t.object as other_id, t.valid_from, t.valid_to, t.confidence, t.source_closet, e.name as other_name
		FROM triples t JOIN entities e ON t.object = e.id
		WHERE t.subject = ?`
	baseIncoming := `
		SELECT t.predicate, t.subject as other_id, t.valid_from, t.valid_to, t.confidence, t.source_closet, e.name as other_name
		FROM triples t JOIN entities e ON t.subject = e.id
		WHERE t.object = ?`

	switch direction {
	case "outgoing":
		query = "SELECT * FROM (" + baseOutgoing + ") q"
		args = []any{eid}
	case "incoming":
		query = "SELECT * FROM (" + baseIncoming + ") q"
		args = []any{eid}
	default:
		query = "SELECT * FROM (" + baseOutgoing + " UNION ALL " + baseIncoming + ") q"
		args = []any{eid, eid}
	}

	if asOf != "" {
		query += " WHERE (valid_from IS NULL OR valid_from <= ?) AND (valid_to IS NULL OR valid_to >= ?)"
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
