//go:build windows

package renameio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTempFileBestEffortReplace(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "index.bin")
	if err := os.WriteFile(destination, []byte("old index"), 0o600); err != nil {
		t.Fatal(err)
	}

	pending, err := TempFile("", destination)
	if err != nil {
		t.Fatal(err)
	}
	defer pending.Cleanup()
	if _, err := pending.Write([]byte("new index")); err != nil {
		t.Fatal(err)
	}
	if err := pending.CloseAtomicallyReplace(); err != nil {
		t.Fatal(err)
	}
	if err := pending.Cleanup(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(destination)
	if err != nil || string(got) != "new index" {
		t.Fatalf("destination=%q err=%v", got, err)
	}
}

func TestTempFileCleanupRemovesUncommittedFile(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "index.bin")
	pending, err := TempFile("", destination)
	if err != nil {
		t.Fatal(err)
	}
	tempPath := pending.Name()
	if err := pending.Cleanup(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("pending path remains after cleanup: %v", err)
	}
}
