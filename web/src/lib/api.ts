const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export interface Source {
  title: string;
  source_type: string;
  score: number;
  url?: string;
}

export interface ChatResponse {
  answer: string;
  sources: Source[];
  model?: string;
}

export async function chat(question: string, vehicleModel?: string): Promise<ChatResponse> {
  const res = await fetch(`${API_URL}/api/chat`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ question, vehicle_model: vehicleModel }),
  });
  if (!res.ok) {
    throw new Error(`API error: ${res.status}`);
  }
  return res.json();
}

export async function* chatStream(
  question: string,
  vehicleModel?: string,
): AsyncGenerator<string> {
  const res = await fetch(`${API_URL}/api/chat/stream`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ question, vehicle_model: vehicleModel }),
  });

  if (!res.ok) {
    throw new Error(`API error: ${res.status}`);
  }

  const reader = res.body?.getReader();
  if (!reader) throw new Error("No response body");

  const decoder = new TextDecoder();
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    yield decoder.decode(value, { stream: true });
  }
}
