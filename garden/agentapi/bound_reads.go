package agentapi

import (
	"context"

	"github.com/ProjectViVy/laputa/mentle/facade"
)

// Explicit reads keep the host-owned principal and session binding; callers
// supply only the read target or query, never identity or authority.
func (b *BoundClient) ReadPersona(ctx context.Context, kind string) (PersonaDocument, error) {
	s, unlock, err := b.service()
	if err != nil {
		return PersonaDocument{}, err
	}
	defer unlock()
	return s.ReadPersona(ctx, b.caller(), b.binding, kind)
}

func (b *BoundClient) ReadACTMEM(ctx context.Context) (ActmemDocument, error) {
	s, unlock, err := b.service()
	if err != nil {
		return ActmemDocument{}, err
	}
	defer unlock()
	return s.ReadACTMEM(ctx, b.caller(), b.binding)
}

func (b *BoundClient) QueryACTMEM(ctx context.Context, q ActmemQuery) (ActmemResult, error) {
	s, unlock, err := b.service()
	if err != nil {
		return ActmemResult{}, err
	}
	defer unlock()
	return s.QueryACTMEM(ctx, b.caller(), b.binding, q)
}

func (b *BoundClient) ReadActivity(ctx context.Context, req ReadRequest) (ActivityResult, error) {
	s, unlock, err := b.service()
	if err != nil {
		return ActivityResult{}, err
	}
	defer unlock()
	return s.ReadActivity(ctx, b.caller(), b.binding, req)
}

func (b *BoundClient) ApplyWorkPatch(ctx context.Context, patch WorkPatch) (ActivityResult, error) {
	s, unlock, err := b.service()
	if err != nil {
		return ActivityResult{}, err
	}
	defer unlock()
	return s.ApplyWorkPatch(ctx, b.caller(), b.binding, patch)
}

func (b *BoundClient) AppendActivity(ctx context.Context, entry Entry) (ActivityResult, error) {
	s, unlock, err := b.service()
	if err != nil {
		return ActivityResult{}, err
	}
	defer unlock()
	return s.AppendActivity(ctx, b.caller(), b.binding, entry)
}

func (b *BoundClient) SearchCards(ctx context.Context, q CardSearch) (CardPage, error) {
	s, unlock, err := b.service()
	if err != nil {
		return CardPage{}, err
	}
	defer unlock()
	return s.SearchCards(ctx, b.caller(), b.binding, q)
}

func (b *BoundClient) ReadEvidence(ctx context.Context, q EvidenceRead) ([]EvidenceFragment, error) {
	s, unlock, err := b.service()
	if err != nil {
		return nil, err
	}
	defer unlock()
	return s.ReadEvidence(ctx, b.caller(), b.binding, q)
}

// IndexHealth preserves the Mentle public health report, including probe
// failure details, while keeping the bound host identity fixed.
func (b *BoundClient) IndexHealth(ctx context.Context) (facade.IndexHealth, error) {
	s, unlock, err := b.service()
	if err != nil {
		return facade.IndexHealth{}, err
	}
	defer unlock()
	return s.IndexHealth(ctx, b.caller(), b.binding)
}
