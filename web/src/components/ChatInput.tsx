"use client";

import { useState, type FormEvent, type KeyboardEvent } from "react";

const STARTERS = [
  "Where is the starter relay in a 2004 Mitsubishi Pajero?",
  "How do I diagnose alternator issues in a Honda Civic?",
  "What fuse controls the headlights in a Toyota Camry?",
];

export default function ChatInput({
  onSend,
  disabled,
}: {
  onSend: (message: string) => void;
  disabled: boolean;
}) {
  const [input, setInput] = useState("");

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    const trimmed = input.trim();
    if (!trimmed || disabled) return;
    onSend(trimmed);
    setInput("");
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  return (
    <div className="border-t border-border bg-bg/80 backdrop-blur-sm">
      <div className="mx-auto max-w-3xl px-4 py-4">
        {!disabled && input === "" && (
          <div className="mb-3 flex flex-wrap gap-2">
            {STARTERS.map((s) => (
              <button
                key={s}
                onClick={() => onSend(s)}
                className="rounded-full border border-border bg-surface px-3 py-1.5 text-xs text-slate-400 transition-colors hover:border-accent hover:text-white"
              >
                {s.length > 50 ? s.slice(0, 50) + "…" : s}
              </button>
            ))}
          </div>
        )}
        <form onSubmit={handleSubmit} className="flex gap-2">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Ask about any vehicle..."
            disabled={disabled}
            rows={1}
            className="flex-1 resize-none rounded-xl border border-border bg-surface px-4 py-3 text-sm text-white placeholder-slate-500 outline-none transition-colors focus:border-accent disabled:opacity-50"
          />
          <button
            type="submit"
            disabled={disabled || !input.trim()}
            className="rounded-xl bg-accent px-5 py-3 text-sm font-medium text-white transition-colors hover:bg-accent-hover disabled:opacity-50"
          >
            Send
          </button>
        </form>
      </div>
    </div>
  );
}
