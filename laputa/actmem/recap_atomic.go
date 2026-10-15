package actmem

import (
	"context"
	"os"
	"path/filepath"
)

func (s *Store) recapBoundary(boundary string) error {
	if s.recapFault != nil {
		return s.recapFault(boundary)
	}
	return nil
}

func (s *Store) persistRecap(ctx context.Context, head RecapHead) error {
	raw, err := encodeRecap(head)
	if err != nil {
		return err
	}
	historyDir := filepath.Join(s.Root(), "recap-history")
	if err := os.MkdirAll(historyDir, 0700); err != nil {
		return recapUnavailable("cannot create Recap history", err)
	}
	// Confirm parent entries for newly created ACTMEM/history directories before
	// making the immutable snapshot eligible for head publication.
	if err := recapSyncDirectory(filepath.Dir(s.Root())); err != nil {
		return recapUnavailable("cannot sync ACTMEM parent", err)
	}
	if err := recapSyncDirectory(s.Root()); err != nil {
		return recapUnavailable("cannot sync ACTMEM directory", err)
	}
	if err := s.recapBoundary("before_revision"); err != nil {
		return recapUnavailable("before revision publication", err)
	}
	if err := recapWriteRevision(s.recapRevisionPath(head), raw); err != nil {
		return recapUnavailable("cannot publish Recap revision", err)
	}
	if err := s.recapBoundary("after_revision"); err != nil {
		return recapUnavailable("after revision publication", err)
	}
	if err := s.publishRecapHead(ctx, raw, ""); err != nil {
		return err
	}
	s.recapConfirmed = true
	s.recapConfirmedDigest = head.Digest
	return nil
}

// publishRecapHead is shared by new commits and identical-byte recovery. Its
// platform replacement/flush protocol is the only positive receipt barrier.
func (s *Store) publishRecapHead(ctx context.Context, raw []byte, prefix string) error {
	temporary, err := recapWriteTemp(s.Root(), raw)
	if err != nil {
		return recapUnavailable("cannot stage Recap head", err)
	}
	defer os.Remove(temporary)
	if err := s.recapBoundary(prefix + "before_replace"); err != nil {
		return recapUnavailable("before head replacement", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	unknown := func(cause error) error {
		s.recapUnknown = true
		return newError("recap_commit_unknown", "head replacement attempted; reopen and resolve operation identity", cause)
	}
	if err := recapReplace(temporary, s.HeadPath()); err != nil {
		return unknown(err)
	}
	if err := s.recapBoundary(prefix + "after_replace"); err != nil {
		return unknown(err)
	}
	if err := recapSyncFile(s.HeadPath()); err != nil {
		return unknown(err)
	}
	if err := s.recapBoundary(prefix + "before_directory_sync"); err != nil {
		return unknown(err)
	}
	if err := recapSyncDirectory(s.Root()); err != nil {
		return unknown(err)
	}
	if err := s.recapBoundary(prefix + "after_directory_sync"); err != nil {
		return unknown(err)
	}
	return nil
}

// recoverRecap republishes the exact selected, validated head bytes. Recovery
// never allocates a logical revision, changes a digest, or discovers an orphan.
func (s *Store) recoverRecap(ctx context.Context, binding RecapBinding, history []RecapHead) error {
	if len(history) == 0 {
		if err := s.recapBoundary("recovery_before_absence_sync"); err != nil {
			return err
		}
		if err := recapSyncDirectory(filepath.Dir(s.Root())); err != nil {
			return err
		}
		if err := recapSyncDirectory(s.Root()); err != nil {
			return err
		}
		// All supported writers hold this same owner. Revalidate after the
		// namespace barrier; ENOENT observation by itself is not a receipt.
		selected, err := s.readRecapChain(ctx, binding)
		if err != nil {
			return err
		}
		if len(selected) != 0 {
			return recapUnavailable("selected head changed during absence recovery", nil)
		}
		return s.recapBoundary("recovery_after_absence_sync")
	}
	raw, err := os.ReadFile(s.HeadPath())
	if err != nil {
		return err
	}
	selected, err := decodeRecap(raw)
	if err != nil {
		return err
	}
	if selected != history[0] {
		return recapUnavailable("selected head changed during recovery", nil)
	}
	// Reachable history was published before the selected head, and is validated
	// above. Reconfirm its file buffers and containing directory as part of this
	// owner's recovery; do not rewrite or promote a different revision file.
	for _, revision := range history {
		if err := recapSyncFile(s.recapRevisionPath(revision)); err != nil {
			return err
		}
	}
	if err := recapSyncDirectory(filepath.Join(s.Root(), "recap-history")); err != nil {
		return err
	}
	if err := recapSyncDirectory(filepath.Dir(s.Root())); err != nil {
		return err
	}
	return s.publishRecapHead(ctx, raw, "recovery_")
}

func recapWriteTemp(directory string, raw []byte) (string, error) {
	file, err := os.CreateTemp(directory, ".recap-*")
	if err != nil {
		return "", err
	}
	path := file.Name()
	complete := false
	defer func() {
		_ = file.Close()
		if !complete {
			_ = os.Remove(path)
		}
	}()
	if err := file.Chmod(0600); err != nil {
		return "", err
	}
	if _, err := file.Write(raw); err != nil {
		return "", err
	}
	if err := file.Sync(); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	complete = true
	return path, nil
}

func recapWriteRevision(path string, raw []byte) error {
	temporary, err := recapWriteTemp(filepath.Dir(path), raw)
	if err != nil {
		return err
	}
	defer os.Remove(temporary)
	if err := recapPublishRevision(temporary, path); err != nil {
		existing, readErr := os.ReadFile(path)
		if readErr != nil || string(existing) != string(raw) {
			return err
		}
	}
	if err := recapSyncFile(path); err != nil {
		return err
	}
	return recapSyncDirectory(filepath.Dir(path))
}

func recapSyncFile(path string) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	err = file.Sync()
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}
