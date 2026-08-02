package activity

import (
	"sync"
	"time"
)

const (
	maxCardIDs     = 50
	maxEvidenceRefs = 50
	maxPendingLeads = 20
)

type ScopeSnapshot struct {
	ActiveCardIDs []string `json:"active_card_ids"`
	EvidenceRefs  []string `json:"evidence_refs"`
	PendingLeads  []string `json:"pending_leads"`
	UpdatedAt     string   `json:"updated_at"`
}

type scopeState struct {
	ActiveCardIDs  []string
	EvidenceRefs   []string
	PendingLeads   []string
	LastCheckpoint time.Time
	UpdatedAt      time.Time
}

type WorkingSet struct {
	mu     sync.RWMutex
	scopes map[string]*scopeState
}

func NewWorkingSet() *WorkingSet {
	return &WorkingSet{scopes: make(map[string]*scopeState)}
}

func (ws *WorkingSet) Update(scope string, cards []string, evidence []string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	st := ws.scope(scope)
	st.ActiveCardIDs = mergeBounded(st.ActiveCardIDs, cards, maxCardIDs)
	st.EvidenceRefs = mergeBounded(st.EvidenceRefs, evidence, maxEvidenceRefs)
	st.UpdatedAt = time.Now().UTC()
}

func (ws *WorkingSet) AddLeads(scope string, leads []string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	st := ws.scope(scope)
	st.PendingLeads = mergeBounded(st.PendingLeads, leads, maxPendingLeads)
	st.UpdatedAt = time.Now().UTC()
}

func (ws *WorkingSet) Get(scope string) ScopeSnapshot {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	st, ok := ws.scopes[scope]
	if !ok {
		return ScopeSnapshot{ActiveCardIDs: []string{}, EvidenceRefs: []string{}, PendingLeads: []string{}}
	}
	return ScopeSnapshot{
		ActiveCardIDs: copySlice(st.ActiveCardIDs),
		EvidenceRefs:  copySlice(st.EvidenceRefs),
		PendingLeads:  copySlice(st.PendingLeads),
		UpdatedAt:     st.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func (ws *WorkingSet) Snapshot(scope string) ScopeSnapshot {
	return ws.Get(scope)
}

func (ws *WorkingSet) Restore(scope string, snap ScopeSnapshot) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	st := ws.scope(scope)
	st.ActiveCardIDs = bound(snap.ActiveCardIDs, maxCardIDs)
	st.EvidenceRefs = bound(snap.EvidenceRefs, maxEvidenceRefs)
	st.PendingLeads = bound(snap.PendingLeads, maxPendingLeads)
	if snap.UpdatedAt != "" {
		st.UpdatedAt, _ = time.Parse(time.RFC3339Nano, snap.UpdatedAt)
	}
}

func (ws *WorkingSet) MarkCheckpoint(scope string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	st := ws.scope(scope)
	st.LastCheckpoint = time.Now().UTC()
}

func (ws *WorkingSet) scope(name string) *scopeState {
	st, ok := ws.scopes[name]
	if !ok {
		st = &scopeState{}
		ws.scopes[name] = st
	}
	return st
}

func mergeBounded(existing, incoming []string, max int) []string {
	seen := make(map[string]bool, len(existing)+len(incoming))
	out := make([]string, 0, max)
	for _, id := range incoming {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	for _, id := range existing {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	if len(out) > max {
		out = out[:max]
	}
	return out
}

func bound(items []string, max int) []string {
	if len(items) > max {
		return items[:max]
	}
	return items
}

func copySlice(items []string) []string {
	out := make([]string, len(items))
	copy(out, items)
	return out
}
