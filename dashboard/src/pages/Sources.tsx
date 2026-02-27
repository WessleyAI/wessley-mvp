import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts';
import { StatusDot } from '@/components/shared/StatusDot';
import type { DashboardData } from '@/lib/types';
import { timeAgo } from '@/lib/format';

const GRID = 'rgba(255,255,255,0.05)';
const TICK = '#6b7280';

export function Sources({ data }: { data: DashboardData }) {
  const { metrics: m } = data;
  if (!m) return null;

  const scrapers = [
    { name: 'Reddit', ...m.scrapers.reddit, count: m.scrapers.reddit.total_posts ?? 0 },
    { name: 'NHTSA', ...m.scrapers.nhtsa, count: m.scrapers.nhtsa.total_docs ?? 0 },
    { name: 'iFixit', ...m.scrapers.ifixit, count: m.scrapers.ifixit.total_docs ?? 0 },
    { name: 'YouTube', ...m.scrapers.youtube, count: m.scrapers.youtube.total_docs ?? 0 },
  ];

  const maxCount = Math.max(1, ...scrapers.map(s => s.count));
  const chartData = scrapers.map(s => ({ name: s.name, docs: s.count }));

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold">Sources</h2>
      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
        {scrapers.map(s => (
          <Card key={s.name}>
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <CardTitle className="text-sm">{s.name}</CardTitle>
                <div className="flex items-center gap-2">
                  <StatusDot color={s.status === 'running' ? 'green' : s.status === 'stopped' ? 'red' : 'gray'} pulse={s.status === 'running'} />
                  <Badge variant={s.status === 'running' ? 'default' : 'secondary'}>{s.status}</Badge>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold mb-2">{s.count.toLocaleString()}</div>
              <Progress value={(s.count / maxCount) * 100} className="mb-2" />
              <p className="text-xs text-muted-foreground">Last: {timeAgo(s.last_scrape)}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader className="pb-2"><CardTitle className="text-sm">Manuals Pipeline</CardTitle></CardHeader>
        <CardContent>
          <div className="grid grid-cols-4 gap-4 text-center">
            {(['discovered', 'downloaded', 'ingested', 'failed'] as const).map(k => (
              <div key={k}><div className="text-xl font-bold">{m.scrapers.manuals[k]}</div><div className="text-xs text-muted-foreground capitalize">{k}</div></div>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2"><CardTitle className="text-sm">Source Comparison</CardTitle></CardHeader>
        <CardContent>
          <div className="h-[220px]">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" stroke={GRID} />
                <XAxis dataKey="name" tick={{ fill: TICK, fontSize: 12 }} />
                <YAxis tick={{ fill: TICK, fontSize: 12 }} />
                <Tooltip contentStyle={{ background: '#1f2937', border: 'none', borderRadius: 8, fontSize: 12 }} />
                <Bar dataKey="docs" fill="#8b5cf6" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
