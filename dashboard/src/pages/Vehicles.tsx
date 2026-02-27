import { useState, useMemo } from 'react';
import { PageTransition } from '@/components/shared/PageTransition';
import { EmptyState } from '@/components/shared/EmptyState';
import { motion, AnimatePresence } from 'framer-motion';
import { Search, ChevronDown, Car } from 'lucide-react';
import type { DashboardData } from '@/lib/types';

type SortKey = 'docs' | 'components' | 'name';

export function Vehicles({ data }: { data: DashboardData }) {
  const { metrics: m } = data;
  const [search, setSearch] = useState('');
  const [sort, setSort] = useState<SortKey>('docs');
  const [expanded, setExpanded] = useState<string | null>(null);

  const vehicles = useMemo(() => {
    if (!m) return [];
    let list = m.knowledge_graph.top_vehicles.filter(v =>
      v.vehicle.toLowerCase().includes(search.toLowerCase())
    );
    list.sort((a, b) => {
      if (sort === 'docs') return b.documents - a.documents;
      if (sort === 'components') return b.components - a.components;
      return a.vehicle.localeCompare(b.vehicle);
    });
    return list;
  }, [m, search, sort]);

  if (!m) return null;

  return (
    <PageTransition>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100">Vehicles</h1>
          <p className="text-sm text-zinc-500 mt-1">Vehicle explorer — {m.knowledge_graph.top_vehicles.length} vehicles tracked</p>
        </div>

        {/* Search + Sort */}
        <div className="flex flex-col sm:flex-row gap-3">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-zinc-500" />
            <input
              type="text" placeholder="Search vehicles..."
              value={search} onChange={e => setSearch(e.target.value)}
              className="w-full rounded-lg bg-zinc-900/50 border border-white/5 pl-9 pr-3 py-2 text-sm text-zinc-200 placeholder-zinc-600 focus:outline-none focus:border-zinc-600"
            />
          </div>
          <select value={sort} onChange={e => setSort(e.target.value as SortKey)}
            className="rounded-lg bg-zinc-900/50 border border-white/5 px-3 py-2 text-sm text-zinc-300 focus:outline-none">
            <option value="docs">Sort by Documents</option>
            <option value="components">Sort by Components</option>
            <option value="name">Sort by Name</option>
          </select>
        </div>

        {/* Vehicle grid */}
        {vehicles.length === 0 ? (
          <EmptyState message="No vehicles found" />
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {vehicles.map((v, i) => {
              const parts = v.vehicle.split(' ');
              const make = parts.length >= 3 ? parts[1] : parts[0];
              return (
                <motion.div key={v.vehicle}
                  initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: Math.min(i * 0.03, 0.3) }}
                  whileHover={{ y: -2 }}
                  className="rounded-xl border border-white/5 bg-zinc-900/50 p-4 cursor-pointer"
                  onClick={() => setExpanded(expanded === v.vehicle ? null : v.vehicle)}
                >
                  <div className="flex items-center gap-2 mb-2">
                    <Car className="h-4 w-4 text-zinc-600" />
                    <span className="text-xs font-medium text-emerald-400 bg-emerald-500/10 rounded px-1.5 py-0.5">{make}</span>
                  </div>
                  <p className="text-sm font-medium text-zinc-200">{v.vehicle}</p>
                  <div className="flex items-center gap-4 mt-2">
                    <span className="text-xs text-zinc-500">{v.documents} docs</span>
                    <span className="text-xs text-zinc-500">{v.components} components</span>
                  </div>
                  {/* Doc count bar */}
                  <div className="mt-2 h-1.5 rounded-full bg-zinc-800 overflow-hidden">
                    <div className="h-full rounded-full bg-emerald-500/60" style={{ width: `${Math.min(100, (v.documents / Math.max(1, vehicles[0]?.documents ?? 1)) * 100)}%` }} />
                  </div>
                  <AnimatePresence>
                    {expanded === v.vehicle && (
                      <motion.div initial={{ height: 0, opacity: 0 }} animate={{ height: 'auto', opacity: 1 }} exit={{ height: 0, opacity: 0 }}
                        className="overflow-hidden mt-3 pt-3 border-t border-white/5">
                        <p className="text-xs text-zinc-400">Component coverage: {v.components} components across documents</p>
                      </motion.div>
                    )}
                  </AnimatePresence>
                </motion.div>
              );
            })}
          </div>
        )}
      </div>
    </PageTransition>
  );
}
