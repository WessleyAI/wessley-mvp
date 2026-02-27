import type { LucideIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface Props {
  icon: LucideIcon;
  message: string;
  action?: string;
  onAction?: () => void;
}

export function EmptyState({ icon: Icon, message, action, onAction }: Props) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
      <Icon className="h-12 w-12 mb-4 opacity-40" />
      <p className="text-sm mb-4">{message}</p>
      {action && onAction && <Button variant="outline" size="sm" onClick={onAction}>{action}</Button>}
    </div>
  );
}
