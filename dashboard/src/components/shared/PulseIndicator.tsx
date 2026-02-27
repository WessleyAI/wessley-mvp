export function PulseIndicator({ color = 'emerald' }: { color?: 'emerald' | 'red' | 'amber' }) {
  const colors = {
    emerald: 'bg-emerald-400',
    red: 'bg-red-400',
    amber: 'bg-amber-400',
  };
  return (
    <span className="relative flex h-2.5 w-2.5">
      <span className={`absolute inline-flex h-full w-full animate-ping rounded-full opacity-75 ${colors[color]}`} />
      <span className={`relative inline-flex h-2.5 w-2.5 rounded-full ${colors[color]}`} />
    </span>
  );
}
