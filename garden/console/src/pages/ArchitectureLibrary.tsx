import { useTranslation } from "react-i18next";
import PageHeader from "../components/PageHeader";
import { DOCS, ROLE_LABELS, type DocRole, type DocStatus } from "../data/docs";
import { useBi } from "../lib/bi";

const ROLES: DocRole[] = ["L0", "L1", "L2", "L3", "L4", "L5"];

const STATUS_TONE: Record<DocStatus, string> = {
  accepted: "s-info",
  implemented: "s-ok",
  proposed: "s-brand",
  superseded: "s-degraded",
  archived: "muted",
};

export default function ArchitectureLibrary() {
  const { t } = useTranslation();
  const bi = useBi();

  return (
    <div className="page">
      <PageHeader title={t("library.title")} lede={t("library.lede")} />

      <div className="lib-stack">
        {ROLES.map((role, idx) => {
          const docs = DOCS.filter((d) => d.role === role);
          if (docs.length === 0) return null;
          return (
            <section key={role} className="lib-role reveal" style={{ animationDelay: `${idx * 50}ms` }}>
              <div className="lib-role-head">
                <span className="lib-role-code telemetry">{role}</span>
                <span className="lib-role-label">{bi(ROLE_LABELS[role])}</span>
                <span className="lib-role-line" />
              </div>
              <div className="lib-cards">
                {docs.map((d) => (
                  <article key={d.id} className={`panel lib-card${d.status === "archived" ? " lib-archived" : ""}`}>
                    <div className="lib-card-head">
                      <h3 className="lib-card-title">{bi(d.title)}</h3>
                      <span className={`chip ${STATUS_TONE[d.status]}`}>{t(`library.status.${d.status}`)}</span>
                    </div>
                    <div className="lib-card-path mono">{d.path}</div>
                    {d.note && <p className="lib-card-note">{bi(d.note)}</p>}
                    <div className="lib-card-meta">
                      <span className="label">{t("library.modules")}</span>
                      <div className="doc-refs mono">
                        {d.modules.map((m) => (
                          <span key={m} className="chip">
                            {m}
                          </span>
                        ))}
                      </div>
                    </div>
                    {d.supersededBy && (
                      <div className="lib-card-super mono">
                        {t("library.supersededBy")}: {d.supersededBy}
                      </div>
                    )}
                    {d.status === "archived" && <div className="lib-archive-note mono">{t("library.archiveNote")}</div>}
                  </article>
                ))}
              </div>
            </section>
          );
        })}
      </div>
    </div>
  );
}
