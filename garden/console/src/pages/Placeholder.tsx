import { useTranslation } from "react-i18next";
import { usePreview } from "../lib/usePreview";

interface Props {
  titleKey: string;
}

export default function Placeholder({ titleKey }: Props) {
  const { t } = useTranslation();
  const { active } = usePreview();

  if (!active) {
    return (
      <div className="page">
        <div className="placeholder reveal">
          <h1 className="placeholder-title">{t(titleKey)}</h1>
          <p className="placeholder-body">{t("placeholder.unavailable")}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="page">
      <div className="placeholder reveal">
        <div className="placeholder-badge mono">{t("status.acceptedDesign")}</div>
        <h1 className="placeholder-title">{t(titleKey)}</h1>
        <p className="placeholder-heading">{t("placeholder.title")}</p>
        <p className="placeholder-body">{t("placeholder.body")}</p>
      </div>
    </div>
  );
}
