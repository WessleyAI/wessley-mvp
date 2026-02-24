"use client";

import { useEffect, useRef } from "react";
import Message, { type MessageData } from "./Message";

export default function ChatMessages({
  messages,
  loading,
}: {
  messages: MessageData[];
  loading: boolean;
}) {
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, loading]);

  return (
    <div className="flex-1 overflow-y-auto px-4 py-6">
      <div className="mx-auto max-w-3xl space-y-4">
        {messages.length === 0 && !loading && (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <span className="text-4xl">⚡</span>
            <h2 className="mt-4 text-xl font-semibold text-white">
              Welcome to Wessley
            </h2>
            <p className="mt-2 max-w-md text-sm text-slate-400">
              Ask me anything about vehicle electrical systems — wiring, relays,
              fuses, alternators, starters, and more.
            </p>
          </div>
        )}

        {messages.map((msg, i) => (
          <Message key={i} msg={msg} />
        ))}

        {loading && (
          <div className="flex justify-start">
            <div className="rounded-2xl border border-border bg-surface px-4 py-3">
              <p className="mb-1 text-xs font-medium text-slate-500">Wessley</p>
              <div className="flex gap-1">
                <span className="typing-dot h-2 w-2 rounded-full bg-slate-500" />
                <span className="typing-dot h-2 w-2 rounded-full bg-slate-500" />
                <span className="typing-dot h-2 w-2 rounded-full bg-slate-500" />
              </div>
            </div>
          </div>
        )}

        <div ref={endRef} />
      </div>
    </div>
  );
}
