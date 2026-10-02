package persona

import (
	"errors"
	"strings"
	"testing"
)

func initMissionPersona(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	svc, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	init := Initialization{Identity: "i", Relationship: "r", Redline: "x", User: "u", World: "w"}
	if _, err := svc.Initialize(init, "user", SourceInit, "init"); err != nil {
		t.Fatal(err)
	}
	return svc
}

// TestMissionKindIdentity pins the appended kind: storage integers 0-6 are
// unmoved, mission is the eighth roster member, and the wire name round-trips.
func TestMissionKindIdentity(t *testing.T) {
	ints := map[Kind]int{KindIdentity: 0, KindRelationship: 1, KindRedline: 2, KindUser: 3, KindWorld: 4, KindDream: 5, KindDark: 6}
	for kind, want := range ints {
		if int(kind) != want {
			t.Fatalf("%s renumbered to %d", kind, int(kind))
		}
	}
	if len(AllKinds) != 8 || AllKinds[7] != KindMission {
		t.Fatalf("AllKinds = %v, want eight kinds with mission last", AllKinds)
	}
	parsed, err := ParseKind("mission")
	if err != nil || parsed != KindMission {
		t.Fatalf("ParseKind(mission) = %v, %v", parsed, err)
	}
	if KindMission.String() != "mission" || KindMission.FileName() != "MISSION.MD" {
		t.Fatalf("wire naming = %q/%q", KindMission.String(), KindMission.FileName())
	}
	if KindMission.RequiredForReady() {
		t.Fatal("mission must not be required for ready")
	}
	if KindMission.ContentLimit() != 400 {
		t.Fatalf("mission content limit = %d, want 400", KindMission.ContentLimit())
	}
}

// TestMissionMissingIsUnassigned: a profile without MISSION.MD is ready and
// the document read reports the explicit unassigned state, never a failure.
func TestMissionMissingIsUnassigned(t *testing.T) {
	svc := initMissionPersona(t)
	view, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != StatusReady {
		t.Fatalf("status without MISSION.MD = %s", view.Status)
	}
	state := view.Files["mission"]
	if state.Exists {
		t.Fatal("mission state should report absent")
	}
	if !state.Valid {
		t.Fatalf("absent mission must be valid (unassigned), got %+v", state)
	}
	doc, err := svc.GetDocument(KindMission)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Exists || doc.Content != "" || doc.Revision != 0 {
		t.Fatalf("unassigned mission doc = %+v", doc)
	}
}

// TestMissionHumanWriteEditClearRestore covers the authenticated-human
// lifecycle: create, edit, clear (unassign), and restore of a prior revision.
func TestMissionHumanWriteEditClearRestore(t *testing.T) {
	svc := initMissionPersona(t)
	out, err := svc.SaveUserDocument(KindMission, "protect the archive", 0, "user", SourceUserDirect, "set mission")
	if err != nil {
		t.Fatalf("human create failed: %v", err)
	}
	if !out.Changed || out.Document.Revision != 1 || out.Document.Content != "protect the archive" {
		t.Fatalf("create outcome = %+v", out)
	}
	out, err = svc.SaveUserDocument(KindMission, "guard the archive and its indices", 1, "user", SourceUserDirect, "edit mission")
	if err != nil {
		t.Fatalf("human edit failed: %v", err)
	}
	if out.Document.Revision != 2 {
		t.Fatalf("edit revision = %d", out.Document.Revision)
	}
	if out, err = svc.SaveUserDocument(KindMission, "", 2, "user", SourceUserDirect, "clear mission"); err != nil {
		t.Fatalf("human clear (unassign) must succeed, got %v", err)
	}
	doc, err := svc.GetDocument(KindMission)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Content != "" {
		t.Fatalf("cleared mission content = %q", doc.Content)
	}
	if out, err = svc.SaveUserDocument(KindMission, "protect the archive", doc.Revision, "user", SourceUserDirect, "restore mission"); err != nil {
		t.Fatalf("human restore failed: %v", err)
	}
	if doc2, _ := svc.GetDocument(KindMission); doc2.Content != "protect the archive" {
		t.Fatalf("restored content = %q", doc2.Content)
	}
	history, err := svc.ListHistory(KindMission)
	if err != nil || len(history) < 4 {
		t.Fatalf("history = %+v, %v", history, err)
	}
	for _, entry := range history {
		if entry.Actor != "user" || entry.Source == string(SourceAgentP16) {
			t.Fatalf("non-human actor recorded: %+v", entry)
		}
	}
}

// TestMissionRejectsNonHumanPaths: every non-human mutation entry rejects
// mission — P16 direct writes, review requests, and non-direct write sources.
func TestMissionRejectsNonHumanPaths(t *testing.T) {
	svc := initMissionPersona(t)
	if _, err := svc.SaveAgentP16(KindMission, "agent mission", "reason"); !errors.Is(err, ErrKindForbidden) {
		t.Fatalf("SaveAgentP16(mission) = %v", err)
	}
	if _, err := svc.SaveAgentP16WithRevision(KindMission, "agent mission", 0, "reason"); !errors.Is(err, ErrKindForbidden) {
		t.Fatalf("SaveAgentP16WithRevision(mission) = %v", err)
	}
	doc, err := svc.GetDocument(KindIdentity)
	if err != nil {
		t.Fatal(err)
	}
	for _, actor := range []RequestActor{ActorAgent, ActorAutodream} {
		_, err = svc.CreateRequest(ChangeRequest{Kind: KindMission, BaseRevision: doc.Revision, BaseHash: doc.ContentHash, ProposedMarkdown: "x", Actor: actor, Reason: "r"})
		if !errors.Is(err, ErrKindForbidden) {
			t.Fatalf("CreateRequest(mission, %s) = %v", actor, err)
		}
	}
	if _, err := svc.Write(KindMission, "x", 0, "workflow", SourceAgentP16, "r"); !errors.Is(err, ErrKindForbidden) {
		t.Fatalf("Write(mission, agent source) = %v", err)
	}
	if _, err := svc.Repair(RepairInput{Documents: map[Kind]string{KindMission: "x"}, Reason: "r"}); !errors.Is(err, ErrKindForbidden) && !errors.Is(err, ErrRepairNotRequired) {
		t.Fatalf("Repair(mission) = %v", err)
	}
}

// TestMissionCapEnforced: 400 visible graphemes is a hard cap — 401 fails and
// nothing is silently truncated.
func TestMissionCapEnforced(t *testing.T) {
	svc := initMissionPersona(t)
	full := strings.Repeat("好", 400)
	if _, err := svc.SaveUserDocument(KindMission, full, 0, "user", SourceUserDirect, "r"); err != nil {
		t.Fatalf("400-grapheme mission rejected: %v", err)
	}
	doc, err := svc.GetDocument(KindMission)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Content != full {
		t.Fatalf("mission content silently altered: %d runes", len([]rune(doc.Content)))
	}
	var capErr *CapExceededError
	if _, err := svc.SaveUserDocument(KindMission, full+"!", doc.Revision, "user", SourceUserDirect, "r"); !errors.As(err, &capErr) {
		t.Fatalf("401-grapheme mission = %v, want cap error", err)
	}
}
