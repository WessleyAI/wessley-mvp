# Spec: web-chat — Next.js Chat Frontend

**Branch:** `spec/web-chat`
**Effort:** 2-3 days
**Priority:** P2 — Phase 4

---

## Scope

Minimal Next.js 15 chat frontend. One page: ask a question about vehicle electrical systems, get an AI answer with sources. Clean, dark theme.

### Files

```
web/
├── src/app/
│   ├── layout.tsx        # Root layout, fonts, metadata
│   ├── page.tsx          # Landing → redirects to /chat
│   └── chat/
│       └── page.tsx      # Chat page
├── src/components/
│   ├── ChatInput.tsx     # Message input with send button
│   ├── ChatMessages.tsx  # Message list (user + AI)
│   ├── Message.tsx       # Single message bubble
│   ├── Sources.tsx       # Source citations under AI response
│   └── Header.tsx        # Simple header with logo/title
├── src/lib/
│   └── api.ts            # API client (POST /api/chat)
├── package.json
├── tailwind.config.ts
├── next.config.ts
└── Dockerfile
```

## UI

```
┌──────────────────────────────────────┐
│  ⚡ Wessley — Vehicle AI             │
├──────────────────────────────────────┤
│                                      │
│  Welcome! Ask me anything about      │
│  vehicle electrical systems.         │
│                                      │
│  ┌─ User ──────────────────────────┐ │
│  │ Where is the starter relay in   │ │
│  │ a 2004 Mitsubishi Pajero?       │ │
│  └─────────────────────────────────┘ │
│                                      │
│  ┌─ Wessley ───────────────────────┐ │
│  │ The starter relay in the 2004   │ │
│  │ Pajero is located in the main   │ │
│  │ fuse box under the hood...      │ │
│  │                                 │ │
│  │ Sources:                        │ │
│  │ • r/MechanicAdvice (0.92)       │ │
│  │ • ChrisFix YouTube (0.87)      │ │
│  └─────────────────────────────────┘ │
│                                      │
│  ┌──────────────────────────┐ [Send] │
│  │ Ask about any vehicle... │        │
│  └──────────────────────────┘        │
└──────────────────────────────────────┘
```

## Features

- Dark theme (bg #0b0f1a, consistent with architecture diagrams)
- Streaming responses (SSE from api)
- Source citations with relevance scores
- Starter prompts ("Where is the alternator in a...", "How do I diagnose...", "What fuse controls...")
- Mobile responsive
- No auth for MVP — open access
- Loading states and error handling

## API Client

```typescript
async function chat(question: string, vehicleModel?: string): Promise<ChatResponse> {
    const res = await fetch(`${API_URL}/api/chat`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ question, vehicle_model: vehicleModel }),
    })
    return res.json()
}

// Streaming variant
async function* chatStream(question: string): AsyncGenerator<string> {
    const res = await fetch(`${API_URL}/api/chat/stream`, { ... })
    const reader = res.body.getReader()
    // yield chunks
}
```

## Acceptance Criteria

- [ ] Single chat page with input + messages
- [ ] Sends question to API, displays response
- [ ] Shows source citations with scores
- [ ] Streaming response display
- [ ] Starter prompts on empty state
- [ ] Dark theme
- [ ] Mobile responsive
- [ ] Error states (API down, no results)
- [ ] Dockerfile for containerized deployment
- [ ] No auth required

## Dependencies

- API server running on configured URL
- Next.js 15, Tailwind CSS, TypeScript

## Reference

- FINAL_ARCHITECTURE.md §2-3 (service architecture)
