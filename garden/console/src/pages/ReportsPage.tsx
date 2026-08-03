import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "../api/hooks";
import type { ReportItem, ReportsListResponse } from "../api/types";
import PageHeader from "../components/PageHeader";

const CADENCES = ["daily", "weekly", "monthly"] as const;

function fmtWindow(item: ReportItem): string {
  return `${item.window_start.slice(0, 10)} → ${item.window_end.slice(0, 10)}`;
}

function ArtifactList({ label, items }: { label: string; items: string[] | null }) {
  const list = items ?? [];
  if (list.length === 0) return null;
  return (
    <div className="reports-artifact-block">
      <span className="reports-artifact-label">{label}</span>
      <ul>
        {list.map((it, i) => (
          <li key={i}>{it}</li>
        ))}
      </ul>
    </div>
  );
}

export default function ReportsPage() {
  const { t } = useTranslation();
  const [cadence, setCadence] = useState<(typeof CADENCES)[number]>("daily");
  const [expanded, setExpanded] = useState<string | null>(null);

  const { data, loading, error } = useApi<ReportsListResponse>(`/v2/reports?cadence=${cadence}&limit=20`);

  const selectCadence = (c: (typeof CADENCES)[number]) => {
    setCadence(c);
    setExpanded(null);
  };

  return (
    <div className="page">
      <PageHeader title={t("reports.title")} lede={t("reports.lede")} />

      <div className="reports-tabs">
        {CADENCES.map((c) => (
          <button
            key={c}
            className={`reports-tab${cadence === c ? " active" : ""}`}
            onClick={() => selectCadence(c)}
          >
            {t(`reports.cadence.${c}`)}
          </button>
        ))}
      </div>

      {loading && <p className="materials-empty">{t("common.loading")}…</p>}
      {error && <p className="materials-empty error">{error}</p>}
      {!loading && data && data.items.length === 0 && (
        <p className="materials-empty">{t("reports.noReports")}</p>
      )}

      <section className="reports-list">
        {data?.items.map((item) => {
          const key = `${item.window_start}:${item.source_hash}`;
          const isOpen = expanded === key;
          return (
            <div key={key} className={`reports-item${isOpen ? " open" : ""}`}>
              <div className="reports-item-head" onClick={() => setExpanded(isOpen ? null : key)}>
                <div className="reports-item-title">
                  <span>{item.title}</span>
                  <span className={`materials-badge materials-badge--${item.generator}`}>{item.generator}</span>
                </div>
                <div className="reports-item-meta mono">
                  <span>{fmtWindow(item)}</span>
                  <span>{t("reports.revision")}: {item.revision}</span>
                  <span>{t("reports.generatedAt")}: {item.generated_at.replace("T", " ").slice(0, 16)}Z</span>
                  <span>{t("reports.sources")}: {(item.source_refs ?? []).length}</span>
                </div>
              </div>
              {isOpen && (
                <div className="reports-detail">
                  <p className="reports-summary">{item.summary}</p>
                  <ArtifactList label={t("reports.artifact.goals")} items={item.goals} />
                  <ArtifactList label={t("reports.artifact.completed")} items={item.completed} />
                  <ArtifactList label={t("reports.artifact.decisions")} items={item.decisions} />
                  <ArtifactList label={t("reports.artifact.openLoops")} items={item.open_loops} />
                  <div className="reports-detail-meta mono">
                    <span>{t("reports.scope")}: {item.scope}</span>
                    <span>{t("reports.hash")}: {item.source_hash.slice(0, 16)}…</span>
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </section>
    </div>
  );
}
