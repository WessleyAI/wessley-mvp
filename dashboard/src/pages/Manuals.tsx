import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { EmptyState } from '@/components/shared/EmptyState';
import { BookOpen } from 'lucide-react';
import type { DashboardData } from '@/lib/types';

export function Manuals({ data }: { data: DashboardData }) {
  const { metrics: m } = data;
  if (!m) return null;
  const man = m.scrapers.manuals;
  const total = man.discovered + man.downloaded + man.ingested;

  if (total === 0) {
    return (
      <div className="space-y-6">
        <h2 className="text-xl font-bold">Manuals</h2>
        <EmptyState icon={BookOpen} message="No manuals discovered yet. The manuals scraper needs to be activated — this is the highest-value untapped data source." />
      </div>
    );
  }

  const stages = [
    { label: 'Discovered', value: man.discovered, color: 'bg-blue-500' },
    { label: 'Downloaded', value: man.downloaded, color: 'bg-indigo-500' },
    { label: 'Ingested', value: man.ingested, color: 'bg-emerald-500' },
    { label: 'Failed', value: man.failed, color: 'bg-red-500' },
  ];
  const maxVal = Math.max(1, ...stages.map(s => s.value));

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold">Manuals</h2>
      <Card>
        <CardHeader className="pb-2"><CardTitle className="text-sm">Manuals Funnel</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          {stages.map(s => (
            <div key={s.label} className="space-y-1">
              <div className="flex justify-between text-sm"><span>{s.label}</span><span className="font-medium">{s.value}</span></div>
              <div className="h-6 bg-muted rounded-md overflow-hidden">
                <div className={`h-full ${s.color} rounded-md transition-all`} style={{ width: `${(s.value / maxVal) * 100}%` }} />
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
