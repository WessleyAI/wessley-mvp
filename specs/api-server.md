# Spec: api-server — Go HTTP API

**Branch:** `spec/api-server`
**Effort:** 3-4 days
**Priority:** P1 — Phase 3

---

## Scope

Go HTTP server providing the chat API endpoint. Receives user questions, orchestrates RAG (search + LLM), returns answers. Publishes async jobs to engine via NATS.

### Files

```
api/
├── server.go            # HTTP setup, routes, middleware chain
├── handler/
│   ├── chat.go          # POST /api/chat — RAG query
│   ├── scrape.go        # POST /api/scrape — trigger scrape job
│   ├── health.go        # GET /health
│   └── search.go        # POST /api/search — vector search only
├── auth/
│   └── auth.go          # API key or Supabase JWT middleware
cmd/api/main.go          # Entrypoint
```

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/chat` | RAG chat — question in, answer + sources out |
| POST | `/api/chat/stream` | SSE streaming RAG chat |
| POST | `/api/search` | Vector search only (no LLM) |
| POST | `/api/scrape` | Trigger scrape job (async via NATS) |
| GET | `/health` | Health check (self + downstream) |

## Chat Flow

```go
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
    var req ChatRequest  // { question, vehicle_model?, top_k? }
    // 1. Validate
    // 2. Call ml-worker: Search(question, topK) → sources
    // 3. Call ml-worker: Chat(question, sources) → answer
    // 4. Return { answer, sources }
}
```

## Key Types

```go
type ChatRequest struct {
    Question     string `json:"question"`
    VehicleModel string `json:"vehicle_model,omitempty"`
    TopK         int    `json:"top_k,omitempty"`  // default 5
}

type ChatResponse struct {
    Answer  string         `json:"answer"`
    Sources []SearchSource `json:"sources"`
}

type SearchSource struct {
    Content string  `json:"content"`
    Source  string  `json:"source"`
    Score   float32 `json:"score"`
}
```

## Server Setup

```go
func NewServer(mlClient *ml.Client, nc *nats.Conn, log *slog.Logger) *Server

func (s *Server) Handler() http.Handler {
    return mid.Chain(s.mux,
        mid.Recover(s.log),
        mid.Logger(s.log),
        mid.CORS("*"),
        // mid.Auth(s.authValidator),  // optional for MVP
    )
}
```

## Middleware

- Recovery (panic → 500)
- Request logging (method, path, status, duration)
- CORS (permissive for MVP)
- Auth (API key header, optional for MVP — can skip initially)
- Rate limiting (later)

## NATS Publishing

```go
// Async job triggers
natsutil.Publish(nc, "engine.scrape", ScrapeRequest{URLs: urls})
natsutil.Publish(nc, "engine.ingest", ScrapedPost{...})
```

## Acceptance Criteria

- [ ] POST /api/chat returns RAG answer + sources
- [ ] SSE streaming endpoint for chat
- [ ] POST /api/search for vector-only search
- [ ] POST /api/scrape publishes async job to NATS
- [ ] GET /health checks self + ml-worker + NATS
- [ ] Middleware chain (recover, log, CORS)
- [ ] Structured JSON error responses
- [ ] Graceful shutdown
- [ ] Configurable via env vars (port, ml-worker addr, NATS URL)
- [ ] Unit tests with mocked ml client

## Dependencies

- `pkg/mid`, `pkg/natsutil`, `ml/client.go`
- NATS, ml-worker (gRPC)

## Reference

- FINAL_ARCHITECTURE.md §8.8
