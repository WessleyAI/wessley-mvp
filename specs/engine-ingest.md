# Spec: engine-ingest — Ingestion Pipeline

**Branch:** `spec/engine-ingest`
**Effort:** 3-4 days
**Priority:** P1 — Phase 2

---

## Scope

Full ingestion pipeline: receive scraped documents → parse → chunk → embed → store in graph + vector DB. Uses `fn.Stage` pipeline composition. Consumes from NATS.

### Files

```
engine/ingest/ingest.go       # Pipeline orchestration
engine/ingest/transform.go    # Parsing and chunking stages
engine/ingest/ingest_test.go
```

## Pipeline

```
ScrapedPost → Parse → Chunk → Embed → Store (Graph + Qdrant)
```

Each step is an `fn.Stage`:

```go
// Parse: extract structured content from raw scraped post
var Parse fn.Stage[scraper.ScrapedPost, ParsedDoc]

// Chunk: split content into embeddable chunks
var ChunkDoc fn.Stage[ParsedDoc, ChunkedDoc]

// Embed: generate embeddings via ml-worker gRPC
var Embed fn.Stage[ChunkedDoc, EmbeddedDoc]  // calls ml-worker

// Store: save to Neo4j (components) + Qdrant (vectors)
var Store fn.Stage[EmbeddedDoc, string]  // returns doc ID

// Full pipeline
var IngestPipeline = fn.Then(fn.Then(fn.Then(Parse, ChunkDoc), Embed), Store)
```

## Key Types

```go
type ParsedDoc struct {
    ID        string
    Source    string
    Title     string
    Content   string
    Vehicle   *scraper.VehicleSignature
    Sentences []string
    Metadata  map[string]string
}

type ChunkedDoc struct {
    ParsedDoc
    Chunks []Chunk
}

type Chunk struct {
    Text   string
    Index  int
    DocID  string
}

type EmbeddedDoc struct {
    ChunkedDoc
    Embeddings [][]float32
}

type IngestReport struct {
    DocsProcessed    int `json:"docs_processed"`
    ChunksCreated    int `json:"chunks_created"`
    ComponentsStored int `json:"components_stored"`
    EmbeddingsStored int `json:"embeddings_stored"`
}
```

## NATS Consumer

```go
// Listens on "engine.ingest" subject
// Receives ScrapedPost, runs through pipeline, acks on success
func StartConsumer(nc *nats.Conn, pipeline fn.Stage[scraper.ScrapedPost, string]) error
```

## Chunking Strategy

- Split by sentences (NLP-lite: split on `.!?` + newlines)
- Chunk size: 512 tokens with 50-token overlap
- Preserve document metadata in each chunk

## Deduplication

- Check doc ID in Redis before processing
- Skip already-ingested documents
- Simple `if seen { ack; return }` — no pattern needed

## Acceptance Criteria

- [ ] End-to-end pipeline: ScrapedPost → Graph + Qdrant
- [ ] Sentence-based chunking with configurable size/overlap
- [ ] Calls ml-worker for embeddings via gRPC
- [ ] Stores components in Neo4j via engine/graph
- [ ] Stores embeddings in Qdrant
- [ ] NATS consumer with ack/nak
- [ ] Deduplication via Redis
- [ ] Batch embedding (chunks grouped by 100)
- [ ] IngestReport with counts
- [ ] Unit tests with mocked gRPC + stores

## Dependencies

- `pkg/fn`, `engine/graph`, `engine/semantic`, `ml/client.go`, `pkg/natsutil`
- NATS, Redis, Neo4j, Qdrant

## Reference

- FINAL_ARCHITECTURE.md §8.7
