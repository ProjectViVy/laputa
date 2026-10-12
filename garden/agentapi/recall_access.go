package agentapi

import (
	"context"
	"github.com/ProjectViVy/laputa/garden/internal/arbiter"
	"github.com/ProjectViVy/laputa/garden/internal/rag"
	"github.com/ProjectViVy/laputa/garden/internal/recall"
	"github.com/ProjectViVy/laputa/mentle/facade"
)

// This adapter exists for one context assembly. It retains only the admitted
// card revisions, so automatic expansion uses the same card-before-evidence
// boundary as explicit tools and rechecks current membership at expansion.
type authorizedCardSearcher struct {
	s         *Service
	b         Binding
	p         Principal
	revisions map[string]uint64
}

func (a *authorizedCardSearcher) SearchCards(ctx context.Context, q facade.CardQuery) (facade.CardPage, error) {
	page, err := a.s.SearchCards(ctx, a.p, a.b, CardSearch{Query: q.Text, Collection: q.Collection, Limit: q.Limit, Cursor: q.Cursor})
	if err != nil {
		return facade.CardPage{}, err
	}
	if a.revisions == nil {
		a.revisions = map[string]uint64{}
	}
	for _, card := range page.Cards {
		a.revisions[card.ID] = uint64(card.Revision)
	}
	return convert[facade.CardPage](page)
}
func (a *authorizedCardSearcher) ReadEvidence(ctx context.Context, q facade.EvidenceQuery) ([]facade.EvidenceFragment, error) {
	refs := []EvidenceRef{}
	for _, id := range q.CardIDs {
		rev := a.revisions[id]
		if rev == 0 {
			return nil, failure("invalid_request", "automatic evidence requires admitted card")
		}
		refs = append(refs, EvidenceRef{CardID: id, ExpectedRevision: rev})
	}
	fragments, err := a.s.ReadEvidence(ctx, a.p, a.b, EvidenceRead{Items: refs, PerItemBudget: q.PerItemBudget, TotalBudget: q.TotalBudget})
	if err != nil {
		return nil, err
	}
	return convert[[]facade.EvidenceFragment](fragments)
}

// DeepRecall reuses orchestration over the same governed cards and evidence.
// Graph/timeline sources without an exact subject contract remain unavailable.
func (s *Service) DeepRecall(ctx context.Context, p Principal, b Binding, q recall.DeepRequest) (recall.DeepResponse, error) {
	if err := s.check(b, p, OpSearch); err != nil {
		return recall.DeepResponse{}, err
	}
	if s.runtime.FastRecall == nil {
		return recall.DeepResponse{}, failure("unavailable", "recall unavailable")
	}
	fast := *s.runtime.FastRecall
	fast.Searcher = &authorizedCardSearcher{s: s, b: b, p: p}
	if s.installation != nil {
		fast.Frozen = &boundFrozen{runtime: s.runtime, binding: b, bindingID: s.bindingID}
	}
	q.SessionID = b.SessionID
	deep := recall.DeepService{Fast: &fast, Planner: rag.RulePlanner{}, Arbiter: arbiter.New()}
	out, err := deep.Recall(ctx, q)
	if err != nil {
		return out, failure("invalid_request", err.Error())
	}
	return out, nil
}
