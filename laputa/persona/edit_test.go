package persona

import (
	"encoding/json"
	"os"
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

func TestPersonaEditGuardsEveryWriter(t *testing.T) {
	for _, which := range []string{"direct", "observations", "agent", "accept", "approve", "repair", "initialize"} {
		t.Run(which, func(t *testing.T) {
			s := editFixture(t)
			kind := KindIdentity
			if which == "observations" {
				kind = KindUser
			}
			if which == "agent" {
				kind = KindDream
			}
			if which == "repair" {
				kind = KindRelationship
			}
			doc, err := s.GetDocument(kind)
			if err != nil {
				t.Fatal(err)
			}
			var review *ChangeRequest
			if which == "accept" || which == "approve" {
				review, err = s.CreateReview(PersonaReviewCreate{Kind: kind, BaseRevision: doc.Revision, BaseHash: doc.ContentHash, ProposedMarkdown: "candidate", Actor: ActorAgent, Reason: "review"})
				if err != nil {
					t.Fatal(err)
				}
			}
			lease, err := s.AcquireEdit(kind, doc.Revision, doc.ContentHash, "editor")
			if err != nil {
				t.Fatal(err)
			}
			switch which {
			case "direct":
				_, err = s.SaveUserDocument(kind, "other", doc.Revision, "user", SourceUserDirect, "write")
			case "observations":
				_, err = s.SaveUserObservations("other", doc.Revision, "write")
			case "agent":
				_, err = s.SaveAgentP16WithRevision(kind, "other", doc.Revision, "write")
			case "accept":
				_, err = s.AcceptRequest(review.ID)
			case "approve":
				_, err = s.ApproveRequest(review.ID)
			case "repair":
				for _, k := range []Kind{KindIdentity, KindRelationship} {
					if e := os.Remove(s.documentPath(k)); e != nil {
						t.Fatal(e)
					}
				}
				_, err = s.Repair(RepairInput{Documents: map[Kind]string{KindIdentity: "repaired", KindRelationship: "repaired"}, Reason: "repair"})
			case "initialize":
				for _, k := range RequiredKinds {
					if e := os.Remove(s.documentPath(k)); e != nil {
						t.Fatal(e)
					}
					if e := os.RemoveAll(s.historyDir(k)); e != nil {
						t.Fatal(e)
					}
				}
				_, err = s.Initialize(Initialization{Identity: "new", Relationship: "new", Redline: "new", User: "new", World: "new"}, "user", SourceInit, "init")
			}
			if CodeOf(err) != "persona_edit_locked" {
				t.Fatalf("%s bypassed lease: %v", which, err)
			}
			state, e := s.EditState(kind)
			if e != nil || !state.Held || state.Generation != lease.Generation {
				t.Fatal(state, e)
			}
			if which == "repair" || which == "initialize" {
				for _, k := range []Kind{KindIdentity, KindRelationship} {
					if _, e := os.Stat(s.documentPath(k)); !os.IsNotExist(e) {
						t.Fatal("multi-document operation partially wrote", k, e)
					}
				}
			} else {
				current, e := s.GetDocument(kind)
				if e != nil || current.ContentHash != doc.ContentHash || current.Revision != doc.Revision {
					t.Fatal("blocked writer changed authority", current, e)
				}
				history, e := s.ListHistory(kind)
				if e != nil || len(history) != int(doc.Revision) {
					t.Fatal("blocked writer changed history", history, e)
				}
			}
		})
	}
}

func TestPersonaEditStaleBaseAndSnapshot(t *testing.T) {
	s := editFixture(t)
	doc, _ := s.GetDocument(KindIdentity)
	a, err := s.AcquireEdit(KindIdentity, doc.Revision, doc.ContentHash, "a")
	if err != nil {
		t.Fatal(err)
	}
	called := false
	if err = s.WithProjectionSnapshot([]Kind{KindIdentity, KindRelationship}, func(ProjectionSnapshot) error { called = true; return nil }); CodeOf(err) != "persona_edit_locked" || called {
		t.Fatal("locked snapshot admitted", err, called)
	}
	in := EditorWrite{Kind: KindIdentity, BaseRevision: doc.Revision, BaseHash: doc.ContentHash, Proof: editProof(a), Scope: "document", Content: "", Reason: "editor"}
	if _, err = s.SaveEditorDocument(in); CodeOf(err) != "persona_invalid_content" {
		t.Fatal(err)
	}
	held, _ := s.EditState(KindIdentity)
	if !held.Held {
		t.Fatal("validation error lost lease")
	}
	in.Content = "new identity"
	in.BaseRevision = 0
	if _, err = s.SaveEditorDocument(in); CodeOf(err) != "persona_revision_conflict" {
		t.Fatal(err)
	}
	in.BaseRevision = doc.Revision
	out, err := s.SaveEditorDocument(in)
	if err != nil || !out.Changed || out.Document.Revision != 2 {
		t.Fatal(out, err)
	}
	if _, err = s.SaveEditorDocument(in); CodeOf(err) != "persona_edit_expired" {
		t.Fatal("consumed proof fell back to normal write", err)
	}
	held, _ = s.EditState(KindIdentity)
	if held.Held {
		t.Fatal("successful save did not release")
	}
	if err = s.WithProjectionSnapshot([]Kind{KindIdentity, KindRelationship}, func(snapshot ProjectionSnapshot) error {
		if snapshot.Documents[KindIdentity].Revision != 2 || snapshot.Documents[KindRelationship].Revision != 1 || snapshot.EditGenerations[KindIdentity] != a.Generation {
			t.Fatal(snapshot)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveUserDocument(KindIdentity, "old generated", 1, "user", SourceUserDirect, "old"); CodeOf(err) != "persona_revision_conflict" {
		t.Fatal(err)
	}
	if err = os.WriteFile(s.documentPath(KindIdentity), []byte("inspected external body"), 0600); err != nil {
		t.Fatal(err)
	}
	called = false
	if err = s.WithProjectionSnapshot([]Kind{KindIdentity}, func(ProjectionSnapshot) error { called = true; return nil }); CodeOf(err) != "persona_authority_drift" || called {
		t.Fatal("uncommitted drift emitted", err, called)
	}
	out, err = s.SaveUserDocument(KindIdentity, "inspected external body", 2, "user", SourceUserDirect, "explicit inspected resave")
	if err != nil || !out.Changed || out.Document.Revision != 3 {
		t.Fatal("external body did not become a new committed revision", out, err)
	}
}

func TestPersonaEditExpiredSaveCannotAffectNewHolder(t *testing.T) {
	s := editFixture(t)
	now := time.Now()
	s.SetClock(func() time.Time { return now })
	doc, _ := s.GetDocument(KindIdentity)
	a, err := s.AcquireEdit(KindIdentity, doc.Revision, doc.ContentHash, "a")
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(121 * time.Second)
	b, err := s.AcquireEdit(KindIdentity, doc.Revision, doc.ContentHash, "b")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.SaveEditorDocument(EditorWrite{Kind: KindIdentity, Content: "old tab", Scope: "document", Reason: "late", BaseRevision: doc.Revision, BaseHash: doc.ContentHash, Proof: editProof(a)})
	if CodeOf(err) != "persona_edit_expired" {
		t.Fatal(err)
	}
	state, _ := s.EditState(KindIdentity)
	if !state.Held || state.Generation != b.Generation {
		t.Fatal("old save displaced new holder", state)
	}
}
