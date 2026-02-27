import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, BarChart, Bar } from 'recharts';
import type { DashboardData } from '@/lib/types';
import { formatCompact, formatPercent } from '@/lib/format';

const GRID = 'rgba(255,255,255,0.05)';
const TICK = '#6b7280';

export function Pipeline({ data }: { data: DashboardData }) {
  const { metrics: m, history } = data;
  if (!m) return null;

  const throughputData = history.slice(-20).map(h => ({
    ts: new Date(h.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    docs: h.new_docs,
    vectors: h.new_vectors,
  }));

  const errorData = history.slice(-20).map(h => ({
    ts: new Date(h.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    errors: h.errors_delta,
  }));

  const embRatio = m.vector_store.total_vectors / Math.max(1, m.ingestion.total_docs_ingested);

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold">Pipeline</h2>
      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="throughput">Throughput</TabsTrigger>
          <TabsTrigger value="errors">Errors</TabsTrigger>
        </TabsList>
        <TabsContent value="overview" className="space-y-4 mt-4">
          {/* Pipeline flow diagram */}
          <Card>
            <CardHeader className="pb-2"><CardTitle className="text-sm">Pipeline Flow</CardTitle></CardHeader>
            <CardContent>
              <div className="flex flex-wrap items-center gap-2 text-xs">
                {['Scrapers', 'Ingest', 'Transform', 'Embed (Ollama)', 'Qdrant', 'Enricher', 'Neo4j'].map((step, i) => (
                  <div key={step} className="flex items-center gap-2">
                    <div className="px-3 py-2 bg-accent rounded-md font-medium">{step}</div>
                    {i < 6 && <span className="text-muted-foreground">→</span>}
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <Card><CardContent className="p-4 text-center"><div className="text-2xl font-bold">{formatCompact(m.ingestion.total_docs_ingested)}</div><div className="text-xs text-muted-foreground">Total Docs</div></CardContent></Card>
            <Card><CardContent className="p-4 text-center"><div className="text-2xl font-bold">{formatCompact(m.vector_store.total_vectors)}</div><div className="text-xs text-muted-foreground">Vectors</div></CardContent></Card>
            <Card><CardContent className="p-4 text-center"><div className="text-2xl font-bold">{formatPercent(embRatio)}</div><div className="text-xs text-muted-foreground">Embed Ratio</div></CardContent></Card>
            <Card><CardContent className="p-4 text-center"><div className="text-2xl font-bold">{m.ingestion.total_errors}</div><div className="text-xs text-muted-foreground">Errors</div></CardContent></Card>
          </div>
        </TabsContent>
        <TabsContent value="throughput" className="mt-4">
          <Card>
            <CardHeader className="pb-2"><CardTitle className="text-sm">Docs & Vectors over Time</CardTitle></CardHeader>
            <CardContent>
              <div className="h-[220px]">
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={throughputData}>
                    <CartesianGrid strokeDasharray="3 3" stroke={GRID} />
                    <XAxis dataKey="ts" tick={{ fill: TICK, fontSize: 10 }} />
                    <YAxis tick={{ fill: TICK, fontSize: 10 }} />
                    <Tooltip contentStyle={{ background: '#1f2937', border: 'none', borderRadius: 8, fontSize: 12 }} />
                    <Line type="monotone" dataKey="docs" stroke="#6366f1" strokeWidth={2} dot={false} />
                    <Line type="monotone" dataKey="vectors" stroke="#10b981" strokeWidth={2} dot={false} />
                  </LineChart>
                </ResponsiveContainer>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="errors" className="mt-4">
          <Card>
            <CardHeader className="pb-2"><CardTitle className="text-sm">Errors over Time</CardTitle></CardHeader>
            <CardContent>
              <div className="h-[220px]">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={errorData}>
                    <CartesianGrid strokeDasharray="3 3" stroke={GRID} />
                    <XAxis dataKey="ts" tick={{ fill: TICK, fontSize: 10 }} />
                    <YAxis tick={{ fill: TICK, fontSize: 10 }} />
                    <Tooltip contentStyle={{ background: '#1f2937', border: 'none', borderRadius: 8, fontSize: 12 }} />
                    <Bar dataKey="errors" fill="#ef4444" radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
