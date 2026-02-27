import { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { EmptyState } from '@/components/shared/EmptyState';
import { ScrollText } from 'lucide-react';
import type { DashboardData } from '@/lib/types';
import { timeAgo } from '@/lib/format';

const CATEGORIES = ['all', 'system', 'infrastructure', 'scraper', 'error'] as const;
const CAT_COLORS: Record<string, string> = {
  system: 'default',
  infrastructure: 'secondary',
  scraper: 'outline',
  error: 'destructive',
};

export function Logs({ data }: { data: DashboardData }) {
  const { logs, history } = data;
  const [filter, setFilter] = useState<string>('all');

  // Merge logs with history entries as log-like items
  const historyLogs = history.map(h => ({
    time: h.timestamp,
    category: 'system' as const,
    message: `Period: +${h.new_docs} docs, +${h.new_vectors} vectors, +${h.new_relations} rels, ${h.errors_delta} errors`,
  }));

  const merged = [...logs, ...historyLogs].sort((a, b) => new Date(b.time).getTime() - new Date(a.time).getTime());
  const filtered = filter === 'all' ? merged : merged.filter(l => l.category === filter);

  if (merged.length === 0) return <EmptyState icon={ScrollText} message="No log entries yet." />;

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold">Logs</h2>
      <div className="flex flex-wrap gap-2">
        {CATEGORIES.map(c => (
          <Button key={c} variant={filter === c ? 'default' : 'outline'} size="sm" onClick={() => setFilter(c)} className="capitalize">{c}</Button>
        ))}
      </div>
      <Card>
        <CardHeader className="pb-2"><CardTitle className="text-sm">{filtered.length} entries</CardTitle></CardHeader>
        <CardContent className="space-y-1 max-h-[600px] overflow-y-auto">
          {filtered.map((l, i) => (
            <div key={i} className="flex items-start gap-2 py-1.5 border-b border-border/50 last:border-0">
              <Badge variant={(CAT_COLORS[l.category] as 'default' | 'secondary' | 'outline' | 'destructive') || 'secondary'} className="text-[10px] shrink-0">{l.category}</Badge>
              <span className="text-xs flex-1">{l.message}</span>
              <span className="text-[10px] text-muted-foreground shrink-0">{timeAgo(l.time)}</span>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
