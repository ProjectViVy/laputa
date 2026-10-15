package actmem

import (
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// acquireOwner is called under mu (or before an Open Store is published).
// New stays filesystem-free; its first mutation acquires the same lifetime lock.
func (s *Store) acquireOwner() error {
	if s.closed {
		return newError("recap_unavailable", "ACTMEM owner is closed", nil)
	}
	if s.owner != nil {
		return nil
	}
	root, err := filepath.Abs(s.Root())
	if err != nil {
		return newError("recap_unavailable", "cannot resolve ACTMEM directory", err)
	}
	if err = os.MkdirAll(root, 0700); err != nil {
		return newError("recap_unavailable", "cannot create ACTMEM directory", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return newError("recap_unavailable", "cannot canonicalize ACTMEM directory", err)
	}
	lock := flock.New(filepath.Join(root, ".owner.lock"))
	locked, err := lock.TryLock()
	if err != nil {
		_ = lock.Close()
		return newError("recap_unavailable", "cannot lock ACTMEM directory", err)
	}
	if !locked {
		_ = lock.Close()
		return newError("recap_owner_busy", "ACTMEM directory already has an owner", nil)
	}
	s.pathMu.Lock()
	s.root = root
	s.pathMu.Unlock()
	s.owner = lock
	return nil
}

// Close releases ownership. A closed Store cannot mutate or reacquire ownership.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.owner != nil {
		return s.owner.Close()
	}
	return nil
}

func (s *Store) loadForMutation() (head, error) {
	if err := s.acquireOwner(); err != nil {
		return head{}, err
	}
	if s.recapUnknown {
		return head{}, newError("recap_commit_unknown", "reopen ACTMEM to resolve the attempted commit", nil)
	}
	return s.load()
}
