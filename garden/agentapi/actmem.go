package agentapi

import (
	"context"

	"github.com/dashimaki/laputa/actmem"
)

// Typed ACTMEM surface (contracts.md section 3): every agent call carries
// the host-issued trusted scope; the store enforces visibility, so hidden
// entries' text, ids and counts never reach the caller.

// ReadActivity returns the scoped ACTMEM projection for the caller's bound
// workspace (personal union). Sections empty means all three.
func (s *Service) ReadActivity(_ context.Context, principal Principal, binding Binding, req ReadRequest) (ActivityResult, error) {
	if err := s.check(binding, principal, OpActmemRead); err != nil {
		return ActivityResult{}, err
	}
	if s.runtime.Actmem == nil {
		return ActivityResult{}, failure("unavailable", "ACTMEM unavailable")
	}
	scope, err := binding.TrustedScope()
	if err != nil {
		return ActivityResult{}, failure("invalid_request", err.Error())
	}
	result, err := s.runtime.Actmem.ReadScoped(scope, req)
	if err != nil {
		return ActivityResult{}, failure(actmem.CodeOf(err), "ACTMEM read failed")
	}
	return ActivityResult{Changed: result.Changed, Revision: result.Revision, Entries: result.Entries}, nil
}

// ApplyWorkPatch applies scoped Work changes against base_revision. The
// store rejects changes to entries outside the caller's admitted scope.
func (s *Service) ApplyWorkPatch(_ context.Context, principal Principal, binding Binding, patch WorkPatch) (ActivityResult, error) {
	if err := s.check(binding, principal, OpActmemWrite); err != nil {
		return ActivityResult{}, err
	}
	if s.runtime.Actmem == nil {
		return ActivityResult{}, failure("unavailable", "ACTMEM unavailable")
	}
	scope, err := binding.TrustedScope()
	if err != nil {
		return ActivityResult{}, failure("invalid_request", err.Error())
	}
	result, err := s.runtime.Actmem.ApplyWorkPatch(scope, patch)
	if err != nil {
		return ActivityResult{}, failure(actmem.CodeOf(err), "ACTMEM work patch failed")
	}
	return ActivityResult{Changed: result.Changed, Revision: result.Revision, Entries: result.Entries}, nil
}

// AppendActivity is the host-only system append: the caller supplies event
// identity; the store allocates the entry id, stamps it and dedupes a
// retained event id. Agent principals cannot impersonate this path — it is
// gated on the maintain operation.
func (s *Service) AppendActivity(_ context.Context, principal Principal, binding Binding, entry Entry) (ActivityResult, error) {
	if err := s.check(binding, principal, OpActmemMaintain); err != nil {
		return ActivityResult{}, err
	}
	if s.runtime.Actmem == nil {
		return ActivityResult{}, failure("unavailable", "ACTMEM unavailable")
	}
	scope, err := binding.TrustedScope()
	if err != nil {
		return ActivityResult{}, failure("invalid_request", err.Error())
	}
	entry.Scope = scope // caller-supplied scope is never trusted
	result, err := s.runtime.Actmem.AppendEntry(entry)
	if err != nil {
		return ActivityResult{}, failure(actmem.CodeOf(err), "ACTMEM append failed")
	}
	return ActivityResult{Changed: result.Changed, Revision: result.Revision, Entries: result.Entries}, nil
}

// FoldSession archives the named session's Pulse/Recap entries into
// deterministic capsules, then removes exactly those sources from the head.
func (s *Service) FoldSession(_ context.Context, principal Principal, binding Binding, sessionID string) ([]actmem.CapsuleSummary, error) {
	if err := s.check(binding, principal, OpActmemMaintain); err != nil {
		return nil, err
	}
	if s.runtime.Actmem == nil {
		return nil, failure("unavailable", "ACTMEM unavailable")
	}
	return s.runtime.Actmem.FoldSession(sessionID)
}
