import { PageTransition } from '@/components/shared/PageTransition';
import { StatusPill } from '@/components/shared/StatusPill';
import { PulseIndicator } from '@/components/shared/PulseIndicator';
import { motion } from 'framer-motion';
import { Database, Layers, Cpu } from 'lucide-react';
import type { DashboardData } from '@/lib/types';

export function Infrastructure({ data }: { data: DashboardData }) {
  const { metrics: m } = data;
  if (!m) return null;

  const infra = m.infrastructure;
  const services = [
    {
      label: 'Neo4j', icon: <Database className="h-6 w-6" />, status: infra.neo4j.status,
      version: infra.neo4j.version ?? '—',
      metric: { label: 'Nodes', value: m.knowledge_graph.total_nodes.toLocaleString() },
      extra: { label: 'Relationships', value: m.knowledge_graph.total_relationships.toLocaleString() },
      color: 'from-blue-500/20 to-blue-600/5',
    },
    {
      label: 'Qdrant', icon: <Layers className="h-6 w-6" />, status: infra.qdrant.status,
      version: infra.qdrant.version ?? '—',
      metric: { label: 'Vectors', value: (infra.qdrant.vectors ?? m.vector_store.total_vectors).toLocaleString() },
      extra: { label: 'Dimensions', value: String(m.vector_store.dimensions) },
      color: 'from-purple-500/20 to-purple-600/5',
    },
    {
      label: 'Ollama', icon: <Cpu className="h-6 w-6" />, status: infra.ollama.status,
      version: infra.ollama.version ?? '—',
      metric: { label: 'Model', value: infra.ollama.model ?? '—' },
      extra: { label: 'Collection', value: m.vector_store.collection },
      color: 'from-emerald-500/20 to-emerald-600/5',
    },
  ];

  return (
    <PageTransition>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100">Infrastructure</h1>
          <p className="text-sm text-zinc-500 mt-1">Service health and connectivity</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {services.map((svc, i) => (
            <motion.div key={svc.label}
              initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.1 }}
              whileHover={{ y: -3 }}
              className={`rounded-xl border border-white/5 bg-gradient-to-b ${svc.color} p-6`}
            >
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-3">
                  <span className="text-zinc-400">{svc.icon}</span>
                  <span className="text-lg font-semibold text-zinc-100">{svc.label}</span>
                </div>
                <StatusPill status={svc.status} />
              </div>

              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-xs text-zinc-500">Version</span>
                  <span className="text-sm text-zinc-300 font-mono">{svc.version}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-xs text-zinc-500">{svc.metric.label}</span>
                  <span className="text-sm font-medium text-zinc-200">{svc.metric.value}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-xs text-zinc-500">{svc.extra.label}</span>
                  <span className="text-sm text-zinc-300">{svc.extra.value}</span>
                </div>
              </div>

              {svc.status === 'connected' && (
                <div className="flex items-center gap-2 mt-4 pt-3 border-t border-white/5">
                  <PulseIndicator />
                  <span className="text-xs text-zinc-500">Connected</span>
                </div>
              )}
            </motion.div>
          ))}
        </div>
      </div>
    </PageTransition>
  );
}
