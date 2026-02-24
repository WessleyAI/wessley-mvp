import type { Source } from "@/lib/api";
import Sources from "./Sources";

export interface MessageData {
  role: "user" | "assistant";
  content: string;
  sources?: Source[];
}

export default function Message({ msg }: { msg: MessageData }) {
  const isUser = msg.role === "user";

  return (
    <div className={`flex ${isUser ? "justify-end" : "justify-start"}`}>
      <div
        className={`max-w-[85%] rounded-2xl px-4 py-3 ${
          isUser
            ? "bg-accent text-white"
            : "bg-surface border border-border text-slate-200"
        }`}
      >
        {!isUser && (
          <p className="mb-1 text-xs font-medium text-slate-500">Wessley</p>
        )}
        <p className="whitespace-pre-wrap text-sm leading-relaxed">{msg.content}</p>
        {msg.sources && <Sources sources={msg.sources} />}
      </div>
    </div>
  );
}
