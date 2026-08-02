import { useTranslation } from "react-i18next";
import { useApi } from "../api/hooks";
import type { AuditResponse, OverviewResponse } from "../api/types";
import PageHeader from "../components/PageHeader";
import StatusDot from "../components/StatusDot";
import { statusTone, TONE_CLASS } from "../lib/status";

function Stat({ value, tone = "ok" }: { value: React.ReactNode; tone?: string }) {
  return <div className={`stat telemetry ${TONE_CLASS[statusTone(tone)] || ""}`}>{value}</div>;
}

export default function Overview() {
  const { t } = useTranslation();
  const { data, error, reload } = useApi<OverviewResponse>("/v2/admin/overview", { poll: 8000 });
  const { data: audit } = useApi<AuditResponse>("/v2/admin/audit?limit=5", { poll: 15000 });

  const components = data?.components ?? {};
  const names = Object.keys(components);
  const degraded = names.filter((n) => components[n] !== "ok");
  const okCount = names.length - degraded.length;
  const ing = data?.ingestion;
  const spool = data?.spool_pending ?? 0;

  return (
    <div className="page">
      <PageHeader
        title={t("overview.title")}
        lede={t("overview.lede")}
        right={
          <button className="btn-ghost mono" onClick={reload}>
            {t("common.retry")}
          </button>
        }
      />

      {error && <div className="callout callout-down">{t("common.error")}: {error}</div>}

      <div className="axis-grid">
        {/* Runtime — live */}
        <section className="panel panel-pad axis reveal">
          <div className="axis-head">
            <span className="label">{t("overview.runtime")}</span>
            <span className="chip s-ok">{t("status.live")}</span>
          </div>
          <Stat value={`${okCount}/${names.length || 0}`} tone={degraded.length ? "degraded" : "ok"} />
          <div className="axis-sub mono">
            {degraded.length ? `${degraded.length} ${t("overview.degradedCount")}` : t("status.ok")}
          </div>
          <div className="component-lines">
            {names.map((n) => (
              <div key={n} className="component-line">
                <StatusDot tone={statusTone(components[n])} size={7} />
                <span className="mono">{n}</span>
                <span className={`mono ${TONE_CLASS[statusTone(components[n])]}`}>{components[n]}</span>
              </div>
            ))}
          </div>
        </section>

        {/* Data — live */}
        <section className="panel panel-pad axis reveal" style={{ animationDelay: "60ms" }}>
          <div className="axis-head">
            <span className="label">{t("overview.data")}</span>
            <span className="chip s-ok">{t("status.live")}</span>
          </div>
          <Stat value={spool} tone={spool > 0 ? "degraded" : "ok"} />
          <div className="axis-sub mono">{t("overview.spoolPending")}</div>
          <div className="mini-stats">
            <div>
              <span className="telemetry">{ing?.total ?? 0}</span>
              <span className="label">{t("overview.ingestionTotal")}</span>
            </div>
            <div>
              <span className={`telemetry ${(ing?.failed ?? 0) > 0 ? "s-down" : ""}`}>{ing?.failed ?? 0}</span>
              <span className="label">{t("overview.failedSources")}</span>
            </div>
          </div>
        </section>

        {/* Cognitive — accepted-design */}
        <section className="panel panel-pad axis axis-dim reveal" style={{ animationDelay: "120ms" }}>
          <div className="axis-head">
            <span className="label">{t("overview.cognitive")}</span>
            <span className="chip s-info">{t("status.acceptedDesign")}</span>
          </div>
          <Stat value="—" tone="muted" />
          <div className="axis-sub mono">{t("overview.stmFreshness")}</div>
          <div className="mini-stats">
            <div>
              <span className="telemetry">—</span>
              <span className="label">{t("overview.worldConflicts")}</span>
            </div>
          </div>
        </section>

        {/* Governance — compat/accepted-design */}
        <section className="panel panel-pad axis axis-dim reveal" style={{ animationDelay: "180ms" }}>
          <div className="axis-head">
            <span className="label">{t("overview.governance")}</span>
            <span className="chip s-degraded">{t("status.compat")}</span>
          </div>
          <Stat value="—" tone="muted" />
          <div className="axis-sub mono">{t("overview.frozenCore")}</div>
          <div className="mini-stats">
            <div>
              <span className="telemetry">—</span>
              <span className="label">{t("overview.rulesVersion")}</span>
            </div>
            <div>
              <span className="telemetry">—</span>
              <span className="label">{t("overview.pendingReview")}</span>
            </div>
          </div>
        </section>
      </div>

      {/* current work surface */}
      <div className="work-grid">
        <section className="panel panel-pad reveal">
          <div className="label">{t("overview.recentAudit")}</div>
          {!audit || audit.entries.length === 0 ? (
            <p className="empty-note">{t("overview.noAudit")}</p>
          ) : (
            <ul className="audit-lines mono">
              {audit.entries.map((e) => (
                <li key={e.sequence}>
                  <span className="audit-seq">#{e.sequence}</span>
                  <span className="audit-action">{e.action}</span>
                  <span className="audit-section">{e.section}</span>
                  <span className="audit-actor">{e.actor}</span>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section className="panel panel-pad reveal" style={{ animationDelay: "80ms" }}>
          <div className="label">{t("overview.componentState")}</div>
          <div className="component-lines">
            {names.map((n) => (
              <div key={n} className="component-line">
                <StatusDot tone={statusTone(components[n])} size={7} />
                <span className="mono">{n}</span>
                <span className={`mono ${TONE_CLASS[statusTone(components[n])]}`}>{components[n]}</span>
              </div>
            ))}
          </div>
        </section>
      </div>
    </div>
  );
}
