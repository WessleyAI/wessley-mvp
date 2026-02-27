import { cn } from '@/lib/utils';

const colors: Record<string, { dot: string; bg: string; text: string }> = {
  connected: { dot: 'bg-emerald-400', bg: 'bg-emerald-500/10', text: 'text-emerald-400' },
  running: { dot: 'bg-emerald-400', bg: 'bg-emerald-500/10', text: 'text-emerald-400' },
  green: { dot: 'bg-emerald-400', bg: 'bg-emerald-500/10', text: 'text-emerald-400' },
  stopped: { dot: 'bg-zinc-500', bg: 'bg-zinc-500/10', text: 'text-zinc-400' },
  error: { dot: 'bg-red-400', bg: 'bg-red-500/10', text: 'text-red-400' },
  warning: { dot: 'bg-amber-400', bg: 'bg-amber-500/10', text: 'text-amber-400' },
};

export function StatusPill({ status, className }: { status: string; className?: string }) {
  const c = colors[status] ?? colors.stopped;
  return (
    <span className={cn('inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium', c.bg, c.text, className)}>
      <span className={cn('h-1.5 w-1.5 rounded-full', c.dot)} />
      {status}
    </span>
  );
}
