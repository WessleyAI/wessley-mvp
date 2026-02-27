import { PageTransition } from '@/components/shared/PageTransition';
import { InsightCard } from '@/components/shared/InsightCard';
import { AnimatedCounter } from '@/components/shared/AnimatedCounter';
import { motion } from 'framer-motion';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';
import { Network, AlertCircle } from 'lucide-react';
import type { DashboardData } from '@/lib/types';

export function Graph({ data }: { data: DashboardData }) {
  const { metrics: m } = data;
  if (!m) return null;

  const kg = m.knowledge_graph;
  const nodeTypes = Object.entries(kg.nodes_by_type).map(([name, count]) => ({ name, count })).sort((a, b) => b.count - a.count);
  const completeness = kg.total_relationships > 0 ? Math.min(100, Math.round((kg.total_relationships / Math.max(1, kg.total_nodes)) * 100)) : 0;

  return (
    <PageTransition>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100">Knowledge Graph</h1>
          <p className="text-sm text-zinc-500 mt-1">Graph structure and entity coverage</p>
        </div>

        {/* Critical alert for zero relationships */}
        {kg.total_relationships === 0 && (
          <InsightCard severity="critical" title="Graph has ZERO relationships — completely flat"
            detail="All nodes exist but no edges connect them. The enricher needs debugging. Structured queries are impossible." />
        )}

        {/* KPI row */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          {[
            { label: 'Total Nodes', value: kg.total_nodes, color: 'text-zinc-100' },
            { label: 'Relationships', value: kg.total_relationships, color: kg.total_relationships === 0 ? 'text-red-400' : 'text-zinc-100' },
            { label: 'Node Types', value: Object.keys(kg.nodes_by_type).length, color: 'text-zinc-100' },
            { label: 'Completeness', value: completeness, color: completeness < 30 ? 'text-red-400' : 'text-zinc-100' },
          ].map((kpi, i) => (
            <motion.div key={kpi.label} initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.05 }}
              className="rounded-xl border border-white/5 bg-zinc-900/50 p-4">
              <p className="text-xs text-zinc-500 uppercase tracking-wider">{kpi.label}</p>
              <p className={`text-2xl font-bold mt-1 ${kpi.color}`}>
                <AnimatedCounter value={kpi.value} suffix={kpi.label === 'Completeness' ? '%' : ''} />
              </p>
            </motion.div>
          ))}
        </div>

        {/* Node type breakdown */}
        {nodeTypes.length > 0 && (
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.2 }}
            className="rounded-xl border border-white/5 bg-zinc-900/50 p-5">
            <h3 className="text-xs text-zinc-500 uppercase tracking-wider mb-4">Nodes by Type</h3>
            <ResponsiveContainer width="100%" height={Math.max(100, nodeTypes.length * 40)}>
              <BarChart data={nodeTypes} layout="vertical" margin={{ left: 80 }}>
                <XAxis type="number" tick={{ fill: '#6b7280', fontSize: 11 }} axisLine={false} tickLine={false} />
                <YAxis type="category" dataKey="name" tick={{ fill: '#a1a1aa', fontSize: 12 }} axisLine={false} tickLine={false} width={75} />
                <Tooltip contentStyle={{ background: '#18181b', border: '1px solid rgba(255,255,255,0.1)', borderRadius: '8px', fontSize: '12px', color: '#a1a1aa' }} />
                <Bar dataKey="count" fill="#3b82f6" radius={[0, 4, 4, 0]} barSize={16} />
              </BarChart>
            </ResponsiveContainer>
          </motion.div>
        )}

        {/* Top Makes */}
        <div>
          <h3 className="text-xs text-zinc-500 uppercase tracking-wider mb-3">Top Makes</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            {kg.top_makes.map((make, i) => (
              <motion.div key={make.name} initial={{ opacity: 0, scale: 0.95 }} animate={{ opacity: 1, scale: 1 }} transition={{ delay: i * 0.03 }}
                whileHover={{ y: -2 }}
                className="rounded-xl border border-white/5 bg-zinc-900/50 p-4">
                <p className="text-sm font-medium text-zinc-200">{make.name}</p>
                <p className="text-xl font-bold text-zinc-100 mt-1">{make.documents}</p>
                <p className="text-xs text-zinc-500">{make.models} models</p>
              </motion.div>
            ))}
          </div>
        </div>

        {/* Top Vehicles */}
        <div>
          <h3 className="text-xs text-zinc-500 uppercase tracking-wider mb-3">Top Vehicles</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {kg.top_vehicles.map((v, i) => (
              <motion.div key={v.vehicle} initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.03 }}
                className="rounded-xl border border-white/5 bg-zinc-900/50 p-4 flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-zinc-200">{v.vehicle}</p>
                  <p className="text-xs text-zinc-500">{v.components} components</p>
                </div>
                <span className="text-lg font-bold text-zinc-100">{v.documents}</span>
              </motion.div>
            ))}
          </div>
        </div>
      </div>
    </PageTransition>
  );
}
