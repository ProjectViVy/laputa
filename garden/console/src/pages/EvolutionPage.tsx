import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "../api/hooks";
import { post } from "../api/client";
import type {
  EvolutionProposal,
  EvolutionRun,
  GeneCandidate,
  ProposalsListResponse,
  ProviderStatus,
  RunsListResponse,
} from "../api/types";
import PageHeader from "../components/PageHeader";

function fmtTime(iso?: string): string {
  if (!iso) return "—";
  return iso.replace("T", " ").slice(0, 19) + "Z";
}

function errMsg(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

export default function EvolutionPage() {
  const { t } = useTranslation();
  const { data: hub, error: hubError } = useApi<ProviderStatus>(
    "/v2/evolution/hub/status",
    { poll: 30000 }
  );
  const degraded = Boolean(hubError);

  return (
    <div className="page">
      <PageHeader title={t("evolution.title")} lede={t("evolution.lede")} />

      <section className="evolution-hub">
        <h2 className="modules-title">{t("evolution.hub.title")}</h2>
        {degraded && (
          <p className="materials-empty error">{t("evolution.hub.degraded")}</p>
        )}
        {!degraded && hub && (
          <div className="evolution-hub-rows">
            <div className="evolution-hub-row">
              <span className="evolution-hub-key">{t("evolution.hub.nodeId")}</span>
              <span className="mono">{hub.node_id}</span>
            </div>
            <div className="evolution-hub-row">
              <span className="evolution-hub-key">{t("evolution.hub.claimed")}</span>
              {hub.claimed ? (
                <span className="modules-badge active">{t("evolution.hub.claimedYes")}</span>
              ) : hub.claim_url ? (
                <a className="evolution-claim-link" href={hub.claim_url} target="_blank" rel="noreferrer">
                  {t("evolution.hub.claimLink")}
                </a>
              ) : (
                <span className="modules-badge">{t("evolution.hub.claimedNo")}</span>
              )}
            </div>
            {hub.has_credit_balance && (
              <div className="evolution-hub-row">
                <span className="evolution-hub-key">{t("evolution.hub.credits")}</span>
                <span>{hub.credit_balance}</span>
              </div>
            )}
            <div className="evolution-hub-row">
              <span className="evolution-hub-key">{t("evolution.hub.survival")}</span>
              <span>{hub.survival_status || "—"}</span>
            </div>
            <div className="evolution-hub-row">
              <span className="evolution-hub-key">{t("evolution.hub.heartbeat")}</span>
              <span className="mono">{fmtTime(hub.last_heartbeat)}</span>
            </div>
          </div>
        )}
      </section>

      <div className="modules-grid">
        <RunsColumn degraded={degraded} />
        <ProposalsColumn />
      </div>
    </div>
  );
}

function RunsColumn({ degraded }: { degraded: boolean }) {
  const { t } = useTranslation();
  const [trigger, setTrigger] = useState("");
  const [outcome, setOutcome] = useState("");
  const [refs, setRefs] = useState("");
  const [traceRef, setTraceRef] = useState("");
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const { data, loading, error, reload } = useApi<RunsListResponse>(
    "/v2/evolution/runs",
    { poll: 8000 }
  );

  const start = async () => {
    if (busy) return;
    setBusy(true);
    setActionError(null);
    try {
      const evidenceRefs = refs.split(",").map((s) => s.trim()).filter(Boolean);
      await post("/v2/evolution/runs", {
        trigger: trigger.trim(),
        outcome: outcome.trim(),
        trace_ref: traceRef.trim(),
        evidence_refs: evidenceRefs,
      });
      setTrigger("");
      setOutcome("");
      setRefs("");
      setTraceRef("");
      reload();
    } catch (e) {
      setActionError(errMsg(e));
    } finally {
      setBusy(false);
    }
  };

  const items = data?.items ?? [];

  return (
    <section className="modules-column">
      <h2 className="modules-title">{t("evolution.runs.title")}</h2>

      <div className="modules-create">
        <input
          className="modules-input"
          value={trigger}
          onChange={(e) => setTrigger(e.target.value)}
          placeholder={t("evolution.runs.triggerPlaceholder")}
        />
        <input
          className="modules-input"
          value={outcome}
          onChange={(e) => setOutcome(e.target.value)}
          placeholder={t("evolution.runs.outcome")}
        />
        <input
          className="modules-input"
          value={refs}
          onChange={(e) => setRefs(e.target.value)}
          placeholder={t("evolution.runs.evidenceRefs")}
        />
        <input
          className="modules-input"
          value={traceRef}
          onChange={(e) => setTraceRef(e.target.value)}
          placeholder={t("evolution.runs.traceRef")}
        />
        <button
          className="modules-btn primary"
          onClick={start}
          disabled={busy || degraded || !trigger.trim()}
        >
          {t("evolution.runs.start")}
        </button>
        {degraded && <p className="materials-empty">{t("evolution.runs.disabled")}</p>}
      </div>

      {actionError && <p className="materials-empty error">{actionError}</p>}
      {loading && <p className="materials-empty">{t("common.loading")}…</p>}
      {error && <p className="materials-empty error">{error}</p>}
      {!loading && !error && items.length === 0 && (
        <p className="materials-empty">{t("evolution.runs.empty")}</p>
      )}

      <ul className="modules-list">
        {items.map((run) => (
          <li
            key={run.run_id}
            className={`modules-item evolution-item${selected === run.run_id ? " selected" : ""}`}
            onClick={() => setSelected(selected === run.run_id ? null : run.run_id)}
          >
            <div className="modules-item-meta">
              <span className={`modules-badge ${run.status}`}>{run.status}</span>
              <span className="modules-badge">{run.provider}</span>
              <span className="modules-item-date mono">{run.run_id}</span>
              <span className="modules-item-date">{fmtTime(run.started_at)}</span>
            </div>
            {run.error && <p className="materials-empty error">{run.error}</p>}
            {selected === run.run_id && <RunDetail runId={run.run_id} />}
          </li>
        ))}
      </ul>
    </section>
  );
}

function RunDetail({ runId }: { runId: string }) {
  const { t } = useTranslation();
  const { data: run, loading } = useApi<EvolutionRun>(
    `/v2/evolution/runs/${encodeURIComponent(runId)}`,
    { poll: 4000 }
  );
  if (loading && !run) return <p className="materials-empty">{t("common.loading")}…</p>;
  if (!run) return null;
  const candidates = run.candidates ?? [];
  return (
    <div className="evolution-detail" onClick={(e) => e.stopPropagation()}>
      {candidates.length === 0 && (
        <p className="materials-empty">{t("evolution.candidates.empty")}</p>
      )}
      {candidates.map((id) => (
        <RunCandidate key={id} candidateId={id} runId={runId} />
      ))}
    </div>
  );
}

function RunCandidate({ candidateId, runId }: { candidateId: string; runId: string }) {
  const { t } = useTranslation();
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const { data } = useApi<GeneCandidate>(
    `/v2/evolution/candidates/${encodeURIComponent(candidateId)}`
  );

  const propose = async () => {
    if (busy || done) return;
    setBusy(true);
    setErr(null);
    try {
      await post("/v2/evolution/proposals", { run_id: runId, candidate_id: candidateId });
      setDone(true);
    } catch (e) {
      setErr(errMsg(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="evolution-candidate">
      <div className="modules-item-meta">
        <span className="modules-badge">{data?.kind ?? "…"}</span>
        <span className="evolution-candidate-name">
          {data?.name ?? candidateId}
        </span>
        {data && (
          <span className="modules-item-date">
            {t("evolution.candidates.confidence")}: {(data.confidence * 100).toFixed(0)}%
          </span>
        )}
      </div>
      {data?.description && <p className="modules-content">{data.description}</p>}
      <div className="modules-actions">
        <button className="modules-btn" onClick={propose} disabled={busy || done}>
          {done ? t("evolution.candidates.proposed") : t("evolution.candidates.propose")}
        </button>
      </div>
      {err && <p className="materials-empty error">{err}</p>}
    </div>
  );
}

function ProposalsColumn() {
  const { t } = useTranslation();
  const [selected, setSelected] = useState<string | null>(null);
  const [revision, setRevision] = useState(0);
  const { data, loading, error } = useApi<ProposalsListResponse>(
    `/v2/evolution/proposals?rev=${revision}`,
    { poll: 8000 }
  );

  const items = data?.items ?? [];
  const bump = () => setRevision((r) => r + 1);

  return (
    <section className="modules-column">
      <h2 className="modules-title">{t("evolution.proposals.title")}</h2>

      {loading && <p className="materials-empty">{t("common.loading")}…</p>}
      {error && <p className="materials-empty error">{error}</p>}
      {!loading && !error && items.length === 0 && (
        <p className="materials-empty">{t("evolution.proposals.empty")}</p>
      )}

      <ul className="modules-list">
        {items.map((p) => (
          <li
            key={p.proposal_id}
            className={`modules-item evolution-item${selected === p.proposal_id ? " selected" : ""}`}
            onClick={() => setSelected(selected === p.proposal_id ? null : p.proposal_id)}
          >
            <div className="modules-item-meta">
              <span className={`modules-badge ${p.status}`}>{p.status}</span>
              <span className="modules-badge">{p.kind}</span>
              <span className="modules-item-date">{fmtTime(p.created_at)}</span>
            </div>
            <p className="modules-content">{p.summary}</p>
            {selected === p.proposal_id && <ProposalDetail proposal={p} onChanged={bump} />}
          </li>
        ))}
      </ul>
    </section>
  );
}

function ProposalDetail({
  proposal,
  onChanged,
}: {
  proposal: EvolutionProposal;
  onChanged: () => void;
}) {
  const { t } = useTranslation();
  const [confirming, setConfirming] = useState<"approved" | "rejected" | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const leakage = proposal.leakage_report;

  const review = async (decision: "approved" | "rejected") => {
    if (confirming !== decision) {
      setConfirming(decision);
      return;
    }
    if (busy) return;
    setBusy(true);
    setErr(null);
    try {
      await post(`/v2/evolution/proposals/${proposal.proposal_id}/review`, { decision });
      setConfirming(null);
      onChanged();
    } catch (e) {
      setErr(errMsg(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="evolution-detail" onClick={(e) => e.stopPropagation()}>
      <div className="modules-item-meta">
        <span className="evolution-hub-key">{t("evolution.proposals.leakage")}</span>
        {leakage.clean ? (
          <span className="modules-badge active">{t("evolution.proposals.clean")}</span>
        ) : (
          <span className="modules-badge failed">{t("evolution.proposals.violated")}</span>
        )}
        <span className="modules-item-date">
          {t("evolution.proposals.checkedRefs")}: {leakage.checked_refs}
        </span>
      </div>
      {(leakage.violations?.length ?? 0) > 0 && (
        <ul className="evolution-leak-list">
          {leakage.violations!.map((v, i) => (
            <li key={i} className="materials-empty error">{v}</li>
          ))}
        </ul>
      )}
      {(leakage.warnings?.length ?? 0) > 0 && (
        <ul className="evolution-leak-list">
          {leakage.warnings!.map((v, i) => (
            <li key={i} className="materials-empty">{v}</li>
          ))}
        </ul>
      )}
      {proposal.reviewer && (
        <p className="modules-item-date">
          {t("evolution.proposals.reviewedBy")}: {proposal.reviewer}
          {proposal.review_note ? ` — ${proposal.review_note}` : ""}
        </p>
      )}
      {proposal.status === "pending" && (
        <div className="modules-actions">
          <button
            className="modules-btn primary"
            onClick={() => review("approved")}
            disabled={busy}
          >
            {confirming === "approved"
              ? t("evolution.proposals.confirmApprove")
              : t("evolution.proposals.approve")}
          </button>
          <button
            className="modules-btn"
            onClick={() => review("rejected")}
            disabled={busy}
          >
            {confirming === "rejected"
              ? t("evolution.proposals.confirmReject")
              : t("evolution.proposals.reject")}
          </button>
        </div>
      )}
      {err && <p className="materials-empty error">{err}</p>}
    </div>
  );
}
