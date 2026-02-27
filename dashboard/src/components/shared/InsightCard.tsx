import { motion } from 'framer-motion';
import { AlertTriangle, AlertCircle, CheckCircle, Info } from 'lucide-react';
import type { ReactNode } from 'react';

const config = {
  critical: { border: 'border-red-500/30', bg: 'bg-red-500/5', icon: <AlertCircle className="h-4 w-4 text-red-400" />, title: 'text-red-400' },
  warning: { border: 'border-amber-500/30', bg: 'bg-amber-500/5', icon: <AlertTriangle className="h-4 w-4 text-amber-400" />, title: 'text-amber-400' },
  healthy: { border: 'border-emerald-500/30', bg: 'bg-emerald-500/5', icon: <CheckCircle className="h-4 w-4 text-emerald-400" />, title: 'text-emerald-400' },
  info: { border: 'border-blue-500/30', bg: 'bg-blue-500/5', icon: <Info className="h-4 w-4 text-blue-400" />, title: 'text-blue-400' },
};

export function InsightCard({ severity, title, detail, action, index = 0 }: {
  severity: 'critical' | 'warning' | 'healthy' | 'info';
  title: string;
  detail?: string;
  action?: ReactNode;
  index?: number;
}) {
  const c = config[severity];
  return (
    <motion.div
      initial={{ opacity: 0, x: -12 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ delay: index * 0.05, duration: 0.3 }}
      className={`rounded-lg border ${c.border} ${c.bg} p-3`}
    >
      <div className="flex items-start gap-2">
        <div className="mt-0.5 shrink-0">{c.icon}</div>
        <div className="min-w-0 flex-1">
          <p className={`text-sm font-medium ${c.title}`}>{title}</p>
          {detail && <p className="text-xs text-zinc-500 mt-1 line-clamp-2">{detail}</p>}
          {action && <div className="mt-2">{action}</div>}
        </div>
      </div>
    </motion.div>
  );
}
