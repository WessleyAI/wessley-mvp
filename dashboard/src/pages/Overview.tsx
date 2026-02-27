import { PageTransition } from '@/components/shared/PageTransition';
import { KPICard } from '@/components/shared/KPICard';
import { HealthRing } from '@/components/shared/HealthRing';
import { StatusPill } from '@/components/shared/StatusPill';
import { InsightCard } from '@/components/shared/InsightCard';
import { motion } from 'framer-motion';
import { PieChart, Pie, Cell, ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip } from 'recharts';
import { Database, Layers, FileText, AlertTriangle, Radio } from 'lucide-react';
import type { DashboardData } from '@/lib/types';
import { timeAgo } from '@/lib/format';
import { generateInsights } from '@/lib/insights';

const SOURCE_COLORS = ['#10b981', '#3b82f6', '#f59e0b', '#8b5cf6', '#ef4444'];

export function Overview({ data }: { data: DashboardData }) {
  const { metrics: m, analysis, history } = data;
  if (!m) return null;

  const insights = generateInsights(m);
  const errRate = m.ingestion.total_errors / Math.max(1, m.ingestion.total_docs_ingested);
  const criticals = analysis?.critical.length ?? 0;
  const warnings = analysis?.warnings.length ?? 0;
  const healthScore = Math.max(0, 100 - criticals * 10 - warnings * 5);

  const scraperSources = [
    { name: 'NHTSA', value: m.scrapers.nhtsa.total_docs ?? 0 },
    { name: 'iFixit', value: m.scrapers.ifixit.total_docs ?? 0 },
    { name: 'Reddit', value: m.scrapers.reddit.total_posts ?? 0 },
    { name: 'YouTube', value: m.scrapers.youtube.total_docs ?? 0 },
    { name: 'Manuals', value: m.scrapers.manuals.ingested },
  ].filter(s => s.value > 0);

  // If no scraper data, use analysis metrics
  if (scraperSources.length === 0 && analysis) {
    scraperSources.push(
      { name: 'NHTSA', value: Math.round(analysis.metrics.total_docs * analysis.metrics.nhtsa_pct) },
      { name: 'Other', value: Math.round(analysis.metrics.total_docs * (1 - analysis.metrics.nhtsa_pct)) },
    );
  }

  const topMakes = m.knowledge_graph.top_makes.slice(0, 6);
  const docsHistory = history.map(h => h.total_docs);
  const vectorsHistory = history.map(h => h.total_vectors);
  const nodesHistory = history.map(h => h.total_nodes);

  const infra = [
    { label: 'Neo4j', status: m.infrastructure.neo4j.status },
    { label: 'Qdrant', status: m.infrastructure.qdrant.status },
    { label: 'Ollama', status: m.infrastructure.ollama.status },
  ];

  return (
    <PageTransition>
      <div className="space-y-6">
        {/* Header */}
        <div>
          <h1 className="text-2xl font-bold text-zinc-100">Dashboard</h1>
          <p className="text-sm text-zinc-500 mt-1">Last updated {timeAgo(m.timestamp)}</p>
        </div>

        {/* KPIs */}
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
          <KPICard title="Total Nodes" value={m.knowledge_graph.total_nodes} icon={<Database className="h-4 w-4" />} sparkData={nodesHistory} />
          <KPICard title="Vectors" value={m.vector_store.total_vectors} icon={<Layers className="h-4 w-4" />} sparkData={vectorsHistory} sparkColor="#3b82f6" />
          <KPICard title="Docs Ingested" value={m.ingestion.total_docs_ingested} icon={<FileText className="h-4 w-4" />} sparkData={docsHistory} sparkColor="#8b5cf6" />
          <KPICard title="Error Rate" value={errRate * 100} suffix="%" decimals={2} icon={<AlertTriangle className="h-4 w-4" />} sparkColor={errRate > 0.01 ? '#ef4444' : '#10b981'} />
          <KPICard title="Active Sources" value={analysis?.metrics.sources_active ?? 0} icon={<Radio className="h-4 w-4" />} />
        </div>

        {/* Health + Source Diversity + Infra */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* Health Ring */}
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.2 }}
            className="rounded-xl border border-white/5 bg-zinc-900/50 p-5 flex flex-col items-center justify-center">
            <HealthRing score={healthScore} size={140} />
            <p className="text-xs text-zinc-500 mt-3">{criticals} critical · {warnings} warnings</p>
          </motion.div>

          {/* Source Diversity */}
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.3 }}
            className="rounded-xl border border-white/5 bg-zinc-900/50 p-5">
            <h3 className="text-xs font-medium text-zinc-500 uppercase tracking-wider mb-3">Source Diversity</h3>
            {scraperSources.length > 0 ? (
              <ResponsiveContainer width="100%" height={160}>
                <PieChart>
                  <Pie data={scraperSources} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={40} outerRadius={65} paddingAngle={2}>
                    {scraperSources.map((_, i) => <Cell key={i} fill={SOURCE_COLORS[i % SOURCE_COLORS.length]} />)}
                  </Pie>
                  <Tooltip contentStyle={{ background: '#18181b', border: '1px solid rgba(255,255,255,0.1)', borderRadius: '8px', fontSize: '12px', color: '#a1a1aa' }} />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <p className="text-sm text-zinc-600 text-center py-10">No source data</p>
            )}
            <div className="flex flex-wrap gap-2 mt-2">
              {scraperSources.map((s, i) => (
                <span key={s.name} className="text-[10px] text-zinc-500 flex items-center gap-1">
                  <span className="h-2 w-2 rounded-full" style={{ background: SOURCE_COLORS[i % SOURCE_COLORS.length] }} />
                  {s.name}
                </span>
              ))}
            </div>
          </motion.div>

          {/* Infrastructure */}
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.4 }}
            className="rounded-xl border border-white/5 bg-zinc-900/50 p-5">
            <h3 className="text-xs font-medium text-zinc-500 uppercase tracking-wider mb-4">Infrastructure</h3>
            <div className="space-y-3">
              {infra.map(({ label, status }) => (
                <div key={label} className="flex items-center justify-between">
                  <span className="text-sm text-zinc-300">{label}</span>
                  <StatusPill status={status} />
                </div>
              ))}
              <div className="flex items-center justify-between pt-2 border-t border-white/5">
                <span className="text-sm text-zinc-300">Relationships</span>
                <StatusPill status={m.knowledge_graph.total_relationships > 0 ? 'connected' : 'error'} />
              </div>
            </div>
          </motion.div>
        </div>

        {/* Top Makes Chart */}
        {topMakes.length > 0 && (
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.5 }}
            className="rounded-xl border border-white/5 bg-zinc-900/50 p-5">
            <h3 className="text-xs font-medium text-zinc-500 uppercase tracking-wider mb-4">Top Makes by Documents</h3>
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={topMakes} layout="vertical" margin={{ left: 60 }}>
                <XAxis type="number" tick={{ fill: '#6b7280', fontSize: 11 }} axisLine={false} tickLine={false} />
                <YAxis type="category" dataKey="name" tick={{ fill: '#a1a1aa', fontSize: 12 }} axisLine={false} tickLine={false} width={55} />
                <Tooltip contentStyle={{ background: '#18181b', border: '1px solid rgba(255,255,255,0.1)', borderRadius: '8px', fontSize: '12px', color: '#a1a1aa' }} />
                <Bar dataKey="documents" fill="#10b981" radius={[0, 4, 4, 0]} barSize={16} />
              </BarChart>
            </ResponsiveContainer>
          </motion.div>
        )}

        {/* Insights */}
        {insights.length > 0 && (
          <div>
            <h3 className="text-xs font-medium text-zinc-500 uppercase tracking-wider mb-3">Insights</h3>
            <div className="space-y-2">
              {insights.map((ins, i) => (
                <InsightCard key={i} index={i} severity={ins.level === 'info' ? 'healthy' : ins.level} title={ins.message} />
              ))}
            </div>
          </div>
        )}
      </div>
    </PageTransition>
  );
}
