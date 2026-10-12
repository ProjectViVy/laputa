package agentapi

import (
	"context"
	"github.com/ProjectViVy/laputa/garden/internal/recall"
	"github.com/ProjectViVy/laputa/garden/internal/registry"
	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApprovedSliceRecallOnlyMustOmitPersona(t *testing.T) {
	ctx := context.Background()
	cfg := installationConfig(t)
	cfg.CatalogOnly = true
	i, err := OpenInstallation(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer i.Close()
	installPersona(t, i, "A")
	writer, err := installedClient(t, i, "A").BindSession("author")
	if err != nil {
		t.Fatal(err)
	}
	createBoundMemory(t, writer, "", "permitted", "needle PERMITTED_MEMORY")
	host, _, err := i.CreateHostBinding(ctx, HostBinding{ProfileID: "A", AgentID: "review", Platform: "test"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := i.BindHostSession(host.BindingID, "chat")
	if err != nil {
		t.Fatal(err)
	}
	// Reuse the same admitted session before, during and after PersonaSync.
	for _, sync := range []bool{false, true, false} {
		if err = i.SetHostControls(ctx, host.BindingID, HostControls{Recall: true, PersonaSync: sync}); err != nil {
			t.Fatal(err)
		}
		if !sync {
			_, err = b.ReadPersona(ctx, "identity")
			assertCode(t, err, "principal_forbidden")
		}
		bootstrap, e1 := b.Bootstrap(ctx, BootstrapRequest{Intent: "needle"})
		fast, e2 := b.FastRecall(ctx, FastRecallRequest{Query: "needle"})
		service, binding := b.AdmittedService()
		deep, e3 := service.DeepRecall(ctx, PrincipalAgent, binding, recall.DeepRequest{Query: "needle", TriggerReason: "explicit"})
		for _, err := range []error{e1, e2, e3} {
			if err != nil {
				t.Fatal(err)
			}
		}
		for name, content := range map[string]string{"bootstrap": bootstrap.Context, "fast": fast.Context, "deep": deep.Context} {
			if strings.Contains(content, "A identity") != sync {
				t.Fatalf("%s PersonaSync=%t context=%q", name, sync, content)
			}
			if !strings.Contains(content, "PERMITTED_MEMORY") {
				t.Fatalf("%s lost admitted memory with PersonaSync=%t: %q", name, sync, content)
			}
		}
		if (bootstrap.FrozenCore.Content(evolution.FrozenKindIdentity) != "") != sync {
			t.Fatal("Frozen snapshot did not follow current PersonaSync")
		}
	}
}

func TestApprovedSliceBoundACTMEMDoesNotReadOtherWorkspace(t *testing.T) {
	ctx := context.Background()
	i, err := OpenInstallation(ctx, installationConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	defer i.Close()
	installPersona(t, i, "A")
	g, err := i.owner.Subject(ctx, "A")
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.Actmem.AppendEntry(evolution.Entry{Section: evolution.SectionPulse, Scope: evolution.Scope{SubjectID: "A", Kind: evolution.ScopeWorkspace, WorkspaceID: "secret"}, SessionID: "private-session", Body: "OTHER_WORKSPACE_SECRET"})
	if err != nil {
		t.Fatal(err)
	}
	for _, scope := range []evolution.Scope{{SubjectID: "A", Kind: evolution.ScopePersonal}, {SubjectID: "A", Kind: evolution.ScopeWorkspace, WorkspaceID: "public"}} {
		if _, err = g.Actmem.AppendEntry(evolution.Entry{Section: evolution.SectionPulse, Scope: scope, SessionID: "allowed", Body: "PERMITTED_" + string(scope.Kind)}); err != nil {
			t.Fatal(err)
		}
	}
	host, _, err := i.CreateHostBinding(ctx, HostBinding{ProfileID: "A", AgentID: "review", Platform: "test", WorkspaceID: "public", Operations: []Operation{OpActmemRead, OpActmemQuery, OpActmemMaintain}})
	if err != nil {
		t.Fatal(err)
	}
	if err = i.SetHostControls(ctx, host.BindingID, HostControls{Recall: true, Capture: true}); err != nil {
		t.Fatal(err)
	}
	b, err := i.BindHostSession(host.BindingID, "chat")
	if err != nil {
		t.Fatal(err)
	}
	q, err := b.QueryACTMEM(ctx, ActmemQuery{Query: "SECRET"})
	if err != nil || len(q.Items) != 0 {
		t.Fatal("scoped baseline", q, err)
	}
	doc, err := b.ReadACTMEM(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Markdown, "PERMITTED_personal") || !strings.Contains(doc.Markdown, "PERMITTED_workspace") {
		t.Fatal("scoped read dropped admitted entries", doc)
	}
	if strings.Contains(doc.Markdown, "OTHER_WORKSPACE_SECRET") {
		t.Fatal("BoundClient.ReadACTMEM disclosed secret workspace content while QueryACTMEM denied it")
	}
	service, binding := b.AdmittedService()
	_, err = service.FoldSession(ctx, PrincipalAgent, binding, "private-session")
	assertCode(t, err, "unavailable")
	owner, err := i.BindProfile("A", ClientConfig{AgentID: "owner", Platform: "test", Principal: PrincipalUser})
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	ownerSession, err := owner.BindSession("owner-read")
	if err != nil {
		t.Fatal(err)
	}
	ownerDoc, err := ownerSession.ReadACTMEM(ctx)
	if err != nil || !strings.Contains(ownerDoc.Markdown, "OTHER_WORKSPACE_SECRET") {
		t.Fatal("explicit owner head lost", ownerDoc, err)
	}
	service, binding = ownerSession.AdmittedService()
	capsules, err := service.FoldSession(ctx, PrincipalUser, binding, "private-session")
	if err != nil || len(capsules) != 1 {
		t.Fatal("explicit owner fold unavailable", capsules, err)
	}
}

func TestApprovedSliceTagOnlyMutationReplay(t *testing.T) {
	f := sharingInstallation(t)
	ctx := context.Background()
	r := createBoundMemory(t, f.a, "", "replay", "old body")
	tags := []string{"tag"}
	m := memory.AuthorizedMutation{OperationID: "tag-op", Operation: evolution.MutationUpdate, RecordID: r.TargetRef, ExpectedRevision: 1, Tags: &tags, Inference: evolution.InferenceObserved}
	original, err := f.a.MutateMemory(ctx, MemoryTarget{}, m)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.a.MutateMemory(ctx, MemoryTarget{}, memory.AuthorizedMutation{OperationID: "body-op", Operation: evolution.MutationUpdate, RecordID: r.TargetRef, ExpectedRevision: 2, Body: "new body", Inference: evolution.InferenceObserved})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := f.a.MutateMemory(ctx, MemoryTarget{}, m)
	if err != nil || replay.OperationID != original.OperationID || replay.Revision != 2 {
		t.Fatalf("same request no longer replays its committed receipt after later edit: original=%+v replay=%+v error=%v", original, replay, err)
	}
	changed := m
	changed.Body = "different request"
	if _, err = f.a.MutateMemory(ctx, MemoryTarget{}, changed); err == nil {
		t.Fatal("changed payload replay accepted")
	}
	current, err := f.a.GetMemory(ctx, r.TargetRef)
	if err != nil || current.Revision != 3 {
		t.Fatal("replay overwrote later body", current, err)
	}
	evidence, err := f.a.ReadEvidence(ctx, EvidenceRead{Items: []EvidenceRef{{CardID: r.TargetRef, ExpectedRevision: 3}}})
	if err != nil || len(evidence) != 1 || evidence[0].Excerpt != "new body" {
		t.Fatal("replay lost newer evidence", evidence, err)
	}
}

func TestApprovedSliceCommittedSourceMutationReplay(t *testing.T) {
	f := sharingInstallation(t)
	ctx := context.Background()
	source := createBoundMemory(t, f.a, "", "source", "source body")
	m := memory.AuthorizedMutation{OperationID: "linked-op", Operation: evolution.MutationCreate, ExpectedAbsent: true, Body: "derived fact", Sources: []evolution.SourceRef{{SourceID: BackendMentle, RecordID: source.TargetRef, Revision: 1, Scope: evolution.Scope{SubjectID: "A", Kind: evolution.ScopePersonal}}}, Inference: evolution.InferenceObserved}
	original, err := f.a.MutateMemory(ctx, MemoryTarget{}, m)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.a.MutateMemory(ctx, MemoryTarget{}, memory.AuthorizedMutation{OperationID: "source-edit", Operation: evolution.MutationUpdate, RecordID: source.TargetRef, ExpectedRevision: 1, Body: "corrected source", Inference: evolution.InferenceObserved})
	if err != nil {
		t.Fatal(err)
	}
	stored, err := f.a.MemoryReceipt(ctx, MemoryTarget{}, original.OperationID)
	if err != nil || stored.TargetRef != original.TargetRef {
		t.Fatal(stored, err)
	}
	_, err = f.a.MutateMemory(ctx, MemoryTarget{}, m)
	if err != nil {
		t.Fatalf("unchanged committed request rejected after source changed despite persisted receipt: %v", err)
	}
	changed := m
	changed.Body = "changed payload"
	if _, err = f.a.MutateMemory(ctx, MemoryTarget{}, changed); err == nil {
		t.Fatal("changed linked replay accepted")
	}
	fresh := m
	fresh.OperationID = "new-stale-source-operation"
	if _, err = f.a.MutateMemory(ctx, MemoryTarget{}, fresh); err == nil {
		t.Fatal("new write reused stale source revision")
	}
	service, _ := f.a.AdmittedService()
	if err = f.i.SetHostControls(ctx, service.bindingID, HostControls{Recall: true}); err != nil {
		t.Fatal(err)
	}
	if _, err = f.a.MutateMemory(ctx, MemoryTarget{}, m); err == nil {
		t.Fatal("replay bypassed current capture control")
	}
}

func TestApprovedSliceHostProjectionCannotReplaceOwnerLock(t *testing.T) {
	ctx := context.Background()
	cfg := installationConfig(t)
	cfg.PalacePath = t.TempDir()
	cfg.CatalogOnly = true
	i, err := OpenInstallation(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer i.Close()
	installPersona(t, i, "A")
	b, _, err := i.CreateHostBinding(ctx, HostBinding{ProfileID: "A", AgentID: "review", Platform: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if err = i.SetHostControls(ctx, b.BindingID, HostControls{PersonaSync: true}); err != nil {
		t.Fatal(err)
	}
	secondCfg := installationConfig(t)
	secondCfg.PalacePath = cfg.PalacePath
	secondCfg.CatalogOnly = true
	blocked, baselineErr := OpenInstallation(ctx, secondCfg)
	if baselineErr == nil {
		blocked.Close()
		t.Fatal("baseline second writer was not rejected")
	}
	external := filepath.Join(t.TempDir(), "authority")
	if _, err = i.RegisterProfile(ctx, ProfileRegistration{ProfileID: "external", AuthorityDir: external, Create: true}); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "lock-alias")
	lock := registry.OwnershipLockPath(cfg.PalacePath)
	if err = os.Symlink(lock, alias); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{lock, registry.OwnershipLockPath(external), alias} {
		_, err = i.PreviewHostProjection(ctx, HostProjectionRequest{BindingID: b.BindingID, Path: path})
		assertCode(t, err, "invalid_request")
	}
	second, err := OpenInstallation(ctx, secondCfg)
	if err == nil {
		defer second.Close()
		t.Fatal("lock guard admitted a second canonical writer")
	}
}
