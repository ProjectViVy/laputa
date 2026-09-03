import { Link } from "react-router-dom";
import { useApi } from "../api/hooks";
import type { EvolutionProposal, IndexHealthResponse, OverviewResponse } from "../api/types";
import PageHeader from "../components/PageHeader";
import StatusDot from "../components/StatusDot";
import { statusTone, TONE_CLASS } from "../lib/status";

interface PersonaReviewsResponse { items: Array<{ id: string; kind: string; reason: string }>; }

export default function Overview() {
  const { data, error, reload } = useApi<OverviewResponse>("/v2/admin/overview", { poll: 10000 });
  const { data: health } = useApi<IndexHealthResponse>("/v2/admin/index-health", { poll: 10000 });
  const { data: reviews } = useApi<PersonaReviewsResponse>("/v2/persona/reviews?state=pending", { poll: 15000 });
  const { data: proposals } = useApi<{ items: EvolutionProposal[] }>("/v2/evolution/proposals?status=pending", { poll: 15000 });
  const components = data?.components ?? {};
  const names = Object.keys(components);
  const degraded = names.filter((name) => components[name] !== "ok");
  const ingestion = data?.ingestion;

  return <div className="page"><PageHeader title="Overview" lede="Live runtime signals and the next actionable work, with each domain retaining its own authority." right={<button className="btn-ghost mono" onClick={reload}>Refresh</button>} />{error && <div className="callout callout-down">Runtime overview unavailable: {error}</div>}<div className="overview-grid"><section className="panel panel-pad overview-card"><div className="axis-head"><span className="label">Runtime</span><span className={`chip ${degraded.length ? "s-degraded" : "s-ok"}`}>{degraded.length ? "degraded" : "live"}</span></div><div className="stat">{names.length - degraded.length}/{names.length || "—"}</div><div className="component-lines">{names.map((name) => <div className="component-line" key={name}><StatusDot tone={statusTone(components[name])} size={7} /><span className="mono">{name}</span><span className={`mono ${TONE_CLASS[statusTone(components[name])]}`}>{components[name]}</span></div>)}</div></section><section className="panel panel-pad overview-card"><div className="axis-head"><span className="label">Recovery</span><Link className="chip" to="/operations">Operations</Link></div><div className={`stat ${health?.status === "ok" ? "s-ok" : "s-degraded"}`}>{health?.status ?? "—"}</div><div className="mini-stats"><div><span className="telemetry">{data?.spool_pending ?? 0}</span><span className="label">spool pending</span></div><div><span className="telemetry">{health?.pending_jobs ?? "—"}</span><span className="label">index jobs</span></div><div><span className="telemetry">{ingestion?.failed ?? 0}</span><span className="label">failed ingest</span></div></div>{health?.reasons?.length ? <p className="source-note">{health.reasons.join(" · ")}</p> : null}</section><section className="panel panel-pad overview-card"><div className="axis-head"><span className="label">Attention</span><span className="chip">live counts</span></div><div className="attention-list"><Link to="/persona"><span>Persona reviews</span><strong>{reviews?.items.length ?? "—"}</strong></Link><Link to="/evolution"><span>EvoMap proposals</span><strong>{proposals?.items.length ?? "—"}</strong></Link><Link to="/chat-approval"><span>Chat approvals</span><strong>—</strong></Link></div></section><section className="panel panel-pad overview-card"><div className="axis-head"><span className="label">Current activity</span><Link className="chip" to="/memory">Memory</Link></div><p className="source-note">Activity sessions are opened explicitly in the Memory workspace. Recent activity is not merged into Persona or Mentle evidence.</p><div className="mini-stats"><div><span className="telemetry">{ingestion?.total ?? 0}</span><span className="label">ingestion total</span></div><div><span className="telemetry">{ingestion?.running ?? 0}</span><span className="label">running</span></div></div></section></div></div>;
}
