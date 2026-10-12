// Package registry owns installation control metadata, never memory or Persona content.
package registry

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ProjectViVy/laputa/garden/internal/sqliteconn"
	"github.com/google/uuid"
)

type Store struct {
	mu             sync.Mutex
	db             *sql.DB
	home           string
	installationID string
}

func Open(path, home string) (*Store, error) {
	if !filepath.IsAbs(path) || !filepath.IsAbs(home) {
		return nil, errors.New("registry requires explicit absolute paths")
	}
	canonical, err := CanonicalPath(home)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := sqliteconn.Open(path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db, home: canonical}
	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS registry_meta(key TEXT PRIMARY KEY,value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS profiles(profile_id TEXT PRIMARY KEY,authority_dir TEXT NOT NULL UNIQUE,installation_id TEXT NOT NULL,display_name TEXT NOT NULL);`); err != nil {
		return nil, errors.Join(err, db.Close())
	}
	if _, err = db.Exec(`INSERT OR IGNORE INTO registry_meta(key,value) VALUES('installation_id',?)`, uuid.NewString()); err != nil {
		return nil, errors.Join(err, db.Close())
	}
	if err = db.QueryRow(`SELECT value FROM registry_meta WHERE key='installation_id'`).Scan(&s.installationID); err != nil {
		return nil, errors.Join(err, db.Close())
	}
	return s, nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}
func (s *Store) InstallationID() string { return s.installationID }

// CanonicalPath resolves existing ancestors too, so a missing leaf beneath a
// symlink cannot become a second authority/lock identity.
func CanonicalPath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("path must be absolute")
	}
	path = filepath.Clean(path)
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	parent := filepath.Dir(path)
	if parent == path {
		return "", fmt.Errorf("cannot resolve path: %w", err)
	}
	resolved, err = CanonicalPath(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolved, filepath.Base(path)), nil
}
