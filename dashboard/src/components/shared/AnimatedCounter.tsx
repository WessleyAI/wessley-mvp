import { useEffect, useRef, useState } from 'react';
import { motion, useSpring, useTransform } from 'framer-motion';

function formatSuffix(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
  if (n >= 10_000) return (n / 1_000).toFixed(1) + 'K';
  return n.toLocaleString();
}

export function AnimatedCounter({ value, suffix = '', prefix = '', decimals = 0 }: { value: number; suffix?: string; prefix?: string; decimals?: number }) {
  const spring = useSpring(0, { duration: 1200, bounce: 0 });
  const display = useTransform(spring, (v) => {
    if (decimals > 0) return prefix + v.toFixed(decimals) + suffix;
    return prefix + formatSuffix(Math.round(v)) + suffix;
  });
  const [text, setText] = useState(prefix + '0' + suffix);

  useEffect(() => {
    spring.set(value);
  }, [value, spring]);

  useEffect(() => {
    const unsub = display.on('change', (v) => setText(v));
    return unsub;
  }, [display]);

  return <motion.span>{text}</motion.span>;
}
