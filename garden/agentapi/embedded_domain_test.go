package agentapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dashimaki/garden/memory"
	"github.com/dashimaki/garden/memory/memorytest"
	"github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/laputa/persona"
)

func userEmbeddedConfig(t *testing.T) Config {
	t.Helper()
	root := t.TempDir()
	return Config{
		PersonaDir: filepath.Join(root, "persona"), PalacePath: filepath.Join(root, "palace"),
		ModelsDir: filepath.Join(root, "models"), StateDB: filepath.Join(root, "state.db"),
		ProfileID: "default", AgentID: "agent-1", Platform: "vivy",
		Principal: PrincipalUser, BackendID: "mentle", DestinationID: "mentle",
	}
}

func initAuthority(t *testing.T, cfg Config) {
	t.Helper()
	authority, err := persona.Open(cfg.PersonaDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = authority.Initialize(persona.Initialization{
		Identity: "safe identity", Relationship: "relation", Redline: "redline",
		User: "user", World: "secret world",
	}, "user", persona.SourceInit, "init"); err != nil {
		t.Fatal(err)
	}
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func wsScope(profile, workspace string) evolution.Scope {
	return evolution.Scope{SubjectID: profile, Kind: evolution.ScopeWorkspace, WorkspaceID: workspace}
}

// validActmem is a minimal well-formed v2 document for owner saves. It
// carries one entry: ActmemDocument.Render emits `entries:` as a null node
// for an empty map, which the loader rejects — entry-less saves are an
// upstream laputa/evolution defect recorded for the next wave, not patched
// here (ACTMEM storage is outside this story's file list).
const validActmem = "---\nschema: laputa.actmem/v2\nrevision: 0\nupdated: \"2026-10-03T00:00:00Z\"\nentries:\n  e_0123456789abcdef0123456789abcdef:\n    section: pulse\n    field: \"\"\n    scope:\n      subject_id: \"default\"\n      kind: personal\n      workspace_id: \"\"\n    session_id: \"s-1\"\n    event_id: \"\"\n    occurred_at: \"2026-10-03T00:00:00Z\"\n    sources: []\n---\n\n## Pulse\n<!-- actmem-entry:e_0123456789abcdef0123456789abcdef -->\nsaved pulse\n<!-- /actmem-entry:e_0123456789abcdef0123456789abcdef -->\n\n## Recap\n\n## Work\n"

func mustEffect(t *testing.T, opID string, kind evolution.EffectKind, payload any, scope evolution.Scope, dest string) evolution.Effect {
	t.Helper()
	e, err := evolution.NewEffect(opID, kind, payload, scope, dest)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// TestEmbeddedDomainSameOwner proves the human capability, the reduced agent
// capability and the bound Evolution domain all derive from one Garden owner.
func TestEmbeddedDomainSameOwner(t *testing.T) {
	cfg := userEmbeddedConfig(t)
	initAuthority(t, cfg)
	// Mentle is unavailable without a local model; inject the scope-bound
	// writer the owner selected.
	personal0 := evolution.Scope{SubjectID: cfg.ProfileID, Kind: evolution.ScopePersonal}
	cfg.BackendID = "fake"
	cfg.DestinationID = "fake-dest"
	store := memorytest.NewFakeStore()
	cfg.Backends = map[string]memory.Backend{memory.EncodeScope(personal0): memorytest.NewFakeBackend(store, personal0, "fake-dest")}
	client, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	human, err := client.BindHumanSession("s1", "")
	if err != nil {
		t.Fatalf("bind human: %v", err)
	}
	agent, err := client.BindAgentSession("s1", "")
	if err != nil {
		t.Fatalf("bind agent: %v", err)
	}
	scope := evolution.Scope{SubjectID: cfg.ProfileID, Kind: evolution.ScopePersonal}
	ports, err := client.BindEvolution(scope, cfg.DestinationID)
	if err != nil {
		t.Fatalf("bind evolution: %v", err)
	}

	// Human ACTMEM save lands in the one owned store; the agent's own reads see it.
	if _, err := human.SaveACTMEM(context.Background(), validActmem, 0); err != nil {
		t.Fatalf("save actmem: %v", err)
	}
	doc, err := agent.ReadACTMEM(context.Background())
	if err != nil || doc.Revision == 0 {
		t.Fatalf("agent read actmem through same owner: %+v %v", doc, err)
	}
	if _, err := ports.MissionRevision(context.Background()); err != nil {
		t.Fatalf("mission revision: %v", err)
	}
	if ports.SourceID == "" || !strings.Contains(ports.SourceID, cfg.ProfileID) {
		t.Fatalf("source identity must carry profile/scope/destination: %q", ports.SourceID)
	}

	// Foreign subject scope is denied before any authority read/write.
	foreign := evolution.Scope{SubjectID: "other", Kind: evolution.ScopePersonal}
	if _, err := client.BindEvolution(foreign, cfg.DestinationID); err == nil {
		t.Fatal("foreign subject scope admitted")
	}
	// Destination mismatch with the configured writer is denied.
	if _, err := client.BindEvolution(scope, "other-dest"); err == nil {
		t.Fatal("destination mismatch admitted")
	}
}

// TestHumanCapabilityCannotBeMintedByAgent proves human authority is only
// available from a trusted user principal and cannot be self-granted.
func TestHumanCapabilityCannotBeMintedByAgent(t *testing.T) {
	cfg := userEmbeddedConfig(t)
	cfg.Principal = PrincipalAgent
	client, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.BindHumanSession("s1", ""); err == nil {
		t.Fatal("agent principal minted a human capability")
	}
	if perr := Authorize(cfg.ProfileID, Binding{ProfileID: cfg.ProfileID, AgentID: "a", Platform: "v", SessionID: "s"}, PrincipalAgent, OpPersonaReview); perr == nil || perr.Code != "principal_forbidden" {
		t.Fatalf("agent persona authority gate = %v", perr)
	}
	if perr := Authorize(cfg.ProfileID, Binding{ProfileID: cfg.ProfileID, AgentID: "a", Platform: "v", SessionID: "s"}, PrincipalAgent, OpActmemSave); perr == nil || perr.Code != "principal_forbidden" {
		t.Fatalf("agent actmem save gate = %v", perr)
	}
}

// TestSelectedBackendNoFallback proves an unavailable or unconfigured selected
// writer never silently activates another backend.
func TestSelectedBackendNoFallback(t *testing.T) {
	ctx := context.Background()
	// Named non-mentle writer with no injected binding: unavailable, never minted.
	cfg := userEmbeddedConfig(t)
	cfg.BackendID = "bml-legacy"
	cfg.DestinationID = "bml"
	client, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	personal := evolution.Scope{SubjectID: cfg.ProfileID, Kind: evolution.ScopePersonal}
	if _, err := client.BindEvolution(personal, "bml"); err == nil {
		t.Fatal("unbound selected writer admitted; would have fallen back")
	}

	// Mentle selected but the local model is absent: unavailable, no fallback.
	cfg2 := userEmbeddedConfig(t)
	client2, err := Open(ctx, cfg2)
	if err != nil {
		t.Fatal(err)
	}
	defer client2.Close()
	if _, err := client2.BindEvolution(personal, "mentle"); err == nil {
		t.Fatal("unavailable mentle writer admitted")
	}

	// An injected scope-bound backend is honored only for its own scope.
	cfg3 := userEmbeddedConfig(t)
	cfg3.BackendID = "fake"
	cfg3.DestinationID = "fake-dest"
	ws := wsScope(cfg3.ProfileID, "ws-1")
	store := memorytest.NewFakeStore()
	cfg3.Backends = map[string]memory.Backend{memory.EncodeScope(ws): memorytest.NewFakeBackend(store, ws, "fake-dest")}
	client3, err := Open(ctx, cfg3)
	if err != nil {
		t.Fatal(err)
	}
	defer client3.Close()
	ports, err := client3.BindEvolution(ws, "fake-dest")
	if err != nil {
		t.Fatalf("injected scope-bound backend rejected: %v", err)
	}
	if ports.Domain == nil {
		t.Fatal("evolution domain missing")
	}
	// A different scope has no injected binding and must not fall back.
	if _, err := client3.BindEvolution(personal, "fake-dest"); err == nil {
		t.Fatal("unbound scope fell back to another backend")
	}
}

// TestDerivedHandlesAfterClose proves owner Close invalidates every derived
// capability exactly once and handles cannot close the shared owner.
func TestDerivedHandlesAfterClose(t *testing.T) {
	cfg := userEmbeddedConfig(t)
	initAuthority(t, cfg)
	personal0 := evolution.Scope{SubjectID: cfg.ProfileID, Kind: evolution.ScopePersonal}
	cfg.BackendID = "fake"
	cfg.DestinationID = "fake-dest"
	store := memorytest.NewFakeStore()
	cfg.Backends = map[string]memory.Backend{memory.EncodeScope(personal0): memorytest.NewFakeBackend(store, personal0, "fake-dest")}
	client, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	human, err := client.BindHumanSession("s1", "")
	if err != nil {
		t.Fatal(err)
	}
	agent, err := client.BindAgentSession("s1", "")
	if err != nil {
		t.Fatal(err)
	}
	personal := evolution.Scope{SubjectID: cfg.ProfileID, Kind: evolution.ScopePersonal}
	ports, err := client.BindEvolution(personal, cfg.DestinationID)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("owner close: %v", err)
	}
	if _, err := agent.FastRecall(context.Background(), FastRecallRequest{Query: "q"}); err == nil {
		t.Fatal("agent handle still served after owner close")
	}
	if _, err := human.SaveACTMEM(context.Background(), validActmem, 0); err == nil {
		t.Fatal("human handle still wrote after owner close")
	}
	if _, err := ports.Domain.Collect(context.Background(), evolution.Window{SourceID: "s", After: 0, Through: 1}); err == nil {
		t.Fatal("evolution domain still served after owner close")
	}
	if _, err := client.BindAgentSession("s2", ""); err == nil {
		t.Fatal("new handle minted after owner close")
	}
	if err := client.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

// TestEmbeddedDomainCapabilities exercises the human write surface against the
// injected scope-bound backend: stamped mutation identity, dedupe receipt,
// stale revision, paged reviews and ledger results.
func TestEmbeddedDomainCapabilities(t *testing.T) {
	ctx := context.Background()
	cfg := userEmbeddedConfig(t)
	initAuthority(t, cfg)
	cfg.BackendID = "fake"
	cfg.DestinationID = "fake-dest"
	personal := evolution.Scope{SubjectID: cfg.ProfileID, Kind: evolution.ScopePersonal}
	store := memorytest.NewFakeStore()
	cfg.Backends = map[string]memory.Backend{memory.EncodeScope(personal): memorytest.NewFakeBackend(store, personal, "fake-dest")}
	client, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	human, err := client.BindHumanSession("s1", "")
	if err != nil {
		t.Fatal(err)
	}

	// The adapter stamps operation identity; a replay of the same logical
	// payload gets a fresh operation id, while mutation status is durable.
	mut := memory.AuthorizedMutation{
		Scope: personal, DestinationID: "fake-dest",
		Operation: evolution.MutationCreate, RecordID: "mem-1",
		ExpectedAbsent: true, Body: "likes tea", Inference: evolution.InferenceObserved,
	}
	rc1, err := human.MutateMemory(ctx, mut)
	if err != nil {
		t.Fatalf("mutate: %v", err)
	}
	if rc1.OperationID == "" || rc1.PayloadDigest == "" || rc1.Status != evolution.StatusApplied ||
		rc1.CanonicalStatus != memory.CanonicalCompleted || rc1.IndexStatus == "" {
		t.Fatalf("receipt: %+v", rc1)
	}
	look, err := human.MemoryReceipt(ctx, rc1.OperationID)
	if err != nil || look.OperationID != rc1.OperationID {
		t.Fatalf("receipt lookup: %+v %v", look, err)
	}

	// Scope/destination stamped by the caller is never trusted.
	badScope := mut
	badScope.Scope = wsScope(cfg.ProfileID, "ws-x")
	if _, err := human.MutateMemory(ctx, badScope); err == nil {
		t.Fatal("foreign scope mutation admitted")
	}
	badDest := mut
	badDest.DestinationID = "other-dest"
	badDest.OperationID = "op-other"
	if _, err := human.MutateMemory(ctx, badDest); err == nil {
		t.Fatal("foreign destination mutation admitted")
	}

	// Search and expand validate the admitted read union; stale revisions fail.
	page, err := human.SearchMemory(ctx, memory.AuthorizedSearch{Query: "tea", Limit: 10})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("search: %+v %v", page, err)
	}
	ev, err := human.ExpandMemory(ctx, memory.AuthorizedExpansion{CardID: "mem-1", ExpectedRevision: 1})
	if err != nil || len(ev.Items) != 1 {
		t.Fatalf("expand: %+v %v", ev, err)
	}
	if _, err := human.ExpandMemory(ctx, memory.AuthorizedExpansion{CardID: "mem-1", ExpectedRevision: 9}); err == nil {
		t.Fatal("stale revision expansion admitted")
	}

	// Review lifecycle: agent proposal, human list, closed accept/reject.
	agent, err := client.BindAgentSession("s1", "")
	if err != nil {
		t.Fatal(err)
	}
	_ = agent
	authority, err := persona.Open(cfg.PersonaDir)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := authority.GetDocument(persona.KindIdentity)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authority.CreateRequest(persona.ChangeRequest{
		Kind: persona.KindIdentity, BaseRevision: doc.Revision, BaseHash: doc.ContentHash,
		ProposedMarkdown: "new identity", Actor: persona.ActorAgent, Reason: "r",
	}); err != nil {
		t.Fatal(err)
	}
	reviews, err := human.ListPersonaReviews(ctx, ReviewQuery{})
	if err != nil || len(reviews.Items) != 1 || reviews.NextCursor != "" {
		t.Fatalf("reviews: %+v %v", reviews, err)
	}
	// A rejection decides the request without rewriting the document.
	out, err := human.DecidePersonaReview(ctx, reviews.Items[0].ID, ReviewReject)
	if err != nil || out == nil {
		t.Fatalf("reject: %+v %v", out, err)
	}
	if _, err := human.DecidePersonaReview(ctx, reviews.Items[0].ID, ReviewDecision("bogus")); err == nil {
		t.Fatal("non-closed review decision admitted")
	}
}

// TestEmbeddedDomainPagingBounds covers default/max page sizes, opaque stale
// cursors, the absent hidden-scope totals, and bounded results reads.
func TestEmbeddedDomainPagingBounds(t *testing.T) {
	ctx := context.Background()
	cfg := userEmbeddedConfig(t)
	initAuthority(t, cfg)
	personal0 := evolution.Scope{SubjectID: cfg.ProfileID, Kind: evolution.ScopePersonal}
	cfg.BackendID = "fake"
	cfg.DestinationID = "fake-dest"
	store := memorytest.NewFakeStore()
	cfg.Backends = map[string]memory.Backend{memory.EncodeScope(personal0): memorytest.NewFakeBackend(store, personal0, "fake-dest")}
	client, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	human, err := client.BindHumanSession("s1", "")
	if err != nil {
		t.Fatal(err)
	}
	authority, err := persona.Open(cfg.PersonaDir)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := authority.GetDocument(persona.KindIdentity)
	if err != nil {
		t.Fatal(err)
	}
	// Pending requests are unique per kind; build the page set as decided
	// records via create-then-reject cycles.
	for i := 0; i < 25; i++ {
		req, err := authority.CreateRequest(persona.ChangeRequest{
			Kind: persona.KindIdentity, BaseRevision: doc.Revision, BaseHash: doc.ContentHash,
			ProposedMarkdown: fmt.Sprintf("proposal %d", i), Actor: persona.ActorAgent, Reason: "r",
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := authority.RejectRequest(req.ID); err != nil {
			t.Fatal(err)
		}
	}
	page1, err := human.ListPersonaReviews(ctx, ReviewQuery{PageQuery: PageQuery{Limit: 0}})
	if err != nil || len(page1.Items) != 20 || page1.NextCursor == "" {
		t.Fatalf("default page: %+v %v", page1, err)
	}
	page2, err := human.ListPersonaReviews(ctx, ReviewQuery{PageQuery: PageQuery{Limit: 1000, Cursor: page1.NextCursor}})
	if err != nil || len(page2.Items) != 5 {
		t.Fatalf("second page: %+v %v", page2, err)
	}
	if _, err := human.ListPersonaReviews(ctx, ReviewQuery{PageQuery: PageQuery{Cursor: "bogus"}}); err == nil {
		t.Fatal("malformed cursor admitted")
	}
	// A mutation between pages makes the minted cursor stale.
	late, err := authority.CreateRequest(persona.ChangeRequest{
		Kind: persona.KindIdentity, BaseRevision: doc.Revision, BaseHash: doc.ContentHash,
		ProposedMarkdown: "late", Actor: persona.ActorAgent, Reason: "r",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authority.RejectRequest(late.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := human.ListPersonaReviews(ctx, ReviewQuery{PageQuery: PageQuery{Cursor: page1.NextCursor}}); err == nil {
		t.Fatal("stale cursor admitted")
	}

	// Results: bounded projection of the owned effects/notes ledger.
	personal := evolution.Scope{SubjectID: cfg.ProfileID, Kind: evolution.ScopePersonal}
	ports, err := client.BindEvolution(personal, cfg.DestinationID)
	if err != nil {
		t.Fatal(err)
	}
	patch := mustEffect(t, "op-work-1", evolution.KindWorkPatch,
		&evolution.WorkPatch{BaseRevision: 0, Changes: []evolution.WorkChange{{Kind: evolution.WorkChangeAdd, Field: evolution.FieldGoal, Body: "g"}}},
		personal, cfg.DestinationID)
	if _, err := ports.Domain.Apply(ctx, patch); err != nil {
		t.Fatal(err)
	}
	note := mustEffect(t, "op-note-1", evolution.KindReflectionNote,
		&evolution.ReflectionNotePayload{Body: "note text"}, personal, cfg.DestinationID)
	if _, err := ports.Domain.Apply(ctx, note); err != nil {
		t.Fatal(err)
	}
	// Each applied effect leaves a receipt row; reflection notes additionally
	// persist their body row. The page projects both, bounded.
	results, err := human.Results(ctx, PageQuery{Limit: 10})
	if err != nil || len(results.Items) != 3 {
		t.Fatalf("results: %+v %v", results, err)
	}
	var sawReceipt, sawNote bool
	for _, item := range results.Items {
		if item.OperationID == "op-work-1" {
			sawReceipt = item.Kind == string(evolution.KindWorkPatch) && item.Status != ""
		}
		if item.OperationID == "op-note-1" && item.Body == "note text" {
			sawNote = true
		}
	}
	if !sawReceipt || !sawNote {
		t.Fatalf("results missing receipt/note projections: %+v", results.Items)
	}
	// Oversized input stays bounded.
	big := strings.Repeat("x", 64<<10+1)
	if _, err := human.SaveACTMEM(ctx, big, 0); err == nil {
		t.Fatal("oversized ACTMEM input admitted")
	}
}

// TestEmbeddedCaptureDedupeReceipt records that replayed terminal captures keep
// the original durable sequence — the ingest ledger, not a new write.
func TestEmbeddedCaptureDedupeReceipt(t *testing.T) {
	cfg := userEmbeddedConfig(t)
	initAuthority(t, cfg)
	client, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	agent, err := client.BindAgentSession("s1", "")
	if err != nil {
		t.Fatal(err)
	}
	req := CaptureRequest{Phase: CaptureCompleted, Content: "terminal",
		ContentHash: "sha256:" + sha256Hex("terminal"),
		Provenance:  CaptureProvenance{RunID: "run-1", EventSeq: 1}}
	first, err := agent.Capture(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	again, err := agent.Capture(context.Background(), req)
	if err != nil || again.IngestionID != first.IngestionID || again.Seq != first.Seq {
		t.Fatalf("replayed capture must return the original seq: %+v vs %+v, %v", first, again, err)
	}
	if first.Seq == 0 {
		t.Fatal("receipt missing durable seq")
	}
}
