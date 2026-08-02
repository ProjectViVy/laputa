import { useTranslation } from "react-i18next";
import type { Bi } from "../data/governance";

export function useBi() {
  const { i18n } = useTranslation();
  const lang = i18n.language.startsWith("zh") ? "zh" : "en";
  return (bi: Bi) => bi[lang];
}
