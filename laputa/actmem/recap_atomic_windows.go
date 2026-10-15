//go:build windows

package actmem

import "golang.org/x/sys/windows"

func recapMove(from, to string, flags uint32) error {
	source, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(source, target, flags|windows.MOVEFILE_WRITE_THROUGH)
}

// Both paths are in the same directory. Replacement never unlinks the old head
// first and never requests a cross-volume copy/delete fallback.
func recapReplace(from, to string) error {
	return recapMove(from, to, windows.MOVEFILE_REPLACE_EXISTING)
}
func recapPublishRevision(from, to string) error { return recapMove(from, to, 0) }

// Win32 has no portable POSIX directory-fsync equivalent. Metadata publication
// requests MOVEFILE_WRITE_THROUGH; file buffers are flushed on both sides of it.
// Native Windows filesystem/crash acceptance remains necessary.
func recapSyncDirectory(string) error { return nil }
