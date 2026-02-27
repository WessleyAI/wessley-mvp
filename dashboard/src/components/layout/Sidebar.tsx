import { useState } from 'react';
import { NavLink } from 'react-router-dom';
import { cn } from '@/lib/utils';
import { PulseIndicator } from '@/components/shared/PulseIndicator';
import {
  LayoutDashboard, GitBranch, Database, Network, Car, BookOpen,
  Settings, BarChart3, Wrench, Server, ScrollText, PanelLeftClose, PanelLeft
} from 'lucide-react';

const links = [
  { to: '/', icon: LayoutDashboard, label: 'Overview' },
  { to: '/pipeline', icon: GitBranch, label: 'Pipeline' },
  { to: '/sources', icon: Database, label: 'Sources' },
  { to: '/graph', icon: Network, label: 'Graph' },
  { to: '/vehicles', icon: Car, label: 'Vehicles' },
  { to: '/manuals', icon: BookOpen, label: 'Manuals' },
  { to: '/controls', icon: Settings, label: 'Controls' },
  { to: '/analysis', icon: BarChart3, label: 'Analysis' },
  { to: '/fixes', icon: Wrench, label: 'Fixes' },
  { to: '/infra', icon: Server, label: 'Infrastructure' },
  { to: '/logs', icon: ScrollText, label: 'Logs' },
];

export function Sidebar() {
  const [collapsed, setCollapsed] = useState(false);

  return (
    <aside className={cn(
      'fixed left-0 top-0 z-40 hidden lg:flex flex-col h-screen border-r border-white/5 bg-zinc-950/80 backdrop-blur-xl transition-all duration-300',
      collapsed ? 'w-14' : 'w-60'
    )}>
      {/* Logo */}
      <div className="flex items-center gap-2 px-4 h-14 border-b border-white/5">
        {!collapsed && (
          <div className="flex items-center gap-2 flex-1">
            <div className="h-7 w-7 rounded-lg bg-gradient-to-br from-emerald-400 to-blue-500 flex items-center justify-center text-xs font-bold text-white">W</div>
            <span className="font-semibold text-zinc-100 text-sm">Wessley AI</span>
            <PulseIndicator />
          </div>
        )}
        <button onClick={() => setCollapsed(!collapsed)} className="p-1 rounded hover:bg-white/5 text-zinc-500">
          {collapsed ? <PanelLeft className="h-4 w-4" /> : <PanelLeftClose className="h-4 w-4" />}
        </button>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto py-2 px-2 space-y-0.5">
        {links.map(({ to, icon: Icon, label }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) => cn(
              'flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-sm transition-colors',
              isActive
                ? 'bg-white/10 text-zinc-100 shadow-sm shadow-emerald-500/10'
                : 'text-zinc-500 hover:text-zinc-300 hover:bg-white/5'
            )}
          >
            <Icon className="h-4 w-4 shrink-0" />
            {!collapsed && <span>{label}</span>}
          </NavLink>
        ))}
      </nav>

      {/* Footer */}
      {!collapsed && (
        <div className="px-4 py-3 border-t border-white/5">
          <p className="text-[10px] text-zinc-600 uppercase tracking-wider">Wessley MVP Dashboard</p>
        </div>
      )}
    </aside>
  );
}
