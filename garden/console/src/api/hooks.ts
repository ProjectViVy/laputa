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

  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);

  const reload = useCallback(() => setTick((t) => t + 1), []);

  useEffect(() => {
    if (!enabled || !path) return;
    let cancelled = false;
    setLoading(true);

    const run = async () => {
      try {
        const result = await get<T>(path);
        if (!cancelled && alive.current) {
          setData(result);
          setError(null);
        }
      } catch (e) {
        if (!cancelled && alive.current) {
          setError(e instanceof Error ? e.message : String(e));
        }
      } finally {
        if (!cancelled && alive.current) setLoading(false);
      }
    };

    run();
    let timer: ReturnType<typeof setInterval> | undefined;
    if (poll && poll > 0) timer = setInterval(run, poll);

    return () => {
      cancelled = true;
      if (timer) clearInterval(timer);
    };
  }, [path, enabled, poll, tick]);

  return { data, error, loading, reload };
}
