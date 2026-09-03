import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "../api/hooks";
import { post } from "../api/client";
import type { MailboxItem, MailboxListResponse } from "../api/types";
import PageHeader from "../components/PageHeader";

function fmtTime(iso?: string): string {
  if (!iso) return "—";
  return iso.replace("T", " ").slice(0, 19) + "Z";
}

function errMsg(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

function reviewable(item: MailboxItem): boolean {
  return item.state === "received" || item.state === "evaluating";
}

export default function MailboxPage() {
  const { t } = useTranslation();
  return (
    <div className="page">
      <PageHeader title={t("mailbox.title")} lede={t("mailbox.lede")} />
      <MailboxPanel />
    </div>
  );
}

export function MailboxPanel() {
  return <div className="modules-grid"><InboxColumn /><OutboxColumn /></div>;
}

function InboxColumn() {
  const { t } = useTranslation();
  const [selected, setSelected] = useState<string | null>(null);
  const [revision, setRevision] = useState(0);
  const { data, loading, error } = useApi<MailboxListResponse>(
    `/v2/mailbox/inbox?rev=${revision}`,
    { poll: 8000 }
  );

  const items = data?.items ?? [];
  const bump = () => setRevision((r) => r + 1);

  return (
    <section className="modules-column">
      <h2 className="modules-title">{t("mailbox.inbox.title")}</h2>

      {loading && <p className="materials-empty">{t("common.loading")}…</p>}
      {error && <p className="materials-empty error">{error}</p>}
      {!loading && !error && items.length === 0 && (
        <p className="materials-empty">{t("mailbox.inbox.empty")}</p>
      )}

      <ul className="modules-list">
        {items.map((item) => (
          <li
            key={item.id}
            className={`modules-item evolution-item${selected === item.id ? " selected" : ""}`}
            onClick={() => setSelected(selected === item.id ? null : item.id)}
          >
            <div className="modules-item-meta">
              <span className={`modules-badge ${item.state}`}>{item.state}</span>
              <span className="modules-item-date mono">{item.id}</span>
              <span className="modules-item-date">{fmtTime(item.updated_at)}</span>
            </div>
            {selected === item.id && <InboxDetail item={item} onChanged={bump} />}
          </li>
        ))}
      </ul>
    </section>
  );
}

function InboxDetail({ item, onChanged }: { item: MailboxItem; onChanged: () => void }) {
  const { t } = useTranslation();
  const [confirming, setConfirming] = useState<"approved" | "rejected" | null>(null);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const review = async (decision: "approved" | "rejected") => {
    if (confirming !== decision) {
      setConfirming(decision);
      return;
    }
    if (busy) return;
    setBusy(true);
    setErr(null);
    try {
      await post(
        `/v2/mailbox/items/${encodeURIComponent(item.id)}/${decision === "approved" ? "approve" : "reject"}`,
        { reason: reason.trim() }
      );
      setConfirming(null);
      setReason("");
      onChanged();
    } catch (e) {
      setErr(errMsg(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="evolution-detail" onClick={(e) => e.stopPropagation()}>
      <ItemFacts item={item} />
      {reviewable(item) && (
        <>
          <div className="modules-actions">
            <button className="modules-btn primary" onClick={() => review("approved")} disabled={busy}>
              {confirming === "approved"
                ? t("mailbox.review.confirmApprove")
                : t("mailbox.review.approve")}
            </button>
            <button className="modules-btn" onClick={() => review("rejected")} disabled={busy}>
              {confirming === "rejected"
                ? t("mailbox.review.confirmReject")
                : t("mailbox.review.reject")}
            </button>
          </div>
          {confirming && (
            <div className="modules-edit">
              <input
                className="modules-input"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder={t("mailbox.review.reasonPlaceholder")}
              />
              <button
                className="modules-btn"
                onClick={() => {
                  setConfirming(null);
                  setReason("");
                }}
                disabled={busy}
              >
                {t("mailbox.review.cancel")}
              </button>
            </div>
          )}
        </>
      )}
      {err && <p className="materials-empty error">{err}</p>}
    </div>
  );
}

function OutboxColumn() {
  const { t } = useTranslation();
  const [selected, setSelected] = useState<string | null>(null);
  const { data, loading, error } = useApi<MailboxListResponse>(
    "/v2/mailbox/outbox",
    { poll: 8000 }
  );

  const items = data?.items ?? [];

  return (
    <section className="modules-column">
      <h2 className="modules-title">{t("mailbox.outbox.title")}</h2>

      {loading && <p className="materials-empty">{t("common.loading")}…</p>}
      {error && <p className="materials-empty error">{error}</p>}
      {!loading && !error && items.length === 0 && (
        <p className="materials-empty">{t("mailbox.outbox.empty")}</p>
      )}

      <ul className="modules-list">
        {items.map((item) => (
          <li
            key={item.id}
            className={`modules-item evolution-item${selected === item.id ? " selected" : ""}`}
            onClick={() => setSelected(selected === item.id ? null : item.id)}
          >
            <div className="modules-item-meta">
              <span className={`modules-badge ${item.state}`}>{item.state}</span>
              <span className="modules-item-date mono">{item.id}</span>
              <span className="modules-item-date">{fmtTime(item.updated_at)}</span>
            </div>
            {selected === item.id && <OutboxDetail item={item} />}
          </li>
        ))}
      </ul>
    </section>
  );
}

function OutboxDetail({ item }: { item: MailboxItem }) {
  const { t } = useTranslation();
  const leakage = item.leakage;
  return (
    <div className="evolution-detail" onClick={(e) => e.stopPropagation()}>
      <ItemFacts item={item} />
      {leakage && (
        <div className="modules-item-meta">
          <span className="evolution-hub-key">{t("mailbox.detail.leakage")}</span>
          {leakage.clean ? (
            <span className="modules-badge active">{t("mailbox.detail.clean")}</span>
          ) : (
            <span className="modules-badge failed">{t("mailbox.detail.violated")}</span>
          )}
          <span className="modules-item-date">
            {t("mailbox.detail.checkedRefs")}: {leakage.checked_refs}
          </span>
        </div>
      )}
      {(leakage?.violations?.length ?? 0) > 0 && (
        <ul className="evolution-leak-list">
          {leakage!.violations!.map((v, i) => (
            <li key={i} className="materials-empty error">{v}</li>
          ))}
        </ul>
      )}
      {(leakage?.warnings?.length ?? 0) > 0 && (
        <ul className="evolution-leak-list">
          {leakage!.warnings!.map((v, i) => (
            <li key={i} className="materials-empty">{v}</li>
          ))}
        </ul>
      )}
      {item.reason && (
        <p className="modules-item-date">
          {t("mailbox.detail.reason")}: {item.reason}
        </p>
      )}
      {item.retry_count > 0 && (
        <p className="modules-item-date">
          {t("mailbox.detail.retryCount")}: {item.retry_count}
        </p>
      )}
    </div>
  );
}

function ItemFacts({ item }: { item: MailboxItem }) {
  const { t } = useTranslation();
  return (
    <>
      <pre className="mailbox-pre">{JSON.stringify(item.payload, null, 2)}</pre>
      {item.evidence_refs.length > 0 && (
        <div className="modules-item-meta">
          <span className="evolution-hub-key">{t("mailbox.detail.evidence")}</span>
          <span className="mono">{item.evidence_refs.join(", ")}</span>
        </div>
      )}
      <p className="modules-item-date">
        {t("mailbox.detail.created", { date: fmtTime(item.created_at) })} ·{" "}
        {t("mailbox.detail.updated", { date: fmtTime(item.updated_at) })}
      </p>
    </>
  );
}
