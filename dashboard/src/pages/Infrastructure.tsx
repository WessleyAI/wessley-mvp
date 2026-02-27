import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { StatusDot } from '@/components/shared/StatusDot';
import { Database, Box, Cpu } from 'lucide-react';
import type { DashboardData } from '@/lib/types';

export function Infrastructure({ data }: { data: DashboardData }) {
  const { metrics: m } = data;
  if (!m) return null;

  const services = [
    { name: 'Neo4j', icon: Database, ...m.infrastructure.neo4j, detail: `Version ${m.infrastructure.neo4j.version || 'N/A'}` },
    { name: 'Qdrant', icon: Box, ...m.infrastructure.qdrant, detail: `${m.infrastructure.qdrant.vectors?.toLocaleString() || 0} vectors` },
    { name: 'Ollama', icon: Cpu, ...m.infrastructure.ollama, detail: `Model: ${m.infrastructure.ollama.model || 'N/A'}` },
  ];

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold">Infrastructure</h2>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {services.map(s => (
          <Card key={s.name}>
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <CardTitle className="text-sm flex items-center gap-2"><s.icon className="h-4 w-4" />{s.name}</CardTitle>
                <StatusDot color={s.status === 'connected' ? 'green' : 'red'} pulse={s.status === 'connected'} />
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-xs text-muted-foreground">{s.detail}</div>
              <div className="text-xs mt-1 capitalize">{s.status}</div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
