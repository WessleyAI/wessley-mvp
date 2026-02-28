# Spec: engine-ingest — Ingestion Pipeline

**Branch:** `spec/engine-ingest`
**Effort:** 3-4 days
**Priority:** P1 — Phase 2

---

## Scope

Full ingestion pipeline: receive scraped documents → **validate** → parse → chunk → embed → store in graph + vector DB. Uses `fn.Stage` pipeline composition. Consumes from NATS. Includes dead letter queue and observability taps.

### Files

```
engine/ingest/ingest.go       # Pipeline orchestration
engine/ingest/transform.go    # Parsing and chunking stages
engine/ingest/ingest_test.go
```

## Pipeline

```
ScrapedPost → Validate → Parse → Chunk → Embed → Store (Graph + Qdrant)
     │            │         │        │       │         │
     └── TapStage at each stage for logging/metrics ──┘
```

Each step is an `fn.Stage`:

```go
// Validate: run domain.ValidateScrapedPost() before processing
var Validate fn.Stage[scraper.ScrapedPost, scraper.ScrapedPost]

// Parse: extract structured content from raw scraped post
var Parse fn.Stage[scraper.ScrapedPost, ParsedDoc]

// Chunk: split content into embeddable chunks
var ChunkDoc fn.Stage[ParsedDoc, ChunkedDoc]

// Embed: generate embeddings via ml-worker gRPC
var Embed fn.Stage[ChunkedDoc, EmbeddedDoc]

// Store: save to Neo4j (components) + Qdrant (vectors)
var Store fn.Stage[EmbeddedDoc, string]  // returns doc ID

// Full pipeline with validation
var IngestPipeline = fn.Then(fn.Then(fn.Then(fn.Then(Validate, Parse), ChunkDoc), Embed), Store)
```

## Validation Stage

```go
// Runs domain.ValidateScrapedPost() on each incoming post.
// Invalid posts are rejected before any processing occurs.
var Validate = fn.StageFunc[scraper.ScrapedPost, scraper.ScrapedPost](
    func(ctx context.Context, post scraper.ScrapedPost) fn.Result[scraper.ScrapedPost] {
        if err := domain.ValidateScrapedPost(post); err != nil {
            return fn.Err[scraper.ScrapedPost](err)
        }
        return fn.Ok(post)
    },
)
```

## Dead Letter Queue

```go
// Messages that fail after 3 retries are published to a NATS DLQ subject.
const DLQSubject = "engine.ingest.dlq"
const MaxRetries = 3

// On repeated failure:
// 1. Increment retry count in message metadata
// 2. If retries >= MaxRetries → publish to DLQSubject with error details
// 3. Ack original message (prevent infinite retry loop)
```

## TapStage (Observability)

```go
// TapStage wraps any stage with logging/metrics at entry and exit.
// Does not modify the data — purely observational.
func TapStage[T any](name string, log *slog.Logger) fn.Stage[T, T] {
    return fn.StageFunc[T, T](func(ctx context.Context, t T) fn.Result[T] {
        log.Info("stage.enter", "stage", name)
        start := time.Now()
        defer func() {
            log.Info("stage.exit", "stage", name, "duration", time.Since(start))
        }()
        return fn.Ok(t)
    })
}
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
```

## NATS Consumer

```go
// Listens on "engine.ingest" subject
// Receives ScrapedPost, runs through pipeline, acks on success
// On failure after MaxRetries → publish to DLQ, ack
func StartConsumer(nc *nats.Conn, pipeline fn.Stage[scraper.ScrapedPost, string]) error
```

## Chunking Strategy

- Split by sentences (NLP-lite: split on `.!?` + newlines)
- Chunk size: 512 tokens with 50-token overlap
- Preserve document metadata in each chunk

## Deduplication

- Check doc ID in Redis before processing
- Skip already-ingested documents

## Acceptance Criteria

- [ ] End-to-end pipeline: ScrapedPost → Validate → Parse → Chunk → Embed → Store
- [ ] Validation stage rejects invalid posts via domain.ValidateScrapedPost()
- [ ] Dead letter queue: failed messages after 3 retries → NATS DLQ subject
- [ ] TapStage logging/metrics at each pipeline stage
- [ ] Sentence-based chunking with configurable size/overlap
- [ ] Calls ml-worker for embeddings via gRPC
- [ ] Stores components in Neo4j via engine/graph
- [ ] Stores embeddings in Qdrant via engine/semantic
- [ ] NATS consumer with ack/nak
- [ ] Deduplication via Redis
- [ ] Batch embedding (chunks grouped by 100)
- [ ] Unit tests with mocked gRPC + stores

## Dependencies

- `pkg/fn`, `engine/domain`, `engine/graph`, `engine/semantic`, `ml/client.go`, `pkg/natsutil`
- NATS, Redis, Neo4j, Qdrant

## Reference

- FINAL_ARCHITECTURE.md §8.7
