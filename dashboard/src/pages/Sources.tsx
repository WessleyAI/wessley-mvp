import { PageTransition } from '@/components/shared/PageTransition';
import { StatusPill } from '@/components/shared/StatusPill';
import { motion } from 'framer-motion';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';
import { Globe, MessageSquare, FileText, Video, BookOpen } from 'lucide-react';
import type { DashboardData } from '@/lib/types';
import { timeAgo } from '@/lib/format';

const sourceConfig = [
  { key: 'nhtsa' as const, label: 'NHTSA', icon: <Globe className="h-4 w-4" />, color: '#10b981', getCount: (s: DashboardData['metrics']) => s?.scrapers.nhtsa.total_docs ?? 0 },
  { key: 'ifixit' as const, label: 'iFixit', icon: <FileText className="h-4 w-4" />, color: '#3b82f6', getCount: (s: DashboardData['metrics']) => s?.scrapers.ifixit.total_docs ?? 0 },
  { key: 'reddit' as const, label: 'Reddit', icon: <MessageSquare className="h-4 w-4" />, color: '#f59e0b', getCount: (s: DashboardData['metrics']) => s?.scrapers.reddit.total_posts ?? 0 },
  { key: 'youtube' as const, label: 'YouTube', icon: <Video className="h-4 w-4" />, color: '#ef4444', getCount: (s: DashboardData['metrics']) => s?.scrapers.youtube.total_docs ?? 0 },
];

export function Sources({ data }: { data: DashboardData }) {
  const { metrics: m } = data;
  if (!m) return null;

  const chartData = sourceConfig.map(s => ({ name: s.label, docs: s.getCount(m), fill: s.color }));
  chartData.push({ name: 'Manuals', docs: m.scrapers.manuals.ingested, fill: '#8b5cf6' });

  return (
    <PageTransition>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100">Sources</h1>
          <p className="text-sm text-zinc-500 mt-1">Data source breakdown and health</p>
        </div>

        {/* Proportional chart */}
        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.1 }}
          className="rounded-xl border border-white/5 bg-zinc-900/50 p-5">
          <h3 className="text-xs text-zinc-500 uppercase tracking-wider mb-4">Document Distribution</h3>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={chartData} margin={{ left: 10 }}>
              <XAxis dataKey="name" tick={{ fill: '#a1a1aa', fontSize: 12 }} axisLine={false} tickLine={false} />
              <YAxis tick={{ fill: '#6b7280', fontSize: 11 }} axisLine={false} tickLine={false} />
              <Tooltip contentStyle={{ background: '#18181b', border: '1px solid rgba(255,255,255,0.1)', borderRadius: '8px', fontSize: '12px', color: '#a1a1aa' }} />
              <Bar dataKey="docs" radius={[4, 4, 0, 0]} barSize={40}>
                {chartData.map((entry, i) => (
                  <Cell key={i} fill={entry.fill} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </motion.div>

        {/* Source cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
          {sourceConfig.map((src, i) => {
            const scraper = m.scrapers[src.key];
            const count = src.getCount(m);
            return (
              <motion.div key={src.key}
                initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.05 }}
                whileHover={{ y: -2 }}
                className="rounded-xl border border-white/5 bg-zinc-900/50 p-4">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <span style={{ color: src.color }}>{src.icon}</span>
                    <span className="text-sm font-medium text-zinc-200">{src.label}</span>
                  </div>
                  <StatusPill status={scraper.status} />
                </div>
                <p className="text-2xl font-bold text-zinc-100">{count.toLocaleString()}</p>
                <p className="text-xs text-zinc-500 mt-1">Last scrape: {timeAgo(scraper.last_scrape)}</p>
                {count === 0 && scraper.status === 'running' && (
                  <p className="text-xs text-amber-400 mt-2">⚠ Running but producing zero results</p>
                )}
              </motion.div>
            );
          })}
        </div>

        {/* Manuals card */}
        <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.3 }}
          className="rounded-xl border border-white/5 bg-zinc-900/50 p-4">
          <div className="flex items-center gap-2 mb-3">
            <BookOpen className="h-4 w-4 text-purple-400" />
            <span className="text-sm font-medium text-zinc-200">Manuals</span>
            <StatusPill status={m.scrapers.manuals.discovered > 0 ? 'running' : 'stopped'} />
          </div>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {(['discovered', 'downloaded', 'ingested', 'failed'] as const).map(k => (
              <div key={k}>
                <p className="text-xs text-zinc-500 capitalize">{k}</p>
                <p className={`text-xl font-bold ${k === 'failed' && m.scrapers.manuals[k] > 0 ? 'text-red-400' : 'text-zinc-100'}`}>
                  {m.scrapers.manuals[k].toLocaleString()}
                </p>
              </div>
            ))}
          </div>
          {m.scrapers.manuals.discovered === 0 && (
            <p className="text-xs text-amber-400 mt-3">⚠ Manuals pipeline completely inactive — highest-value untapped source</p>
          )}
        </motion.div>
      </div>
    </PageTransition>
  );
}

// Need Cell import
import { Cell } from 'recharts';
