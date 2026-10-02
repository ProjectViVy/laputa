package actmem

import (
	"github.com/dashimaki/laputa/evolution"
)

// Scope visibility (spec section 4): a caller's admitted union is the
// personal scope plus their current workspace. Entries carry trusted scope
// metadata; nothing else may widen or relabel it.

// visibleTo reports whether an entry scoped by entryScope is inside the
// caller's admitted union. Personal entries are subject-visible; workspace
// entries only ever reach the same workspace.
func visibleTo(entryScope evolution.Scope, caller evolution.Scope) bool {
	if entryScope.SubjectID == "" || entryScope.SubjectID != caller.SubjectID {
		return false
	}
	switch entryScope.Kind {
	case evolution.ScopePersonal:
		return true
	case evolution.ScopeWorkspace:
		return caller.Kind == evolution.ScopeWorkspace && caller.WorkspaceID != "" && caller.WorkspaceID == entryScope.WorkspaceID
	default:
		return false
	}
}

// ReadScoped returns the revision plus the entries visible to the caller's
// admitted scope. Unclassified (legacy) content and other scopes' text, ids
// and counts stay absent — the revision is a liveness hint, not a lookup key.
func (s *Store) ReadScoped(caller evolution.Scope, req evolution.ReadRequest) (evolution.ActivityResult, error) {
	if err := req.Validate(); err != nil {
		return evolution.ActivityResult{}, newError("actmem_invalid_edit", err.Error(), nil)
	}
	if err := caller.Validate(); err != nil {
		return evolution.ActivityResult{}, newError("actmem_invalid_edit", "caller scope: "+err.Error(), nil)
	}
	h, err := s.load()
	if err != nil {
		return evolution.ActivityResult{}, err
	}
	result := evolution.ActivityResult{Revision: h.revision, Entries: []evolution.Entry{}}
	if !h.classified() {
		return result, nil
	}
	wanted := func(section evolution.EntrySection) bool {
		if len(req.Sections) == 0 {
			return true
		}
		for _, s := range req.Sections {
			if s == section {
				return true
			}
		}
		return false
	}
	for _, entry := range h.entries() {
		if !wanted(entry.Section) || !visibleTo(entry.Scope, caller) {
			continue
		}
		result.Entries = append(result.Entries, entry)
	}
	return result, nil
}
