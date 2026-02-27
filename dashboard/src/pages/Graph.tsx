import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts';
import { AlertTriangle } from 'lucide-react';
import type { DashboardData } from '@/lib/types';

const GRID = 'rgba(255,255,255,0.05)';
const TICK = '#6b7280';

export function Graph({ data }: { data: DashboardData }) {
  const { metrics: m, history } = data;
  if (!m) return null;

  const kg = m.knowledge_graph;
  const nodeTypeData = Object.entries(kg.nodes_by_type).map(([name, count]) => ({ name, count }));
  const growthData = history.slice(-20).map(h => ({
    ts: new Date(h.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    nodes: h.total_nodes,
    rels: h.new_relations,
  }));

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold">Knowledge Graph</h2>
      {kg.total_relationships === 0 && (
        <div className="flex items-center gap-2 p-3 rounded-md bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
          <AlertTriangle className="h-4 w-4" />
          <span>CRITICAL: Graph has 0 relationships — completely flat. Enricher needs debugging.</span>
        </div>
      )}
      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="topology">Topology</TabsTrigger>
          <TabsTrigger value="growth">Growth</TabsTrigger>
        </TabsList>
        <TabsContent value="overview" className="space-y-4 mt-4">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <Card><CardContent className="p-4 text-center"><div className="text-2xl font-bold">{kg.total_nodes.toLocaleString()}</div><div className="text-xs text-muted-foreground">Nodes</div></CardContent></Card>
            <Card><CardContent className="p-4 text-center"><div className="text-2xl font-bold">{kg.total_relationships}</div><div className="text-xs text-muted-foreground">Relationships</div></CardContent></Card>
            <Card><CardContent className="p-4 text-center"><div className="text-2xl font-bold">{Object.keys(kg.nodes_by_type).length}</div><div className="text-xs text-muted-foreground">Node Types</div></CardContent></Card>
            <Card><CardContent className="p-4 text-center"><div className="text-2xl font-bold">{kg.top_makes.length}</div><div className="text-xs text-muted-foreground">Makes</div></CardContent></Card>
          </div>
          {nodeTypeData.length > 0 && (
            <Card>
              <CardHeader className="pb-2"><CardTitle className="text-sm">Node Types</CardTitle></CardHeader>
              <CardContent>
                <div className="h-[220px]">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={nodeTypeData}>
                      <CartesianGrid strokeDasharray="3 3" stroke={GRID} />
                      <XAxis dataKey="name" tick={{ fill: TICK, fontSize: 12 }} />
                      <YAxis tick={{ fill: TICK, fontSize: 12 }} />
                      <Tooltip contentStyle={{ background: '#1f2937', border: 'none', borderRadius: 8, fontSize: 12 }} />
                      <Bar dataKey="count" fill="#6366f1" radius={[4, 4, 0, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              </CardContent>
            </Card>
          )}
          <Card>
            <CardHeader className="pb-2"><CardTitle className="text-sm">Top Makes</CardTitle></CardHeader>
            <CardContent>
              <Table>
                <TableHeader><TableRow><TableHead>Make</TableHead><TableHead className="text-right">Models</TableHead><TableHead className="text-right">Documents</TableHead></TableRow></TableHeader>
                <TableBody>
                  {kg.top_makes.map(mk => (
                    <TableRow key={mk.name}><TableCell>{mk.name}</TableCell><TableCell className="text-right">{mk.models}</TableCell><TableCell className="text-right">{mk.documents.toLocaleString()}</TableCell></TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="topology" className="mt-4">
          <Card>
            <CardHeader className="pb-2"><CardTitle className="text-sm">Expected Graph Topology</CardTitle></CardHeader>
            <CardContent>
              <div className="flex flex-col items-center gap-2 py-8 text-sm">
                {['Make', 'Model', 'Year', 'System', 'Component'].map((n, i) => (
                  <div key={n} className="flex flex-col items-center">
                    <div className="px-6 py-3 bg-accent rounded-lg font-medium">{n}</div>
                    {i < 4 && <div className="text-muted-foreground text-lg">↓</div>}
                  </div>
                ))}
              </div>
              {kg.total_relationships === 0 && <p className="text-center text-xs text-red-400">Currently flat — all nodes are Component with no edges.</p>}
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="growth" className="mt-4">
          <Card>
            <CardHeader className="pb-2"><CardTitle className="text-sm">Node Growth</CardTitle></CardHeader>
            <CardContent>
              <div className="h-[220px]">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={growthData}>
                    <CartesianGrid strokeDasharray="3 3" stroke={GRID} />
                    <XAxis dataKey="ts" tick={{ fill: TICK, fontSize: 10 }} />
                    <YAxis tick={{ fill: TICK, fontSize: 10 }} />
                    <Tooltip contentStyle={{ background: '#1f2937', border: 'none', borderRadius: 8, fontSize: 12 }} />
                    <Bar dataKey="nodes" fill="#10b981" radius={[4, 4, 0, 0]} />
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
