package persona

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func editFixture(t *testing.T) *Service {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Initialize(Initialization{Identity: "identity", Relationship: "relationship", Redline: "redline", User: "user", World: "world"}, "user", SourceInit, "init"); err != nil {
		t.Fatal(err)
	}
	return s
}
func editProof(l *EditLease) EditProof { return EditProof{Token: l.Token, Generation: l.Generation} }
func TestPersonaEditLeaseLifecycle(t *testing.T) {
	s := editFixture(t)
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	s.SetClock(func() time.Time { return now })
	doc, err := s.GetDocument(KindIdentity)
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.AcquireEdit(KindIdentity, doc.Revision, doc.ContentHash, "tab-a")
	if err != nil {
		t.Fatal(err)
	}
	if a.ExpiresAt.Sub(now) != 120*time.Second || a.Token == "" {
		t.Fatal(a)
	}
	retry, err := s.AcquireEdit(KindIdentity, doc.Revision, doc.ContentHash, "tab-a")
	if err != nil || retry.Token != a.Token {
		t.Fatal(retry, err)
	}
	if _, err = s.AcquireEdit(KindIdentity, doc.Revision, doc.ContentHash, "tab-b"); CodeOf(err) != "persona_edit_locked" {
		t.Fatal(err)
	}
	state, err := s.EditState(KindIdentity)
	raw, _ := json.Marshal(state)
	if err != nil || !state.Held || strings.Contains(string(raw), a.Token) {
		t.Fatal(state, err)
	}
	now = now.Add(121 * time.Second)
	b, err := s.AcquireEdit(KindIdentity, doc.Revision, doc.ContentHash, "tab-b")
	if err != nil || b.Generation <= a.Generation {
		t.Fatal(b, err)
	}
	if _, err = s.RenewEdit(KindIdentity, editProof(a)); CodeOf(err) != "persona_edit_expired" {
		t.Fatal(err)
	}
	if err = s.ReleaseEdit(KindIdentity, editProof(a)); CodeOf(err) != "persona_edit_expired" {
		t.Fatal(err)
	}
	state, err = s.EditState(KindIdentity)
	if err != nil || !state.Held || state.Generation != b.Generation {
		t.Fatal(state, err)
	}
	renewed, err := s.RenewEdit(KindIdentity, editProof(b))
	if err != nil || renewed.Token != b.Token {
		t.Fatal(renewed, err)
	}
	reopened, err := Open(strings.TrimSuffix(s.Root(), "/persona"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = reopened.RenewEdit(KindIdentity, editProof(b)); CodeOf(err) != "persona_edit_expired" {
		t.Fatal(err)
	}
	if err = s.ReleaseEdit(KindIdentity, editProof(b)); err != nil {
		t.Fatal(err)
	}
	state, err = s.EditState(KindIdentity)
	if err != nil || state.Held || state.Generation != b.Generation {
		t.Fatal(state, err)
	}
}
