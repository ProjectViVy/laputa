package actmem

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"

	"github.com/ProjectViVy/laputa/laputa/evolution"
)

// AppendSnapshot identifies one canonical head before a host system append.
// It is a precondition, not authority or an effect receipt.
type AppendSnapshot struct {
	Revision uint64 `json:"revision"`
	SHA256   string `json:"sha256"`
}

func snapshotOf(h head) AppendSnapshot {
	sum := sha256.Sum256([]byte(h.markdown))
	return AppendSnapshot{Revision: h.revision, SHA256: hex.EncodeToString(sum[:])}
}

func (s *Store) AppendSnapshot() (AppendSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, err := s.load()
	if err != nil {
		return AppendSnapshot{}, err
	}
	if !h.classified() {
		return AppendSnapshot{}, newError("actmem_unclassified_head", "unclassified ACTMEM refuses captured activity", nil)
	}
	return snapshotOf(h), nil
}

func prepareCaptured(entries []evolution.Entry) ([]evolution.Entry, error) {
	if len(entries) != 2 || entries[0].Section != evolution.SectionPulse || entries[1].Section != evolution.SectionRecap {
		return nil, newError("actmem_invalid_edit", "capture requires one Pulse/Recap pair", nil)
	}
	pair := append([]evolution.Entry(nil), entries...)
	first := pair[0]
	if first.SessionID == "" || first.EventID == "" || len(first.Sources) == 0 {
		return nil, newError("actmem_invalid_edit", "capture identity and sources required", nil)
	}
	for i := range pair {
		e := &pair[i]
		if err := e.Scope.Validate(); err != nil {
			return nil, newError("actmem_invalid_edit", "capture scope invalid", err)
		}
		if !e.Scope.Equal(first.Scope) || e.SessionID != first.SessionID || e.EventID != first.EventID || !reflect.DeepEqual(e.Sources, first.Sources) || e.Field != "" {
			return nil, newError("actmem_invalid_edit", "capture pair identity differs", nil)
		}
		for _, ref := range e.Sources {
			if err := ref.Validate(); err != nil {
				return nil, newError("actmem_invalid_edit", "capture source invalid", err)
			}
		}
		e.Body = normalizeSection(e.Body)
		cap := PulseItemCapChars
		if e.Section == evolution.SectionRecap {
			cap = RecapItemCapChars
		}
		// Captured entries must remain archivable with their complete provenance.
		// IDs and UTC timestamps have fixed native widths; digest width is fixed.
		sizing := *e
		sizing.ID = "e_" + strings.Repeat("0", 32)
		sizing.OccurredAt = "2000-01-01T00:00:00.000Z"
		sizing.Body = ""
		overhead := len([]rune(renderFoldCapsule(e.SessionID, []evolution.ActmemEntry{storedFromEntry(sizing)})))
		if available := ACTMEMCapsuleCap - overhead; available < cap {
			cap = available
		}
		if cap <= 0 {
			return nil, newError("actmem_cap_exceeded", "captured provenance leaves no capsule body capacity", nil)
		}
		e.Body = truncateChars(e.Body, cap)
		if strings.TrimSpace(e.Body) == "" {
			return nil, newError("actmem_invalid_edit", "capture body required", nil)
		}
		if err := rejectReservedDelimiters(e.Body); err != nil {
			return nil, err
		}
	}
	return pair, nil
}

// LookupCaptured rejoins exact native entries in head or recorded capsules.
// Missing entries alone never prove that an effect did not happen.
func (s *Store) LookupCaptured(entries []evolution.Entry) (evolution.ActivityResult, bool, error) {
	pair, err := prepareCaptured(entries)
	if err != nil {
		return evolution.ActivityResult{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	h, err := s.load()
	if err != nil {
		return evolution.ActivityResult{}, false, err
	}
	return s.lookupCaptured(h, pair, true)
}

func (s *Store) lookupCaptured(h head, pair []evolution.Entry, includeArchive bool) (evolution.ActivityResult, bool, error) {
	if !h.classified() {
		return evolution.ActivityResult{}, false, newError("actmem_unclassified_head", "unclassified ACTMEM refuses receipt lookup", nil)
	}
	candidates := h.entries()
	// Archive scanning belongs to recovery, never the fresh append path.
	if includeArchive {
		capsules, err := s.sessionCapsules(pair[0].SessionID)
		if err != nil {
			return evolution.ActivityResult{}, false, err
		}
		for _, capsule := range capsules {
			candidates = append(candidates, entriesOf(capsule.sources)...)
		}
	}
	found := make([]evolution.Entry, len(pair))
	count := 0
	for i, want := range pair {
		for _, entry := range candidates {
			if entry.Section != want.Section || entry.EventID != want.EventID {
				continue
			}
			if entry.SessionID != want.SessionID || !entry.Scope.Equal(want.Scope) || entry.Body != want.Body || !reflect.DeepEqual(entry.Sources, want.Sources) {
				return evolution.ActivityResult{}, false, newError("actmem_capture_conflict", "original captured activity identity or payload differs", nil)
			}
			if found[i].ID != "" && found[i].ID != entry.ID {
				return evolution.ActivityResult{}, false, newError("actmem_capture_conflict", "multiple native entries claim one capture", nil)
			}
			if found[i].ID == "" {
				count++
			}
			found[i] = entry
		}
	}
	if count == 0 {
		return evolution.ActivityResult{Revision: h.revision, Entries: []evolution.Entry{}}, false, nil
	}
	if count != len(pair) {
		return evolution.ActivityResult{}, false, newError("actmem_recovery_required", "partial captured activity cannot be replayed", nil)
	}
	return evolution.ActivityResult{Revision: h.revision, Entries: found}, true, nil
}

// AppendCaptured admits and commits both ring entries in one head revision.
// Its caller persists intent/receipts separately and retains unknown writes.
func (s *Store) AppendCaptured(entries []evolution.Entry, before AppendSnapshot) (evolution.ActivityResult, error) {
	pair, err := prepareCaptured(entries)
	if err != nil {
		return evolution.ActivityResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	h, err := s.load()
	if err != nil {
		return evolution.ActivityResult{}, err
	}
	replay, found, err := s.lookupCaptured(h, pair, false)
	if err != nil || found {
		return replay, err
	}
	if snapshotOf(h) != before {
		return evolution.ActivityResult{}, revisionConflict(before.Revision, h.revision)
	}
	for i := range pair {
		pair[i].ID, err = newEntryID()
		if err != nil {
			return evolution.ActivityResult{}, newError("actmem_storage_error", "cannot allocate captured entry", err)
		}
		pair[i].OccurredAt = s.now().Format("2006-01-02T15:04:05.000Z07:00")
		ring := append(h.doc.Sections[pair[i].Section], storedFromEntry(pair[i]))
		for sectionChars(ring) > ACTMEMRingCapChars && len(ring) > 1 {
			ring = ring[1:]
		}
		h.doc.Sections[pair[i].Section] = ring
	}
	if err := s.commitHead(&h); err != nil {
		return evolution.ActivityResult{}, err
	}
	return evolution.ActivityResult{Changed: true, Revision: h.revision, Entries: pair}, nil
}
