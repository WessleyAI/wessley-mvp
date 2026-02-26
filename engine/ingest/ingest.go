// Package ingest provides the ingestion pipeline that processes scraped content
// through validation, parsing, chunking, embedding, and storage stages.
package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/domain"
	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/engine/scraper"
	"github.com/WessleyAI/wessley-mvp/engine/semantic"
	"github.com/WessleyAI/wessley-mvp/pkg/fn"
	mlpb "github.com/WessleyAI/wessley-mvp/ml/proto/wessley/ml/v1"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

const (
	// IngestSubject is the NATS subject for incoming scraped posts.
	IngestSubject = "engine.ingest"
	// DLQSubject is the dead letter queue subject for failed messages.
	DLQSubject = "engine.ingest.dlq"
	// MaxRetries before sending to DLQ.
	MaxRetries = 3
	// EmbedBatchSize is the max chunks per embedding request.
	EmbedBatchSize = 100
)

// Deps holds the external dependencies for the ingestion pipeline.
type Deps struct {
	Embedder     mlpb.EmbedServiceClient
	VectorStore  *semantic.VectorStore
	GraphStore   *graph.GraphStore
	DeduplicateF func(ctx context.Context, docID string) (bool, error) // returns true if already ingested
	Logger       *slog.Logger
}

// --- Pipeline Stages ---

// Validate checks a ScrapedPost via domain validation.
var Validate fn.Stage[scraper.ScrapedPost, scraper.ScrapedPost] = func(ctx context.Context, post scraper.ScrapedPost) fn.Result[scraper.ScrapedPost] {
	if err := domain.ValidateScrapedPost(post); err != nil {
		return fn.Err[scraper.ScrapedPost](err)
	}
	return fn.Ok(post)
}

// Parse converts a ScrapedPost into a ParsedDoc.
var Parse fn.Stage[scraper.ScrapedPost, ParsedDoc] = func(_ context.Context, post scraper.ScrapedPost) fn.Result[ParsedDoc] {
	return fn.Ok(parsedDocFromPost(post))
}

// ChunkDoc splits a ParsedDoc into a ChunkedDoc.
var ChunkDoc fn.Stage[ParsedDoc, ChunkedDoc] = func(_ context.Context, doc ParsedDoc) fn.Result[ChunkedDoc] {
	chunks := chunkSentences(doc.ID, doc.Sentences, DefaultChunkSize, DefaultOverlap)
	if len(chunks) == 0 {
		// Single chunk fallback for short content.
		chunks = []Chunk{{Text: doc.Content, Index: 0, DocID: doc.ID}}
	}
	return fn.Ok(ChunkedDoc{ParsedDoc: doc, Chunks: chunks})
}

// NewEmbed creates an Embed stage that calls the ml-worker EmbedService.
func NewEmbed(client mlpb.EmbedServiceClient) fn.Stage[ChunkedDoc, EmbeddedDoc] {
	return func(ctx context.Context, doc ChunkedDoc) fn.Result[EmbeddedDoc] {
		embeddings := make([][]float32, len(doc.Chunks))

		// Batch in groups of EmbedBatchSize.
		for i := 0; i < len(doc.Chunks); i += EmbedBatchSize {
			end := i + EmbedBatchSize
			if end > len(doc.Chunks) {
				end = len(doc.Chunks)
			}

			texts := make([]string, end-i)
			for j, c := range doc.Chunks[i:end] {
				texts[j] = c.Text
			}

			resp, err := client.EmbedBatch(ctx, &mlpb.EmbedBatchRequest{Texts: texts})
			if err != nil {
				return fn.Err[EmbeddedDoc](fmt.Errorf("embed batch: %w", err))
			}

			for j, emb := range resp.GetEmbeddings() {
				embeddings[i+j] = emb.GetValues()
			}
		}

		return fn.Ok(EmbeddedDoc{ChunkedDoc: doc, Embeddings: embeddings})
	}
}

// NewStore creates a Store stage that writes to Neo4j and Qdrant.
func NewStore(vs *semantic.VectorStore, gs *graph.GraphStore) fn.Stage[EmbeddedDoc, string] {
	return func(ctx context.Context, doc EmbeddedDoc) fn.Result[string] {
		// Store component in knowledge graph.
		comp := graph.Component{
			ID:      doc.ID,
			Name:    doc.Title,
			Type:    "document",
			Vehicle: doc.Vehicle,
			Properties: map[string]string{
				"source": doc.Source,
			},
		}
		if err := gs.SaveComponent(ctx, comp); err != nil {
			return fn.Err[string](fmt.Errorf("graph save: %w", err))
		}

		// If VehicleInfo is present, ensure the vehicle hierarchy exists.
		if doc.VehicleInfo != nil {
			vi := graph.VehicleInfo{
				Make:  doc.VehicleInfo.Make,
				Model: doc.VehicleInfo.Model,
				Year:  doc.VehicleInfo.Year,
				Trim:  doc.VehicleInfo.Trim,
			}
			if err := gs.EnsureVehicleHierarchy(ctx, vi); err != nil {
				// Log but don't fail the pipeline for hierarchy errors.
				slog.Warn("ingest: vehicle hierarchy", "error", err, "doc_id", doc.ID)
			}
		}

		// Store vectors in Qdrant.
		records := make([]semantic.VectorRecord, len(doc.Chunks))
		for i, chunk := range doc.Chunks {
			// Generate deterministic UUID from doc ID and chunk index
			pointID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(fmt.Sprintf("%s-%d", doc.ID, chunk.Index))).String()
			payload := map[string]any{
				"content":     chunk.Text,
				"doc_id":      doc.ID,
				"source":      doc.Source,
				"vehicle":     doc.Vehicle,
				"chunk_index": chunk.Index,
			}
			// Add structured vehicle info to Qdrant payload.
			if doc.VehicleInfo != nil {
				payload["vehicle_make"] = doc.VehicleInfo.Make
				payload["vehicle_model"] = doc.VehicleInfo.Model
				payload["vehicle_year"] = doc.VehicleInfo.Year
				if doc.VehicleInfo.Trim != "" {
					payload["vehicle_trim"] = doc.VehicleInfo.Trim
				}
			}
			records[i] = semantic.VectorRecord{
				ID:        pointID,
				Embedding: doc.Embeddings[i],
				Payload:   payload,
			}
		}
		if err := vs.Upsert(ctx, records); err != nil {
			return fn.Err[string](fmt.Errorf("vector upsert: %w", err))
		}

		return fn.Ok(doc.ID)
	}
}

