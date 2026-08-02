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
