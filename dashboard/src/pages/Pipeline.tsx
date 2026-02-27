import { PageTransition } from '@/components/shared/PageTransition';
import { StatusPill } from '@/components/shared/StatusPill';
import { AnimatedCounter } from '@/components/shared/AnimatedCounter';
import { motion } from 'framer-motion';
import { ArrowRight, Database, FileSearch, FileInput, Cpu, Network } from 'lucide-react';
import type { DashboardData } from '@/lib/types';

interface Stage {
  label: string;
  icon: React.ReactNode;
  count: number;
  status: string;
  detail: string;
}

export function Pipeline({ data }: { data: DashboardData }) {
  const { metrics: m, analysis } = data;
  if (!m) return null;

  const totalScraper = (m.scrapers.nhtsa.total_docs ?? 0) + (m.scrapers.ifixit.total_docs ?? 0) +
    (m.scrapers.reddit.total_posts ?? 0) + (m.scrapers.youtube.total_docs ?? 0) + m.scrapers.manuals.ingested;

  const stages: Stage[] = [
    { label: 'Sources', icon: <Database className="h-5 w-5" />, count: analysis?.metrics.sources_active ?? 0, status: 'running', detail: `${analysis?.metrics.sources_active ?? 0} active, ${analysis?.metrics.sources_dead ?? 0} dead` },
    { label: 'Scraping', icon: <FileSearch className="h-5 w-5" />, count: totalScraper, status: 'running', detail: `${totalScraper.toLocaleString()} raw docs` },
    { label: 'Ingestion', icon: <FileInput className="h-5 w-5" />, count: m.ingestion.total_docs_ingested, status: m.ingestion.total_errors > 50 ? 'warning' : 'running', detail: `${m.ingestion.total_errors} errors` },
    { label: 'Embedding', icon: <Cpu className="h-5 w-5" />, count: m.vector_store.total_vectors, status: m.vector_store.total_vectors < m.ingestion.total_docs_ingested * 0.5 ? 'warning' : 'running', detail: `${((m.vector_store.total_vectors / Math.max(1, m.ingestion.total_docs_ingested)) * 100).toFixed(1)}% coverage` },
    { label: 'Graph', icon: <Network className="h-5 w-5" />, count: m.knowledge_graph.total_nodes, status: m.knowledge_graph.total_relationships === 0 ? 'error' : 'running', detail: `${m.knowledge_graph.total_relationships} relationships` },
  ];

  return (
    <PageTransition>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100">Pipeline</h1>
          <p className="text-sm text-zinc-500 mt-1">Data flow from sources to knowledge graph</p>
        </div>

        {/* Flow diagram */}
        <div className="flex flex-col lg:flex-row items-stretch gap-0 overflow-x-auto pb-2">
          {stages.map((stage, i) => (
            <div key={stage.label} className="flex items-center">
              <motion.div
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ delay: i * 0.1 }}
                className="rounded-xl border border-white/5 bg-zinc-900/50 p-5 min-w-[160px] flex-1"
              >
                <div className="flex items-center gap-2 mb-3">
                  <span className="text-zinc-500">{stage.icon}</span>
                  <span className="text-sm font-medium text-zinc-300">{stage.label}</span>
                </div>
                <div className="text-2xl font-bold text-zinc-100 mb-2">
                  <AnimatedCounter value={stage.count} />
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-xs text-zinc-500">{stage.detail}</span>
                  <StatusPill status={stage.status} />
                </div>
              </motion.div>
              {i < stages.length - 1 && (
                <motion.div
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: i * 0.1 + 0.2 }}
                  className="hidden lg:flex items-center px-2"
                >
                  <ArrowRight className="h-5 w-5 text-zinc-700" />
                </motion.div>
              )}
            </div>
          ))}
        </div>

        {/* Stage details */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.5 }}
            className="rounded-xl border border-white/5 bg-zinc-900/50 p-4">
            <h3 className="text-xs text-zinc-500 uppercase tracking-wider mb-2">Embedding Coverage</h3>
            <div className="relative h-3 rounded-full bg-zinc-800 overflow-hidden">
              <motion.div
                initial={{ width: 0 }}
                animate={{ width: `${Math.min(100, (m.vector_store.total_vectors / Math.max(1, m.ingestion.total_docs_ingested)) * 100)}%` }}
                transition={{ duration: 1, delay: 0.6 }}
                className="absolute inset-y-0 left-0 rounded-full bg-gradient-to-r from-emerald-500 to-blue-500"
              />
            </div>
            <p className="text-xs text-zinc-500 mt-2">{m.vector_store.total_vectors.toLocaleString()} / {m.ingestion.total_docs_ingested.toLocaleString()} documents embedded</p>
          </motion.div>

          <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.6 }}
            className="rounded-xl border border-white/5 bg-zinc-900/50 p-4">
            <h3 className="text-xs text-zinc-500 uppercase tracking-wider mb-2">Error Rate</h3>
            <p className="text-2xl font-bold text-zinc-100">{((m.ingestion.total_errors / Math.max(1, m.ingestion.total_docs_ingested)) * 100).toFixed(2)}%</p>
            <p className="text-xs text-zinc-500 mt-1">{m.ingestion.total_errors} errors out of {m.ingestion.total_docs_ingested.toLocaleString()} docs</p>
          </motion.div>

          <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.7 }}
            className="rounded-xl border border-white/5 bg-zinc-900/50 p-4">
            <h3 className="text-xs text-zinc-500 uppercase tracking-wider mb-2">Graph Completeness</h3>
            {m.knowledge_graph.total_relationships === 0 ? (
              <p className="text-sm text-red-400 font-medium">⚠ No relationships — graph is flat</p>
            ) : (
              <p className="text-2xl font-bold text-zinc-100">{m.knowledge_graph.total_relationships.toLocaleString()}</p>
            )}
            <p className="text-xs text-zinc-500 mt-1">{m.knowledge_graph.total_nodes.toLocaleString()} nodes, {Object.keys(m.knowledge_graph.nodes_by_type).length} types</p>
          </motion.div>
        </div>
      </div>
    </PageTransition>
  );
}
