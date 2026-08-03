export type Source = "live" | "compat" | "accepted-design";
export type ComponentStatus = "ok" | "degraded" | "offline" | string;

export interface IngestionStats {
  accepted: number;
  running: number;
  spooled: number;
  completed: number;
  failed: number;
  total: number;
}

export interface OverviewResponse {
  status: string;
  components: Record<string, ComponentStatus>;
  ingestion?: IngestionStats;
  spool_pending?: number;
  source: Source;
}

export interface ComponentEntry {
  name: string;
  status: ComponentStatus;
  source: Source;
}

export interface ComponentsResponse {
  components: ComponentEntry[];
  api_contract: string;
}

export interface TraceStep {
  step: string;
  status: string;
  duration_ms: number;
  error_code?: string;
}

export interface BudgetConsumption {
  budget_chars: number;
  used_chars: number;
  kg_queries: number;
  timeline_queries: number;
  card_searches: number;
}

export interface RecallTrace {
  trace_id: string;
  query: string;
  scope: string;
  trigger_reason: string;
  source_set: string[];
  filter_conditions: string[];
  candidate_ids: string[];
  evidence_refs: string[];
  budget: BudgetConsumption;
  degraded: boolean;
  failure_state?: string;
  steps: TraceStep[];
  started_at: string;
  duration_ms: number;
}

export interface ContextManifestResponse {
  trace: RecallTrace;
  source: Source;
}

export interface SpoolEntry {
  event_id: string;
  session_id: string;
  scope: string;
  kind: string;
  created_at: string;
}

export interface SpoolResponse {
  pending_count: number;
  entries: SpoolEntry[];
  source: Source;
}

export interface AuditEntry {
  sequence: number;
  section: string;
  action: string;
  actor: string;
  reason: string;
  request_id: string;
  rollback_ref: string;
  timestamp: string;
}

export interface AuditResponse {
  entries: AuditEntry[];
  count: number;
  source: Source;
}

export interface HealthResponse {
  status: string;
  components: Record<string, ComponentStatus>;
  api_contract: string;
}

export interface PipelineDefinition {
  name: string;
  version: string;
  enabled?: boolean;
  capabilities: string[];
  max_steps: number;
  max_visits_per_step: number;
}

export interface PipelinesResponse {
  revision: string;
  pipelines: PipelineDefinition[];
}

export interface RunTrace {
  trace_id: string;
  pipeline: string;
  revision: string;
  started_at: string;
  duration_ms: number;
  status: string;
  warnings?: string[];
}

export interface PipelineRunsResponse {
  runs: RunTrace[];
}

// ============ Materials & Evidence ============

export interface MemoryCard {
  id: string;
  kind: string;
  collection: string;
  scope: string;
  title: string;
  summary: string;
  source_ref: string;
  revision: number;
  status: string;
  valid_from: string;
  valid_to?: string;
  superseded_by?: string;
  tags: string[];
  heat_score: number;
  last_activated?: string;
  candidate_score: number;
}

export interface CardsResponse {
  cards: MemoryCard[];
  next_cursor?: string;
  source: Source;
}

export interface EvidenceFragment {
  card_id: string;
  material_ref: string;
  source_uri?: string;
  source_rev?: string;
  excerpt: string;
  start_offset: number;
  end_offset: number;
  content_hash: string;
  validity: string;
  evidence_refs?: string[];
}

export interface EvidenceResponse {
  card_id: string;
  fragments: EvidenceFragment[];
  source: Source;
}

export interface CollectionInfo {
  name: string;
  count: number;
}

export interface CollectionsResponse {
  collections: CollectionInfo[];
  source: Source;
}

// ============ Reports ============

export interface ReportItem {
  cadence: string;
  window_start: string;
  window_end: string;
  source_ids: string[];
  source_hash: string;
  title: string;
  summary: string;
  highlights: string[];
  open_questions: string[];
  generated_at: string;
  scope: string;
  goals: string[];
  completed: string[];
  decisions: string[];
  open_loops: string[];
  source_refs: string[];
  revision: number;
  generator: string;
}

export interface ReportsListResponse {
  cadence: string;
  items: ReportItem[];
  count: number;
}

// ============ Human modules (AMBITION / USER SUGGESTIONS) ============

export type ModuleKind = "ambition" | "suggestion";
export type ModuleStatus = "active" | "dismissed";

export interface HumanModule {
  id: string;
  kind: ModuleKind;
  content: string;
  status: ModuleStatus;
  created_at: string;
  updated_at: string;
}

export interface ModulesListResponse {
  kind: ModuleKind;
  status: string;
  items: HumanModule[];
  count: number;
}

// ============ Evolution (EvoMap hub transport, ADR-0010) ============

export interface ProviderStatus {
  node_id: string;
  claimed: boolean;
  claim_url?: string;
  credit_balance?: number;
  has_credit_balance: boolean;
  survival_status: string;
  last_heartbeat: string;
}

export interface EvolutionRun {
  run_id: string;
  status: string;
  bundle_id: string;
  provider: string;
  candidates?: string[];
  error?: string;
  started_at: string;
  completed_at?: string;
}

export interface RunsListResponse {
  items: EvolutionRun[];
  count: number;
}

export interface GeneCandidate {
  candidate_id: string;
  run_id: string;
  kind: string;
  name: string;
  description: string;
  payload: Record<string, unknown>;
  evidence_refs: string[];
  trace_ref: string;
  confidence: number;
  created_at: string;
}

export interface LeakageReport {
  clean: boolean;
  violations?: string[];
  warnings?: string[];
  checked_refs: number;
}

export interface EvolutionProposal {
  proposal_id: string;
  run_id: string;
  candidate_id: string;
  kind: string;
  status: string;
  summary: string;
  leakage_report: LeakageReport;
  reviewer?: string;
  review_note?: string;
  reviewed_at?: string;
  created_at: string;
}

export interface ProposalsListResponse {
  items: EvolutionProposal[];
  count: number;
}
