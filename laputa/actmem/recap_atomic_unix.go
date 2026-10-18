//go:build !windows

package actmem

import "os"

const recapSupportsAbsenceConfirmation = true

func recapReplace(from, to string) error { return os.Rename(from, to) }

// The exclusive hard link cannot overwrite an immutable revision. Its staging
// name is later removed; the revision itself is never cleaned up or rewritten.
func recapPublishRevision(from, to string) error { return os.Link(from, to) }
func recapSyncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	err = directory.Sync()
	closeErr := directory.Close()
	if err != nil {
		return err
	}
	return closeErr
}
