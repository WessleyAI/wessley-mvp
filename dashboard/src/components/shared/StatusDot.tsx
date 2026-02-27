const colors: Record<string, string> = {
  green: 'bg-emerald-500',
  yellow: 'bg-yellow-500',
  red: 'bg-red-500',
  gray: 'bg-gray-500',
};

export function StatusDot({ color = 'gray', pulse = false }: { color?: string; pulse?: boolean }) {
  const c = colors[color] || colors.gray;
  return (
    <span className="relative flex h-3 w-3">
      {pulse && <span className={`animate-ping absolute inline-flex h-full w-full rounded-full opacity-75 ${c}`} />}
      <span className={`relative inline-flex rounded-full h-3 w-3 ${c}`} />
    </span>
  );
}
