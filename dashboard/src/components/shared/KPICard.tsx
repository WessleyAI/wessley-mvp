import { motion } from 'framer-motion';
import { AnimatedCounter } from './AnimatedCounter';
import { SparklineChart } from './SparklineChart';
import type { ReactNode } from 'react';

export function KPICard({ title, value, suffix, prefix, decimals, delta, deltaLabel, sparkData, sparkColor, icon }: {
  title: string;
  value: number;
  suffix?: string;
  prefix?: string;
  decimals?: number;
  delta?: number;
  deltaLabel?: string;
  sparkData?: number[];
  sparkColor?: string;
  icon?: ReactNode;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      whileHover={{ y: -2 }}
      transition={{ duration: 0.3 }}
      className="rounded-xl border border-white/5 bg-zinc-900/50 p-4 backdrop-blur-sm"
    >
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs font-medium text-zinc-500 uppercase tracking-wider">{title}</span>
        {icon && <span className="text-zinc-600">{icon}</span>}
      </div>
      <div className="text-2xl font-bold text-zinc-100">
        <AnimatedCounter value={value} suffix={suffix} prefix={prefix} decimals={decimals} />
      </div>
      <div className="flex items-center justify-between mt-2">
        {delta !== undefined ? (
          <span className={`text-xs font-medium ${delta >= 0 ? 'text-emerald-400' : 'text-red-400'}`}>
            {delta >= 0 ? '↑' : '↓'} {Math.abs(delta).toLocaleString()} {deltaLabel ?? ''}
          </span>
        ) : <span />}
        {sparkData && sparkData.length > 1 && (
          <div className="w-20">
            <SparklineChart data={sparkData} color={sparkColor} height={24} />
          </div>
        )}
      </div>
    </motion.div>
  );
}
