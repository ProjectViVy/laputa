// Package sqlite provides pure-Go SQLite database connectivity.
package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Open configures a single connection so the connection-scoped pragmas remain
// effective for every operation, including transactions.
func Open(path string) (*sql.DB, error) { return open(path, false) }

// OpenReadOnly opens an existing SQLite database without changing its journal
// mode. Both SQLite's file mode and the connection's query_only pragma deny writes.
func OpenReadOnly(path string) (*sql.DB, error) { return open(path, true) }

func open(path string, readOnly bool) (*sql.DB, error) {
	name := filepath.ToSlash(path)
	if path == ":memory:" {
		name = ":memory:"
	}
	params := url.Values{"_pragma": {"busy_timeout(5000)", "foreign_keys(ON)"}}
	if readOnly {
		params.Set("mode", "ro")
		params.Add("_pragma", "query_only(ON)")
	} else {
		params.Add("_pragma", "journal_mode(WAL)")
	}
	dsn := "file:" + (&url.URL{Path: name}).EscapedPath() + "?" + params.Encode()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	if err = db.Ping(); err != nil {
		return nil, errors.Join(fmt.Errorf("open sqlite %s: %w", path, err), db.Close())
	}
	return db, nil
}
