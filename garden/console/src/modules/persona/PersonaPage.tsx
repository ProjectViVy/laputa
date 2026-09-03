import { useState } from "react";
import { useTranslation } from "react-i18next";
import { post, put } from "../../api/client";
import { useApi } from "../../api/hooks";
import PageHeader from "../../components/PageHeader";
import { fileStateOf, PERSONA_KINDS } from "./api";
import type { PersonaDocument, PersonaFileState, PersonaHistoryEntry, PersonaHistoryRevision, PersonaStatusView } from "./types";

function fmtTime(iso?: string | null): string {
  if (!iso) return "—";
  return iso.replace("T", " ").slice(0, 19) + "Z";
}

function errMsg(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

export default function PersonaPage() {
  const { t } = useTranslation();
  const [selected, setSelected] = useState("identity");
  const [reviewMode, setReviewMode] = useState(false);
  const { data, loading, error, reload } = useApi<PersonaStatusView>("/v2/persona/documents");

  if (data?.status === "uninitialized") {
    return <PersonaInitialization onComplete={reload} />;
  }

  const state = fileStateOf(data ?? { status: "incomplete", documents: [] }, selected);
  return (
    <div className="page">
      <PageHeader title={t("persona.title")} lede="Seven Markdown authority documents, immutable history, and a separate review queue." />
      <div className="workspace-note mono">source: laputa/persona · profile={data?.status ?? "loading"} · tool-only documents stay explicit</div>
      {loading && !data && <p className="materials-empty">{t("common.loading")}</p>}
      {error && <InlineError message={error} onRetry={reload} />}
      {data && (
        <div className="persona-layout">
          <section className="persona-list-col panel panel-pad">
            <div className="persona-list-head">
              <h2 className="section-title">Documents</h2>
              <span className={`persona-profile-badge persona-profile-${data.status}`}>{data.status}</span>
            </div>
            <ul className="persona-list">
              {PERSONA_KINDS.map((kind) => {
                const item = fileStateOf(data, kind);
                return (
                  <li key={kind}>
                    <button type="button" className={`persona-item${selected === kind ? " selected" : ""}`} onClick={() => { setSelected(kind); setReviewMode(false); }}>
                      <span className="persona-item-file mono">{item?.file_name ?? kind}</span>
                      <PersonaFileStatus state={item} />
                    </button>
                  </li>
                );
              })}
            </ul>
            <button type="button" className="btn-ghost" onClick={() => setReviewMode(!reviewMode)}>{reviewMode ? "Document" : "Review queue"}</button>
          </section>
          <section className="persona-detail-col panel panel-pad">
            {reviewMode ? <ReviewQueue /> : <PersonaDocumentDetail kind={selected} state={state} />}
          </section>
          {!reviewMode && <section className="persona-history-col panel panel-pad"><PersonaHistory kind={selected} /></section>}
        </div>
      )}
    </div>
  );
}

function PersonaFileStatus({ state }: { state?: PersonaFileState }) {
  if (!state) return <span className="persona-badge persona-missing">missing</span>;
  if (!state.exists) return <span className={`persona-badge ${state.required ? "persona-missing" : "persona-optional"}`}>{state.required ? "missing" : "optional"}</span>;
  if (!state.valid) return <span className="persona-badge persona-invalid">invalid</span>;
  return <span className="persona-badge persona-ready">ready · {state.revision}</span>;
}

function PersonaDocumentDetail({ kind, state }: { kind: string; state?: PersonaFileState }) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState<string | null>(null);
  const [scope, setScope] = useState<"document" | "observations">("document");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const { data, loading, error, reload } = useApi<PersonaDocument>(`/v2/persona/documents/${encodeURIComponent(kind)}`);
  const current = draft ?? data?.content ?? "";

  const save = async () => {
    if (!data || busy) return;
    setBusy(true); setMessage(null);
    try {
      await put(`/v2/persona/documents/${encodeURIComponent(kind)}`, { base_revision: data.revision, content: current, scope: kind === "user" ? scope : "document", reason: reason.trim() || "User-authorized Persona document edit" });
      setEditing(false); setDraft(null); setReason(""); reload();
    } catch (error) { setMessage(errMsg(error)); }
    finally { setBusy(false); }
  };

  if (loading && !data) return <p className="materials-empty">Loading document…</p>;
  if (error) return <InlineError message={error} onRetry={reload} />;
  if (!data) return <p className="materials-empty">Document is missing or unavailable.</p>;
  return (
    <div className="persona-detail">
      <div className="persona-detail-head"><h2 className="section-title mono">{data.file_name}</h2><span className="persona-meta mono">rev {data.revision} · {fmtTime(data.updated_at)}</span></div>
      {state?.tool_only && <p className="persona-tool-note">Tool-only document. It is never added to bootstrap, recall, or automatic context.</p>}
      <div className="persona-meta-row"><span className="persona-meta">{current.length} / {state?.content_limit ?? "—"} chars</span><span className="persona-meta mono">{data.content_hash}</span></div>
      {editing ? <textarea className="persona-editor mono" value={current} onChange={(event) => setDraft(event.target.value)} autoFocus /> : <pre className="persona-content mono">{data.content || "—"}</pre>}
      {editing && <>
        {kind === "user" && <select className="modules-input" value={scope} onChange={(event) => setScope(event.target.value as "document" | "observations")}><option value="document">User preference document</option><option value="observations">Agent observation append</option></select>}
        <input className="modules-input" value={reason} onChange={(event) => setReason(event.target.value)} placeholder="Reason for this document edit" />
        <div className="modules-actions"><button className="btn-primary" onClick={save} disabled={busy || !current.trim()}>{busy ? "Saving…" : "Save document"}</button><button className="btn-ghost" onClick={() => { setEditing(false); setDraft(null); }}>Cancel</button></div>
      </>}
      {!editing && <button className="btn-ghost" onClick={() => { setDraft(data.content); setEditing(true); }}>Edit document</button>}
      {message && <p className="materials-empty error">{message}</p>}
    </div>
  );
}

