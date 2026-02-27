import { Inbox } from 'lucide-react';

export function EmptyState({ message = 'No data available' }: { message?: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-zinc-600">
      <Inbox className="h-10 w-10 mb-3" />
      <p className="text-sm">{message}</p>
    </div>
  );
}
