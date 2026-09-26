package sqlite

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenUsesExactFilenameWithURIMetacharacters(t *testing.T) {
	files := []string{"data # & spaces.sqlite3"}
	if runtime.GOOS != "windows" {
		files = append(files, "data ? # & spaces.sqlite3")
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			root := t.TempDir()
			parent := filepath.Join(root, "parent # & spaces")
			if err := os.Mkdir(parent, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(parent, file)
			db, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`CREATE TABLE retained (value TEXT); INSERT INTO retained VALUES ('yes')`); err != nil {
				db.Close()
				t.Fatal(err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("requested database path %q: %v", path, err)
			}
			db, err = Open(path)
			if err != nil {
				t.Fatal(err)
			}
			var value string
			if err := db.QueryRow(`SELECT value FROM retained`).Scan(&value); err != nil {
				db.Close()
				t.Fatal(err)
			}
			if value != "yes" {
				t.Errorf("reopened value = %q, want yes", value)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			if err := filepath.WalkDir(root, func(found string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !entry.IsDir() && found != path && found != path+"-wal" && found != path+"-shm" {
					t.Errorf("unexpected sibling database file: %q", found)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestOpenPortableFileAndConnectionPragmas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.sqlite3")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	var journal string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&journal); err != nil {
		t.Fatal(err)
	}
	if journal != "wal" {
		t.Fatalf("journal_mode = %q", journal)
	}
	var fk, timeout int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if fk != 1 {
		t.Fatalf("foreign_keys = %d", fk)
	}
	if err := db.QueryRow(`PRAGMA busy_timeout`).Scan(&timeout); err != nil {
		t.Fatal(err)
	}
	if timeout != 5000 {
		t.Fatalf("busy_timeout = %d", timeout)
	}
	if _, err := db.Exec(`CREATE TABLE legacy (id INTEGER PRIMARY KEY, value TEXT NOT NULL); INSERT INTO legacy(value) VALUES ('kept')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	var value string
	if err := reopened.QueryRow(`SELECT value FROM legacy`).Scan(&value); err != nil {
		t.Fatal(err)
	}
	if value != "kept" {
		t.Fatalf("legacy value = %q", value)
	}
}

func TestOpenInvalidPathReturnsError(t *testing.T) {
	if db, err := Open(filepath.Join(t.TempDir(), "missing", "db.sqlite3")); err == nil {
		db.Close()
		t.Fatal("expected open error")
	}
}
