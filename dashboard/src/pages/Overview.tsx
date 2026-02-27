import { Database, Cpu, FileText, Layers } from 'lucide-react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { KPICard } from '@/components/shared/KPICard';
import type { DashboardData } from '@/lib/types';
import { formatCompact, formatPercent } from '@/lib/format';
import { generateInsights } from '@/lib/insights';

const CHART_GRID = 'rgba(255,255,255,0.05)';
const CHART_TICK = '#6b7280';

export function Overview({ data }: { data: DashboardData }) {
  const { metrics: m, analysis } = data;
  if (!m) return null;

  const insights = generateInsights(m);
  const embRatio = m.vector_store.total_vectors / Math.max(1, m.ingestion.total_docs_ingested);
  const errRate = m.ingestion.total_errors / Math.max(1, m.ingestion.total_docs_ingested);

  const healthItems = [
    { label: 'Neo4j', ok: m.infrastructure.neo4j.status === 'connected' },
    { label: 'Qdrant', ok: m.infrastructure.qdrant.status === 'connected' },
    { label: 'Ollama', ok: m.infrastructure.ollama.status === 'connected' },
    { label: 'Relationships', ok: m.knowledge_graph.total_relationships > 0 },
  ];

  const sourceData = Object.entries(m.scrapers)
    .filter(([k]) => k !== 'manuals')
    .map(([k, v]) => ({
      name: k,
      docs: 'total_docs' in v ? (v.total_docs ?? 0) : ('total_posts' in v ? (v.total_posts ?? 0) : 0),
    }));

  const makeData = m.knowledge_graph.top_makes.slice(0, 8).map(mk => ({ name: mk.name, docs: mk.documents }));

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold mb-1">Dashboard Overview</h2>
        <p className="text-sm text-muted-foreground">
          {formatCompact(m.ingestion.total_docs_ingested)} docs ingested across {Object.keys(m.scrapers).length - 1} sources •
          {' '}{formatCompact(m.knowledge_graph.total_nodes)} graph nodes •
          {' '}{formatCompact(m.vector_store.total_vectors)} vectors
        </p>
      </div>

      <div className="flex flex-wrap gap-2">
        {healthItems.map(h => (
          <Badge key={h.label} variant={h.ok ? 'default' : 'destructive'}>{h.label}: {h.ok ? '✓' : '✗'}</Badge>
        ))}
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
        <KPICard title="Documents" value={formatCompact(m.ingestion.total_docs_ingested)} icon={FileText} subtitle={`${m.ingestion.total_errors} errors`} trend={formatPercent(1 - errRate) + ' success'} trendUp />
        <KPICard title="Graph Nodes" value={formatCompact(m.knowledge_graph.total_nodes)} icon={Share2Icon} subtitle={`${m.knowledge_graph.total_relationships} relationships`} trend={m.knowledge_graph.total_relationships === 0 ? '⚠ Flat' : undefined} trendUp={false} />
        <KPICard title="Vectors" value={formatCompact(m.vector_store.total_vectors)} icon={Database} subtitle={m.vector_store.collection} trend={formatPercent(embRatio) + ' embedded'} trendUp={embRatio > 0.5} />
        <KPICard title="Makes Tracked" value={String(m.knowledge_graph.top_makes.length)} icon={Layers} subtitle={`${m.knowledge_graph.top_vehicles.length} top vehicles`} />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <Card className="lg:col-span-1">
          <CardHeader className="pb-2"><CardTitle className="text-sm">Insights</CardTitle></CardHeader>
          <CardContent className="space-y-2">
            {insights.map((ins, i) => (
              <div key={i} className={`text-xs p-2 rounded ${ins.level === 'critical' ? 'bg-red-500/10 text-red-400' : ins.level === 'warning' ? 'bg-yellow-500/10 text-yellow-400' : 'bg-emerald-500/10 text-emerald-400'}`}>
                {ins.message}
              </div>
            ))}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm">Sources</CardTitle></CardHeader>
          <CardContent>
            <div className="h-[220px]">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={sourceData}>
                  <CartesianGrid strokeDasharray="3 3" stroke={CHART_GRID} />
                  <XAxis dataKey="name" tick={{ fill: CHART_TICK, fontSize: 12 }} />
                  <YAxis tick={{ fill: CHART_TICK, fontSize: 12 }} />
                  <Tooltip contentStyle={{ background: '#1f2937', border: 'none', borderRadius: 8, fontSize: 12 }} />
                  <Bar dataKey="docs" fill="#6366f1" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm">Top Makes</CardTitle></CardHeader>
          <CardContent>
            <div className="h-[220px]">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={makeData} layout="vertical">
                  <CartesianGrid strokeDasharray="3 3" stroke={CHART_GRID} />
                  <XAxis type="number" tick={{ fill: CHART_TICK, fontSize: 12 }} />
                  <YAxis dataKey="name" type="category" width={70} tick={{ fill: CHART_TICK, fontSize: 11 }} />
                  <Tooltip contentStyle={{ background: '#1f2937', border: 'none', borderRadius: 8, fontSize: 12 }} />
                  <Bar dataKey="docs" fill="#10b981" radius={[0, 4, 4, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function Share2Icon(props: React.SVGProps<SVGSVGElement>) {
  return <Cpu {...props} />;
}
