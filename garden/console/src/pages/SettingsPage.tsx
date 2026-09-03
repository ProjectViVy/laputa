import { useState } from "react";
import { useTranslation } from "react-i18next";
import { setCapabilityToken } from "../api/client";
import { useApi } from "../api/hooks";
import type { PipelinesResponse, ProviderStatus } from "../api/types";
import PageHeader from "../components/PageHeader";

export default function SettingsPage() {
  const { t, i18n } = useTranslation();
  const { data: pipelines } = useApi<PipelinesResponse>("/v2/pipelines");
  const { data: hubStatus } = useApi<ProviderStatus>("/v2/evolution/hub/status");
  const [token, setToken] = useState("");
  const [theme, setTheme] = useState(() => localStorage.getItem("garden.console.theme") ?? "cyber-dark");

  const toggleLanguage = () => {
    i18n.changeLanguage(i18n.language === "zh" ? "en" : "zh");
  };

  const saveToken = () => setCapabilityToken(token);
  const changeTheme = (value: string) => {
    setTheme(value);
    localStorage.setItem("garden.console.theme", value);
    if (value === "cyber-dark") document.documentElement.removeAttribute("data-theme");
    else document.documentElement.setAttribute("data-theme", value);
  };

  return (
    <div className="page">
      <PageHeader title={t("settings.title")} lede={t("settings.lede")} />

      {/* General Settings & Language */}
      <div className="panel panel-pad reveal" style={{ marginBottom: "1.5rem" }}>
        <h3 style={{ margin: "0 0 1rem 0", fontSize: "1.1rem" }}>{t("settings.environment")}</h3>
        <div className="mono" style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))", gap: "1rem" }}>
          <div>
            <span className="label" style={{ display: "block", color: "var(--text-muted, #a0aec0)" }}>{t("settings.contract")}</span>
            <span style={{ fontWeight: 600, color: "var(--text-main, #e2e8f0)" }}>garden-hermes/1</span>
          </div>
          <div>
            <span className="label" style={{ display: "block", color: "var(--text-muted, #a0aec0)" }}>{t("topbar.language")}</span>
            <button className="btn-ghost mono" onClick={toggleLanguage} style={{ padding: "0.2rem 0.6rem" }}>
              {i18n.language.toUpperCase()}
            </button>
          </div>
          <div>
            <span className="label" style={{ display: "block", color: "var(--text-muted, #a0aec0)" }}>{t("topbar.scope")}</span>
            <span style={{ fontWeight: 600 }}>single-host · local</span>
          </div>
        </div>
      </div>

      <div className="panel panel-pad reveal" style={{ marginBottom: "1.5rem" }}>
        <h3 style={{ margin: "0 0 1rem 0", fontSize: "1.1rem" }}>Capability session</h3>
        <p className="source-note">Write actions use the server capability contract. The token is held in session storage and is never sent to a different origin.</p>
        <div style={{ display: "flex", gap: "0.75rem", flexWrap: "wrap", marginTop: "0.75rem" }}>
          <input className="modules-input" type="password" value={token} onChange={(event) => setToken(event.target.value)} placeholder="Paste a capability token" />
          <button className="btn-ghost" onClick={saveToken}>Save for session</button>
          <select className="modules-input" value={theme} onChange={(event) => changeTheme(event.target.value)} aria-label="Theme">
            <option value="cyber-dark">Cyber dark</option>
            <option value="warm-editorial">Warm editorial</option>
            <option value="neo-brutal">Neo brutal</option>
          </select>
        </div>
      </div>

      {/* EvoMap Hub Provider Status */}
      <div className="panel panel-pad reveal" style={{ marginBottom: "1.5rem" }}>
        <div className="axis-head" style={{ marginBottom: "1rem" }}>
          <h3 style={{ margin: 0, fontSize: "1.1rem" }}>{t("settings.hubStatus")}</h3>
          <span className={`chip ${hubStatus?.claimed ? "s-ok" : "s-degraded"}`}>
            {hubStatus?.claimed ? "CLAIMED" : "UNCLAIMED"}
          </span>
        </div>

        {!hubStatus ? (
          <p className="mono" style={{ color: "var(--text-muted, #718096)", fontStyle: "italic" }}>
            Hub provider status offline or unavailable.
          </p>
        ) : (
          <div className="mono" style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))", gap: "1rem" }}>
            <div>
              <span className="label" style={{ display: "block", color: "var(--text-muted, #a0aec0)" }}>{t("settings.nodeId")}</span>
              <span style={{ fontWeight: 600 }}>{hubStatus.node_id || "—"}</span>
            </div>
            <div>
              <span className="label" style={{ display: "block", color: "var(--text-muted, #a0aec0)" }}>{t("settings.credits")}</span>
              <span style={{ fontWeight: 600, color: hubStatus.has_credit_balance ? "#48bb78" : "#f56565" }}>
                {hubStatus.credit_balance ?? 0}
              </span>
            </div>
            <div>
              <span className="label" style={{ display: "block", color: "var(--text-muted, #a0aec0)" }}>Survival Status</span>
              <span>{hubStatus.survival_status}</span>
            </div>
          </div>
        )}
      </div>

      {/* Configured Pipelines */}
      <div className="panel panel-pad reveal">
        <div className="axis-head" style={{ marginBottom: "1rem" }}>
          <h3 style={{ margin: 0, fontSize: "1.1rem" }}>{t("settings.pipelines")}</h3>
          <span className="mono label">{pipelines?.revision || "v2.0"}</span>
        </div>

        {!pipelines || pipelines.pipelines.length === 0 ? (
          <p className="mono" style={{ color: "var(--text-muted, #718096)", fontStyle: "italic" }}>
            No pipelines configured.
          </p>
        ) : (
          <div style={{ overflowX: "auto" }}>
            <table className="mono" style={{ width: "100%", fontSize: "0.85rem", borderCollapse: "collapse" }}>
              <thead>
                <tr style={{ borderBottom: "1px solid var(--border-subtle, #2d3748)", textTransform: "uppercase", textAlign: "left" }}>
                  <th style={{ padding: "0.5rem" }}>Pipeline Name</th>
                  <th style={{ padding: "0.5rem" }}>Version</th>
                  <th style={{ padding: "0.5rem" }}>Capabilities</th>
                  <th style={{ padding: "0.5rem" }}>Max Steps</th>
                </tr>
              </thead>
              <tbody>
                {pipelines.pipelines.map((p) => (
                  <tr key={p.name} style={{ borderBottom: "1px solid var(--border-subtle, #1a202c)" }}>
                    <td style={{ padding: "0.5rem", fontWeight: 600, color: "var(--text-accent, #63b3ed)" }}>{p.name}</td>
                    <td style={{ padding: "0.5rem" }}>{p.version}</td>
                    <td style={{ padding: "0.5rem" }}>
                      {p.capabilities.map((c) => (
                        <span key={c} className="chip s-info" style={{ marginRight: "0.3rem", fontSize: "0.75rem" }}>{c}</span>
                      ))}
                    </td>
                    <td style={{ padding: "0.5rem" }}>{p.max_steps}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
