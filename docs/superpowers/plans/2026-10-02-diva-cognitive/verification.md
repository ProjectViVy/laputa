# Planning verification

> **2026-10-11 policy supersession:** The Pulse/Recap/Work, tool-only ACTMEM and fixed-cap/timer contract below is the prior cognitive baseline. Preserve its receipts, scope, Mission/Dream fences and recorded results; replan selected-policy Recap and compatibility through the existing context policy. See [ADR-0018](../../../architecture/0018-vivy-unified-context-and-recap-policy.md). Original design, interface and verification records below remain evidence of their recorded snapshots; this notice does not change their pass/fail results or claim the new policy is implemented.

Scope: document structure, contract consistency and handoff completeness only. No product code was written and no runtime tests were run.

Run from package root: `python3 scripts/check_plan.py`.

The checker validates all nine Story IDs, predecessor endpoints, absence of cycles, topological waves, required plan fields and local Markdown file links. The shared contract records scope types, typed effect variants, operation identity, Markdown grammar, FrozenCore v2 and backend receipt rules; each Story links it instead of independently declaring them.

Manual consistency review: strategy semantics remain in Laputa; Garden is the governed domain adapter; ViVy owns execution and durable runtime state; INOFY is reused. Mission/Dream effects cannot be encoded by the evolution output union. Unknown outcomes do not authorize blind retries. ACTMEM scope metadata remains inside its Markdown authority. S08 remains blocked on the external desktop bridge and cannot claim desktop verification.

Package readiness is distinct from Story execution readiness. S01 is the first planned execution candidate; implementing it requires execution authorization and accepted shared-contract review. Later Stories require actual accepted predecessor evidence. This package does not change AGENTS.md, publish module tags, update issues, push branches or migrate data.