// TapStage wraps any stage with logging at entry and exit.
func TapStage[T any](name string, log *slog.Logger) fn.Stage[T, T] {
	return fn.TapStage(func(ctx context.Context, t T) {
		log.Info("stage.enter", "stage", name)
		// Note: exit logging happens via the deferred log below.
		// For simplicity, entry is logged via TapStage; duration is approximate.
	})
}

// LoggedTap returns a stage that logs entry/exit with duration.
func LoggedTap[T any](name string, log *slog.Logger) fn.Stage[T, T] {
	return func(ctx context.Context, t T) fn.Result[T] {
		log.Info("stage.enter", "stage", name)
		start := time.Now()
		defer func() {
			log.Info("stage.exit", "stage", name, "duration", time.Since(start))
		}()
		return fn.Ok(t)
	}
}

// NewPipeline constructs the full ingestion pipeline with all stages wired.
func NewPipeline(deps Deps) fn.Stage[scraper.ScrapedPost, string] {
	log := deps.Logger
	if log == nil {
		log = slog.Default()
	}

	// Compose: Validate → Parse → Chunk → Embed → Store
	// with logging taps between stages.
	validated := fn.Then(LoggedTap[scraper.ScrapedPost]("validate", log), Validate)
	parsed := fn.Then(validated, fn.Then(LoggedTap[scraper.ScrapedPost]("parse", log), Parse))
	chunked := fn.Then(parsed, fn.Then(LoggedTap[ParsedDoc]("chunk", log), ChunkDoc))
	embedded := fn.Then(chunked, fn.Then(LoggedTap[ChunkedDoc]("embed", log), NewEmbed(deps.Embedder)))
	stored := fn.Then(embedded, fn.Then(LoggedTap[EmbeddedDoc]("store", log), NewStore(deps.VectorStore, deps.GraphStore)))

	return stored
}

// dlqMessage is published to the DLQ on repeated failure.
type dlqMessage struct {
	Post    scraper.ScrapedPost `json:"post"`
	Error   string              `json:"error"`
	Retries int                 `json:"retries"`
}

// StartConsumer starts a NATS JetStream consumer that runs scraped posts
// through the ingestion pipeline with retry and DLQ support.
func StartConsumer(nc *nats.Conn, deps Deps) (*nats.Subscription, error) {
	pipeline := NewPipeline(deps)
	log := deps.Logger
	if log == nil {
		log = slog.Default()
	}

	return nc.Subscribe(IngestSubject, func(msg *nats.Msg) {
		var post scraper.ScrapedPost
		if err := json.Unmarshal(msg.Data, &post); err != nil {
			log.Error("ingest: unmarshal failed", "error", err)
			return
		}

		ctx := context.Background()

		// Deduplication check.
		if deps.DeduplicateF != nil {
			docID := post.Source + ":" + post.SourceID
			exists, err := deps.DeduplicateF(ctx, docID)
			if err != nil {
				log.Warn("ingest: dedup check failed", "error", err)
			} else if exists {
				log.Info("ingest: skipping duplicate", "doc_id", docID)
				if msg.Reply != "" {
					_ = msg.Ack()
				}
				return
			}
		}

		// Get retry count from header.
		retries := 0
		if msg.Header != nil {
			if v := msg.Header.Get("X-Retry-Count"); v != "" {
				fmt.Sscanf(v, "%d", &retries)
			}
		}

		result := pipeline(ctx, post)
		if result.IsErr() {
			_, pipeErr := result.Unwrap()
			retries++
			log.Error("ingest: pipeline failed",
				"error", pipeErr,
				"source_id", post.SourceID,
				"retry", retries,
			)

			if retries >= MaxRetries {
				// Send to DLQ.
				dlq := dlqMessage{
					Post:    post,
					Error:   pipeErr.Error(),
					Retries: retries,
				}
				data, _ := json.Marshal(dlq)
				if err := nc.Publish(DLQSubject, data); err != nil {
					log.Error("ingest: DLQ publish failed", "error", err)
				}
			} else {
				// Re-publish with incremented retry count.
				retryMsg := nats.NewMsg(IngestSubject)
				retryMsg.Data = msg.Data
				retryMsg.Header = nats.Header{}
				retryMsg.Header.Set("X-Retry-Count", fmt.Sprintf("%d", retries))
				if err := nc.PublishMsg(retryMsg); err != nil {
					log.Error("ingest: retry publish failed", "error", err)
				}
			}
		} else {
			docID, _ := result.Unwrap()
			log.Info("ingest: success", "doc_id", docID)
		}

		// Ack if JetStream.
		if msg.Reply != "" {
			_ = msg.Ack()
		}
	})
}
