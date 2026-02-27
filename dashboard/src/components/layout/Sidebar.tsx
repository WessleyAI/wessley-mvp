import { useLocation, Link } from 'react-router-dom';
import { LayoutDashboard, GitBranch, Radio, Share2, Car, BookOpen, Settings, FileSearch, Wrench, Server, ScrollText, Timer } from 'lucide-react';
import { useEffect, useState } from 'react';

const sections = [
  {
    label: 'MONITOR', items: [
      { to: '/', icon: LayoutDashboard, label: 'Overview' },
      { to: '/pipeline', icon: GitBranch, label: 'Pipeline' },
      { to: '/sources', icon: Radio, label: 'Sources' },
    ]
  },
  {
    label: 'KNOWLEDGE', items: [
      { to: '/graph', icon: Share2, label: 'Graph' },
      { to: '/vehicles', icon: Car, label: 'Vehicles' },
      { to: '/manuals', icon: BookOpen, label: 'Manuals' },
    ]
  },
  {
    label: 'OPERATIONS', items: [
      { to: '/controls', icon: Settings, label: 'Controls' },
      { to: '/analysis', icon: FileSearch, label: 'Analysis' },
      { to: '/fixes', icon: Wrench, label: 'Fixes' },
    ]
  },
  {
    label: 'SYSTEM', items: [
      { to: '/infra', icon: Server, label: 'Infra' },
      { to: '/logs', icon: ScrollText, label: 'Logs' },
    ]
  },
];

export function Sidebar() {
  const { pathname } = useLocation();
  const [countdown, setCountdown] = useState(30);

  useEffect(() => {
    const iv = setInterval(() => setCountdown(c => c <= 1 ? 30 : c - 1), 1000);
    return () => clearInterval(iv);
  }, []);

  return (
    <aside className="hidden lg:flex flex-col w-60 h-screen fixed left-0 top-0 bg-card border-r border-border p-4 overflow-y-auto">
      <div className="mb-6">
        <h1 className="text-lg font-bold">🚗 Wessley AI</h1>
        <p className="text-xs text-muted-foreground">Knowledge Engine</p>
      </div>
      {sections.map(s => (
        <div key={s.label} className="mb-4">
          <p className="text-[10px] font-semibold text-muted-foreground tracking-widest mb-2">{s.label}</p>
          {s.items.map(item => {
            const active = pathname === item.to;
            return (
              <Link key={item.to} to={item.to} className={`flex items-center gap-2 px-3 py-1.5 rounded-md text-sm transition-colors ${active ? 'bg-accent text-accent-foreground font-medium' : 'text-muted-foreground hover:text-foreground hover:bg-accent/50'}`}>
                <item.icon className="h-4 w-4" />
                {item.label}
              </Link>
            );
          })}
        </div>
      ))}
      <div className="mt-auto pt-4 border-t border-border flex items-center gap-2 text-xs text-muted-foreground">
        <Timer className="h-3 w-3" />
        <span>Refresh in {countdown}s</span>
      </div>
    </aside>
  );
}
