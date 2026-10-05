// Package testkit provides the frozen S01 compatibility fixtures for the
// evolution contract tests and downstream stories: two subjects, scoped
// workspaces, a read-only cross-workspace mount, a legacy record, and the
// three principal classes. Fixtures carry no runtime behavior.
package testkit

import (
	"strings"

	"github.com/ProjectViVy/laputa/laputa/evolution"
)

const (
	SubjectAID   = "subject-a"
	SubjectBID   = "subject-b"
	WorkspaceXID = "ws-x"
	WorkspaceYID = "ws-y"
)

// SubjectAPersonal is subject A's implicit personal scope.
func SubjectAPersonal() evolution.Scope {
	return evolution.Scope{SubjectID: SubjectAID, Kind: evolution.ScopePersonal}
}

// SubjectAWorkspaceX is subject A bound to workspace X.
func SubjectAWorkspaceX() evolution.Scope {
	return evolution.Scope{SubjectID: SubjectAID, Kind: evolution.ScopeWorkspace, WorkspaceID: WorkspaceXID}
}

// SubjectAWorkspaceY is subject A bound to workspace Y.
func SubjectAWorkspaceY() evolution.Scope {
	return evolution.Scope{SubjectID: SubjectAID, Kind: evolution.ScopeWorkspace, WorkspaceID: WorkspaceYID}
}

// SubjectBWorkspaceX is a second subject's scope in workspace X; distinct
// from subject A's even though the workspace id matches.
func SubjectBWorkspaceX() evolution.Scope {
	return evolution.Scope{SubjectID: SubjectBID, Kind: evolution.ScopeWorkspace, WorkspaceID: WorkspaceXID}
}

// Mount is one scoped access grant. ReadOnly mounts admit no write
// operations; the cross-workspace read mount never becomes a write grant.
type Mount struct {
	Scope    evolution.Scope
	ReadOnly bool
}

// SubjectAMounts returns subject A's workspace mounts: X read/write, Y
// read-only cross-workspace.
func SubjectAMounts() []Mount {
	return []Mount{
		{Scope: SubjectAWorkspaceX(), ReadOnly: false},
		{Scope: SubjectAWorkspaceY(), ReadOnly: true},
	}
}

// UnknownScopeLegacyRecordJSON is a persisted pre-v1 scope record whose kind
// is outside the closed vocabulary; strict decoders must reject it rather
// than silently reinterpret it.
const UnknownScopeLegacyRecordJSON = `{"subject_id":"subject-legacy","kind":"vault","workspace_id":"ws-legacy"}`

// Principal is a fixture provenance identity.
type Principal struct {
	Class evolution.PrincipalClass
	ID    string
}

func HumanPrincipal() Principal { return Principal{Class: evolution.PrincipalHuman, ID: "owner"} }
func ConversationPrincipal() Principal {
	return Principal{Class: evolution.PrincipalConversation, ID: "session-1"}
}
func WorkflowPrincipal() Principal {
	return Principal{Class: evolution.PrincipalWorkflow, ID: "laputa.evolution.reflect@1"}
}

// Binding returns a populated trusted run binding for a scope. Callers
// mutate the returned value for negative cases.
func Binding(scope evolution.Scope) evolution.RunBinding {
	return evolution.RunBinding{
		SubjectID:       scope.SubjectID,
		WorkspaceID:     scope.WorkspaceID,
		DestinationID:   "mentle.default",
		PolicyRevision:  "policy-2026-10-02",
		StrategyDigest:  "sha256:" + strings.Repeat("a", 64),
		MissionRevision: 0,
	}
}

const (
	ActmemGoldenPulseID = "e_00000000000000000000000000000001"
	ActmemGoldenPulse2  = "e_00000000000000000000000000000002"
	ActmemGoldenRecapID = "e_00000000000000000000000000000003"
	ActmemGoldenWorkID  = "e_00000000000000000000000000000004"
	ActmemGoldenWork2ID = "e_00000000000000000000000000000005"
	ActmemOrphanID      = "e_00000000000000000000000000000009"
)

