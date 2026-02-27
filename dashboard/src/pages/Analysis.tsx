import { PageTransition } from '@/components/shared/PageTransition';
import { InsightCard } from '@/components/shared/InsightCard';
import { EmptyState } from '@/components/shared/EmptyState';
import { motion } from 'framer-motion';
import { Clock, CheckCircle, Lightbulb, Target } from 'lucide-react';
import type { DashboardData } from '@/lib/types';
import { timeAgo } from '@/lib/format';

const effortColors: Record<string, string> = { low: 'text-emerald-400 bg-emerald-500/10', med: 'text-amber-400 bg-amber-500/10', high: 'text-red-400 bg-red-500/10' };
const impactLabels: Record<string, string> = { low: 'Low Impact', med: 'Med Impact', high: 'High Impact' };

export function Analysis({ data }: { data: DashboardData }) {
  const { analysis: a } = data;
  if (!a) return <EmptyState message="No analysis data available" />;

  return (
    <PageTransition>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100">Analysis</h1>
          <div className="flex items-center gap-2 mt-1">
            <Clock className="h-3.5 w-3.5 text-zinc-500" />
            <p className="text-sm text-zinc-500">Updated {timeAgo(a.timestamp)}</p>
          </div>
        </div>

        {/* Critical */}
        {a.critical.length > 0 && (
          <div>
            <h2 className="text-xs font-medium text-red-400 uppercase tracking-wider mb-3">Critical ({a.critical.length})</h2>
            <div className="space-y-2">
              {a.critical.map((item, i) => (
                <InsightCard key={i} index={i} severity="critical" title={item.title} detail={item.detail} />
              ))}
            </div>
          </div>
        )}

        {/* Warnings */}
        {a.warnings.length > 0 && (
          <div>
            <h2 className="text-xs font-medium text-amber-400 uppercase tracking-wider mb-3">Warnings ({a.warnings.length})</h2>
            <div className="space-y-2">
              {a.warnings.map((item, i) => (
                <InsightCard key={i} index={i} severity="warning" title={item.title} detail={item.detail} />
              ))}
            </div>
          </div>
        )}

        {/* Healthy */}
        {a.healthy.length > 0 && (
          <div>
            <h2 className="text-xs font-medium text-emerald-400 uppercase tracking-wider mb-3">Healthy ({a.healthy.length})</h2>
            <div className="space-y-1.5">
              {a.healthy.map((item, i) => (
                <motion.div key={i} initial={{ opacity: 0, x: -8 }} animate={{ opacity: 1, x: 0 }} transition={{ delay: i * 0.03 }}
                  className="flex items-center gap-2 text-sm text-zinc-300">
                  <CheckCircle className="h-3.5 w-3.5 text-emerald-400 shrink-0" />
                  <span>{item}</span>
                </motion.div>
              ))}
            </div>
          </div>
        )}

        {/* Suggestions */}
        {a.suggestions.length > 0 && (
          <div>
            <h2 className="text-xs font-medium text-blue-400 uppercase tracking-wider mb-3 flex items-center gap-1.5">
              <Lightbulb className="h-3.5 w-3.5" /> Suggestions ({a.suggestions.length})
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {a.suggestions.map((s, i) => (
                <motion.div key={i} initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.05 }}
                  whileHover={{ y: -2 }}
                  className="rounded-xl border border-white/5 bg-zinc-900/50 p-4">
                  <p className="text-sm font-medium text-zinc-200 mb-2">{s.title}</p>
                  <p className="text-xs text-zinc-500 mb-3 line-clamp-2">{s.impact}</p>
                  <div className="flex gap-2">
                    <span className={`text-[10px] font-medium rounded px-2 py-0.5 ${effortColors[s.effort] ?? effortColors.med}`}>
                      {s.effort.toUpperCase()} effort
                    </span>
                  </div>
                </motion.div>
              ))}
            </div>
          </div>
        )}

        {/* Strategy */}
        {a.strategy.length > 0 && (
          <div>
            <h2 className="text-xs font-medium text-zinc-400 uppercase tracking-wider mb-3 flex items-center gap-1.5">
              <Target className="h-3.5 w-3.5" /> Strategy
            </h2>
            <div className="rounded-xl border border-white/5 bg-zinc-900/50 p-5 space-y-3">
              {a.strategy.map((s, i) => (
                <motion.div key={i} initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: i * 0.05 }}
                  className="flex gap-3">
                  <span className="text-xs font-bold text-zinc-600 mt-0.5 shrink-0 w-5 text-right">{i + 1}</span>
                  <p className="text-sm text-zinc-300">{s}</p>
                </motion.div>
              ))}
            </div>
          </div>
        )}
      </div>
    </PageTransition>
  );
}
