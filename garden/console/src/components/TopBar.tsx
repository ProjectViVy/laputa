import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { useApi } from "../api/hooks";
import type { HealthResponse } from "../api/types";
import { useProfiles } from "../api/profiles";
import StatusDot from "./StatusDot";
import { statusTone } from "../lib/status";

export default function TopBar() {
  const { profiles, selected, select } = useProfiles();
 const { t, i18n } = useTranslation();
  const { data } = useApi<HealthResponse>("/health", { poll: 10000 });
  const isZh = i18n.language.startsWith("zh");

  const toggleLang = () => {
    const next = isZh ? "en" : "zh";
    i18n.changeLanguage(next);
    localStorage.setItem("console.lang", next);
  };

  const tone = statusTone(data?.status);

  return (
    <header className="topbar">
      <div className="topbar-left">
        <span className="scope">
          <span className="label">{t("topbar.scope")}</span>
          <select aria-label="当前人格" value={selected} onChange={(e) => select(e.target.value)}>{profiles.length === 0 && <option value="">尚无人格</option>}{profiles.map((p) => <option key={p.profile_id} value={p.profile_id}>{p.display_name || p.profile_id}</option>)}</select>
 <span className="scope-value mono">{selected}</span>
 <Link to="/access">共享空间</Link>
        </span>
      </div>
      <div className="topbar-right">
        <span className="live-tag">
          <StatusDot tone={tone} size={7} />
          <span className="mono">{data?.status ?? t("topbar.live")}</span>
        </span>
        <Link className="lang-toggle mono" to="/operations">runtime</Link>
        <button className="lang-toggle mono" onClick={toggleLang} aria-label={t("topbar.language")}>
          <span className={!isZh ? "lang-on" : "lang-off"}>EN</span>
          <span className="lang-sep">/</span>
          <span className={isZh ? "lang-on" : "lang-off"}>中文</span>
        </button>
      </div>
    </header>
  );
}
