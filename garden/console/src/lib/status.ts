export type Tone = "ok" | "degraded" | "down" | "info" | "brand" | "muted";

export function statusTone(status: string | undefined): Tone {
  switch ((status || "").toLowerCase()) {
    case "ok":
      return "ok";
    case "degraded":
      return "degraded";
    case "offline":
    case "down":
    case "failed":
      return "down";
    default:
      return "muted";
  }
}

export const TONE_CLASS: Record<Tone, string> = {
  ok: "s-ok",
  degraded: "s-degraded",
  down: "s-down",
  info: "s-info",
  brand: "s-brand",
  muted: "",
};

export const TONE_VAR: Record<Tone, string> = {
  ok: "var(--ok)",
  degraded: "var(--degraded)",
  down: "var(--down)",
  info: "var(--info)",
  brand: "var(--brand)",
  muted: "var(--ink-3)",
};
