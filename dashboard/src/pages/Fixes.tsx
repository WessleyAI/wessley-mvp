import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { EmptyState } from '@/components/shared/EmptyState';
import { Wrench, CheckCircle, SkipForward } from 'lucide-react';
import type { DashboardData } from '@/lib/types';
import { timeAgo } from '@/lib/format';

export function Fixes({ data }: { data: DashboardData }) {
  const { fixes } = data;
  if (fixes.length === 0) return <EmptyState icon={Wrench} message="No fix runs recorded yet." />;

  const sorted = [...fixes].sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold">Fix History</h2>
      {sorted.map((run, i) => (
        <Card key={i}>
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between flex-wrap gap-2">
              <CardTitle className="text-sm">{timeAgo(run.timestamp)} — commit {run.commit}</CardTitle>
              <div className="flex gap-2">
                <Badge variant={run.build_status === 'pass' ? 'default' : 'destructive'}>{run.build_status}</Badge>
                <Badge variant="secondary">{run.tests_after.passed}/{run.tests_after.passed + run.tests_after.failed} tests</Badge>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-3">
            <p className="text-xs text-muted-foreground">{run.summary}</p>
            {run.fixed.length > 0 && (
              <div className="space-y-2">
                <p className="text-xs font-medium text-emerald-400 flex items-center gap-1"><CheckCircle className="h-3 w-3" /> Fixed ({run.fixed.length})</p>
                {run.fixed.map((f, j) => (
                  <div key={j} className="pl-4 text-xs space-y-0.5">
                    <div className="font-medium">{f.title} <Badge variant="secondary" className="ml-1 text-[10px]">{f.category}</Badge></div>
                    <div className="text-muted-foreground">{f.detail}</div>
                    <div className="font-mono text-muted-foreground">{f.files_changed.join(', ')}</div>
                  </div>
                ))}
              </div>
            )}
            {run.skipped.length > 0 && (
              <div className="space-y-2">
                <p className="text-xs font-medium text-yellow-400 flex items-center gap-1"><SkipForward className="h-3 w-3" /> Skipped ({run.skipped.length})</p>
                {run.skipped.map((s, j) => (
                  <div key={j} className="pl-4 text-xs space-y-0.5">
                    <div className="font-medium">{s.title}</div>
                    <div className="text-muted-foreground">{s.reason}</div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
