package sqlite

import (
	"bytes"
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

// This fixture was created by Python's stdlib sqlite3, not by Open or modernc.
// Copy it so WAL mode and writes cannot alter the checked-in external file.
func TestOpenExternalV1SQLiteFile(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "external_v1.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(fixture, []byte("SQLite format 3\x00")) {
		t.Fatal("external v1 fixture is not a SQLite database")
	}
	path := filepath.Join(t.TempDir(), "external_v1.sqlite")
	if err := os.WriteFile(path, fixture, 0600); err != nil {
		t.Fatal(err)
	}

	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("user_version = %d, want 1", version)
	}
	var content string
	if err := db.QueryRow(`SELECT content FROM legacy_notes WHERE id = 7`).Scan(&content); err != nil {
		t.Fatal(err)
	}
	if content != "written by external sqlite3" {
		t.Fatalf("legacy content = %q", content)
	}
	if _, err := db.Exec(`INSERT INTO legacy_notes(id, content) VALUES (8, 'written by mentle')`); err != nil {
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
	for id, want := range map[int]string{7: "written by external sqlite3", 8: "written by mentle"} {
		var got string
		if err := reopened.QueryRow(`SELECT content FROM legacy_notes WHERE id = ?`, id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("note %d = %q, want %q", id, got, want)
		}
	}
	if err := reopened.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Errorf("reopened user_version = %d, want 1", version)
	}
}

func TestOpenInvalidPathReturnsError(t *testing.T) {
	if db, err := Open(filepath.Join(t.TempDir(), "missing", "db.sqlite3")); err == nil {
		db.Close()
		t.Fatal("expected open error")
	}
}
