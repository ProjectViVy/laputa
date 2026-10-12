import { get } from "../../api/client";
import type { PersonaFileState, PersonaStatusView, PersonaDocument, PersonaHistoryEntry, PersonaHistoryRevision } from "./types";

export function fetchPersonaStatus(): Promise<PersonaStatusView> {
  return get<PersonaStatusView>("/v2/persona/documents");
}

export function fetchPersonaDocument(kind: string): Promise<PersonaDocument> {
  return get<PersonaDocument>(`/v2/persona/documents/${encodeURIComponent(kind)}`);
}

export function fetchPersonaHistory(kind: string): Promise<{ kind: string; entries: PersonaHistoryEntry[] }> {
  return get<{ kind: string; items: PersonaHistoryEntry[] }>(`/v2/persona/history/${encodeURIComponent(kind)}`).then((value) => ({ kind: value.kind, entries: value.items }));
}

export function fetchPersonaHistoryRevision(kind: string, revision: number): Promise<PersonaHistoryRevision> {
  return get<PersonaHistoryRevision>(`/v2/persona/history/${encodeURIComponent(kind)}/${revision}`);
}

export const PERSONA_KINDS = ["identity", "relationship", "redline", "user", "dream", "dark", "world"] as const;
export type PersonaKindName = (typeof PERSONA_KINDS)[number];

export function fileStateOf(view: PersonaStatusView, kind: string): PersonaFileState | undefined {
  return view.documents.find((document) => document.kind === kind);
}
