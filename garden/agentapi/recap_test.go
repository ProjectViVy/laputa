package agentapi

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ProjectViVy/laputa/laputa/actmem"
	"github.com/ProjectViVy/laputa/laputa/persona"
)

type recapFacade interface {
	ReadRecap(context.Context) (actmem.RecapHead, error)
	CommitRecap(context.Context, actmem.RecapCommit) (actmem.RecapHead, error)
	LookupRecapOperation(context.Context, string) (actmem.RecapHead, bool, error)
	ListRecapHistory(context.Context, uint64, int) ([]actmem.RecapHead, error)
	ReadRecapRevision(context.Context, uint64) (actmem.RecapHead, error)
}

func recapConfig(t *testing.T) Config {
	t.Helper()
	cfg := clientConfig(t)
	cfg.Principal = PrincipalUser
	cfg.ResolveRecapBinding = func(_ context.Context, root string) (actmem.RecapBinding, error) {
		if root != filepath.Join(cfg.PersonaDir, "actmem") {
			t.Fatalf("resolver root=%q", root)
		}
		return actmem.RecapBinding{SubjectID: "organism-uuid", DomainID: "shared-personal-v1"}, nil
	}
	return cfg
}

func openRecap(t *testing.T, cfg Config) (*Client, recapFacade) {
	t.Helper()
	c, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	bound, err := c.BindRecap()
	if err != nil {
		t.Fatal(err)
	}
	facade, ok := any(bound).(recapFacade)
	if !ok {
		t.Fatal("bound owner is missing C1 Recap facade")
	}
	return c, facade
}

