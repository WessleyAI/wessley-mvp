# Spec: api-server — Go HTTP API

**Branch:** `spec/api-server`
**Effort:** 2-3 days
**Priority:** P1 — Phase 3

---

## Scope

Thin Go HTTP server providing the chat API endpoint. Receives user questions, delegates to `engine/rag` for RAG orchestration, returns answers. Publishes async jobs to engine via NATS.

### Files

```
api/
├── server.go            # HTTP setup, routes, middleware chain
├── handler/
│   ├── chat.go          # POST /api/chat — thin: parse → rag.Query() → respond
│   ├── scrape.go        # POST /api/scrape — trigger scrape job
│   └── health.go        # GET /health — aggregated health check
├── auth/
│   └── auth.go          # API key or Supabase JWT middleware
cmd/api/main.go          # Entrypoint
```

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/chat` | RAG chat — delegates to engine/rag.Query() |
| POST | `/api/chat/stream` | SSE streaming RAG chat |
| POST | `/api/scrape` | Trigger scrape job (async via NATS) |
| GET | `/health` | Aggregated health check |

## Chat Handler (THIN)

The chat handler does NOT orchestrate RAG directly. It delegates entirely to `engine/rag`:

```go
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
    var req ChatRequest  // { question, vehicle_model?, top_k? }
    // 1. Parse & validate request
    // 2. Call engine/rag.Query(ctx, RAGQuery{...}) → RAGResponse
    // 3. Serialize response as JSON
}
```

No search logic, no prompt building, no LLM calls in the API layer.

## Health Check Aggregator

```go
// GET /health checks all downstream dependencies
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    // Check: self, NATS, ml-worker (gRPC), Neo4j, Qdrant, Redis
    // Return: { status: "ok"|"degraded"|"down", checks: {...} }
}

type HealthResponse struct {
    Status string            `json:"status"`
    Checks map[string]string `json:"checks"` // service → "ok"|"error: ..."
}
```

Checked services: self, NATS, ml-worker, Neo4j, Qdrant, Redis.

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
    Mode    string         `json:"mode"`
}
```

## Server Setup

```go
func NewServer(ragService *rag.RAGService, nc *nats.Conn, log *slog.Logger) *Server

func (s *Server) Handler() http.Handler {
    return mid.Chain(s.mux,
        mid.Recover(s.log),
        mid.Logger(s.log),
        mid.CORS("*"),
    )
}
```

## Middleware

- Recovery (panic → 500)
- Request logging (method, path, status, duration)
- CORS (permissive for MVP)
- Auth (API key header, optional for MVP)

## NATS Publishing

```go
natsutil.Publish(nc, "engine.scrape", ScrapeRequest{URLs: urls})
```

## Acceptance Criteria

- [ ] POST /api/chat delegates to engine/rag.Query() — no RAG logic in api
- [ ] SSE streaming endpoint for chat
- [ ] POST /api/scrape publishes async job to NATS
- [ ] GET /health aggregates: self + NATS + ml-worker + Neo4j + Qdrant + Redis
- [ ] Middleware chain (recover, log, CORS)
- [ ] Structured JSON error responses
- [ ] Graceful shutdown
- [ ] Configurable via env vars (port, NATS URL, etc.)
- [ ] Unit tests with mocked RAG service

## Dependencies

- `engine/rag`, `pkg/mid`, `pkg/natsutil`
- NATS

## Reference

- FINAL_ARCHITECTURE.md §8.8
