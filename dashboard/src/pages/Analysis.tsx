import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { AlertTriangle, AlertCircle, CheckCircle, Lightbulb, Bug, Target } from 'lucide-react';
import type { DashboardData } from '@/lib/types';
import { EmptyState } from '@/components/shared/EmptyState';

export function Analysis({ data }: { data: DashboardData }) {
  const { analysis: a } = data;
  if (!a) return <EmptyState icon={AlertCircle} message="No analysis data available." />;

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold">Analysis</h2>

      {a.critical.length > 0 && (
        <Card className="border-red-500/20">
          <CardHeader className="pb-2"><CardTitle className="text-sm flex items-center gap-2"><AlertTriangle className="h-4 w-4 text-red-400" /> Critical ({a.critical.length})</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {a.critical.map((c, i) => (
              <div key={i} className="p-3 rounded-md bg-red-500/5 space-y-1">
                <div className="text-sm font-medium text-red-400">{c.title}</div>
                <div className="text-xs text-muted-foreground">{c.detail}</div>
                <div className="text-xs text-emerald-400">Fix: {c.fix}</div>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      {a.warnings.length > 0 && (
        <Card className="border-yellow-500/20">
          <CardHeader className="pb-2"><CardTitle className="text-sm flex items-center gap-2"><AlertCircle className="h-4 w-4 text-yellow-400" /> Warnings ({a.warnings.length})</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {a.warnings.map((w, i) => (
              <div key={i} className="p-3 rounded-md bg-yellow-500/5 space-y-1">
                <div className="text-sm font-medium text-yellow-400">{w.title}</div>
                <div className="text-xs text-muted-foreground">{w.detail}</div>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      {a.healthy.length > 0 && (
        <Card className="border-emerald-500/20">
          <CardHeader className="pb-2"><CardTitle className="text-sm flex items-center gap-2"><CheckCircle className="h-4 w-4 text-emerald-400" /> Healthy ({a.healthy.length})</CardTitle></CardHeader>
          <CardContent>
            <ul className="space-y-1">
              {a.healthy.map((h, i) => <li key={i} className="text-xs text-emerald-400">✓ {h}</li>)}
            </ul>
          </CardContent>
        </Card>
      )}

      {a.suggestions.length > 0 && (
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm flex items-center gap-2"><Lightbulb className="h-4 w-4 text-blue-400" /> Suggestions</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {a.suggestions.map((s, i) => (
              <div key={i} className="p-3 rounded-md bg-blue-500/5 space-y-1">
                <div className="flex items-center gap-2"><span className="text-sm font-medium">{s.title}</span><Badge variant="secondary">{s.effort}</Badge></div>
                <div className="text-xs text-muted-foreground">{s.impact}</div>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      {a.bugs.length > 0 && (
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm flex items-center gap-2"><Bug className="h-4 w-4 text-orange-400" /> Bugs ({a.bugs.length})</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {a.bugs.map((b, i) => (
              <div key={i} className="p-3 rounded-md bg-orange-500/5 space-y-1">
                <div className="text-sm font-medium">{b.title}</div>
                <div className="text-xs font-mono text-muted-foreground">{b.file}:{b.line}</div>
                <div className="text-xs text-muted-foreground">{b.detail}</div>
                <div className="text-xs text-emerald-400">Fix: {b.fix}</div>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      {a.strategy.length > 0 && (
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm flex items-center gap-2"><Target className="h-4 w-4 text-purple-400" /> Strategy</CardTitle></CardHeader>
          <CardContent>
            <ol className="space-y-2 list-decimal list-inside">
              {a.strategy.map((s, i) => <li key={i} className="text-xs text-muted-foreground">{s}</li>)}
            </ol>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
