package sqliteconn

import (
	"database/sql"
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

func TestOpenConfiguresEveryConnectionAndPersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "garden.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	check := func(db *sql.DB) {
		t.Helper()
		for pragma, want := range map[string]int{"journal_mode": 0, "busy_timeout": 5000, "foreign_keys": 1} {
			if pragma == "journal_mode" {
				var mode string
				if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
					t.Fatal(err)
				}
				if mode != "wal" {
					t.Fatalf("journal_mode=%q, want wal", mode)
				}
				continue
			}
			var got int
			if err := db.QueryRow("PRAGMA " + pragma).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("%s=%d, want %d", pragma, got, want)
			}
		}
	}
	check(db)
	if _, err := db.Exec(`CREATE TABLE parent(id INTEGER PRIMARY KEY); CREATE TABLE child(parent_id INTEGER REFERENCES parent(id)); INSERT INTO parent(id) VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO child(parent_id) VALUES (999)`); err == nil {
		t.Fatal("foreign key violation was accepted")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	check(db)
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM parent`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("persisted rows=%d, want 1", count)
	}
}
