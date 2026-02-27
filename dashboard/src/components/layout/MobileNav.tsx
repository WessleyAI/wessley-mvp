import { useLocation, Link } from 'react-router-dom';
import { LayoutDashboard, Share2, Settings, Server, FileSearch } from 'lucide-react';

const tabs = [
  { to: '/', icon: LayoutDashboard, label: 'Overview' },
  { to: '/graph', icon: Share2, label: 'Graph' },
  { to: '/controls', icon: Settings, label: 'Controls' },
  { to: '/analysis', icon: FileSearch, label: 'Analysis' },
  { to: '/infra', icon: Server, label: 'Infra' },
];

export function MobileNav() {
  const { pathname } = useLocation();
  return (
    <nav className="lg:hidden fixed bottom-0 left-0 right-0 bg-card border-t border-border flex justify-around py-2 z-50">
      {tabs.map(t => {
        const active = pathname === t.to;
        return (
          <Link key={t.to} to={t.to} className={`flex flex-col items-center gap-0.5 text-[10px] ${active ? 'text-primary' : 'text-muted-foreground'}`}>
            <t.icon className="h-5 w-5" />
            {t.label}
          </Link>
        );
      })}
    </nav>
  );
}
