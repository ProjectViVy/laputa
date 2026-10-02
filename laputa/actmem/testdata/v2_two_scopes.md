---
schema: laputa.actmem/v2
revision: 4
updated: "2026-10-02T09:00:00Z"
entries:
  e_0123456789abcdef0123456789abcdef:
    section: pulse
    field: ""
    scope:
      subject_id: "sub"
      kind: workspace
      workspace_id: "ws-x"
    session_id: "s-1"
    event_id: "ev-1"
    occurred_at: "2026-10-02T08:59:00Z"
    sources: []
  e_22222222222222222222222222222222:
    section: work
    field: open
    scope:
      subject_id: "sub"
      kind: workspace
      workspace_id: "ws-y"
    session_id: "s-2"
    event_id: ""
    occurred_at: "2026-10-02T08:58:00Z"
    sources: []
  e_33333333333333333333333333333333:
    section: work
    field: goal
    scope:
      subject_id: "sub"
      kind: workspace
      workspace_id: "ws-x"
    session_id: "s-2"
    event_id: ""
    occurred_at: "2026-10-02T08:57:00Z"
    sources: []
  e_44444444444444444444444444444444:
    section: work
    field: open
    scope:
      subject_id: "sub"
      kind: personal
      workspace_id: ""
    session_id: "s-3"
    event_id: ""
    occurred_at: "2026-10-02T08:56:00Z"
    sources: []
---

## Pulse
<!-- actmem-entry:e_0123456789abcdef0123456789abcdef -->
ws-x pulse text
<!-- /actmem-entry:e_0123456789abcdef0123456789abcdef -->

## Recap

## Work
<!-- actmem-entry:e_33333333333333333333333333333333 -->
ws-x goal text
<!-- /actmem-entry:e_33333333333333333333333333333333 -->
<!-- actmem-entry:e_22222222222222222222222222222222 -->
ws-y secret text
<!-- /actmem-entry:e_22222222222222222222222222222222 -->
<!-- actmem-entry:e_44444444444444444444444444444444 -->
personal open text
<!-- /actmem-entry:e_44444444444444444444444444444444 -->
