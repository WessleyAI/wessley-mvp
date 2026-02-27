import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';

const sourceColors: Record<string, string> = {
  nhtsa: 'bg-blue-500',
  ifixit: 'bg-emerald-500',
  reddit: 'bg-orange-500',
  youtube: 'bg-red-500',
  forums: 'bg-purple-500',
};

interface Props {
  sources: Record<string, number>;
  total: number;
}

export function SourceBar({ sources, total }: Props) {
  if (total === 0) return null;
  const entries = Object.entries(sources).filter(([, v]) => v > 0).sort((a, b) => b[1] - a[1]);
  return (
    <TooltipProvider>
      <div className="flex h-4 rounded-full overflow-hidden bg-muted">
        {entries.map(([name, count]) => (
          <Tooltip key={name}>
            <TooltipTrigger asChild>
              <div className={`${sourceColors[name] || 'bg-gray-500'} transition-all`} style={{ width: `${(count / total) * 100}%` }} />
            </TooltipTrigger>
            <TooltipContent><span className="capitalize">{name}</span>: {count.toLocaleString()}</TooltipContent>
          </Tooltip>
        ))}
      </div>
      <div className="flex gap-3 mt-2 flex-wrap">
        {entries.map(([name, count]) => (
          <div key={name} className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <span className={`w-2 h-2 rounded-full ${sourceColors[name] || 'bg-gray-500'}`} />
            <span className="capitalize">{name}</span>
            <span>({((count / total) * 100).toFixed(1)}%)</span>
          </div>
        ))}
      </div>
    </TooltipProvider>
  );
}
