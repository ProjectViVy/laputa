export interface PersonaFileState {
  kind: string;
  file_name: string;
  exists: boolean;
  valid: boolean;
  reason: string | null;
  revision: number;
  updated_at: string | null;
  pending_count: number;
  content_limit: number;
  frozen_limit: number | null;
  required: boolean;
  tool_only: boolean;
}

export interface PersonaStatusView {
  status: "uninitialized" | "ready" | "incomplete";
  documents: PersonaFileState[];
}

export interface PersonaDocument {
  kind: string;
  file_name: string;
  exists: boolean;
  valid: boolean;
  content: string;
  revision: number;
  content_hash: string;
  updated_at: string | null;
  pending_count: number;
}

export interface PersonaHistoryEntry {
  revision: number;
  content_hash: string;
  actor: string;
  source: string;
  reason: string;
  base_revision: number;
  created_at: string;
}

export interface PersonaHistoryRevision {
  revision: number;
  content_hash: string;
  actor: string;
  source: string;
  reason: string;
  base_revision: number;
  created_at: string;
  content: string;
  unified_diff: string;
}
