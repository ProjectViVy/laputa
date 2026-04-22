# Laputa Development Documentation

> Central repository for all Laputa design, research, and BMAD output documents.
> Total: 105 documents

## Directory Structure

```
docs/dev/
├── bmad-output/          # BMAD methodology outputs
│   ├── planning-artifacts/   # PRD, Architecture, Epics (6 docs)
│   └── implementation-artifacts/  # Stories, Patches, Deferred work (39 docs)
├── design/               # Core design documents (1 doc)
├── research/             # Research and analysis documents (57 docs)
│   ├── architecture-analysis/    # Laputa-DIVA architecture analysis
│   ├── architecture-reports/     # Agent-diva architecture reports (NEW)
│   ├── agent-diva-dev-docs/      # Agent-diva development docs (NEW)
│   ├── agent-diva-integration/   # Memory evolution research
│   ├── migration-guide/          # Nano migration guides
│   ├── nano-architecture/        # Nano architecture docs (NEW)
│   ├── nanobot-architecture/     # Nanobot sync research
│   ├── mempalace-baseline/       # mempalace-rs baseline docs (NEW)
│   ├── qa-checklists/            # QA checklists (NEW)
│   ├── roadmaps/                 # Provider roadmaps (NEW)
│   ├── standalone-bundle/        # Standalone bundle research (NEW)
│   └── upsp-study/               # UPSP conceptual analysis
```

## BMAD Output

### Planning Artifacts (`bmad-output/planning-artifacts/`)

| File | Description |
|------|-------------|
| `prd.md` | Initial Product Requirements Document |
| `prd-laputa-agent-diva-integration.md` | Laputa-agent-diva integration PRD |
| `architecture.md` | Technical architecture specification |
| `epics.md` | Epic breakdown and story mapping |
| `implementation-readiness-report-2026-04-13.md` | Pre-implementation validation |
| `sprint-change-proposal-2026-04-20.md` | Sprint adjustment proposal |

### Implementation Artifacts (`bmad-output/implementation-artifacts/`)

Story-level implementation specs organized by Epic:

**Epic 1 - Project Foundation**
- `1-1-project-structure-setup.md`
- `1-2-identity-initialization.md`
- `1-3-memoryrecord-extension.md`
- `1-2-1-code-review-fixes.md`
- `1-3-patch-heat-validation.md`

**Epic 2 - Diary & Memory Processing**
- `2-1-diary-write.md`
- `2-2-memory-filter-merge.md`
- `2-3-emotion-anchor.md`

**Epic 3 - Recall & Search**
- `3-1-timeline-recall.md`
- `3-2-semantic-search.md`
- `3-3-wakepack-generate.md`
- `3-4-hybrid-search.md`

**Epic 4 - Rhythm & Capsules**
- `4-1-weekly-capsule.md`
- `4-2-rhythm-scheduler.md`

**Epic 5 - Heat & Lifecycle**
- `5-1-heat-service.md`
- `5-2-mixed-trigger.md`
- `5-3-user-intervention.md`
- `5-4-archive-candidate.md`

**Epic 6 - CLI & MCP**
- `6-1-cli-commands.md`
- `6-2-mcp-tools.md`

**Epic 7 - Knowledge Graph**
- `7-1-relation-node.md`
- `7-2-emotion-dimension.md`

**Epic 8 - Archive & Export**
- `8-1-archive-export.md`
- `8-2-data-export.md`

**Epic 9 - Standalone Build**
- `9-1-standalone-build-decoupling.md`
- `9-2-repo-metadata-doc-independence.md`
- `9-3-clean-server-migration-validation.md`
- `9-3-migration-validation-report.md`

**Epic 10 - Integration & Examples**
- `10-1-e2e-acceptance-script.md`
- `10-2-cli-mcp-db-path-unification.md`
- `10-3-laputa-nano-integration-design.md`
- `10-4-laputa-tui-example-setup.md`

**Patch-Level Work**
- `patch-1a-heat-performance.md`
- `patch-1b-heat-validation.md`
- `patch-1c-test-supplement.md`
- `patch-2-cli-mcp-critical.md`
- `epic-1-patch-security-validation.md`

**Deferred Work**
- `deferred-work.md` - Consolidated deferred items

## Design Documents (`design/`)

| File | Description |
|------|-------------|
| `brain-memory-system-design.md` | Original brain-memory system design |

## Research Documents (`research/`)

### Architecture Analysis (`architecture-analysis/`)
- `00-architecture-analysis-summary.md` - Laputa-DIVA integration architecture analysis

### Architecture Reports (`architecture-reports/`) NEW
Agent-diva architecture analysis reports:
- `openclaw-session-reset-analysis.md` - Session reset mechanism analysis
- `soul-mechanism-analysis.md` - Soul mechanism deep dive
- `zeroclaw-style-memory-architecture-for-agent-diva.md` - Memory architecture design
- `上下文管理调研记录.md` - Context management research notes

