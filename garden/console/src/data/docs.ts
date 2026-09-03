export type Bi = { en: string; zh: string };
export type DocStatus = "accepted" | "implemented" | "proposed" | "superseded" | "archived";
export type DocRole = "L0" | "L1" | "L2" | "L3" | "L4" | "L5";

export interface DocEntry {
  id: string;
  role: DocRole;
  title: Bi;
  path: string;
  status: DocStatus;
  modules: string[];
  supersededBy?: string;
  note?: Bi;
}

export const ROLE_LABELS: Record<DocRole, Bi> = {
  L0: { en: "Product vision", zh: "产品愿景" },
  L1: { en: "Accepted decisions", zh: "已接受决策" },
  L2: { en: "Target architecture", zh: "目标架构" },
  L3: { en: "Runtime contracts", zh: "运行时契约" },
  L4: { en: "Implementation evidence", zh: "实现证据" },
  L5: { en: "Historical evidence", zh: "历史证据" },
};

export const DOCS: DocEntry[] = [
  { id: "readme", role: "L0", title: { en: "MemoryOS purpose and boundaries", zh: "MemoryOS 目的与边界" }, path: "README.md / AGENTS.md", status: "implemented", modules: ["Laputa", "Mentle", "Garden", "EvoMap"], note: { en: "Module ownership and the local-first operating model.", zh: "模块所有权与本地优先运行模型。" } },
  { id: "0012", role: "L1", title: { en: "ADR-0012 — Laputa Markdown clean break", zh: "ADR-0012 — Laputa Markdown clean break" }, path: "docs/architecture/0012-laputa-markdown-clean-break.md", status: "accepted", modules: ["Laputa", "Garden"], note: { en: "Seven Markdown authority files, explicit activity memory, and bounded context.", zh: "七份 Markdown 权威文件、显式活动记忆与有界上下文。" } },
  { id: "0013", role: "L2", title: { en: "ADR-0013 — implementation architecture", zh: "ADR-0013 — 实施架构" }, path: "docs/architecture/0013-laputa-clean-break-implementation-architecture.md", status: "accepted", modules: ["Laputa", "Garden", "EvoMap"], note: { en: "Replacement order, adapters, and deletion proof matrix.", zh: "替换顺序、适配器与删除证明矩阵。" } },
  { id: "0014", role: "L2", title: { en: "ADR-0014 — canonical authority and index recovery", zh: "ADR-0014 — 权威与索引恢复" }, path: "docs/architecture/0014-mentle-canonical-authority-and-derived-index-recovery.md", status: "accepted", modules: ["Mentle", "Garden"], note: { en: "Canonical SQLite is authoritative; derived indexes recover from transactional jobs.", zh: "Canonical SQLite 是唯一权威；派生索引通过事务任务恢复。" } },
  { id: "api-contract", role: "L3", title: { en: "Garden HTTP API contracts", zh: "Garden HTTP API 契约" }, path: "docs/bmad/garden-authority-recovery-2026-09/api-contract.md", status: "implemented", modules: ["Garden", "Mentle", "Laputa"] },
  { id: "implementation", role: "L4", title: { en: "Recovery implementation records", zh: "恢复实施记录" }, path: "docs/bmad/garden-authority-recovery-2026-09/implementation/", status: "implemented", modules: ["Garden", "Mentle", "Laputa", "EvoMap"] },
  { id: "archive", role: "L5", title: { en: "Superseded architecture archive", zh: "已取代的架构归档" }, path: "docs/archive/2026-08-14-laputa-clean-break/", status: "archived", modules: ["Laputa", "Garden", "Mentle"], note: { en: "Read-only historical evidence; it is not an implementation source.", zh: "只读历史证据，不是实施来源。" } },
];
