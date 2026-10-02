package agentapi

import (
	"context"
	"strings"
	"testing"

	"github.com/dashimaki/garden/internal/runtimecore"
	"github.com/dashimaki/laputa/actmem"
	"github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/laputa/persona"
)

// A bound host principal and session are supplied by Client, never by the read request.
func TestBoundReadPersonaExplicitWorld(t *testing.T) {
	p, err := persona.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Initialize(persona.Initialization{Identity: "identity", Relationship: "relationship", Redline: "redline", User: "user", World: "private world"}, "user", persona.SourceInit, "setup"); err != nil {
		t.Fatal(err)
	}
	b := &BoundClient{client: &Client{runtime: &runtimecore.Garden{ProfileID: "default", Persona: p}, principal: PrincipalAgent}, binding: binding()}
	doc, err := b.ReadPersona(context.Background(), "world")
	if err != nil || doc.Kind != "world" || !strings.Contains(doc.Content, "private world") {
		t.Fatalf("world document: %+v, %v", doc, err)
	}
	_, err = b.ReadPersona(context.Background(), "../world")
	assertCode(t, err, "invalid_request")
}

func TestBoundReadACTMEMAndQueryACTMEM(t *testing.T) {
	a := actmem.New(t.TempDir())
	if _, err := a.AppendEntry(evolution.Entry{Section: evolution.SectionPulse, Scope: evolution.Scope{SubjectID: "default", Kind: evolution.ScopePersonal}, SessionID: "session", Body: "unique milestone"}); err != nil {
		t.Fatal(err)
	}
	b := &BoundClient{client: &Client{runtime: &runtimecore.Garden{ProfileID: "default", Actmem: a}, principal: PrincipalAgent}, binding: binding()}
	doc, err := b.ReadACTMEM(context.Background())
	if err != nil || !strings.Contains(doc.Markdown, "unique milestone") {
		t.Fatalf("ACTMEM: %+v, %v", doc, err)
	}
	result, err := b.QueryACTMEM(context.Background(), ActmemQuery{Query: "unique", MaxHits: 1, MaxChars: 100})
	if err != nil || len(result.Items) != 1 || !strings.Contains(result.Items[0].Excerpt, "unique") {
		t.Fatalf("query: %+v, %v", result, err)
	}
	_, err = b.QueryACTMEM(context.Background(), ActmemQuery{})
	assertCode(t, err, "invalid_request")
}

func TestBoundMaterialReadsDelegateAndEnforcePrincipal(t *testing.T) {
	b := &BoundClient{client: &Client{runtime: &runtimecore.Garden{ProfileID: "default"}, principal: PrincipalAgent}, binding: binding()}
	_, err := b.SearchCards(context.Background(), CardSearch{Query: "signal"})
	assertCode(t, err, "unavailable")
	_, err = b.ReadEvidence(context.Background(), EvidenceRead{Items: []EvidenceRef{{CardID: "card", ExpectedRevision: 1}}})
	assertCode(t, err, "unavailable")
	_, err = b.ReadEvidence(context.Background(), EvidenceRead{})
	assertCode(t, err, "invalid_request")
	b.client.principal = PrincipalAutodream
	_, err = b.ReadACTMEM(context.Background())
	assertCode(t, err, "principal_forbidden")
	b.client.principal = ""
	_, err = b.SearchCards(context.Background(), CardSearch{Query: "signal"})
	assertCode(t, err, "authentication_required")
}

func TestBoundIndexHealthRespectsHostBindingAndClosedHandle(t *testing.T) {
	b := &BoundClient{client: &Client{runtime: &runtimecore.Garden{ProfileID: "default"}, principal: PrincipalAgent}, binding: binding()}
	health, err := b.IndexHealth(context.Background())
	assertCode(t, err, "index_health_unavailable")
	if health.Status != "unavailable" {
		t.Fatalf("health status: %+v", health)
	}
	b.client.principal = ""
	_, err = b.IndexHealth(context.Background())
	assertCode(t, err, "authentication_required")
	b.client.runtime = nil
	_, err = b.IndexHealth(context.Background())
	assertCode(t, err, "unavailable")
	var nilBound *BoundClient
	_, err = nilBound.IndexHealth(context.Background())
	assertCode(t, err, "unavailable")
}

func TestBoundExplicitReadsRejectUnavailableHandles(t *testing.T) {
	var b *BoundClient
	_, err := b.ReadPersona(context.Background(), "world")
	assertCode(t, err, "unavailable")
	_, err = b.ReadACTMEM(context.Background())
	assertCode(t, err, "unavailable")
	_, err = b.QueryACTMEM(context.Background(), ActmemQuery{Query: "hello"})
	assertCode(t, err, "unavailable")
	_, err = b.SearchCards(context.Background(), CardSearch{Query: "hello"})
	assertCode(t, err, "unavailable")
	_, err = b.ReadEvidence(context.Background(), EvidenceRead{Items: []EvidenceRef{{CardID: "card", ExpectedRevision: 1}}})
	assertCode(t, err, "unavailable")
	b = &BoundClient{client: &Client{principal: PrincipalAgent}, binding: binding()}
	_, err = b.ReadPersona(context.Background(), "world")
	assertCode(t, err, "unavailable")
}
