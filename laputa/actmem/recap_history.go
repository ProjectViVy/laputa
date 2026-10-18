package actmem

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

func (s *Store) recapRevisionPath(head RecapHead) string {
	return filepath.Join(s.Root(), "recap-history", fmt.Sprintf("%020d-%s.md", head.Revision, head.Digest))
}

// recapChain serves authoritative C1 state only after the same exclusive owner
// validates and confirms its selected namespace. The unknown bit is deliberately
// not cleared by observation: an uncertain live owner must Close/reopen.
func (s *Store) recapChain(ctx context.Context, binding RecapBinding, allowUninitialized bool) ([]RecapHead, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !validRecapBinding(binding) {
		return nil, newError("recap_scope_denied", "subject and domain are required", nil)
	}
	if err := s.acquireOwner(); err != nil {
		return nil, err
	}
	if s.recapUnknown {
		return nil, newError("recap_commit_unknown", "reopen ACTMEM and durably recover before resolving operation identity", nil)
	}
	history, err := s.readRecapChain(ctx, binding)
	if err != nil {
		return nil, err
	}
	selectedDigest := ""
	if len(history) > 0 {
		selectedDigest = history[0].Digest
	}
	if s.recapConfirmed && s.recapConfirmedDigest == selectedDigest {
		return history, nil
	}
	if len(history) == 0 && !recapSupportsAbsenceConfirmation {
		// Win32 lacks our required qualified absence barrier. Permit only an
		// uninitialized current view or publication of an actual requested initial
		// head; never turn a bare missing name into a negative operation receipt.
		if allowUninitialized {
			return history, nil
		}
		return nil, newError("recap_commit_unknown", "missing Recap head cannot establish durable negative resolution on this platform", nil)
	}
	if err := s.recoverRecap(ctx, binding, history); err != nil {
		// A canceled or expired context stops recovery before any head
		// replacement is attempted (publishRecapHead marks the owner unknown
		// itself once a replacement starts). Nothing is uncertain, so the next
		// call simply recovers again instead of wedging the owner until reopen.
		if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
			return nil, err
		}
		s.recapUnknown = true
		return nil, newError("recap_commit_unknown", "Recap recovery durability is unconfirmed; reopen and retry recovery", err)
	}
	s.recapConfirmed = true
	s.recapConfirmedDigest = selectedDigest
	return history, nil
}

// readRecapChain follows only digests reachable from ACTMEM.MD. An orphan revision
// is never discovered, returned, replayed, or promoted, including after restart.
func (s *Store) readRecapChain(ctx context.Context, binding RecapBinding) ([]RecapHead, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.closed {
		return nil, recapUnavailable("ACTMEM owner is closed", nil)
	}
	if !validRecapBinding(binding) {
		return nil, newError("recap_scope_denied", "subject and domain are required", nil)
	}
	raw, err := os.ReadFile(s.HeadPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, recapUnavailable("cannot read Recap head", err)
	}
	if !declaresV3Schema(string(raw)) {
		legacy, err := loadHead(string(raw))
		if err != nil {
			return nil, recapUnavailable("cannot read legacy ACTMEM", err)
		}
		doc := legacy.projection()
		if len(legacy.entries()) != 0 || doc.Pulse != "" || doc.Recap != "" || doc.Work != emptyWork() {
			return nil, newError("recap_legacy_nonempty", "nonempty legacy ACTMEM is preserved without conversion", nil)
		}
		return nil, nil
	}
	current, err := decodeRecap(raw)
	if err != nil {
		return nil, err
	}
	if current.Binding != binding {
		return nil, newError("recap_scope_denied", "Recap belongs to another subject or domain", nil)
	}
	var history []RecapHead
	operations := map[string]bool{}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if operations[current.OperationID] {
			return nil, recapUnavailable("duplicate reachable operation identity", nil)
		}
		operations[current.OperationID] = true
		archive, err := os.ReadFile(s.recapRevisionPath(current))
		if err != nil {
			return nil, recapUnavailable("cannot read reachable Recap revision", err)
		}
		stored, err := decodeRecap(archive)
		if err != nil {
			return nil, err
		}
		if stored != current {
			return nil, recapUnavailable("head and immutable revision differ", nil)
		}
		history = append(history, current)
		if current.Revision == 1 {
			if err := validateRecapTransition(RecapHead{Binding: binding}, current); err != nil {
				return nil, err
			}
			break
		}
		archive, err = os.ReadFile(s.recapRevisionPath(RecapHead{Revision: current.Revision - 1, Digest: current.PreviousDigest}))
		if err != nil {
			return nil, recapUnavailable("Recap digest chain is incomplete", err)
		}
		previous, err := decodeRecap(archive)
		if err != nil {
			return nil, err
		}
		if err := validateRecapTransition(previous, current); err != nil {
			return nil, err
		}
		current = previous
	}
	for _, h := range history {
		if h.Cause != "restore" {
			continue
		}
		restored := false
		for _, source := range history {
			if source.Revision == h.RestoredFrom && source.Text == h.Text {
				restored = true
				break
			}
		}
		if !restored {
			return nil, recapUnavailable("restore text differs from its reachable source", nil)
		}
	}
	return history, nil
}

func validateRecapTransition(previous, current RecapHead) error {
	fail := func() error { return recapUnavailable("invalid Recap digest chain transition", nil) }
	if previous.Binding != current.Binding || previous.Revision == math.MaxUint64 || current.Revision != previous.Revision+1 || current.PreviousDigest != previous.Digest {
		return fail()
	}
	epoch := previous.InvalidationEpoch
	if current.Cause == "clear" {
		if epoch == math.MaxUint64 {
			return fail()
		}
		epoch++
	}
	if current.InvalidationEpoch != epoch {
		return fail()
	}
	if current.Cause == "summary" {
		if current.Sources.After != previous.CoveredThrough || current.CoveredThrough != current.Sources.Through {
			return fail()
		}
	} else if current.CoveredThrough != previous.CoveredThrough {
		return fail()
	}
	if current.Cause == "reconcile" && (current.Sources.After != previous.CoveredThrough || current.Sources.Through != previous.CoveredThrough) {
		return fail()
	}
	request := RecapCommit{Binding: current.Binding, BaseRevision: previous.Revision, OperationID: current.OperationID, Text: current.Text, Cause: current.Cause, Sources: current.Sources, RestoredFrom: current.RestoredFrom}
	if current.Cause == "restore" {
		request.Text = ""
	}
	if recapHash(request) != current.OperationDigest {
		return fail()
	}
	return nil
}
