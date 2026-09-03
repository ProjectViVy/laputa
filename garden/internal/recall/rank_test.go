package recall

import (
	"testing"

	"github.com/dashimaki/mentle/facade"
)

func TestFilterCardsCopiesCandidatesWithoutAuthorityProjection(t *testing.T) {
	input := []facade.MemoryCard{{ID: "one"}, {ID: "two"}}
	output := FilterCards(input)
	if len(output) != 2 || &output[0] == &input[0] {
		t.Fatalf("output=%v", output)
	}
}

func TestRankCardsAndDeduplicate(t *testing.T) {
	cards := []facade.MemoryCard{{ID: "a", Scope: "project", CandidateScore: .2}, {ID: "b", Scope: "other", CandidateScore: .9}, {ID: "a", Scope: "project", CandidateScore: .2}}
	ranked := RankCards(cards, "project")
	unique := DeduplicateCards(ranked)
	if len(unique) != 2 || unique[0].ID != "b" {
		t.Fatalf("unique=%+v", unique)
	}
}
