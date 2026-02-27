import type { Metrics } from './types';

export interface Insight {
  level: 'critical' | 'warning' | 'info';
  message: string;
}

export function generateInsights(m: Metrics): Insight[] {
  const insights: Insight[] = [];
  if (m.knowledge_graph.total_relationships === 0) {
    insights.push({ level: 'critical', message: 'Knowledge graph has ZERO relationships — completely flat. Enricher needs debugging.' });
  }
  const ratio = m.vector_store.total_vectors / Math.max(1, m.ingestion.total_docs_ingested);
  if (ratio < 0.5) {
    insights.push({ level: 'critical', message: `Only ${(ratio * 100).toFixed(1)}% of documents are embedded. RAG quality severely degraded.` });
  }
  if (m.scrapers.manuals.discovered === 0) {
    insights.push({ level: 'warning', message: 'Manuals scraper inactive — 0 discovered. Highest-value untapped source.' });
  }
  if (m.scrapers.reddit.status === 'stopped' || (m.scrapers.reddit.total_posts ?? 0) === 0) {
    insights.push({ level: 'warning', message: 'Reddit scraper producing zero posts.' });
  }
  const sources = m.ingestion.docs_by_source;
  const total = Object.values(sources).reduce((a, b) => a + b, 0);
  if (total > 0) {
    const max = Math.max(...Object.values(sources));
    if (max / total > 0.9) {
      insights.push({ level: 'warning', message: 'Extreme source imbalance — one source dominates >90% of documents.' });
    }
  }
  if (m.knowledge_graph.recent_vehicles.length === 0) {
    insights.push({ level: 'warning', message: 'No new vehicles being discovered recently.' });
  }
  if (m.infrastructure.neo4j.status === 'connected' && m.infrastructure.qdrant.status === 'connected' && m.infrastructure.ollama.status === 'connected') {
    insights.push({ level: 'info', message: 'All infrastructure services connected and healthy.' });
  }
  if (m.ingestion.total_errors / Math.max(1, m.ingestion.total_docs_ingested) < 0.005) {
    insights.push({ level: 'info', message: `Error rate ${(m.ingestion.total_errors / Math.max(1, m.ingestion.total_docs_ingested) * 100).toFixed(2)}% — excellent.` });
  }
  return insights;
}
