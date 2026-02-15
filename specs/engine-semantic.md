# Spec: engine-semantic — Qdrant Vector Store (Sole Owner)

**Branch:** `spec/engine-semantic`
**Effort:** 2-3 days
**Priority:** P1 — Phase 3

---

## Scope

Qdrant vector store integration — the **sole owner** of all Qdrant operations. Handles both embedding storage during ingestion AND similarity search during RAG. No other service accesses Qdrant directly (ml-worker does NOT touch Qdrant).

### Files

```
engine/semantic/store.go       # Qdrant operations (read + write)
engine/semantic/model.go       # Search types
engine/semantic/store_test.go
```

## Key Types

```go
type VectorStore struct {
    client qdrant.Client
    collection string
}

type SearchResult struct {
    ID      string            `json:"id"`
    Score   float32           `json:"score"`
    Content string            `json:"content"`
    DocID   string            `json:"doc_id"`
    Source  string            `json:"source"`
    Meta    map[string]string `json:"meta"`
}

type VectorRecord struct {
    ID        string
    Embedding []float32
    Payload   map[string]any  // content, doc_id, source, vehicle, chunk_index
}
```

## Operations

```go
func New(addr string, collection string) (*VectorStore, error)

// Ensure collection exists with correct dimensions
func (v *VectorStore) EnsureCollection(ctx context.Context, dims int) error

// === WRITE (ingestion) ===

// Store embeddings — called by engine/ingest during ingestion pipeline
func (v *VectorStore) Upsert(ctx context.Context, records []VectorRecord) error

// Delete by doc ID (for re-ingestion)
func (v *VectorStore) DeleteByDocID(ctx context.Context, docID string) error

// === READ (RAG search) ===

// Search by embedding vector — called by engine/rag during query
func (v *VectorStore) Search(ctx context.Context, embedding []float32, topK int) ([]SearchResult, error)

// Search with metadata filter (e.g. vehicle, source)
func (v *VectorStore) SearchFiltered(ctx context.Context, embedding []float32, topK int, filters map[string]string) ([]SearchResult, error)
```

## Ownership Model

```
engine/semantic is the SOLE Qdrant owner:
  ├── WRITES: called by engine/ingest (ingestion pipeline stores embeddings)
  └── READS:  called by engine/rag (RAG pipeline searches vectors)

ml-worker does NOT access Qdrant.
ml-worker only provides: embeddings (EmbedService) + chat (ChatService).
```

## Collection Schema

```
Collection: "wessley_docs"
Vector: 384 dims (all-MiniLM-L6-v2) or 768 (nomic-embed)
Payload indices: doc_id, source, vehicle_make, vehicle_model, vehicle_year
Distance: Cosine
```

## Acceptance Criteria

- [ ] Create/ensure Qdrant collection on startup
- [ ] Batch upsert vectors with metadata (ingestion path)
- [ ] Similarity search with top-K (RAG path)
- [ ] Filtered search by vehicle/source
- [ ] Delete by doc ID
- [ ] Sole owner of Qdrant — no other service accesses it directly
- [ ] Handles connection errors gracefully
- [ ] Unit tests with testcontainers or mock

## Dependencies

- Qdrant Go client (`github.com/qdrant/go-client`)
- Qdrant (Docker)

## Reference

- FINAL_ARCHITECTURE.md §8.3
