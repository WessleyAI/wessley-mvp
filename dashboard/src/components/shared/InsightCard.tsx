import { Badge } from '@/components/ui/badge';
import type { Insight } from '@/lib/types';
import { cn } from '@/lib/utils';

const styles: Record<string, { badge: string; border: string }> = {
  critical: { badge: 'bg-red-500/20 text-red-400 border-red-500/30', border: 'border-l-red-500' },
  warning: { badge: 'bg-amber-500/20 text-amber-400 border-amber-500/30', border: 'border-l-amber-500' },
  info: { badge: 'bg-blue-500/20 text-blue-400 border-blue-500/30', border: 'border-l-blue-500' },
  ok: { badge: 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30', border: 'border-l-emerald-500' },
};

export function InsightCard({ insight }: { insight: Insight }) {
  const s = styles[insight.severity] || styles.info;
  return (
    <div className={cn('flex items-start gap-3 border-l-2 pl-3 py-1.5', s.border)}>
      <Badge variant="outline" className={cn('text-[10px] shrink-0 mt-0.5', s.badge)}>
        {insight.severity}
      </Badge>
      <span className="text-sm text-muted-foreground">{insight.message}</span>
    </div>
  );
}