### Agent-DIVA Dev Docs (`agent-diva-dev-docs/`) NEW
Development documentation from agent-diva:
- `architecture.md` - Agent-diva architecture overview
- `bug-fixing-lessons-learned.md` - Bug fixing experiences
- `development.md` - Development guide
- `migration.md` - Migration documentation
- `nano-runtime-packaging-plan.md` - Nano runtime packaging

### Agent-DIVA Integration (`agent-diva-integration/`)
Memory evolution and integration research:
- `2026-03-26-agent-diva-integrated-memory-design.md`
- `2026-03-26-agent-diva-memory-capability-parity-plan.md`
- `2026-03-26-agent-diva-memory-implementation-plan.md`
- `2026-03-26-agent-diva-memory-phase-a-spec.md`
- `2026-03-26-agent-diva-rag-implementation-based-on-zeroclaw.md`
- `2026-03-26-agent-diva-rag-research.md`

### Migration Guide (`migration-guide/`)
- `agent-diva-nano-migration-guide.md` - Nano migration reference

### Nano Architecture (`nano-architecture/`) NEW
Agent-diva-nano architecture documentation:
- `agent-diva-nano-architecture.md` - Nano architecture overview
- `agent-diva-nano-implementation-plan.md` - Implementation plan
- `agent-diva-nano-master-spec.md` - Master specification
- `minimal-gui-agent-diva-implementation-plan.md` - Minimal GUI plan
- `nano-decoupling-preparation-plan.md` - Decoupling preparation
- `crates-io-publish-strategy.md` - Crates.io publishing strategy

### Nanobot Architecture (`nanobot-architecture/`)
- `2026-03-26-clawhub-registry-integration-plan.md`
- `2026-03-26-dev-research-summary.md`
- `2026-03-26-nanobot-gap-analysis.md`
- `2026-03-26-onboarding-wizard-p2-assessment.md`
- `2026-03-26-plugin-architecture-reassessment.md`
- `2026-03-26-provider-login-delivery-plan.md`
- `2026-03-26-provider-parity-map-from-zeroclaw.md`
- `2026-03-26-provider-phase1-implementation-checklist.md`

### Mempalace Baseline (`mempalace-baseline/`) NEW
Historical baseline from mempalace-rs project:
- `TASKS_V0.4.md` - V0.4 task list
- `benchmarking_plan_2026.md` - Benchmarking plan
- `parity_report.md` - Parity report
- `aaak/evolution_plan.md` - AAAK evolution plan
- `adrs/aaak-evolution-hardening.md` - AAAK hardening ADR

### QA Checklists (`qa-checklists/`) NEW
- `blackbox-test-checklist.md` - Blackbox testing checklist

### Roadmaps (`roadmaps/`) NEW
Provider and roadmap documents:
- `provider-catalog-refactor-plan.md` - Provider catalog refactor
- `provider-selection-followups.md` - Provider selection follow-ups
- `soul-persona-gap-implementation-checklist.md` - Soul-persona gap checklist

### Standalone Bundle (`standalone-bundle/`) NEW
Standalone application bundle research:
- `standalone-bundle-research.md` - Standalone bundle analysis
- `windows-standalone-app-solution.md` - Windows standalone solution

### UPSP Study (`upsp-study/`)
UPSP (主体协议) conceptual research for Laputa resonance concepts:
- `UPSP.md` - Original UPSP specification
- `UPSP架构分析.md` - Architecture analysis
- `UPSP与DIVA-Soul兼容性分析.md` - Compatibility analysis
- `UPSP开发路线.md` - Development roadmap
- `UPSP-Rust-Crate架构方案.md` - Rust crate architecture
- `upsp-rs-architecture-design.md` - Architecture design
- `phase2-memory-integration-plan.md` - Memory integration plan
- `executive-summary.md` - Executive summary
- `SUMMARY.md` - Overall summary
- `README.md` - UPSP directory guide
- `2026-04-05 UPSP现状对其研究.md` - Status research

## Related Project Documents

For project-level documents, see:
- [../AGENTS.md](../../AGENTS.md) - Project guide for AI agents
- [../README.md](../../README.md) - Project overview
- [../DECISIONS.md](../../DECISIONS.md) - Core decision log
- [../STATUS.md](../../STATUS.md) - Repository status

## Document Categories Summary

| Category | Documents |
|----------|-----------|
| BMAD Planning | 6 |
| BMAD Implementation | 39 |
| Design | 1 |
| Research | 57 |
| **Total** | **105** |

## Document Status

All documents in this directory are historical research and design outputs.
They are preserved for reference and do not represent current implementation state.

For current implementation status, refer to:
- Repository source code under `src/`
- Test files under `tests/`
- STATUS.md for tracking updates