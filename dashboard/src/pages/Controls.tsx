import { useState } from 'react';
import { PageTransition } from '@/components/shared/PageTransition';
import { StatusPill } from '@/components/shared/StatusPill';
import { motion } from 'framer-motion';
import { Play, Square, RefreshCw, Shield } from 'lucide-react';
import type { DashboardData } from '@/lib/types';
import { timeAgo } from '@/lib/format';

const isLocalhost = typeof window !== 'undefined' && (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1');

interface ServiceConfig {
  label: string;
  key: string;
  status: string;
  lastRun: string;
  action: string;
}

export function Controls({ data }: { data: DashboardData }) {
  const { metrics: m } = data;
  if (!m) return null;

  const services: ServiceConfig[] = [
    { label: 'NHTSA Scraper', key: 'nhtsa', status: m.scrapers.nhtsa.status, lastRun: m.scrapers.nhtsa.last_scrape, action: 'Trigger Crawl' },
    { label: 'iFixit Scraper', key: 'ifixit', status: m.scrapers.ifixit.status, lastRun: m.scrapers.ifixit.last_scrape, action: 'Trigger Crawl' },
    { label: 'Reddit Scraper', key: 'reddit', status: m.scrapers.reddit.status, lastRun: m.scrapers.reddit.last_scrape, action: 'Trigger Crawl' },
    { label: 'YouTube Scraper', key: 'youtube', status: m.scrapers.youtube.status, lastRun: m.scrapers.youtube.last_scrape, action: 'Trigger Crawl' },
    { label: 'Ingestion Engine', key: 'ingest', status: 'running', lastRun: m.ingestion.last_ingestion, action: 'Restart Ingest' },
    { label: 'Graph Enricher', key: 'enricher', status: m.knowledge_graph.total_relationships > 0 ? 'running' : 'error', lastRun: m.timestamp, action: 'Run Enricher' },
  ];

  return (
    <PageTransition>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-zinc-100">Controls</h1>
            <p className="text-sm text-zinc-500 mt-1">Service management panel</p>
          </div>
          {!isLocalhost && (
            <div className="flex items-center gap-2 text-amber-400 text-xs">
              <Shield className="h-4 w-4" />
              Controls disabled (not localhost)
            </div>
          )}
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
          {services.map((svc, i) => (
            <motion.div key={svc.key}
              initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.05 }}
              className="rounded-xl border border-white/5 bg-zinc-900/50 p-4">
              <div className="flex items-center justify-between mb-3">
                <span className="text-sm font-medium text-zinc-200">{svc.label}</span>
                <StatusPill status={svc.status} />
              </div>
              <p className="text-xs text-zinc-500 mb-4">Last run: {timeAgo(svc.lastRun)}</p>
              <div className="flex gap-2">
                <button disabled={!isLocalhost}
                  className="flex items-center gap-1.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20 px-3 py-1.5 text-xs font-medium text-emerald-400 hover:bg-emerald-500/20 disabled:opacity-30 disabled:cursor-not-allowed transition-colors">
                  <Play className="h-3 w-3" />{svc.action}
                </button>
                <button disabled={!isLocalhost}
                  className="flex items-center gap-1.5 rounded-lg bg-red-500/10 border border-red-500/20 px-3 py-1.5 text-xs font-medium text-red-400 hover:bg-red-500/20 disabled:opacity-30 disabled:cursor-not-allowed transition-colors">
                  <Square className="h-3 w-3" />Stop
                </button>
              </div>
            </motion.div>
          ))}
        </div>
      </div>
    </PageTransition>
  );
}
