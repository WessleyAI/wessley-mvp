import { useState, useMemo } from 'react';
import { PageTransition } from '@/components/shared/PageTransition';
import { EmptyState } from '@/components/shared/EmptyState';
import { motion } from 'framer-motion';
import { Search } from 'lucide-react';
import type { DashboardData } from '@/lib/types';
import { timeAgo } from '@/lib/format';

const categories = ['All', 'ingestion', 'sources', 'errors', 'system'] as const;
const catColors: Record<string, string> = {
  ingestion: 'text-blue-400 bg-blue-500/10',
  sources: 'text-purple-400 bg-purple-500/10',
  errors: 'text-red-400 bg-red-500/10',
  system: 'text-zinc-400 bg-zinc-500/10',
};

export function Logs({ data }: { data: DashboardData }) {
  const { logs } = data;
  const [filter, setFilter] = useState<string>('All');
  const [search, setSearch] = useState('');

  const filtered = useMemo(() => {
    return logs.filter(l => {
      if (filter !== 'All' && l.category !== filter) return false;
      if (search && !l.message.toLowerCase().includes(search.toLowerCase())) return false;
      return true;
    });
  }, [logs, filter, search]);

  return (
    <PageTransition>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100">Logs</h1>
          <p className="text-sm text-zinc-500 mt-1">{logs.length} log entries</p>
        </div>

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-zinc-500" />
          <input type="text" placeholder="Search logs..." value={search} onChange={e => setSearch(e.target.value)}
            className="w-full rounded-lg bg-zinc-900/50 border border-white/5 pl-9 pr-3 py-2 text-sm text-zinc-200 placeholder-zinc-600 focus:outline-none focus:border-zinc-600" />
        </div>

        {/* Filter chips */}
        <div className="flex gap-2 flex-wrap">
          {categories.map(cat => (
            <button key={cat} onClick={() => setFilter(cat)}
              className={`rounded-full px-3 py-1 text-xs font-medium transition-colors ${filter === cat ? 'bg-white/10 text-zinc-100' : 'bg-zinc-900/50 text-zinc-500 hover:text-zinc-300'}`}>
              {cat === 'All' ? 'All' : cat.charAt(0).toUpperCase() + cat.slice(1)}
            </button>
          ))}
        </div>

        {/* Log entries */}
        {filtered.length === 0 ? (
          <EmptyState message="No log entries match your filters" />
        ) : (
          <div className="rounded-xl border border-white/5 bg-zinc-900/50 divide-y divide-white/5 overflow-hidden">
            {filtered.slice(0, 200).map((log, i) => (
              <motion.div key={i} initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: Math.min(i * 0.01, 0.5) }}
                className="px-4 py-2.5 flex items-start gap-3 hover:bg-white/[0.02]">
                <span className="text-[10px] text-zinc-600 font-mono shrink-0 mt-0.5 w-16">{timeAgo(log.time)}</span>
                <span className={`text-[10px] font-medium rounded px-1.5 py-0.5 shrink-0 ${catColors[log.category] ?? catColors.system}`}>
                  {log.category}
                </span>
                <span className="text-xs text-zinc-300 break-all">{log.message}</span>
              </motion.div>
            ))}
          </div>
        )}
      </div>
    </PageTransition>
  );
}
