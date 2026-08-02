import { TONE_VAR, type Tone } from "../lib/status";

interface Props {
  tone: Tone;
  pulse?: boolean;
  size?: number;
}

export default function StatusDot({ tone, pulse = true, size = 8 }: Props) {
  const color = TONE_VAR[tone];
  return (
    <span
      aria-hidden
      style={{
        display: "inline-block",
        width: size,
        height: size,
        borderRadius: "50%",
        background: color,
        boxShadow: `0 0 8px ${color}`,
        animation: pulse ? "pulse 2s ease-in-out infinite" : undefined,
        flexShrink: 0,
      }}
    />
  );
}
