import { PageTransition } from '@/components/shared/PageTransition';
import { EmptyState } from '@/components/shared/EmptyState';
import { AnimatedCounter } from '@/components/shared/AnimatedCounter';
import { InsightCard } from '@/components/shared/InsightCard';
import { motion } from 'framer-motion';
import { BookOpen, Download, FileInput, AlertTriangle } from 'lucide-react';
import type { DashboardData } from '@/lib/types';

export function Manuals({ data }: { data: DashboardData }) {
  const { metrics: m } = data;
  if (!m) return null;

  const man = m.scrapers.manuals;
  const stages = [
    { label: 'Discovered', value: man.discovered, icon: <BookOpen className="h-5 w-5" />, color: 'text-blue-400' },
    { label: 'Downloaded', value: man.downloaded, icon: <Download className="h-5 w-5" />, color: 'text-purple-400' },
    { label: 'Ingested', value: man.ingested, icon: <FileInput className="h-5 w-5" />, color: 'text-emerald-400' },
    { label: 'Failed', value: man.failed, icon: <AlertTriangle className="h-5 w-5" />, color: 'text-red-400' },
  ];

  const discoveredToDownloaded = man.discovered > 0 ? ((man.downloaded / man.discovered) * 100).toFixed(0) : '—';
  const downloadedToIngested = man.downloaded > 0 ? ((man.ingested / man.downloaded) * 100).toFixed(0) : '—';

  const isEmpty = man.discovered === 0 && man.downloaded === 0 && man.ingested === 0;

  return (
    <PageTransition>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100">Manuals</h1>
          <p className="text-sm text-zinc-500 mt-1">Owner manual discovery and ingestion pipeline</p>
        </div>

        {isEmpty && (
          <InsightCard severity="warning" title="Manuals pipeline completely inactive"
            detail="Despite having 26+ manufacturer sources coded, no manuals have been discovered. This is the highest-value untapped data source for Wessley." />
        )}

        {/* Funnel */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          {stages.map((s, i) => (
            <motion.div key={s.label} initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.05 }}
              className="rounded-xl border border-white/5 bg-zinc-900/50 p-5 text-center">
              <span className={s.color}>{s.icon}</span>
              <p className="text-2xl font-bold text-zinc-100 mt-2"><AnimatedCounter value={s.value} /></p>
              <p className="text-xs text-zinc-500 mt-1">{s.label}</p>
            </motion.div>
          ))}
        </div>

        {/* Conversion rates */}
        {!isEmpty && (
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.3 }}
            className="rounded-xl border border-white/5 bg-zinc-900/50 p-5">
            <h3 className="text-xs text-zinc-500 uppercase tracking-wider mb-4">Conversion Rates</h3>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-sm text-zinc-300">Discovered → Downloaded</p>
                <p className="text-xl font-bold text-zinc-100">{discoveredToDownloaded}%</p>
              </div>
              <div>
                <p className="text-sm text-zinc-300">Downloaded → Ingested</p>
                <p className="text-xl font-bold text-zinc-100">{downloadedToIngested}%</p>
              </div>
            </div>
          </motion.div>
        )}

        {isEmpty && <EmptyState message="No manual data yet — activate the manuals scraper to begin" />}
      </div>
    </PageTransition>
  );
}
