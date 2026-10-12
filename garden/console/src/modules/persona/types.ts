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
  edit_status?: { held: boolean; generation: number; expires_at?: string; blocked_writes: number };
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
  kind: string;
  entry: PersonaHistoryEntry;
  content: string;
  unified_diff: string;
}

export interface EditLease { token: string; generation: number; kind: string; base_revision: number; base_hash: string; expires_at: string }
