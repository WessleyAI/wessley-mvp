import { useState } from 'react';
import { PageTransition } from '@/components/shared/PageTransition';
import { EmptyState } from '@/components/shared/EmptyState';
import { motion, AnimatePresence } from 'framer-motion';
import { Clock, CheckCircle, XCircle, GitCommit, ChevronDown, ChevronUp } from 'lucide-react';
import type { DashboardData } from '@/lib/types';
import { timeAgo } from '@/lib/format';

export function Fixes({ data }: { data: DashboardData }) {
  const { fixes } = data;
  const [expanded, setExpanded] = useState<number | null>(null);

  if (fixes.length === 0) return <PageTransition><EmptyState message="No fix runs recorded yet" /></PageTransition>;

  return (
    <PageTransition>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100">Fix History</h1>
          <p className="text-sm text-zinc-500 mt-1">{fixes.length} fix runs recorded</p>
        </div>

        {/* Timeline */}
        <div className="relative">
          <div className="absolute left-4 top-0 bottom-0 w-px bg-zinc-800" />
          <div className="space-y-4">
            {fixes.map((run, i) => (
              <motion.div key={i} initial={{ opacity: 0, x: -12 }} animate={{ opacity: 1, x: 0 }} transition={{ delay: i * 0.05 }}
                className="relative pl-10">
                {/* Timeline dot */}
                <div className={`absolute left-2.5 top-4 h-3 w-3 rounded-full border-2 ${run.build_status === 'pass' ? 'border-emerald-400 bg-emerald-400/20' : 'border-red-400 bg-red-400/20'}`} />

                <div className="rounded-xl border border-white/5 bg-zinc-900/50 p-4 cursor-pointer"
                  onClick={() => setExpanded(expanded === i ? null : i)}>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Clock className="h-3.5 w-3.5 text-zinc-500" />
                      <span className="text-xs text-zinc-500">{timeAgo(run.timestamp)}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-medium text-emerald-400 bg-emerald-500/10 rounded px-2 py-0.5">
                        {run.fixed.length} fixed
                      </span>
                      {run.skipped.length > 0 && (
                        <span className="text-xs font-medium text-zinc-500 bg-zinc-500/10 rounded px-2 py-0.5">
                          {run.skipped.length} skipped
                        </span>
                      )}
                      {expanded === i ? <ChevronUp className="h-4 w-4 text-zinc-500" /> : <ChevronDown className="h-4 w-4 text-zinc-500" />}
                    </div>
                  </div>
                  <p className="text-sm text-zinc-200">{run.summary}</p>

                  <AnimatePresence>
                    {expanded === i && (
                      <motion.div initial={{ height: 0, opacity: 0 }} animate={{ height: 'auto', opacity: 1 }} exit={{ height: 0, opacity: 0 }}
                        className="overflow-hidden mt-4 pt-4 border-t border-white/5 space-y-4">
                        {/* Fixed items */}
                        {run.fixed.length > 0 && (
                          <div>
                            <h4 className="text-xs text-emerald-400 uppercase tracking-wider mb-2">Fixed</h4>
                            <div className="space-y-1.5">
                              {run.fixed.map((f, j) => (
                                <div key={j} className="flex items-start gap-2 text-xs">
                                  <CheckCircle className="h-3.5 w-3.5 text-emerald-400 mt-0.5 shrink-0" />
                                  <div>
                                    <span className="text-zinc-300">{f.title}</span>
                                    <span className="text-zinc-600 ml-2">[{f.category}]</span>
                                  </div>
                                </div>
                              ))}
                            </div>
                          </div>
                        )}

                        {/* Skipped items */}
                        {run.skipped.length > 0 && (
                          <div>
                            <h4 className="text-xs text-zinc-500 uppercase tracking-wider mb-2">Skipped</h4>
                            <div className="space-y-1.5">
                              {run.skipped.map((s, j) => (
                                <div key={j} className="flex items-start gap-2 text-xs">
                                  <XCircle className="h-3.5 w-3.5 text-zinc-600 mt-0.5 shrink-0" />
                                  <div>
                                    <span className="text-zinc-400">{s.title}</span>
                                    <span className="text-zinc-600 ml-2">— {s.reason}</span>
                                  </div>
                                </div>
                              ))}
                            </div>
                          </div>
                        )}

                        {/* Test comparison + build */}
                        <div className="flex flex-wrap gap-4 text-xs">
                          <div>
                            <span className="text-zinc-500">Tests before: </span>
                            <span className="text-emerald-400">{run.tests_before.passed}✓</span>
                            <span className="text-red-400 ml-1">{run.tests_before.failed}✗</span>
                          </div>
                          <div>
                            <span className="text-zinc-500">Tests after: </span>
                            <span className="text-emerald-400">{run.tests_after.passed}✓</span>
                            <span className="text-red-400 ml-1">{run.tests_after.failed}✗</span>
                          </div>
                          <div className="flex items-center gap-1">
                            <span className={`h-2 w-2 rounded-full ${run.build_status === 'pass' ? 'bg-emerald-400' : 'bg-red-400'}`} />
                            <span className="text-zinc-400">Build: {run.build_status}</span>
                          </div>
                          {run.commit && (
                            <div className="flex items-center gap-1">
                              <GitCommit className="h-3 w-3 text-zinc-600" />
                              <span className="text-zinc-500 font-mono">{run.commit.slice(0, 8)}</span>
                            </div>
                          )}
                        </div>
                      </motion.div>
                    )}
                  </AnimatePresence>
                </div>
              </motion.div>
            ))}
          </div>
        </div>
      </div>
    </PageTransition>
  );
}
