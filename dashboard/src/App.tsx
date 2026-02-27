import { HashRouter, Routes, Route } from 'react-router-dom';
import { Sidebar } from '@/components/layout/Sidebar';
import { MobileNav } from '@/components/layout/MobileNav';
import { useData } from '@/hooks/useData';
import { Overview } from '@/pages/Overview';
import { Pipeline } from '@/pages/Pipeline';
import { Sources } from '@/pages/Sources';
import { Graph } from '@/pages/Graph';
import { Vehicles } from '@/pages/Vehicles';
import { Manuals } from '@/pages/Manuals';
import { Controls } from '@/pages/Controls';
import { Analysis } from '@/pages/Analysis';
import { Fixes } from '@/pages/Fixes';
import { Infrastructure } from '@/pages/Infrastructure';
import { Logs } from '@/pages/Logs';
import { Loader2 } from 'lucide-react';

function App() {
  const data = useData();

  if (data.loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (data.error && !data.metrics) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-center space-y-2">
          <p className="text-red-400">Failed to load data</p>
          <p className="text-xs text-muted-foreground">{data.error}</p>
        </div>
      </div>
    );
  }

  return (
    <HashRouter>
      <div className="dark min-h-screen bg-background text-foreground">
        <Sidebar />
        <main className="lg:ml-60 p-4 md:p-6 pb-20 lg:pb-6">
          <Routes>
            <Route path="/" element={<Overview data={data} />} />
            <Route path="/pipeline" element={<Pipeline data={data} />} />
            <Route path="/sources" element={<Sources data={data} />} />
            <Route path="/graph" element={<Graph data={data} />} />
            <Route path="/vehicles" element={<Vehicles data={data} />} />
            <Route path="/manuals" element={<Manuals data={data} />} />
            <Route path="/controls" element={<Controls data={data} />} />
            <Route path="/analysis" element={<Analysis data={data} />} />
            <Route path="/fixes" element={<Fixes data={data} />} />
            <Route path="/infra" element={<Infrastructure data={data} />} />
            <Route path="/logs" element={<Logs data={data} />} />
          </Routes>
        </main>
        <MobileNav />
      </div>
    </HashRouter>
  );
}

export default App;