// ActmemGoldenV2 is a valid v2 document: two Pulse entries, one Recap, two
// Work entries, all bound to subject A workspace X.
const ActmemGoldenV2 = `---
schema: laputa.actmem/v2
revision: 3
updated: "2026-10-02T08:00:00Z"
entries:
  e_00000000000000000000000000000001:
    section: pulse
    scope: {subject_id: "subject-a", kind: workspace, workspace_id: "ws-x"}
    session_id: "sess-1"
    event_id: "evt-1"
    occurred_at: "2026-10-02T06:00:00Z"
    sources: []
  e_00000000000000000000000000000002:
    section: pulse
    scope: {subject_id: "subject-a", kind: workspace, workspace_id: "ws-x"}
    session_id: "sess-1"
    event_id: "evt-2"
    occurred_at: "2026-10-02T07:00:00Z"
    sources:
      - {source_id: "journal", record_id: "r-1", revision: 4, scope: {subject_id: "subject-a", kind: workspace, workspace_id: "ws-x"}}
  e_00000000000000000000000000000003:
    section: recap
    scope: {subject_id: "subject-a", kind: workspace, workspace_id: "ws-x"}
    session_id: "sess-1"
    event_id: "evt-3"
    occurred_at: "2026-10-02T07:30:00Z"
    sources: []
  e_00000000000000000000000000000004:
    section: work
    field: goal
    scope: {subject_id: "subject-a", kind: workspace, workspace_id: "ws-x"}
    session_id: "sess-1"
    event_id: "evt-4"
    occurred_at: "2026-10-02T07:45:00Z"
    sources: []
  e_00000000000000000000000000000005:
    section: work
    field: next
    scope: {subject_id: "subject-a", kind: workspace, workspace_id: "ws-x"}
    session_id: "sess-1"
    event_id: "evt-5"
    occurred_at: "2026-10-02T07:50:00Z"
    sources: []
---

## Pulse

<!-- actmem-entry:e_00000000000000000000000000000001 -->
- 2026-10-02T06:00:00Z user asked for a **quiet shift** <!-- session-hex:sess1 -->
<!-- /actmem-entry:e_00000000000000000000000000000001 -->

<!-- actmem-entry:e_00000000000000000000000000000002 -->
- second pulse line with *inline code* and a
  wrapped second line of body text
<!-- /actmem-entry:e_00000000000000000000000000000002 -->

## Recap

<!-- actmem-entry:e_00000000000000000000000000000003 -->
Recap of the morning session across two paragraphs of
Markdown text, kept under the 200-char cap.

<!-- /actmem-entry:e_00000000000000000000000000000003 -->

## Work

<!-- actmem-entry:e_00000000000000000000000000000004 -->
Ship the contract freeze with strict decoders.
<!-- /actmem-entry:e_00000000000000000000000000000004 -->

<!-- actmem-entry:e_00000000000000000000000000000005 -->
1. Land laputa/evolution package
2. Wire the backend contract
<!-- /actmem-entry:e_00000000000000000000000000000005 -->
`

// ActmemOrphanHeaderV2 carries a header record with no body entry.
const ActmemOrphanHeaderV2 = `---
schema: laputa.actmem/v2
revision: 1
updated: "2026-10-02T08:00:00Z"
entries:
  e_00000000000000000000000000000009:
    section: pulse
    scope: {subject_id: "subject-a", kind: personal, workspace_id: ""}
    session_id: "sess-9"
    event_id: "evt-9"
    occurred_at: "2026-10-02T07:00:00Z"
    sources: []
---

## Pulse

## Recap

## Work
`

// ActmemDuplicateHeaderKeyV2 repeats a mapping key in the front matter.
const ActmemDuplicateHeaderKeyV2 = `---
schema: laputa.actmem/v2
revision: 1
updated: "2026-10-02T08:00:00Z"
revision: 2
entries: {}
---

## Pulse

## Recap

## Work
`

// ActmemUnknownHeaderKeyV2 adds a key outside the closed vocabulary.
const ActmemUnknownHeaderKeyV2 = `---
schema: laputa.actmem/v2
revision: 1
updated: "2026-10-02T08:00:00Z"
entries: {}
owner: "root"
---

## Pulse

## Recap

## Work
`
