import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import en from "./en.json";
import zh from "./zh.json";

const stored = localStorage.getItem("console.lang");

i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    zh: { translation: zh },
  },
  lng: stored === "zh" ? "zh" : "en",
  fallbackLng: "en",
  interpolation: { escapeValue: false },
});

export default i18n;
