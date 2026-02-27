import { useState, useCallback } from 'react';

const API_BASE = 'http://localhost:8080';

export function useIsLocal(): boolean {
  return ['localhost', '127.0.0.1'].includes(location.hostname);
}

export function useApi() {
  const isLocal = useIsLocal();
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const call = useCallback(async (path: string, opts?: RequestInit) => {
    if (!isLocal) { setError('Only available on localhost'); return null; }
    setLoading(true); setError(null); setResult(null);
    try {
      const res = await fetch(API_BASE + path, opts);
      const text = await res.text();
      if (!res.ok) throw new Error(text);
      setResult(text);
      return text;
    } catch (e) {
      setError((e as Error).message);
      return null;
    } finally {
      setLoading(false);
    }
  }, [isLocal]);

  return { call, loading, result, error, isLocal };
}
