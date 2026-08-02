import { useTranslation } from "react-i18next";
import type { GovNode } from "../data/governance";
import { useBi } from "../lib/bi";
import { statusTone, TONE_CLASS } from "../lib/status";
import StatusDot from "./StatusDot";

interface Props {
  node: GovNode | null;
  runtimeStatus?: string;
  onClose: () => void;
}

export default function Inspector({ node, runtimeStatus, onClose }: Props) {
  const { t } = useTranslation();
  const bi = useBi();
  if (!node) return null;

  const tone = statusTone(runtimeStatus);
  const phaseTone =
    node.phase === "existing" ? "ok" : node.phase === "compat" ? "degraded" : "info";

  const rows: { label: string; body: React.ReactNode }[] = [
    { label: t("governance.inspector.responsibility"), body: bi(node.responsibility) },
    {
      label: t("governance.inspector.runtime"),
      body: runtimeStatus ? (
        <span className={`chip ${TONE_CLASS[tone]}`}>
          <StatusDot tone={tone} size={6} /> {runtimeStatus}
        </span>
      ) : (
        <span className="chip">{t("status.acceptedDesign")}</span>
      ),
    },
    { label: t("governance.inspector.boundary"), body: bi(node.boundary) },
    {
      label: t("governance.inspector.io"),
      body: (
        <>
          <div>→ {bi(node.inputs)}</div>
          <div>← {bi(node.outputs)}</div>
        </>
      ),
    },
    {
      label: t("governance.inspector.apis"),
      body: (
        <ul className="api-list mono">
          {node.apis.map((a) => (
            <li key={a}>{a}</li>
          ))}
        </ul>
      ),
    },
    { label: t("governance.inspector.rules"), body: bi(node.rules) },
    { label: t("governance.inspector.limitations"), body: bi(node.limitations) },
    {
      label: t("governance.inspector.phase"),
      body: <span className={`chip ${TONE_CLASS[phaseTone]}`}>{t(`status.${node.phase === "existing" ? "live" : node.phase}`)}</span>,
    },
    {
      label: t("governance.inspector.docs"),
      body: (
        <div className="doc-refs mono">
          {node.docs.map((d) => (
            <span key={d} className="chip">
              {d}
            </span>
          ))}
        </div>
      ),
    },
  ];

  return (
    <>
      <div className="drawer-scrim" onClick={onClose} />
      <aside className="drawer" role="dialog" aria-label={node.name}>
        <div className="drawer-head">
          <div>
            <div className="drawer-title">{node.name}</div>
            <div className="label">{t("governance.title")}</div>
          </div>
          <button className="drawer-close" onClick={onClose} aria-label="close">
            ✕
          </button>
        </div>
        <div className="drawer-body">
          {rows.map((row) => (
            <section key={row.label} className="drawer-section">
              <div className="label">{row.label}</div>
              <div className="drawer-content">{row.body}</div>
            </section>
          ))}
        </div>
      </aside>
    </>
  );
}
