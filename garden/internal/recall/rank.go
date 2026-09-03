package recall

import "sort"

import "github.com/dashimaki/mentle/facade"

func FilterCards(cards []facade.MemoryCard) []facade.MemoryCard {
	out := make([]facade.MemoryCard, len(cards))
	copy(out, cards)
	return out
}

func RankCards(cards []facade.MemoryCard, scope string) []facade.MemoryCard {
	sorted := make([]facade.MemoryCard, len(cards))
	copy(sorted, cards)
	sort.SliceStable(sorted, func(i, j int) bool {
		return compositeScore(sorted[i], scope) > compositeScore(sorted[j], scope)
	})
	return sorted
}

func compositeScore(card facade.MemoryCard, scope string) float64 {
	scopeBonus := 0.0
	if scope != "" && card.Scope == scope {
		scopeBonus = 1.0
	}
	return card.CandidateScore*0.6 + card.HeatScore*0.2 + scopeBonus*0.2
}

func DeduplicateCards(cards []facade.MemoryCard) []facade.MemoryCard {
	seen := make(map[string]bool, len(cards))
	out := make([]facade.MemoryCard, 0, len(cards))
	for _, card := range cards {
		if seen[card.ID] {
			continue
		}
		seen[card.ID] = true
		out = append(out, card)
	}
	return out
}
