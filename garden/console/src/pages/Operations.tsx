import { useTranslation } from "react-i18next";
import { useApi } from "../api/hooks";
import type {
  AuditResponse,
  ComponentsResponse,
  OverviewResponse,
  PipelinesResponse,
  SpoolResponse,
} from "../api/types";
import PageHeader from "../components/PageHeader";
import StatusDot from "../components/StatusDot";
import { statusTone, TONE_CLASS } from "../lib/status";

export default function Operations() {
  const { t } = useTranslation();
  const { data: comps } = useApi<ComponentsResponse>("/v2/admin/components", { poll: 10000 });
  const { data: spool } = useApi<SpoolResponse>("/v2/admin/spool", { poll: 8000 });
  const { data: audit } = useApi<AuditResponse>("/v2/admin/audit?limit=20", { poll: 15000 });
  const { data: pipes } = useApi<PipelinesResponse>("/v1/pipelines");
  const { data: overview } = useApi<OverviewResponse>("/v2/admin/overview", { poll: 10000 });
  const ing = overview?.ingestion;

  return (
    <div className="page">
      <PageHeader title={t("operations.title")} lede={t("operations.lede")} />

      <div className="ops-grid">
        {/* component health */}
        <section className="panel panel-pad reveal">
          <div className="label">{t("operations.componentHealth")}</div>
          <table className="table mono">
            <tbody>
              {(comps?.components ?? []).map((c) => (
                <tr key={c.name}>
                  <td>
                    <span className="cell-status">
                      <StatusDot tone={statusTone(c.status)} size={7} />
                      {c.name}
                    </span>
                  </td>
                  <td className={TONE_CLASS[statusTone(c.status)]}>{c.status}</td>
                  <td className="muted">{c.source}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {ing && (
            <div className="ingest-strip">
              <span className="label">{t("operations.ingestion")}</span>
              <div className="ingest-nums mono">
                <span>{ing.accepted} acc</span>
                <span>{ing.running} run</span>
                <span className={ing.spooled ? "s-degraded" : ""}>{ing.spooled} spool</span>
                <span className="s-ok">{ing.completed} done</span>
                <span className={ing.failed ? "s-down" : ""}>{ing.failed} fail</span>
              </div>
            </div>
          )}
        </section>

        {/* spool */}
        <section className="panel panel-pad reveal" style={{ animationDelay: "60ms" }}>
          <div className="spool-head">
            <span className="label">{t("operations.spool")}</span>
            <span className={`chip ${spool && spool.pending_count > 0 ? "s-degraded" : "s-ok"}`}>
              {spool?.pending_count ?? 0} {t("operations.pendingCount")}
            </span>
          </div>
          {!spool || spool.entries.length === 0 ? (
            <p className="empty-note">{t("operations.noSpool")}</p>
          ) : (
            <ul className="spool-list mono">
              {spool.entries.map((e) => (
                <li key={e.event_id}>
                  <span className="spool-id">{e.event_id.slice(0, 12)}</span>
                  <span>{e.session_id.slice(0, 10) || "—"}</span>
                  <span className="muted">{e.kind}</span>
                  <span className="muted">{e.created_at.slice(11, 19)}</span>
                </li>
              ))}
            </ul>
          )}
        </section>

        {/* pipelines */}
        <section className="panel panel-pad reveal" style={{ animationDelay: "120ms" }}>
          <div className="spool-head">
            <span className="label">{t("operations.pipelines")}</span>
            {pipes && <span className="chip">{t("operations.revision")} {pipes.revision.slice(0, 8)}</span>}
          </div>
          {!pipes || pipes.pipelines.length === 0 ? (
            <p className="empty-note">{t("operations.noPipelines")}</p>
          ) : (
            <ul className="pipe-list">
              {pipes.pipelines.map((p) => (
                <li key={p.name} className="pipe-row">
                  <span className="pipe-name mono">{p.name}</span>
                  <span className="mono muted">v{p.version}</span>
                  <span className="chip">{p.capabilities?.length ?? 0} cap</span>
                </li>
              ))}
            </ul>
          )}
        </section>

        {/* audit */}
        <section className="panel panel-pad reveal" style={{ animationDelay: "180ms" }}>
          <div className="label">{t("operations.audit")}</div>
          {!audit || audit.entries.length === 0 ? (
            <p className="empty-note">{t("operations.noAudit")}</p>
          ) : (
            <table className="table table-audit mono">
              <thead>
                <tr>
                  <th>{t("operations.seq")}</th>
                  <th>{t("operations.action")}</th>
                  <th>{t("operations.section")}</th>
                  <th>{t("operations.actor")}</th>
                  <th>{t("operations.when")}</th>
                </tr>
              </thead>
              <tbody>
                {audit.entries.map((e) => (
                  <tr key={e.sequence}>
                    <td className="muted">#{e.sequence}</td>
                    <td className="s-info">{e.action}</td>
                    <td>{e.section}</td>
                    <td>{e.actor}</td>
                    <td className="muted">{e.timestamp.slice(0, 19).replace("T", " ")}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      </div>
    </div>
  );
}
