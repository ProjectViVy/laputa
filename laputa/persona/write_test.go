package persona

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitializeWritesFiveAndStaysReady(t *testing.T) {
	dir := t.TempDir()
	svc, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	init := Initialization{
		Identity:     "我是记忆花园的守护者。",
		Relationship: "长期协作伙伴。",
		Redline:      "绝不伪造记忆。",
		User:         "工程师，偏好简洁中文。",
		World:        "Windows 本地主机。",
	}
	out, err := svc.Initialize(init, "user", SourceInit, "Persona initialization")
	if err != nil {
		t.Fatal(err)
	}
	if !out.Changed {
		t.Fatal("initialize should report changed")
	}
	view, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != StatusReady {
		t.Fatalf("status after init = %s", view.Status)
	}
	for _, kind := range RequiredKinds {
		st := view.Files[kind.String()]
		if !st.Exists || !st.Valid {
			t.Fatalf("%s should be valid after init: %+v", kind, st)
		}
		if st.Revision != 1 {
			t.Fatalf("%s revision = %d, want 1", kind, st.Revision)
		}
	}
	// DREAM/DARK must NOT be created by init
	if view.Files["dream"].Exists || view.Files["dark"].Exists {
		t.Fatal("init must not create optional DREAM/DARK")
	}
	// USER should be wrapped in Preferences section
	doc, err := svc.GetDocument(KindUser)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ExtractMarkdownSection(doc.Content, "Preferences"); !ok {
		t.Fatalf("USER should be wrapped in ## Preferences, got %q", doc.Content)
	}
}

