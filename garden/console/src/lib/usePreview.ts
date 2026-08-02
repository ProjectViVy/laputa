import { useCallback, useSyncExternalStore } from "react";

const KEY = "console.preview";

function getSnapshot() {
  return localStorage.getItem(KEY) === "1";
}

function subscribe(cb: () => void) {
  window.addEventListener("storage", cb);
  return () => window.removeEventListener("storage", cb);
}

export function usePreview() {
  const active = useSyncExternalStore(subscribe, getSnapshot);
  const toggle = useCallback(() => {
    localStorage.setItem(KEY, getSnapshot() ? "0" : "1");
    window.dispatchEvent(new Event("storage"));
  }, []);
  return { active, toggle };
}
