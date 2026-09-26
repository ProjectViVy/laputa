// Package sqliteconn opens Garden's local state databases with consistent SQLite settings.
package sqliteconn

import (
	"database/sql"
	"net/url"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Open returns a single-connection pool so connection-local pragmas remain in force.
// Callers own the returned DB and their domain schema.
func Open(path string) (*sql.DB, error) {
	pragmas := url.Values{}
	pragmas.Add("_pragma", "busy_timeout(5000)")
	pragmas.Add("_pragma", "journal_mode(WAL)")
	pragmas.Add("_pragma", "foreign_keys(1)")
	db, err := sql.Open("sqlite", "file:"+(&url.URL{Path: filepath.ToSlash(path)}).EscapedPath()+"?"+pragmas.Encode())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
