package agentapi

import (
	"context"
	"strings"
	"testing"

	"github.com/dashimaki/garden/internal/runtimecore"
	"github.com/dashimaki/laputa/actmem"
	"github.com/dashimaki/laputa/persona"
)

func TestReadPersonaExplicitWorldAndBinding(t *testing.T) {
	p, err := persona.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Initialize(persona.Initialization{Identity: "identity", Relationship: "relationship", Redline: "redline", User: "user", World: "world only"}, "user", persona.SourceInit, "setup")
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(&runtimecore.Garden{ProfileID: "default", Persona: p})
	doc, err := s.ReadPersona(context.Background(), PrincipalAgent, binding(), "world")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Kind != "world" || !strings.Contains(doc.Content, "world only") {
		t.Fatalf("unexpected world: %+v", doc)
	}
	_, err = s.ReadPersona(context.Background(), "", binding(), "world")
	assertCode(t, err, "authentication_required")
	b := binding()
	b.ProfileID = "other"
	_, err = s.ReadPersona(context.Background(), PrincipalAgent, b, "world")
	assertCode(t, err, "profile_mismatch")
	_, err = s.ReadPersona(context.Background(), PrincipalAgent, binding(), "../world")
	assertCode(t, err, "invalid_request")
}

func TestExplicitActmemReadAndQuery(t *testing.T) {
	a := actmem.New(t.TempDir())
	_, err := a.AppendPulse("session", "unique milestone")
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(&runtimecore.Garden{ProfileID: "default", Actmem: a})
	doc, err := s.ReadACTMEM(context.Background(), PrincipalAgent, binding())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Markdown, "unique milestone") {
		t.Fatalf("ACTMEM: %+v", doc)
	}
	result, err := s.QueryACTMEM(context.Background(), PrincipalAgent, binding(), ActmemQuery{Query: "unique", MaxHits: 1, MaxChars: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || !strings.Contains(result.Items[0].Excerpt, "unique") {
		t.Fatalf("query: %+v", result)
	}
	_, err = s.ReadACTMEM(context.Background(), PrincipalAutodream, binding())
	assertCode(t, err, "principal_forbidden")
	_, err = s.QueryACTMEM(context.Background(), PrincipalAgent, binding(), ActmemQuery{})
	assertCode(t, err, "invalid_request")
}

func TestExplicitMaterialReadsAreBoundAndUnavailableWithoutMentle(t *testing.T) {
	s := NewService(&runtimecore.Garden{ProfileID: "default"})
	_, err := s.SearchCards(context.Background(), "", binding(), CardSearch{Query: "signal"})
	assertCode(t, err, "authentication_required")
	_, err = s.SearchCards(context.Background(), PrincipalAgent, binding(), CardSearch{Query: "signal"})
	assertCode(t, err, "unavailable")
	_, err = s.ReadEvidence(context.Background(), PrincipalAgent, binding(), EvidenceRead{CardIDs: []string{"card"}, PerItemBudget: 20, TotalBudget: 20})
	assertCode(t, err, "unavailable")
	_, err = s.ReadEvidence(context.Background(), PrincipalAgent, binding(), EvidenceRead{})
	assertCode(t, err, "invalid_request")
	b := binding()
	b.ProfileID = "other"
	_, err = s.ReadEvidence(context.Background(), PrincipalAgent, b, EvidenceRead{CardIDs: []string{"card"}})
	assertCode(t, err, "profile_mismatch")
}
