package hosts

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func hash(body string) string { return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(body))) }
func readFile(path string) (string, bool, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return "", false, errors.New("clean absolute host path required")
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if !info.Mode().IsRegular() {
		return "", false, errors.New("host target must be regular, not a symlink")
	}
	if info.Size() > 1<<20 {
		return "", false, errors.New("host file exceeds 1 MiB")
	}
	raw, err := os.ReadFile(path)
	return string(raw), true, err
}
func managedBlock(id, body string) string {
	return "<!-- Garden BEGIN binding=" + id + " -->\n" + body + "<!-- Garden END binding=" + id + " -->\n"
}
func managedRegion(body, id string) (int, int, error) {
	begin := "<!-- Garden BEGIN binding=" + id + " -->"
	end := "<!-- Garden END binding=" + id + " -->"
	if strings.Count(body, "<!-- Garden BEGIN") != strings.Count(body, begin) || strings.Count(body, "<!-- Garden END") != strings.Count(body, end) {
		return 0, 0, failure("conflict", "malformed or foreign managed markers")
	}
	if strings.Count(body, begin) == 0 && strings.Count(body, end) == 0 {
		return -1, -1, nil
	}
	if strings.Count(body, begin) != 1 || strings.Count(body, end) != 1 {
		return 0, 0, failure("conflict", "duplicate managed markers")
	}
	a, z := strings.Index(body, begin), strings.Index(body, end)
	if z < a {
		return 0, 0, failure("conflict", "reversed managed markers")
	}
	z += len(end)
	if z < len(body) && body[z] == '\n' {
		z++
	}
	return a, z, nil
}

func backup(dir, body string) (string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, "original-*")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if err = f.Chmod(0600); err == nil {
		_, err = f.WriteString(body)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		os.Remove(name)
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	return name, nil
}
func replace(path, body string) error {
	mode := os.FileMode(0600)
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return failure("conflict", "target is no longer a regular file")
		}
		mode = info.Mode().Perm()
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".garden-projection-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(mode); err == nil {
		_, err = f.WriteString(body)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, path)
}
