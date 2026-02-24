import type { Source } from "@/lib/api";

export default function Sources({ sources }: { sources: Source[] }) {
  if (!sources || sources.length === 0) return null;

  return (
    <div className="mt-3 border-t border-border pt-3">
      <p className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-500">Sources</p>
      <ul className="space-y-1">
        {sources.map((s, i) => (
          <li key={i} className="flex items-center gap-2 text-sm text-slate-400">
            <span className="text-slate-500">•</span>
            {s.url ? (
              <a href={s.url} target="_blank" rel="noopener noreferrer" className="hover:text-accent">
                {s.title || s.source_type}
              </a>
            ) : (
              <span>{s.title || s.source_type}</span>
            )}
            <span className="ml-auto rounded bg-surface px-1.5 py-0.5 text-xs text-slate-500">
              {(s.score * 100).toFixed(0)}%
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