func TestIssue58RecapClearRestore(t *testing.T) {
	ctx := context.Background()
	c, recap := openRecap(t, recapConfig(t))
	if _, err := c.runtime.Persona.Initialize(persona.Initialization{Identity: "identity", Relationship: "relation", Redline: "redline", User: "user", World: "world"}, "user", persona.SourceInit, "init"); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []persona.Kind{persona.KindDream, persona.KindMission} {
		if _, err := c.runtime.Persona.SaveUserDocument(kind, "preserved "+kind.String(), 0, "user", persona.SourceUserDirect, "test"); err != nil {
			t.Fatal(err)
		}
	}
	before := map[persona.Kind]persona.Document{}
	for _, kind := range []persona.Kind{persona.KindIdentity, persona.KindDream, persona.KindMission} {
		doc, err := c.runtime.Persona.GetDocument(kind)
		if err != nil {
			t.Fatal(err)
		}
		before[kind] = *doc
	}
	initial, err := recap.ReadRecap(ctx)
	if err != nil || initial.Revision != 0 || initial.Binding.SubjectID == c.identity.ProfileID {
		t.Fatalf("initial=%+v err=%v", initial, err)
	}
	summary := actmem.RecapCommit{OperationID: "summary", Cause: "summary", Text: "remembered", Sources: actmem.SourceRange{Through: 12, Digest: "opaque-host-receipt"}}
	first, err := recap.CommitRecap(ctx, summary)
	if err != nil {
		t.Fatal(err)
	}
	edit, err := recap.CommitRecap(ctx, actmem.RecapCommit{BaseRevision: first.Revision, OperationID: "edit", Cause: "edit", Text: "corrected"})
	if err != nil {
		t.Fatal(err)
	}
	advanced, err := recap.CommitRecap(ctx, actmem.RecapCommit{BaseRevision: edit.Revision, OperationID: "advance", Cause: "summary", Text: "later history", Sources: actmem.SourceRange{After: 12, Through: 20, Digest: "second-host-receipt"}})
	if err != nil {
		t.Fatal(err)
	}
	cleared, err := recap.CommitRecap(ctx, actmem.RecapCommit{BaseRevision: advanced.Revision, OperationID: "clear", Cause: "clear"})
	if err != nil || cleared.Text != "" || cleared.CoveredThrough != 20 || cleared.InvalidationEpoch != 1 {
		t.Fatalf("clear=%+v err=%v", cleared, err)
	}
	stale := actmem.RecapCommit{BaseRevision: advanced.Revision, OperationID: "stale", Cause: "summary", Text: "resurrected", Sources: actmem.SourceRange{After: 20, Through: 30, Digest: "host"}}
	if _, err := recap.CommitRecap(ctx, stale); err == nil {
		t.Fatal("stale summary resurrected cleared text")
	} else {
		assertCode(t, err, "recap_conflict")
	}
	head, err := recap.ReadRecap(ctx)
	if err != nil || head != cleared {
		t.Fatalf("stale write changed head: %+v %v", head, err)
	}
	restored, err := recap.CommitRecap(ctx, actmem.RecapCommit{BaseRevision: cleared.Revision, OperationID: "restore", Cause: "restore", RestoredFrom: first.Revision})
	if err != nil || restored.Revision != 5 || restored.Text != first.Text || restored.CoveredThrough != 20 || restored.InvalidationEpoch != 1 {
		t.Fatalf("restore=%+v err=%v", restored, err)
	}
	replay, err := recap.CommitRecap(ctx, summary)
	if err != nil || replay != first {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	receipt, found, err := recap.LookupRecapOperation(ctx, "clear")
	if err != nil || !found || receipt != cleared {
		t.Fatalf("lookup=%+v found=%v err=%v", receipt, found, err)
	}
	history, err := recap.ListRecapHistory(ctx, 0, 2)
	if err != nil || len(history) != 2 || history[0] != restored || history[1] != cleared {
		t.Fatalf("history=%+v %v", history, err)
	}
	older, err := recap.ListRecapHistory(ctx, cleared.Revision, 2)
	if err != nil || len(older) != 2 || older[0] != advanced || older[1] != edit {
		t.Fatalf("older=%+v %v", older, err)
	}
	revision, err := recap.ReadRecapRevision(ctx, first.Revision)
	if err != nil || revision != first {
		t.Fatalf("revision=%+v %v", revision, err)
	}
	for kind, want := range before {
		got, err := c.runtime.Persona.GetDocument(kind)
		if err != nil || !reflect.DeepEqual(*got, want) {
			t.Fatalf("%s changed: %v", kind, err)
		}
	}
	// The owner is exactly the Store used by the rest of Garden.
	direct, err := c.runtime.Actmem.ReadRecap(ctx, initial.Binding)
	if err != nil || direct != restored {
		t.Fatalf("second owner/store: %+v %v", direct, err)
	}
}

func assertRecapDenied(t *testing.T, recap recapFacade, code string) {
	t.Helper()
	ctx := context.Background()
	head, err := recap.ReadRecap(ctx)
	assertCode(t, err, code)
	if head != (actmem.RecapHead{}) {
		t.Fatal("denied read leaked content")
	}
	head, found, err := recap.LookupRecapOperation(ctx, "secret")
	assertCode(t, err, code)
	if found || head != (actmem.RecapHead{}) {
		t.Fatal("denied lookup leaked receipt")
	}
	history, err := recap.ListRecapHistory(ctx, 0, 50)
	assertCode(t, err, code)
	if len(history) != 0 {
		t.Fatal("denied history leaked count")
	}
	head, err = recap.ReadRecapRevision(ctx, 1)
	assertCode(t, err, code)
	if head != (actmem.RecapHead{}) {
		t.Fatal("denied revision leaked content")
	}
	head, err = recap.CommitRecap(ctx, actmem.RecapCommit{OperationID: "denied", Cause: "edit", Text: "overwrite"})
	assertCode(t, err, code)
	if head != (actmem.RecapHead{}) {
		t.Fatal("denied write leaked receipt")
	}
}

func TestIssue58RecapFacadeScopeDenied(t *testing.T) {
	ctx := context.Background()
	cfg := recapConfig(t)
	c, recap := openRecap(t, cfg)
	original, err := recap.CommitRecap(ctx, actmem.RecapCommit{OperationID: "secret", Cause: "edit", Text: "secret text"})
	if err != nil {
		t.Fatal(err)
	}
	for _, binding := range []actmem.RecapBinding{{SubjectID: "foreign", DomainID: original.Binding.DomainID}, {SubjectID: original.Binding.SubjectID, DomainID: "foreign"}} {
		got, err := recap.CommitRecap(ctx, actmem.RecapCommit{Binding: binding, BaseRevision: 1, OperationID: "foreign", Cause: "edit", Text: "overwrite"})
		assertCode(t, err, "recap_scope_denied")
		if got != (actmem.RecapHead{}) {
			t.Fatal("foreign commit leaked head")
		}
	}
	worker, err := c.BindAgentSession("worker", "restricted")
	if err != nil {
		t.Fatal(err)
	}
	assertRecapDenied(t, worker, "recap_scope_denied")
	// A separately configured trusted binding still cannot read this head.
	for _, foreign := range []actmem.RecapBinding{{SubjectID: "foreign", DomainID: original.Binding.DomainID}, {SubjectID: original.Binding.SubjectID, DomainID: "foreign"}} {
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
		cfg.ResolveRecapBinding = func(context.Context, string) (actmem.RecapBinding, error) { return foreign, nil }
		c, recap = openRecap(t, cfg)
		assertRecapDenied(t, recap, "recap_scope_denied")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	assertRecapDenied(t, recap, "recap_unavailable")
}

func TestIssue58RecapOptionalBindingAndLegacyPreservation(t *testing.T) {
	cfg := clientConfig(t)
	cfg.Principal = PrincipalUser
	c, recap := openRecap(t, cfg)
	assertRecapDenied(t, recap, "recap_unavailable")
	bound, err := c.BindHumanSession("unnamed", "")
	if err != nil {
		t.Fatal(err)
	}
	if status, err := bound.PersonaStatus(context.Background()); err != nil || status.Status != persona.StatusUninitialized {
		t.Fatalf("ordinary uninitialized persona: %+v %v", status, err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	cfg = recapConfig(t)
	legacy := actmem.New(cfg.PersonaDir)
	seedEntry(t, legacy, "recap", personalScope("default"), "legacy-session", "legacy recap")
	raw, err := os.ReadFile(legacy.HeadPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	_, recap = openRecap(t, cfg)
	assertRecapDenied(t, recap, "recap_legacy_nonempty")
	after, err := os.ReadFile(legacy.HeadPath())
	if err != nil || string(after) != string(raw) {
		t.Fatal("legacy head changed")
	}
}

func TestIssue58RecapResolverFailureClosesPartialOwner(t *testing.T) {
	cfg := recapConfig(t)
	want := errors.New("subject path mismatch")
	cfg.ResolveRecapBinding = func(context.Context, string) (actmem.RecapBinding, error) { return actmem.RecapBinding{}, want }
	c, err := Open(context.Background(), cfg)
	if c != nil || !errors.Is(err, want) {
		t.Fatalf("client=%v err=%v", c, err)
	}
	if _, err := os.Stat(cfg.StateDB); !os.IsNotExist(err) {
		t.Fatalf("resolver ran after state/ingestion startup: %v", err)
	}
}

func TestIssue58RecapOwnerLifecycle(t *testing.T) {
	cfg := clientConfig(t)
	c, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	store := c.runtime.Actmem
	binding := actmem.RecapBinding{SubjectID: "subject", DomainID: "personal"}
	if _, err := store.CommitRecap(context.Background(), actmem.RecapCommit{Binding: binding, OperationID: "first", Cause: "edit", Text: "owned"}); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadRecap(context.Background(), binding); actmem.CodeOf(err) != "recap_unavailable" {
		t.Fatalf("closed Garden left ACTMEM owner readable: %v", err)
	}
	reopened, err := actmem.Open(cfg.PersonaDir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
}

func TestIssue58RecapFacadeOwnerErrors(t *testing.T) {
	ctx := context.Background()
	cfg := recapConfig(t)
	_, recap := openRecap(t, cfg)
	// Recap has no legacy per-entry/ring text cap, and the host range digest is opaque.
	request := actmem.RecapCommit{OperationID: "original", Cause: "summary", Text: strings.Repeat("exact free text\n", 2000), Sources: actmem.SourceRange{Through: 7, Digest: "opaque-receipt"}}
	first, err := recap.CommitRecap(ctx, request)
	if err != nil || first.Text != request.Text {
		t.Fatalf("text changed: %v", err)
	}
	request.Text = "different request"
	_, err = recap.CommitRecap(ctx, request)
	assertCode(t, err, "recap_operation_conflict")
	_, err = recap.CommitRecap(ctx, actmem.RecapCommit{BaseRevision: 1, OperationID: "gap", Cause: "summary", Sources: actmem.SourceRange{After: 8, Through: 9, Digest: "host"}})
	assertCode(t, err, "recap_invalid_range")
	// Armed construction pins the owner before binding or ingestion, so a
	// competitor must be rejected before it can obtain a bound facade.
	competing, err := Open(ctx, cfg)
	if competing != nil {
		_ = competing.Close()
		t.Fatal("competing armed owner constructed a client")
	}
	if actmem.CodeOf(err) != "recap_owner_busy" {
		t.Fatalf("armed construction lost owner error: %v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = recap.ReadRecap(canceled)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("context cancellation changed: %v", err)
	}
}
