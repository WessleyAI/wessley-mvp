import { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { useApi, useIsLocal } from '@/hooks/useApi';
import type { DashboardData } from '@/lib/types';
import { Lock } from 'lucide-react';

function RemoteWrap({ children, isLocal }: { children: React.ReactNode; isLocal: boolean }) {
  if (isLocal) return <>{children}</>;
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild><div className="opacity-50 cursor-not-allowed">{children}</div></TooltipTrigger>
        <TooltipContent><p className="flex items-center gap-1"><Lock className="h-3 w-3" /> Only available on localhost</p></TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export function Controls({ data }: { data: DashboardData }) {
  const { metrics: m } = data;
  const isLocal = useIsLocal();
  const api = useApi();
  const [cypher, setCypher] = useState('');
  const [cypherResult, setCypherResult] = useState<string | null>(null);
  const [vectorQuery, setVectorQuery] = useState('');
  const [vectorResult, setVectorResult] = useState<string | null>(null);

  if (!m) return null;

  const scrapers = ['reddit', 'nhtsa', 'ifixit', 'youtube'] as const;
  const actions = [
    { label: 'Run Enricher', path: '/api/enricher/run' },
    { label: 'Rebuild Graph', path: '/api/graph/rebuild' },
    { label: 'Snapshot', path: '/api/snapshot' },
    { label: 'Clear Cache', path: '/api/cache/clear' },
  ];
  const crawlers = [
    { label: 'Crawl Manuals', path: '/api/manuals/crawl' },
    { label: 'Crawl Forums', path: '/api/forums/crawl' },
  ];

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold">Control Panel</h2>
      {!isLocal && (
        <div className="flex items-center gap-2 p-3 rounded-md bg-yellow-500/10 border border-yellow-500/20 text-yellow-400 text-sm">
          <Lock className="h-4 w-4" />
          <span>Controls disabled — connect from localhost to enable.</span>
        </div>
      )}

      <Card>
        <CardHeader className="pb-2"><CardTitle className="text-sm">Scrapers</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          {scrapers.map(name => {
            const s = m.scrapers[name];
            return (
              <RemoteWrap key={name} isLocal={isLocal}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <span className="text-sm font-medium capitalize w-20">{name}</span>
                    <Badge variant={s.status === 'running' ? 'default' : 'secondary'}>{s.status}</Badge>
                  </div>
                  <Switch disabled={!isLocal} checked={s.status === 'running'} onCheckedChange={() => api.call(`/api/scraper/${name}/toggle`, { method: 'POST' })} />
                </div>
              </RemoteWrap>
            );
          })}
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm">Pipeline Actions</CardTitle></CardHeader>
          <CardContent className="flex flex-wrap gap-2">
            {actions.map(a => (
              <RemoteWrap key={a.label} isLocal={isLocal}>
                <Button variant="outline" size="sm" disabled={!isLocal} onClick={() => api.call(a.path, { method: 'POST' })}>{a.label}</Button>
              </RemoteWrap>
            ))}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm">Manual Crawlers</CardTitle></CardHeader>
          <CardContent className="flex flex-wrap gap-2">
            {crawlers.map(c => (
              <RemoteWrap key={c.label} isLocal={isLocal}>
                <Button variant="outline" size="sm" disabled={!isLocal} onClick={() => api.call(c.path, { method: 'POST' })}>{c.label}</Button>
              </RemoteWrap>
            ))}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="pb-2"><CardTitle className="text-sm">Cypher Query</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          <RemoteWrap isLocal={isLocal}>
            <textarea
              className="w-full h-24 bg-muted rounded-md p-3 text-sm font-mono resize-none border-0 focus:outline-none focus:ring-1 focus:ring-ring"
              placeholder="MATCH (n) RETURN labels(n), count(n)"
              value={cypher}
              onChange={e => setCypher(e.target.value)}
              disabled={!isLocal}
            />
          </RemoteWrap>
          <RemoteWrap isLocal={isLocal}>
            <Button size="sm" disabled={!isLocal || !cypher} onClick={async () => {
              const r = await api.call('/api/cypher', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ query: cypher }) });
              if (r) setCypherResult(r);
            }}>Run Query</Button>
          </RemoteWrap>
          {cypherResult && <pre className="bg-muted rounded-md p-3 text-xs overflow-auto max-h-48">{cypherResult}</pre>}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2"><CardTitle className="text-sm">Vector Search</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          <div className="flex gap-2">
            <RemoteWrap isLocal={isLocal}>
              <Input placeholder="Search vectors..." value={vectorQuery} onChange={e => setVectorQuery(e.target.value)} disabled={!isLocal} />
            </RemoteWrap>
            <RemoteWrap isLocal={isLocal}>
              <Button size="sm" disabled={!isLocal || !vectorQuery} onClick={async () => {
                const r = await api.call(`/api/search?q=${encodeURIComponent(vectorQuery)}`);
                if (r) setVectorResult(r);
              }}>Search</Button>
            </RemoteWrap>
          </div>
          {vectorResult && <pre className="bg-muted rounded-md p-3 text-xs overflow-auto max-h-48">{vectorResult}</pre>}
        </CardContent>
      </Card>

      {api.error && <div className="text-red-400 text-sm p-3 bg-red-500/10 rounded-md">{api.error}</div>}
      {api.result && !cypherResult && !vectorResult && <div className="text-emerald-400 text-sm p-3 bg-emerald-500/10 rounded-md">{api.result}</div>}
    </div>
  );
}
