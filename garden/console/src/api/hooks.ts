import { useCallback, useEffect, useRef, useState } from "react";
import { get } from "./client";

interface UseApiOptions {
  poll?: number;
  enabled?: boolean;
}

interface UseApiResult<T> {
  data: T | null;
  error: string | null;
  loading: boolean;
  reload: () => void;
}

export function useApi<T>(path: string | null, opts: UseApiOptions = {}): UseApiResult<T> {
  const { poll, enabled = true } = opts;
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [tick, setTick] = useState(0);
  const alive = useRef(true);
  const resource = useRef("");

  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);

  const reload = useCallback(() => setTick((t) => t + 1), []);

  useEffect(() => {
    const key = `${enabled}:${path}`;
    if (resource.current !== key) { setData(null); resource.current = key; }
    setError(null);
 if (!enabled || !path) { setLoading(false); return; }
 const controller = new AbortController();
    let cancelled = false;
    setLoading(true);

    let sequence = 0;
    const run = async () => {
      if (poll && document.visibilityState !== "visible") return;
      const request = ++sequence;
      try {
        const result = await get<T>(path, controller.signal);
        if (!cancelled && alive.current && request === sequence) {
          setData(result);
          setError(null);
        }
      } catch (e) {
        if (!cancelled && alive.current && request === sequence) {
          setError(e instanceof Error ? e.message : String(e));
        }
      } finally {
        if (!cancelled && alive.current && request === sequence) setLoading(false);
      }
    };

    run();
    let timer: ReturnType<typeof setInterval> | undefined;
    if (poll && poll > 0) timer = setInterval(run, poll);

    const focus = () => { if (poll) void run(); };
    window.addEventListener("focus", focus); document.addEventListener("visibilitychange", focus);
    return () => {
      window.removeEventListener("focus", focus); document.removeEventListener("visibilitychange", focus);
      cancelled = true;
 controller.abort();
      if (timer) clearInterval(timer);
    };
  }, [path, enabled, poll, tick]);

  return { data, error, loading, reload };
}
