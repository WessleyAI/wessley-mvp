# Spec: engine-rag — RAG Orchestration

**Branch:** `spec/engine-rag`
**Effort:** 2-3 days
**Priority:** P1 — Phase 3

---

## Scope

RAG (Retrieval-Augmented Generation) orchestration layer extracted from the API server. Owns the full query flow: embed → search → enrich → prompt → generate → format. Lives in the engine as a reusable pipeline stage.

### Files

```
engine/rag/rag.go          # RAGService + Query method
engine/rag/rag_test.go
```

## RAGService

```go
type RAGService struct {
    ml       *ml.Client            // gRPC client for embeddings + LLM
    semantic *semantic.VectorStore  // Qdrant search
    graph    *graph.Store           // Neo4j context enrichment
    log      *slog.Logger
}

func New(ml *ml.Client, semantic *semantic.VectorStore, graph *graph.Store, log *slog.Logger) *RAGService
```

## Query Flow

```go
type RAGQuery struct {
    Question     string            `json:"question"`
    VehicleModel string            `json:"vehicle_model,omitempty"`
    TopK         int               `json:"top_k,omitempty"`  // default 5
    Filters      map[string]string `json:"filters,omitempty"`
}

type RAGResponse struct {
    Answer  string         `json:"answer"`
    Sources []SearchSource `json:"sources"`
    Mode    string         `json:"mode"`  // "full", "graph-only", "raw-results"
}

func (s *RAGService) Query(ctx context.Context, q RAGQuery) fn.Result[RAGResponse]
```

### Full flow

1. **Embed query** — call `ml.Embed(q.Question)` → `[]float32`
2. **Vector search** — call `semantic.Search(embedding, topK)` → sources
3. **Graph context** — call `graph.GetContext(q.VehicleModel)` → component relationships
4. **Build prompt** — combine sources + graph context into LLM prompt
5. **LLM generate** — call `ml.Chat(prompt)` → answer
6. **Format** — assemble `RAGResponse` with answer + sources + mode

## Graceful Degradation

| Failure | Fallback | Mode |
|---------|----------|------|
| Vector search fails | Use graph context only for LLM | `"graph-only"` |
| LLM fails | Return raw search results without generated answer | `"raw-results"` |
| Both vector + graph fail | Return error | error |

## Pipeline Stage

Internally implemented as an `fn.Stage[RAGQuery, RAGResponse]`:

```go
// RAGStage returns the RAG pipeline as an fn.Stage for composition
func (s *RAGService) Stage() fn.Stage[RAGQuery, RAGResponse] {
    return fn.StageFunc[RAGQuery, RAGResponse](func(ctx context.Context, q RAGQuery) fn.Result[RAGResponse] {
        return s.Query(ctx, q)
    })
}
```

## Acceptance Criteria

- [ ] Full RAG flow: embed → search → graph → prompt → LLM → response
- [ ] Graceful degradation on vector search failure (graph-only mode)
- [ ] Graceful degradation on LLM failure (raw-results mode)
- [ ] Configurable top-K and filters
- [ ] Vehicle-aware graph context enrichment
- [ ] Implements `fn.Stage[RAGQuery, RAGResponse]`
- [ ] Structured logging at each step
- [ ] Unit tests with mocked ML client, semantic store, graph store

## Dependencies

- `ml/client.go`, `engine/semantic`, `engine/graph`, `pkg/fn`

## Reference

- FINAL_ARCHITECTURE.md §8.8
