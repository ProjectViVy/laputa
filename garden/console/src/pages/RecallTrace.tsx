import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "../api/hooks";
import type { RecallTrace as Trace } from "../api/types";
import PageHeader from "../components/PageHeader";
import StatusDot from "../components/StatusDot";
import { statusTone, TONE_CLASS } from "../lib/status";

export default function RecallTrace() {
  const { t } = useTranslation();
  const [input, setInput] = useState("");
  const [traceId, setTraceId] = useState<string | null>(null);

  const { data, error, loading } = useApi<Trace>(
    traceId ? `/v2/recall/traces/${encodeURIComponent(traceId)}` : null,
  );

  const notFound = error != null && traceId != null;

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const v = input.trim();
    if (v) setTraceId(v);
  };

  return (
    <div className="page">
      <PageHeader title={t("trace.title")} lede={t("trace.lede")} />

      <form className="trace-form" onSubmit={submit}>
        <label className="label" htmlFor="trace-id">
          {t("trace.inputLabel")}
        </label>
        <div className="trace-input-row">
          <input
            id="trace-id"
            className="trace-input mono"
            placeholder={t("trace.inputPlaceholder")}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            spellCheck={false}
          />
          <button className="btn-primary mono" type="submit">
            {t("trace.load")}
          </button>
        </div>
        <p className="hint mono">{t("trace.hint")}</p>
      </form>

      {loading && <p className="empty-note">{t("common.loading")}…</p>}
      {notFound && <div className="callout callout-down">{t("trace.notFound")}: {error}</div>}

      {data && (
        <div className="trace-grid">
          {/* request */}
          <section className="panel panel-pad reveal">
            <div className="label">{t("trace.request")}</div>
            <dl className="kv mono">
              <div>
                <dt>{t("trace.query")}</dt>
                <dd>{data.query || "—"}</dd>
              </div>
              <div>
                <dt>{t("trace.scope")}</dt>
                <dd>{data.scope || "—"}</dd>
              </div>
              <div>
                <dt>{t("trace.trigger")}</dt>
                <dd>{data.trigger_reason || "—"}</dd>
              </div>
            </dl>
          </section>

          {/* timeline */}
          <section className="panel panel-pad reveal" style={{ animationDelay: "60ms" }}>
            <div className="label">{t("trace.timeline")}</div>
            <ol className="timeline">
              {data.steps.map((s, i) => (
                <li key={i} className="timeline-step">
                  <StatusDot tone={statusTone(s.status === "ok" ? "ok" : s.status)} size={9} pulse={false} />
                  <div className="timeline-body">
                    <span className="mono timeline-name">{s.step}</span>
                    <span className={`mono ${TONE_CLASS[statusTone(s.status === "ok" ? "ok" : s.status)]}`}>
                      {s.status} · {s.duration_ms}ms{s.error_code ? ` · ${s.error_code}` : ""}
                    </span>
                  </div>
                </li>
              ))}
              {data.steps.length === 0 && <p className="empty-note">{t("trace.none")}</p>}
            </ol>
          </section>

          {/* selected / read / rejected */}
          <section className="panel panel-pad reveal" style={{ animationDelay: "120ms" }}>
            <RefBlock title={t("trace.selected")} tone="s-ok" items={data.candidate_ids} empty={t("trace.none")} />
            <RefBlock title={t("trace.read")} tone="s-info" items={data.evidence_refs} empty={t("trace.none")} />
            <RefBlock title={t("trace.rejected")} tone="s-degraded" items={data.filter_conditions} empty={t("trace.none")} />
          </section>

          {/* budget + performance */}
          <section className="panel panel-pad reveal" style={{ animationDelay: "180ms" }}>
            <div className="label">{t("trace.budget")}</div>
            <div className="budget-grid mono">
              <Budget k={t("trace.budgetChars")} v={data.budget.budget_chars} />
              <Budget k={t("trace.usedChars")} v={data.budget.used_chars} />
              <Budget k={t("trace.kgQueries")} v={data.budget.kg_queries} />
              <Budget k={t("trace.timelineQueries")} v={data.budget.timeline_queries} />
              <Budget k={t("trace.cardSearches")} v={data.budget.card_searches} />
            </div>
            <hr className="divider" style={{ margin: "16px 0" }} />
            <div className="label">{t("trace.performance")}</div>
            <div className="budget-grid mono">
              <Budget k={t("trace.duration")} v={`${data.duration_ms}ms`} />
              <Budget k={t("trace.degraded")} v={data.degraded ? "yes" : "no"} tone={data.degraded ? "s-degraded" : "s-ok"} />
            </div>
          </section>
        </div>
      )}
    </div>
  );
}

function RefBlock({ title, items, tone, empty }: { title: string; items: string[]; tone: string; empty: string }) {
  return (
    <div className="ref-block">
      <div className={`label ${tone}`}>
        {title} · {items.length}
      </div>
      {items.length === 0 ? (
        <p className="empty-note">{empty}</p>
      ) : (
        <ul className="ref-list mono">
          {items.map((r, i) => (
            <li key={`${r}-${i}`}>{r}</li>
          ))}
        </ul>
      )}
    </div>
  );
}

function Budget({ k, v, tone }: { k: string; v: React.ReactNode; tone?: string }) {
  return (
    <div className="budget-cell">
      <span className={`telemetry ${tone ?? ""}`}>{v}</span>
      <span className="label">{k}</span>
    </div>
  );
}