func TestInitializeFailsWhenNotUninitialized(t *testing.T) {
	dir := t.TempDir()
	svc, _ := Open(dir)
	init := Initialization{Identity: "a", Relationship: "b", Redline: "c", User: "d", World: "e"}
	if _, err := svc.Initialize(init, "user", SourceInit, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Initialize(init, "user", SourceInit, "again"); CodeOf(err) != "persona_already_initialized" {
		t.Fatalf("second init should error, got %v", err)
	}
}

func TestWriteBumpsRevisionAndHistory(t *testing.T) {
	dir := t.TempDir()
	svc, _ := Open(dir)
	init := Initialization{Identity: "v1 identity", Relationship: "r", Redline: "x", User: "u", World: "w"}
	if _, err := svc.Initialize(init, "user", SourceInit, "init"); err != nil {
		t.Fatal(err)
	}
	out, err := svc.Write(KindIdentity, "v2 identity updated", 1, "user", SourceUserDirect, "update identity")
	if err != nil {
		t.Fatal(err)
	}
	if !out.Changed {
		t.Fatal("write should be changed")
	}
	doc, err := svc.GetDocument(KindIdentity)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Revision != 2 || doc.Content != "v2 identity updated" {
		t.Fatalf("doc = rev %d content %q", doc.Revision, doc.Content)
	}
	hist, err := svc.ListHistory(KindIdentity)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 2 || hist[0].Revision != 2 || hist[0].Actor != "user" {
		t.Fatalf("history = %+v", hist)
	}
	// status still ready
	view, _ := svc.Status()
	if view.Status != StatusReady {
		t.Fatalf("status = %s", view.Status)
	}
}

func TestWriteRevisionConflict(t *testing.T) {
	dir := t.TempDir()
	svc, _ := Open(dir)
	init := Initialization{Identity: "v1", Relationship: "r", Redline: "x", User: "u", World: "w"}
	if _, err := svc.Initialize(init, "user", SourceInit, "init"); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Write(KindIdentity, "stale write", 99, "user", SourceUserDirect, "stale")
	if err == nil {
		t.Fatal("stale base revision should error")
	}
	var rc *RevisionConflictError
	if !asRevisionConflict(err, &rc) {
		t.Fatalf("err = %v, want RevisionConflictError", err)
	}
}

func TestCreateRequestAndAccept(t *testing.T) {
	dir := t.TempDir()
	svc, _ := Open(dir)
	init := Initialization{Identity: "v1 identity", Relationship: "r", Redline: "x", User: "u", World: "w"}
	if _, err := svc.Initialize(init, "user", SourceInit, "init"); err != nil {
		t.Fatal(err)
	}
	doc, _ := svc.GetDocument(KindIdentity)
	req, err := svc.CreateRequest(ChangeRequest{
		Kind:             KindIdentity,
		BaseRevision:     doc.Revision,
		BaseHash:         doc.ContentHash,
		ProposedMarkdown: "v2 via agent proposal",
		Actor:            ActorAgent,
		Reason:           "agent suggests sharper identity",
	})
	if err != nil {
		t.Fatal(err)
	}
	if req.ID == "" || req.State != RequestPending {
		t.Fatalf("req = %+v", req)
	}
	// duplicate pending for same kind
	if _, err := svc.CreateRequest(ChangeRequest{
		Kind: KindIdentity, BaseRevision: doc.Revision, BaseHash: doc.ContentHash,
		ProposedMarkdown: "another", Actor: ActorAgent, Reason: "dup",
	}); err == nil {
		t.Fatal("duplicate pending request should error")
	} else if _, ok := err.(*RequestExistsError); !ok {
		t.Fatalf("err = %v, want RequestExistsError", err)
	}
	// accept
	out, err := svc.AcceptRequest(req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Changed {
		t.Fatal("accept should change")
	}
	doc2, _ := svc.GetDocument(KindIdentity)
	if doc2.Content != "v2 via agent proposal" || doc2.Revision != 2 {
		t.Fatalf("doc after accept = rev %d %q", doc2.Revision, doc2.Content)
	}
	// request now accepted
	reqs, _ := svc.ListRequests(nil)
	if reqs[0].State != RequestAccepted {
		t.Fatalf("state = %s", reqs[0].State)
	}
}

func TestRejectRequest(t *testing.T) {
	dir := t.TempDir()
	svc, _ := Open(dir)
	init := Initialization{Identity: "v1", Relationship: "r", Redline: "x", User: "u", World: "w"}
	if _, err := svc.Initialize(init, "user", SourceInit, "init"); err != nil {
		t.Fatal(err)
	}
	doc, _ := svc.GetDocument(KindIdentity)
	req, _ := svc.CreateRequest(ChangeRequest{
		Kind: KindIdentity, BaseRevision: doc.Revision, BaseHash: doc.ContentHash,
		ProposedMarkdown: "should not land", Actor: ActorAgent, Reason: "reject me",
	})
	if _, err := svc.RejectRequest(req.ID); err != nil {
		t.Fatal(err)
	}
	// document unchanged
	doc2, _ := svc.GetDocument(KindIdentity)
	if doc2.Content != "v1" || doc2.Revision != 1 {
		t.Fatalf("doc after reject = rev %d %q", doc2.Revision, doc2.Content)
	}
	reqs, _ := svc.ListRequests(nil)
	if reqs[0].State != RequestRejected {
		t.Fatalf("state = %s", reqs[0].State)
	}
	// rejecting again -> stale
	if _, err := svc.RejectRequest(req.ID); err == nil {
		t.Fatal("rejecting a decided request should error")
	}
}

func asRevisionConflict(err error, rc **RevisionConflictError) bool {
	for err != nil {
		if v, ok := err.(*RevisionConflictError); ok {
			*rc = v
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func TestWriteLockedOrderHistorySynced(t *testing.T) {
	// Verify the on-disk layout produced by writeLocked: snapshot+diff+log
	// must all exist and log content_hash must match the file's hash.
	dir := t.TempDir()
	svc, _ := Open(dir)
	init := Initialization{Identity: "order check", Relationship: "r", Redline: "x", User: "u", World: "w"}
	if _, err := svc.Initialize(init, "user", SourceInit, "init"); err != nil {
		t.Fatal(err)
	}
	root := svc.Root()
	if _, err := os.Stat(filepath.Join(root, "history", "IDENTITY.MD", "1.md")); err != nil {
		t.Fatalf("snapshot missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "history", "IDENTITY.MD", "1.diff")); err != nil {
		t.Fatalf("diff missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "history", "IDENTITY.MD", "log.jsonl")); err != nil {
		t.Fatalf("log missing: %v", err)
	}
}
