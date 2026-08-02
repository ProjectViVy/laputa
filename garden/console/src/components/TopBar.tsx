import { useTranslation } from "react-i18next";
import { useApi } from "../api/hooks";
import type { HealthResponse } from "../api/types";
import StatusDot from "./StatusDot";
import { statusTone } from "../lib/status";
import { usePreview } from "../lib/usePreview";

export default function TopBar() {
  const { t, i18n } = useTranslation();
  const { data } = useApi<HealthResponse>("/health", { poll: 10000 });
  const { active: previewActive, toggle: togglePreview } = usePreview();
  const isZh = i18n.language.startsWith("zh");

  const toggleLang = () => {
    const next = isZh ? "en" : "zh";
    i18n.changeLanguage(next);
    localStorage.setItem("console.lang", next);
  };

  const tone = statusTone(data?.status === "ok" ? "ok" : "degraded");

  return (
    <header className="topbar">
      <div className="topbar-left">
        <span className="scope">
          <span className="label">{t("topbar.scope")}</span>
          <span className="scope-value mono">{t("topbar.host")}</span>
        </span>
      </div>
      <div className="topbar-right">
        <span className="live-tag">
          <StatusDot tone={tone} size={7} />
          <span className="mono">{t("topbar.live")}</span>
        </span>
        <button className="lang-toggle mono" onClick={togglePreview} aria-label={t("topbar.preview")}>
          <span className={previewActive ? "lang-on" : "lang-off"}>{t("topbar.preview")}</span>
        </button>
        <button className="lang-toggle mono" onClick={toggleLang} aria-label={t("topbar.language")}>
          <span className={!isZh ? "lang-on" : "lang-off"}>EN</span>
          <span className="lang-sep">/</span>
          <span className={isZh ? "lang-on" : "lang-off"}>中文</span>
        </button>
      </div>
    </header>
  );
}
