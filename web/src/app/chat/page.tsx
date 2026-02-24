"use client";

import { useState, useCallback } from "react";
import Header from "@/components/Header";
import ChatMessages from "@/components/ChatMessages";
import ChatInput from "@/components/ChatInput";
import { chat, chatStream, type Source } from "@/lib/api";
import type { MessageData } from "@/components/Message";

export default function ChatPage() {
  const [messages, setMessages] = useState<MessageData[]>([]);
  const [loading, setLoading] = useState(false);

  const handleSend = useCallback(async (question: string) => {
    const userMsg: MessageData = { role: "user", content: question };
    setMessages((prev) => [...prev, userMsg]);
    setLoading(true);

    try {
      // Try streaming first, fall back to regular
      let content = "";
      let sources: Source[] = [];

      try {
        const assistantIdx = messages.length + 1; // index after user msg
        for await (const chunk of chatStream(question)) {
          content += chunk;
          setMessages((prev) => {
            const updated = [...prev];
            if (updated[assistantIdx]) {
              updated[assistantIdx] = { role: "assistant", content };
            } else {
              updated.push({ role: "assistant", content });
            }
            return updated;
          });
        }
        // Try parsing final chunk as JSON for sources
        try {
          const parsed = JSON.parse(content);
          if (parsed.answer) {
            content = parsed.answer;
            sources = parsed.sources || [];
          }
        } catch {
          // Content is plain text streaming, that's fine
        }
      } catch {
        // Streaming not available, fall back to regular request
        const response = await chat(question);
        content = response.answer;
        sources = response.sources || [];
      }

      setMessages((prev) => {
        const updated = prev.filter((m) => m === userMsg || m.role === "user" || m.content !== content);
        // Remove any partial streaming message and add final
        const withoutPartial = updated.filter((_, i) => i <= prev.indexOf(userMsg));
        return [
          ...withoutPartial,
          { role: "assistant", content, sources },
        ];
      });
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : "Something went wrong";
      setMessages((prev) => [
        ...prev,
        {
          role: "assistant",
          content: `⚠️ ${errorMsg}. Make sure the API server is running.`,
        },
      ]);
    } finally {
      setLoading(false);
    }
  }, [messages.length]);

  return (
    <div className="flex h-dvh flex-col bg-bg">
      <Header />
      <ChatMessages messages={messages} loading={loading} />
      <ChatInput onSend={handleSend} disabled={loading} />
    </div>
  );
}
