import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "../api/hooks";
import { patch, post } from "../api/client";
import type {
  HumanModule,
  ModuleKind,
  ModulesListResponse,
  ModuleStatus,
} from "../api/types";
import PageHeader from "../components/PageHeader";

const KINDS: ModuleKind[] = ["ambition", "suggestion"];

function fmtDate(iso: string): string {
  return iso.slice(0, 10);
}

export default function ModulesPage() {
  const { t } = useTranslation();
  return (
    <div className="page">
      <PageHeader title={t("modules.title")} lede={t("modules.lede")} />
      <div className="modules-grid">
        {KINDS.map((kind) => (
          <ModuleColumn key={kind} kind={kind} />
        ))}
      </div>
    </div>
  );
}

function ModuleColumn({ kind }: { kind: ModuleKind }) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [editing, setEditing] = useState<string | null>(null);
  const [editText, setEditText] = useState("");
  const { data, loading, error, reload } = useApi<ModulesListResponse>(
    `/v2/reports/modules?kind=${kind}&status=all`
  );

  const run = async (fn: () => Promise<unknown>) => {
    if (busy) return;
    setBusy(true);
    setActionError(null);
    try {
      await fn();
      reload();
    } catch (e) {
      setActionError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const create = () =>
    run(async () => {
      await post("/v2/reports/modules", { kind, content: draft });
      setDraft("");
    });

  const setStatus = (m: HumanModule, status: ModuleStatus) =>
    run(async () => {
      await patch(`/v2/reports/modules/${m.id}`, { status });
    });

  const saveEdit = (m: HumanModule) =>
    run(async () => {
      await patch(`/v2/reports/modules/${m.id}`, { content: editText });
      setEditing(null);
    });

  const items = data?.items ?? [];

  return (
    <section className="modules-column">
      <h2 className="modules-title">{t(`modules.${kind}`)}</h2>

      <div className="modules-create">
        <textarea
          className="modules-input"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder={t("modules.placeholder")}
          rows={2}
        />
        <button
          className="modules-btn primary"
          onClick={create}
          disabled={busy || !draft.trim()}
        >
          {t("modules.create")}
        </button>
      </div>

      {actionError && <p className="materials-empty error">{actionError}</p>}
      {loading && <p className="materials-empty">{t("common.loading")}…</p>}
      {error && <p className="materials-empty error">{error}</p>}
      {!loading && !error && items.length === 0 && (
        <p className="materials-empty">{t("modules.empty")}</p>
      )}

      <ul className="modules-list">
        {items.map((m) => (
          <li
            key={m.id}
            className={`modules-item${m.status === "dismissed" ? " dismissed" : ""}`}
          >
            <div className="modules-item-meta">
              <span className={`modules-badge ${m.status}`}>
                {t(m.status === "active" ? "modules.statusActive" : "modules.statusDismissed")}
              </span>
              <span className="modules-item-date">
                {t("modules.created", { date: fmtDate(m.created_at) })}
              </span>
            </div>
            {editing === m.id ? (
              <div className="modules-edit">
                <textarea
                  className="modules-input"
                  value={editText}
                  onChange={(e) => setEditText(e.target.value)}
                  rows={2}
                />
                <div className="modules-actions">
                  <button
                    className="modules-btn primary"
                    onClick={() => saveEdit(m)}
                    disabled={busy || !editText.trim()}
                  >
                    {t("modules.save")}
                  </button>
                  <button className="modules-btn" onClick={() => setEditing(null)}>
                    {t("modules.cancel")}
                  </button>
                </div>
              </div>
            ) : (
              <>
                <p className="modules-content">{m.content}</p>
                <div className="modules-actions">
                  <button
                    className="modules-btn"
                    onClick={() => {
                      setEditing(m.id);
                      setEditText(m.content);
                    }}
                  >
                    {t("modules.edit")}
                  </button>
                  {m.status === "active" ? (
                    <button className="modules-btn" onClick={() => setStatus(m, "dismissed")}>
                      {t("modules.dismiss")}
                    </button>
                  ) : (
                    <button className="modules-btn" onClick={() => setStatus(m, "active")}>
                      {t("modules.reactivate")}
                    </button>
                  )}
                </div>
              </>
            )}
          </li>
        ))}
      </ul>
    </section>
  );
}