function PersonaHistory({ kind }: { kind: string }) {
  const { data, loading, error, reload } = useApi<{ kind: string; items: PersonaHistoryEntry[] }>(`/v2/persona/history/${encodeURIComponent(kind)}`);
  const [revision, setRevision] = useState<number | null>(null);
  if (loading) return <p className="materials-empty">Loading history…</p>;
  if (error) return <InlineError message={error} onRetry={reload} />;
  const entries = data?.items ?? [];
  return <div><div className="axis-head"><h2 className="section-title">History</h2><span className="chip">{entries.length} revisions</span></div>{entries.length === 0 ? <p className="materials-empty">No immutable revisions yet.</p> : <ul className="persona-history-list">{entries.map((entry) => <li key={entry.revision}><button type="button" className={`persona-history-item${revision === entry.revision ? " selected" : ""}`} onClick={() => setRevision(entry.revision)}><span className="mono">rev {entry.revision}</span><span className="persona-history-meta">{entry.actor} · {entry.source} · {fmtTime(entry.created_at)}</span></button></li>)}</ul>}{revision !== null && <HistoryDetail kind={kind} revision={revision} />}</div>;
}

function HistoryDetail({ kind, revision }: { kind: string; revision: number }) {
  const { data, loading, error } = useApi<PersonaHistoryRevision>(`/v2/persona/history/${encodeURIComponent(kind)}/${revision}`);
  if (loading) return <p className="materials-empty">Loading revision…</p>;
  if (error || !data) return <p className="materials-empty error">{error ?? "Revision unavailable"}</p>;
  return <div className="persona-history-detail"><div className="persona-meta-row"><span>rev {data.revision}</span><span>{data.actor} · base {data.base_revision}</span></div><pre className="persona-content mono">{data.content}</pre>{data.unified_diff && <pre className="persona-diff mono">{data.unified_diff}</pre>}</div>;
}

function ReviewQueue() {
  const { data, loading, error, reload } = useApi<{ items: Array<{ id: string; kind: string; base_revision: number; reason: string; actor: string; state: string; proposed_markdown: string }> }>("/v2/persona/reviews?state=pending");
  const [busy, setBusy] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const decide = async (id: string, action: "approve" | "reject") => { setBusy(id); setMessage(null); try { await post(`/v2/persona/reviews/${encodeURIComponent(id)}/${action}`, {}); reload(); } catch (error) { setMessage(errMsg(error)); } finally { setBusy(null); } };
  if (loading) return <p className="materials-empty">Loading review queue…</p>;
  if (error) return <InlineError message={error} onRetry={reload} />;
  const items = data?.items ?? [];
  return <div><div className="axis-head"><h2 className="section-title">Persona review queue</h2><span className="chip">{items.length} pending</span></div>{items.length === 0 ? <p className="materials-empty">No Persona proposals require review.</p> : <ul className="review-list">{items.map((item) => <li key={item.id} className="review-item"><div className="modules-item-meta"><span className="modules-badge">{item.kind}</span><span className="mono">base rev {item.base_revision}</span><span>{item.actor}</span></div><p>{item.reason}</p><pre className="persona-diff mono">{item.proposed_markdown}</pre><div className="modules-actions"><button className="btn-primary" disabled={busy === item.id} onClick={() => decide(item.id, "approve")}>{busy === item.id ? "Working…" : "Approve Persona"}</button><button className="btn-ghost" disabled={busy === item.id} onClick={() => decide(item.id, "reject")}>Reject</button></div></li>)}</ul>}{message && <p className="materials-empty error">{message}</p>}</div>;
}

function PersonaInitialization({ onComplete }: { onComplete: () => void }) {
  const [values, setValues] = useState<Record<string, string>>({ identity: "", relationship: "", redline: "", user: "", world: "" });
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const initialize = async () => { setBusy(true); setMessage(null); try { await post("/v2/persona/initialize", values); onComplete(); } catch (error) { setMessage(errMsg(error)); } finally { setBusy(false); } };
  return <div className="page"><PageHeader title="Initialize Persona" lede="Create the five required Markdown authority documents. Optional files remain absent until explicitly written." /><section className="panel panel-pad setup-card">{["identity", "relationship", "redline", "user", "world"].map((kind) => <label key={kind} className="setup-field"><span className="label">{kind}.md</span><textarea className="persona-editor mono" value={values[kind]} onChange={(event) => setValues((current) => ({ ...current, [kind]: event.target.value }))} rows={3} /></label>)}<button className="btn-primary" disabled={busy || Object.values(values).some((value) => !value.trim())} onClick={initialize}>{busy ? "Initializing…" : "Initialize Persona"}</button>{message && <p className="materials-empty error">{message}</p>}</section></div>;
}

function InlineError({ message, onRetry }: { message: string; onRetry: () => void }) { return <p className="materials-empty error">{message} <button type="button" className="link-button" onClick={onRetry}>Retry</button></p>; }
