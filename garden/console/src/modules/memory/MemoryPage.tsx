import { useState } from "react";
import { get, post } from "../../api/client";
import { useApi } from "../../api/hooks";
import PageHeader from "../../components/PageHeader";
import type { CardsResponse, EvidenceResponse, MemoryCard, SpoolResponse } from "../../api/types";

interface ActmemView {
  revision: number;
  updated_at: string;
  markdown: string;
  truncated: boolean;
  sections?: { pulse: string; recap: string; work: string };
}

interface ActmemQueryResult {
  revision: number;
  items: Array<{ section: string; work_section?: string; line_index: number; excerpt: string }>;
  returned_chars: number;
  truncated: boolean;
}

interface ActivityResponse {
  session_id: string;
  events: Array<{ id: string; type: string; created_at?: string; data?: Record<string, unknown> }>;
}

function errorText(value: unknown): string { return value instanceof Error ? value.message : String(value); }

export default function MemoryPage() {
  return <div className="page"><PageHeader title="Memory" lede="Explicit activity memory, Mentle evidence, and Garden activity remain separate sources." /><div className="workspace-note mono">ACTMEM and evidence are explicit reads; no background promotion or combined memory projection.</div><div className="memory-grid"><ActmemPanel /><MentlePanel /></div><ActivityPanel /></div>;
}

function ActmemPanel() {
  const [document, setDocument] = useState<ActmemView | null>(null);
  const [query, setQuery] = useState("");
  const [queryResult, setQueryResult] = useState<ActmemQueryResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const read = async () => { setBusy(true); setMessage(null); try { setDocument(await get<ActmemView>("/v2/actmem?max_chars=1200")); } catch (error) { setMessage(errorText(error)); } finally { setBusy(false); } };
  const runQuery = async (event: React.FormEvent) => { event.preventDefault(); if (!query.trim()) return; setBusy(true); setMessage(null); try { setQueryResult(await post<ActmemQueryResult>("/v2/actmem/query", { query: query.trim(), max_hits: 12, max_chars: 1200 })); } catch (error) { setMessage(errorText(error)); } finally { setBusy(false); } };
  return <section className="panel panel-pad memory-source memory-source-actmem"><div className="axis-head"><div><span className="label">Activity memory</span><h2 className="section-title">ACTMEM</h2></div><span className="chip">explicit</span></div><p className="source-note">Tool-only, bounded, and never part of automatic context.</p><div className="modules-actions"><button className="btn-primary" disabled={busy} onClick={read}>{busy ? "Reading…" : "Read ACTMEM"}</button></div><form className="memory-query" onSubmit={runQuery}><input className="modules-input" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Query activity memory" /><button className="btn-ghost" disabled={busy || !query.trim()}>Query</button></form>{message && <p className="materials-empty error">{message}</p>}{document && <pre className="memory-body mono">rev {document.revision} · {document.updated_at}{document.truncated ? " · truncated" : ""}{"\n\n"}{document.markdown}</pre>}{queryResult && <div className="memory-query-result"><div className="label">Query results · rev {queryResult.revision}</div>{queryResult.items.length === 0 ? <p className="empty-note">No matching activity entries.</p> : queryResult.items.map((item, index) => <div className="memory-hit" key={`${item.section}-${item.line_index}-${index}`}><span className="mono">{item.section}{item.work_section ? `/${item.work_section}` : ""}:{item.line_index}</span><span>{item.excerpt}</span></div>)}</div>}</section>;
}

function MentlePanel() {
  const [query, setQuery] = useState("");
  const [submitted, setSubmitted] = useState("");
  const [selected, setSelected] = useState<MemoryCard | null>(null);
  const cardsPath = submitted ? `/v2/materials/cards?query=${encodeURIComponent(submitted)}&limit=20` : null;
  const { data, loading, error } = useApi<CardsResponse>(cardsPath);
  const evidence = useApi<EvidenceResponse>(selected ? `/v2/materials/cards/${encodeURIComponent(selected.id)}/evidence` : null);
  return <section className="panel panel-pad memory-source memory-source-mentle"><div className="axis-head"><div><span className="label">Canonical evidence</span><h2 className="section-title">Mentle materials</h2></div><span className="chip">cards → evidence</span></div><form className="memory-query" onSubmit={(event) => { event.preventDefault(); setSubmitted(query.trim()); setSelected(null); }}><input className="modules-input" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search cards" /><button className="btn-primary" disabled={!query.trim()}>Search</button></form>{loading && <p className="materials-empty">Loading cards…</p>}{error && <p className="materials-empty error">{error}</p>}{submitted && !loading && !error && (data?.cards.length ?? 0) === 0 && <p className="materials-empty">No cards match this submitted query.</p>}<div className="memory-card-list">{(data?.cards ?? []).map((card) => <button type="button" className={`memory-card${selected?.id === card.id ? " selected" : ""}`} key={card.id} onClick={() => setSelected(card)}><span className="memory-card-title">{card.title || card.summary || card.id}</span><span className="mono">{card.kind} · rev {card.revision} · {card.status}</span><span className="memory-card-summary">{card.summary}</span></button>)}</div>{selected && <div className="evidence-detail"><div className="label">Evidence for {selected.id}</div>{evidence.loading && <p className="materials-empty">Loading evidence…</p>}{evidence.error && <p className="materials-empty error">{evidence.error}</p>}{(evidence.data?.fragments ?? []).map((fragment, index) => <article className="evidence-fragment" key={`${fragment.content_hash}-${index}`}><p>{fragment.excerpt}</p><span className="mono">{fragment.material_ref} · {fragment.validity} · {fragment.content_hash}</span></article>)}</div>}</section>;
}

function ActivityPanel() {
  const [input, setInput] = useState("");
  const [session, setSession] = useState("");
  const { data, loading, error } = useApi<ActivityResponse>(session ? `/v2/activity/sessions/${encodeURIComponent(session)}?limit=40` : null);
  const { data: spool } = useApi<SpoolResponse>("/v2/admin/spool");
  return <section className="panel panel-pad activity-source"><div className="axis-head"><div><span className="label">Garden runtime</span><h2 className="section-title">Activity sessions</h2></div><span className="chip">spool {spool?.pending_count ?? "—"}</span></div><form className="memory-query" onSubmit={(event) => { event.preventDefault(); setSession(input.trim()); }}><input className="modules-input" value={input} onChange={(event) => setInput(event.target.value)} placeholder="Enter a session id" /><button className="btn-ghost" disabled={!input.trim()}>Open session</button></form>{loading && <p className="materials-empty">Loading session…</p>}{error && <p className="materials-empty error">{error}</p>}{data && <div className="activity-events">{data.events.length === 0 ? <p className="empty-note">No events for this session.</p> : data.events.map((event) => <div className="activity-event" key={event.id}><span className="mono">{event.type}</span><span>{event.id}</span><span className="muted">{event.data ? JSON.stringify(event.data) : ""}</span></div>)}</div>}</section>;
}
