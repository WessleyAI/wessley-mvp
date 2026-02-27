import { useState, useEffect, useRef, useCallback } from 'react';
import type { Metrics, MetricsHistoryEntry, Analysis, FixRun, LogEntry, DashboardData } from '@/lib/types';

const BASE = import.meta.env.BASE_URL + 'data/';

async function fetchJSON<T>(file: string): Promise<T> {
  const res = await fetch(BASE + file + '?t=' + Date.now());
  if (!res.ok) throw new Error(`Failed to fetch ${file}`);
  return res.json();
}

export function useData(): DashboardData {
  const [data, setData] = useState<DashboardData>({
    metrics: null, history: [], analysis: null, fixes: [], logs: [],
    loading: true, error: null, lastRefresh: 0,
  });
  const tsRef = useRef('');

  const load = useCallback(async () => {
    try {
      const [metrics, history, analysis, fixes, logs] = await Promise.all([
        fetchJSON<Metrics>('metrics-latest.json'),
        fetchJSON<MetricsHistoryEntry[]>('metrics-history.json'),
        fetchJSON<Analysis>('analysis-latest.json'),
        fetchJSON<FixRun[]>('fixes-history.json'),
        fetchJSON<LogEntry[]>('logs-latest.json'),
      ]);
      if (metrics.timestamp !== tsRef.current) {
        tsRef.current = metrics.timestamp;
        setData({ metrics, history, analysis, fixes, logs, loading: false, error: null, lastRefresh: Date.now() });
      }
    } catch (e) {
      setData(prev => ({ ...prev, loading: false, error: (e as Error).message }));
    }
  }, []);

  useEffect(() => {
    load();
    const iv = setInterval(load, 30_000);
    return () => clearInterval(iv);
  }, [load]);

  return data;
}
