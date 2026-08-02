import { useTranslation } from "react-i18next";
import { useApi } from "../api/hooks";
import type { OverviewResponse } from "../api/types";
import StatusDot from "./StatusDot";

export default function SafetyBanner() {
  const { t } = useTranslation();
  const { data } = useApi<OverviewResponse>("/v2/admin/overview", { poll: 8000 });

  if (!data) return null;

  const degraded = Object.entries(data.components)
    .filter(([, s]) => s !== "ok")
    .map(([name]) => name);
  const spool = data.spool_pending ?? 0;
  const healthy = degraded.length === 0 && spool === 0;
  if (healthy) return null;

  const tone = degraded.some((n) => data.components[n] === "offline") ? "down" : "degraded";

  return (
    <div className={`banner banner-${tone}`} role="status">
      <span className="banner-item">
        <StatusDot tone="info" size={7} />
        {t("banner.localOnly")}
      </span>
      {degraded.length > 0 && (
        <span className="banner-item">
          <StatusDot tone={tone} size={7} />
          {t("banner.degraded", { names: degraded.join(" · ") })}
        </span>
      )}
      {spool > 0 && (
        <span className="banner-item">
          <StatusDot tone="degraded" size={7} />
          {t("banner.pendingSpool", { count: spool })}
        </span>
      )}
      <span className="banner-item banner-compat">{t("banner.compat")}</span>
    </div>
  );
}
