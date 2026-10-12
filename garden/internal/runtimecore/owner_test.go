package runtimecore

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestOwnerLockAcrossProcesses(t *testing.T) {
	if dir := os.Getenv("GARDEN_TEST_LOCK_DIR"); dir != "" {
		lock, err := acquireOwnership(dir)
		if err != nil {
			if strings.Contains(err.Error(), "owner_conflict") {
				os.Exit(23)
			}
			t.Fatal(err)
		}
		defer lock.Close()
		return
	}
	dir := filepath.Join(t.TempDir(), "authority")
	_ = os.Mkdir(dir, 0700)
	lock, err := acquireOwnership(dir)
	if err != nil {
		t.Fatal(err)
	}
	child := func() error {
		cmd := exec.Command(os.Args[0], "-test.run=^TestOwnerLockAcrossProcesses$")
		cmd.Env = append(os.Environ(), "GARDEN_TEST_LOCK_DIR="+dir)
		return cmd.Run()
	}
	err = child()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 23 {
		t.Fatalf("second process acquired owner: %v", err)
	}
	alias := filepath.Join(filepath.Dir(dir), "alias")
	if err = os.Symlink(dir, alias); err != nil {
		t.Fatal(err)
	}
	if _, err = acquireOwnership(alias); err == nil {
		t.Fatal("alias bypassed owner lock")
	}
	if err = lock.Close(); err != nil {
		t.Fatal(err)
	}
	if err = child(); err != nil {
		t.Fatalf("lock not released: %v", err)
	}
}

func TestOwnerLockPartialAcquisitionReleases(t *testing.T) {
	a, b := filepath.Join(t.TempDir(), "a"), filepath.Join(t.TempDir(), "b")
	held, err := acquireOwnership(b)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	if _, err = acquireOwnership(a, b); err == nil {
		t.Fatal("partial acquisition ignored conflict")
	}
	lock, err := acquireOwnership(a)
	if err != nil {
		t.Fatal("partial open leaked lock", err)
	}
	_ = lock.Close()
}
