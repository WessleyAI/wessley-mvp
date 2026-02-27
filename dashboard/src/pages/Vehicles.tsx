import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Progress } from '@/components/ui/progress';
import type { DashboardData } from '@/lib/types';
import { EmptyState } from '@/components/shared/EmptyState';
import { Car } from 'lucide-react';

export function Vehicles({ data }: { data: DashboardData }) {
  const { metrics: m } = data;
  if (!m) return null;
  const kg = m.knowledge_graph;
  const maxDocs = Math.max(1, ...kg.top_makes.map(mk => mk.documents));

  if (kg.top_makes.length === 0) {
    return <EmptyState icon={Car} message="No vehicle makes tracked yet." />;
  }

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold">Vehicles</h2>
      <Card>
        <CardHeader className="pb-2"><CardTitle className="text-sm">Makes</CardTitle></CardHeader>
        <CardContent>
          <Table>
            <TableHeader><TableRow><TableHead>Make</TableHead><TableHead>Models</TableHead><TableHead>Documents</TableHead><TableHead className="w-[200px]">Coverage</TableHead></TableRow></TableHeader>
            <TableBody>
              {kg.top_makes.map(mk => (
                <TableRow key={mk.name}>
                  <TableCell className="font-medium">{mk.name}</TableCell>
                  <TableCell>{mk.models}</TableCell>
                  <TableCell>{mk.documents.toLocaleString()}</TableCell>
                  <TableCell><Progress value={(mk.documents / maxDocs) * 100} /></TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="pb-2"><CardTitle className="text-sm">Top Vehicles</CardTitle></CardHeader>
        <CardContent>
          {kg.top_vehicles.length === 0 ? (
            <p className="text-sm text-muted-foreground py-4 text-center">No vehicle data available.</p>
          ) : (
            <Table>
              <TableHeader><TableRow><TableHead>Vehicle</TableHead><TableHead className="text-right">Documents</TableHead><TableHead className="text-right">Components</TableHead></TableRow></TableHeader>
              <TableBody>
                {kg.top_vehicles.map(v => (
                  <TableRow key={v.vehicle}><TableCell>{v.vehicle}</TableCell><TableCell className="text-right">{v.documents}</TableCell><TableCell className="text-right">{v.components}</TableCell></TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
