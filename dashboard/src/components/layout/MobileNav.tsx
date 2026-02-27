import { NavLink } from 'react-router-dom';
import { cn } from '@/lib/utils';
import { LayoutDashboard, Database, Network, BarChart3, Server, MoreHorizontal } from 'lucide-react';
import { useState } from 'react';
import { GitBranch, Car, BookOpen, Settings, Wrench, ScrollText } from 'lucide-react';

const main = [
  { to: '/', icon: LayoutDashboard, label: 'Overview' },
  { to: '/sources', icon: Database, label: 'Sources' },
  { to: '/graph', icon: Network, label: 'Graph' },
  { to: '/analysis', icon: BarChart3, label: 'Analysis' },
  { to: '/infra', icon: Server, label: 'Infra' },
];

const more = [
  { to: '/pipeline', icon: GitBranch, label: 'Pipeline' },
  { to: '/vehicles', icon: Car, label: 'Vehicles' },
  { to: '/manuals', icon: BookOpen, label: 'Manuals' },
  { to: '/controls', icon: Settings, label: 'Controls' },
  { to: '/fixes', icon: Wrench, label: 'Fixes' },
  { to: '/logs', icon: ScrollText, label: 'Logs' },
];

export function MobileNav() {
  const [showMore, setShowMore] = useState(false);

  return (
    <>
      {showMore && (
        <div className="fixed inset-0 z-40 bg-black/50 lg:hidden" onClick={() => setShowMore(false)}>
          <div className="absolute bottom-16 left-0 right-0 bg-zinc-900 border-t border-white/5 p-2 grid grid-cols-3 gap-1" onClick={(e) => e.stopPropagation()}>
            {more.map(({ to, icon: Icon, label }) => (
              <NavLink key={to} to={to} onClick={() => setShowMore(false)}
                className={({ isActive }) => cn('flex flex-col items-center gap-1 rounded-lg py-2 text-xs', isActive ? 'text-zinc-100 bg-white/10' : 'text-zinc-500')}>
                <Icon className="h-4 w-4" />{label}
              </NavLink>
            ))}
          </div>
        </div>
      )}
      <nav className="fixed bottom-0 left-0 right-0 z-50 lg:hidden border-t border-white/5 bg-zinc-950/90 backdrop-blur-xl">
        <div className="flex items-center justify-around h-14">
          {main.map(({ to, icon: Icon, label }) => (
            <NavLink key={to} to={to}
              className={({ isActive }) => cn('flex flex-col items-center gap-0.5 text-[10px]', isActive ? 'text-zinc-100' : 'text-zinc-600')}>
              <Icon className="h-4 w-4" />{label}
            </NavLink>
          ))}
          <button onClick={() => setShowMore(!showMore)} className="flex flex-col items-center gap-0.5 text-[10px] text-zinc-600">
            <MoreHorizontal className="h-4 w-4" />More
          </button>
        </div>
      </nav>
    </>
  );
}
