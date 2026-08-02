export type Bi = { en: string; zh: string };
export type Phase = "existing" | "compat" | "accepted-design" | "deferred";

export interface GovNode {
  id: string;
  name: string;
  tier: number;
  col: number;
  component?: string;
  phase: Phase;
  compat?: boolean;
  responsibility: Bi;
  boundary: Bi;
  inputs: Bi;
  outputs: Bi;
  apis: string[];
  rules: Bi;
  limitations: Bi;
  governanceNote: Bi;
  roadmapNote: Bi;
  docs: string[];
}

export const GOV_NODES: GovNode[] = [
  {
    id: "hosts",
    name: "Hosts",
    tier: 0,
    col: 1,
    phase: "deferred",
    responsibility: {
      en: "Integration surfaces (Hermes, Claude Code, Codex, OpenClaw) that consume ContextView and emit activity events.",
      zh: "集成端(Hermes、Claude Code、Codex、OpenClaw),消费 ContextView 并发出活动事件。",
    },
    boundary: {
      en: "Receives only approved PortableSkill data and bounded context; never authority.",
      zh: "仅接收已批准的 PortableSkill 数据与有界上下文;绝非权威。",
    },
    inputs: { en: "ContextView, governance projection", zh: "ContextView、治理投影" },
    outputs: { en: "Activity events, session submissions", zh: "活动事件、会话提交" },
    apis: ["POST /v2/recall/fast", "POST /v2/activity/events", "POST /v1/sessions"],
    rules: {
      en: "Host adapters receive only approved PortableSkill data (architecture §13).",
      zh: "Host 适配器仅接收已批准的 PortableSkill 数据(架构 §13)。",
    },
    limitations: {
      en: "No host adapter implemented yet (Wave 7, downstream of Gates A–C).",
      zh: "尚无 host 适配器实现(Wave 7,位于 Gate A–C 之后)。",
    },
    governanceNote: { en: "read-only projection", zh: "只读投影" },
    roadmapNote: { en: "deferred — Wave 7", zh: "推迟 — Wave 7" },
    docs: ["0001"],
  },
  {
    id: "garden",
    name: "Garden",
    tier: 1,
    col: 1,
    component: "garden",
    phase: "existing",
    responsibility: {
      en: "HTTP gateway and orchestration: source ingestion, recall (Fast/Deep), ContextView assembly, admin aggregation.",
      zh: "HTTP 网关与编排:源摄取、召回(Fast/Deep)、ContextView 组装、管理聚合。",
    },
    boundary: {
      en: "May orchestrate; may not become authority. Aggregates the single /v2/admin/* console surface.",
      zh: "可编排;不可成为权威。聚合唯一的 /v2/admin/* 控制台台面。",
    },
    inputs: { en: "Host events, recall intents", zh: "Host 事件、召回意图" },
    outputs: { en: "ContextView, recall traces, admin overview", zh: "ContextView、召回 trace、管理概览" },
    apis: ["POST /v2/recall/fast", "POST /v2/recall/deep", "GET /v2/admin/overview", "GET /v1/pipelines"],
    rules: {
      en: "No silent high-impact mutation; Fast Recall never calls Planner/KG/graph by default.",
      zh: "无静默高影响变更;Fast Recall 默认绝不调用 Planner/KG/图谱。",
    },
    limitations: {
      en: "Deep Recall requires explicit capability declaration; evolution provider not wired.",
      zh: "Deep Recall 需显式能力声明;evolution provider 未接入。",
    },
    governanceNote: { en: "orchestrator, not authority", zh: "编排者,非权威" },
    roadmapNote: { en: "implemented — Waves 1–6", zh: "已实现 — Waves 1–6" },
    docs: ["0001", "0003"],
  },
  {
    id: "laputa",
    name: "Laputa",
    tier: 2,
    col: 0,
    component: "laputa",
    phase: "existing",
    responsibility: {
      en: "Cognitive governance: identity, authority, lifecycle, policy, audit. Holds the Frozen Core and STM checkpoint.",
      zh: "认知治理:身份、权威、生命周期、策略、审计。持有 Frozen Core 与 STM 检查点。",
    },
    boundary: {
      en: "May approve and apply authority. MEMRULES human-only; WORLD user-editable, never default-injected.",
      zh: "可批准并应用权威。MEMRULES 仅人类;WORLD 用户可编辑,绝不默认注入。",
    },
    inputs: { en: "Governed mutation requests", zh: "受治理的变更请求" },
    outputs: { en: "Governance projection, audit entries", zh: "治理投影、审计条目" },
    apis: ["POST /v2/governance/projection", "POST /v2/governance/mutations", "GET /v2/governance/audit"],
    rules: {
      en: "Actor-aware authorization; append-only audit log; legacy 06/13/14 are compatibility-only.",
      zh: "按执行者授权;仅追加审计日志;遗留 06/13/14 仅为兼容。",
    },
    limitations: {
      en: "MEMRULES.MD / WORLD.MD runtime not implemented (Gate A pending).",
      zh: "MEMRULES.MD / WORLD.MD 运行时未实现(Gate A 待定)。",
    },
    governanceNote: { en: "authority layer", zh: "权威层" },
    roadmapNote: { en: "implemented; MEMRULES/WORLD accepted-design", zh: "已实现;MEMRULES/WORLD 为已接受设计" },
    docs: ["0001", "0002"],
  },
  {
    id: "mentle",
    name: "Mentle",
    tier: 2,
    col: 1,
    component: "mentle",
    phase: "existing",
    responsibility: {
      en: "Material universe: canonical material, evidence lake, retrieval, taxonomy, knowledge graph.",
      zh: "材料宇宙:规范材料、证据湖、检索、分类法、知识图谱。",
    },
    boundary: {
      en: "May store and retrieve; may not promote authority. Cards expose no full content; evidence is budgeted.",
      zh: "可存储与检索;不可提升权威。卡片不暴露全文;证据有预算限制。",
    },
    inputs: { en: "Source artifacts, ingest writes", zh: "源工件、摄取写入" },
    outputs: { en: "Cards, bounded evidence fragments, graph facts", zh: "卡片、有界证据片段、图谱事实" },
    apis: ["SearchCards", "ReadEvidence", "KG / timeline (deep only)"],
    rules: {
      en: "Two-layer model: card discovery → bounded evidence read. Superseded/deleted cards never reappear.",
      zh: "两层模型:卡片发现 → 有界证据读取。被取代/删除的卡片绝不复现。",
    },
    limitations: {
      en: "No HTTP admin adapter yet (/v2/materials/* is a design gap); embedder may be offline.",
      zh: "尚无 HTTP 管理适配(/v2/materials/* 为设计缺口);embedder 可能离线。",
    },
    governanceNote: { en: "material, not cognition", zh: "材料,非认知" },
    roadmapNote: { en: "implemented; /v2/materials/* accepted-design", zh: "已实现;/v2/materials/* 为已接受设计" },
    docs: ["0001"],
  },
  {
    id: "legacy",
    name: "Legacy 14-section registry",
    tier: 2,
    col: 2,
    phase: "compat",
    compat: true,
    responsibility: {
      en: "Pre-vNext 14-section Laputa registry. Retained as read-only compatibility evidence during migration.",
      zh: "vNext 之前的 14 区段 Laputa 注册表。迁移期间保留为只读兼容证据。",
    },
    boundary: {
      en: "No new target-architecture features. 06-history, 13-report_indexes, 14-aaak_summaries are frozen.",
      zh: "不新增目标架构特性。06-history、13-report_indexes、14-aaak_summaries 已冻结。",
    },
    inputs: { en: "— (frozen)", zh: "—(已冻结)" },
    outputs: { en: "Read-only legacy reads", zh: "只读遗留读取" },
    apis: ["GET /v1/memories/{key} (compat)"],
    rules: {
      en: "ADR-0002 deletes 13/14 as target concepts; 06 excluded from projection.",
      zh: "ADR-0002 将 13/14 从目标概念中删除;06 从投影中排除。",
    },
    limitations: {
      en: "Compatibility-only; must not be presented as target architecture.",
      zh: "仅兼容;不得呈现为目标架构。",
    },
    governanceNote: { en: "compatibility-only", zh: "仅兼容" },
    roadmapNote: { en: "frozen — migration pending", zh: "冻结 — 迁移待定" },
    docs: ["0002"],
  },
  {
    id: "contextview",
    name: "ContextView / trace",
    tier: 3,
    col: 1,
    phase: "existing",
    responsibility: {
      en: "Disposable final context assembled for one request, with a manifest and (for deep) a persisted recall trace.",
      zh: "为单次请求组装的一次性最终上下文,带清单,以及(deep 时)持久化的召回 trace。",
    },
    boundary: {
      en: "Not written to authority and not auto-promoted to memory.",
      zh: "不写入权威,也不自动提升为记忆。",
    },
    inputs: { en: "Governance, cards, evidence", zh: "治理、卡片、证据" },
    outputs: { en: "Bounded context string + manifest", zh: "有界上下文字符串 + 清单" },
    apis: ["GET /v2/recall/traces/{id}", "GET /v2/admin/context-manifest/{id}"],
    rules: {
      en: "Deep Recall always emits a RecallTrace; rejected paths are as visible as selected.",
      zh: "Deep Recall 总是产出 RecallTrace;被拒绝路径与选中路径同等可见。",
    },
    limitations: { en: "Fast Recall trace is compact, not full.", zh: "Fast Recall trace 为精简版,非完整。" },
    governanceNote: { en: "disposable, explainable", zh: "一次性、可解释" },
    roadmapNote: { en: "implemented", zh: "已实现" },
    docs: ["0001", "0003"],
  },
];

export const FLOW_DESCRIPTIONS: Record<string, Bi> = {
  context: {
    en: "Host → Garden → STM / scoped WORLD / Mentle evidence → ContextView.",
    zh: "Host → Garden → STM / 有界 WORLD / Mentle 证据 → ContextView。",
  },
  ingestion: {
    en: "Event → Mentle SourceArtifact → ACK → STM; Mentle unavailable → transient spool → receipt-bound drain.",
    zh: "事件 → Mentle SourceArtifact → ACK → STM;Mentle 不可用 → 瞬态缓冲 → 凭据约束的排空。",
  },
  recall: {
    en: "Intent → Fast Recall → Cards → Evidence Read → ContextView → trace.",
    zh: "意图 → Fast Recall → 卡片 → 证据读取 → ContextView → trace。",
  },
  change: {
    en: "Human edit → policy validation → protected write → audit log (not an ordinary agent auto-write chain).",
    zh: "人类编辑 → 策略校验 → 受保护写入 → 审计日志(非常规 agent 自动写入链)。",
  },
};
