import type { Bi } from "./governance";

export type DocStatus = "accepted" | "implemented" | "proposed" | "superseded" | "archived";
export type DocRole = "L0" | "L1" | "L2" | "L3" | "L4" | "L5";

export interface DocEntry {
  id: string;
  role: DocRole;
  title: Bi;
  path: string;
  status: DocStatus;
  modules: string[];
  supersedes?: string;
  supersededBy?: string;
  note?: Bi;
}

export const ROLE_LABELS: Record<DocRole, Bi> = {
  L0: { en: "Product Vision", zh: "产品愿景" },
  L1: { en: "Accepted Decisions", zh: "已接受决策" },
  L2: { en: "Target Architecture", zh: "目标架构" },
  L3: { en: "Runtime Contracts", zh: "运行时契约" },
  L4: { en: "Implementation Evidence", zh: "实现证据" },
  L5: { en: "Historical Evidence", zh: "历史证据" },
};

export const DOCS: DocEntry[] = [
  {
    id: "readme",
    role: "L0",
    title: { en: "MemoryOS — purpose & boundaries", zh: "MemoryOS — 目的与边界" },
    path: "README.md / AGENTS.md",
    status: "implemented",
    modules: ["Laputa", "Garden", "Mentle"],
    note: {
      en: "Defines the governed-memory vision and the three-module ownership rule.",
      zh: "定义受治理记忆的愿景与三模块所有权规则。",
    },
  },
  {
    id: "0002",
    role: "L1",
    title: { en: "ADR-0002 — Laputa Cognitive Partition", zh: "ADR-0002 — Laputa 认知分区" },
    path: "docs/architecture/0002-laputa-cognitive-partition-decision.md",
    status: "accepted",
    modules: ["Laputa", "Garden"],
    note: {
      en: "Deletes LTM/LONGMEM; defines MEMRULES & WORLD; reclassifies 07–11 as the report system.",
      zh: "删除 LTM/LONGMEM;定义 MEMRULES 与 WORLD;将 07–11 重分类为报告系统。",
    },
  },
  {
    id: "0003",
    role: "L1",
    title: { en: "ADR-0003 — Operations Console Design", zh: "ADR-0003 — 运营台设计" },
    path: "docs/architecture/0003-operations-console-design.md",
    status: "accepted",
    modules: ["Garden"],
    note: {
      en: "This console: workbench-first admin UI, /v2/admin/* aggregation, i18n, MVP phasing.",
      zh: "本控制台:工作台优先的管理 UI、/v2/admin/* 聚合、i18n、MVP 分期。",
    },
  },
  {
    id: "0001",
    role: "L2",
    title: { en: "0001 — MemoryOS vNext Architecture", zh: "0001 — MemoryOS vNext 架构" },
    path: "docs/architecture/0001-memoryos-vnext-architecture.md",
    status: "implemented",
    modules: ["Laputa", "Garden", "Mentle", "Hosts"],
    note: {
      en: "Master plan: modules, interfaces, flows, migration waves 0–7, security, performance targets.",
      zh: "总体规划:模块、接口、流程、迁移 Wave 0–7、安全、性能目标。",
    },
  },
  {
    id: "http-contracts",
    role: "L3",
    title: { en: "Garden HTTP API contracts (v1 + v2)", zh: "Garden HTTP API 契约(v1 + v2)" },
    path: "AGENTS.md §HTTP API Contracts · internal/server",
    status: "implemented",
    modules: ["Garden"],
    note: {
      en: "v2 recall/activity/governance/evolution/admin routes; v1 legacy translator.",
      zh: "v2 recall/activity/governance/evolution/admin 路由;v1 遗留翻译器。",
    },
  },
  {
    id: "test-suite",
    role: "L4",
    title: { en: "Garden internal + E2E test suite", zh: "Garden internal + E2E 测试套件" },
    path: "garden/internal/** · garden/e2e",
    status: "implemented",
    modules: ["Garden", "Mentle", "Laputa"],
    note: {
      en: "Behavioral proofs: capability gate, budget enforcement, degradation, no unauthorized mutation.",
      zh: "行为证明:能力门、预算强制、降级、无未授权变更。",
    },
  },
  {
    id: "archive-2026-08-01",
    role: "L5",
    title: { en: "Pre-MemoryOS redesign archive", zh: "MemoryOS 重设计前归档" },
    path: "docs/archive/2026-08-01-pre-memoryos-redesign/",
    status: "archived",
    modules: ["Laputa", "Garden", "Mentle"],
    supersededBy: "0001",
    note: {
      en: "Superseded design history. Read-only; explicitly not the current contract.",
      zh: "已被取代的设计历史。只读;明确不是当前契约。",
    },
  },
];
