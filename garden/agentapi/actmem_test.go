package agentapi

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dashimaki/garden/internal/runtimecore"
	"github.com/dashimaki/laputa/actmem"
	"github.com/dashimaki/laputa/evolution"
)

func seedEntry(t *testing.T, a *actmem.Store, section evolution.EntrySection, scope evolution.Scope, sessionID, body string) {
	t.Helper()
	if _, err := a.AppendEntry(evolution.Entry{Section: section, Scope: scope, SessionID: sessionID, Body: body}); err != nil {
		t.Fatal(err)
	}
}

func personalScope(subject string) evolution.Scope {
	return evolution.Scope{SubjectID: subject, Kind: evolution.ScopePersonal}
}

func workspaceScope(subject, workspace string) evolution.Scope {
	return evolution.Scope{SubjectID: subject, Kind: evolution.ScopeWorkspace, WorkspaceID: workspace}
}

func TestReadActivityHonoursTrustedScope(t *testing.T) {
	a := actmem.New(t.TempDir())
	seedEntry(t, a, evolution.SectionPulse, personalScope("default"), "s-1", "personal pulse")
	seedEntry(t, a, evolution.SectionPulse, workspaceScope("default", "ws-a"), "s-2", "workspace pulse")
	seedEntry(t, a, evolution.SectionPulse, workspaceScope("other", "ws-a"), "s-3", "foreign subject pulse")

	// Personal caller: sees only their personal + nothing foreign.
	s := NewService(&runtimecore.Garden{ProfileID: "default", Actmem: a})
	result, err := s.ReadActivity(context.Background(), PrincipalAgent, binding(), ReadRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 1 || result.Entries[0].Body != "personal pulse" {
		t.Fatalf("personal entries=%+v", result.Entries)
	}

	// Workspace-bound caller: sees personal + their workspace union.
	bound := Binding{ProfileID: "default", AgentID: "agent-1", Platform: "vivy", SessionID: "s-9", WorkspaceID: "ws-a"}
	result, err = s.ReadActivity(context.Background(), PrincipalAgent, bound, ReadRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("workspace entries=%+v", result.Entries)
	}
}

func TestReadActivityOnUnclassifiedHeadReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	a := actmem.New(dir)
	// Unclassified legacy head: owner read still sees it, scoped read must not.
	legacy := "---\nrevision: 7\nupdated_at: 2026-09-03T12:00:00Z\n---\n\n# ACTMEM\n\n## Pulse\n- legacy pulse\n\n## Recap\n\n## Work\n### Goal\n### Open\n### Next\n### Constraints\n### Pointers\n"
	if err := os.WriteFile(filepath.Join(dir, actmem.ACTMEMFileName), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewService(&runtimecore.Garden{ProfileID: "default", Actmem: a})
	result, err := s.ReadActivity(context.Background(), PrincipalAgent, binding(), ReadRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 0 {
		t.Fatalf("unclassified head leaked %d entries", len(result.Entries))
	}
}

func TestApplyWorkPatchScoped(t *testing.T) {
	a := actmem.New(t.TempDir())
	s := NewService(&runtimecore.Garden{ProfileID: "default", Actmem: a})
	bound := Binding{ProfileID: "default", AgentID: "agent-1", Platform: "vivy", SessionID: "s-1", WorkspaceID: "ws-a"}
	result, err := s.ApplyWorkPatch(context.Background(), PrincipalAgent, bound, WorkPatch{
		BaseRevision: 0,
		Changes: []WorkChange{
			{Kind: "add", Field: "goal", Body: "workspace goal"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || len(result.Entries) != 1 {
		t.Fatalf("patch result=%+v", result)
	}
	// Personal (unbound) caller cannot drop the workspace entry.
	_, err = s.ApplyWorkPatch(context.Background(), PrincipalAgent, binding(), WorkPatch{
		BaseRevision: result.Revision,
		Changes:      []WorkChange{{Kind: "drop", EntryID: result.Entries[0].ID}},
	})
	if err == nil {
		t.Fatal("cross-scope drop accepted")
	}
	// Stale base conflicts.
	_, err = s.ApplyWorkPatch(context.Background(), PrincipalAgent, bound, WorkPatch{
		BaseRevision: result.Revision - 1,
		Changes:      []WorkChange{{Kind: "add", Field: "goal", Body: "stale"}},
	})
	assertCode(t, err, "actmem_revision_conflict")
}

func TestAppendActivityDedupesEventID(t *testing.T) {
	a := actmem.New(t.TempDir())
	s := NewService(&runtimecore.Garden{ProfileID: "default", Actmem: a})
	entry := Entry{Section: evolution.SectionPulse, Scope: personalScope("default"), SessionID: "s-1", EventID: "evt-1", Body: "once"}
	first, err := s.AppendActivity(context.Background(), PrincipalAgent, binding(), entry)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.AppendActivity(context.Background(), PrincipalAgent, binding(), entry)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed || second.Changed {
		t.Fatalf("dedupe: first=%+v second=%+v", first, second)
	}
	// Read principal cannot append.
	_, err = s.AppendActivity(context.Background(), PrincipalRead, binding(), entry)
	assertCode(t, err, "principal_forbidden")
}

func TestBoundClientFoldSessionDelegates(t *testing.T) {
	a := actmem.New(t.TempDir())
	seedEntry(t, a, evolution.SectionPulse, personalScope("default"), "s-fold", "folded")
	b := &BoundClient{client: &Client{runtime: &runtimecore.Garden{ProfileID: "default", Actmem: a}, principal: PrincipalAgent}, binding: binding()}
	s, unlock, err := b.service()
	if err != nil {
		t.Fatal(err)
	}
	capsules, err := s.FoldSession(context.Background(), b.client.principal, b.binding, "s-fold")
	unlock()
	if err != nil {
		t.Fatal(err)
	}
	if len(capsules) == 0 {
		t.Fatal("no capsule written")
	}
}
