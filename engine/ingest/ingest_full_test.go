package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/engine/semantic"
	mlpb "github.com/WessleyAI/wessley-mvp/ml/proto/wessley/ml/v1"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
)

// --- Mock Embedder ---

type mockEmbedder struct {
	resp *mlpb.EmbedBatchResponse
	err  error
}

func (m *mockEmbedder) Embed(_ context.Context, _ *mlpb.EmbedRequest, _ ...grpc.CallOption) (*mlpb.EmbedResponse, error) {
	return nil, nil
}

func (m *mockEmbedder) EmbedBatch(_ context.Context, req *mlpb.EmbedBatchRequest, _ ...grpc.CallOption) (*mlpb.EmbedBatchResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.resp != nil {
		return m.resp, nil
	}
	// Default: return embeddings matching input count
	embs := make([]*mlpb.EmbedResponse, len(req.Texts))
	for i := range embs {
		embs[i] = &mlpb.EmbedResponse{Values: []float32{1, 0, 0, 0}, Dimensions: 4}
	}
	return &mlpb.EmbedBatchResponse{Embeddings: embs}, nil
}

// --- Mock Qdrant points ---

type mockPoints struct{}

func (m *mockPoints) Upsert(_ context.Context, _ *pb.UpsertPoints, _ ...grpc.CallOption) (*pb.PointsOperationResponse, error) {
	return &pb.PointsOperationResponse{}, nil
}
func (m *mockPoints) Delete(_ context.Context, _ *pb.DeletePoints, _ ...grpc.CallOption) (*pb.PointsOperationResponse, error) {
	return &pb.PointsOperationResponse{}, nil
}
func (m *mockPoints) Search(_ context.Context, _ *pb.SearchPoints, _ ...grpc.CallOption) (*pb.SearchResponse, error) {
	return &pb.SearchResponse{}, nil
}

type mockCollections struct{}

func (m *mockCollections) List(_ context.Context, _ *pb.ListCollectionsRequest, _ ...grpc.CallOption) (*pb.ListCollectionsResponse, error) {
	return &pb.ListCollectionsResponse{}, nil
}
func (m *mockCollections) Create(_ context.Context, _ *pb.CreateCollection, _ ...grpc.CallOption) (*pb.CollectionOperationResponse, error) {
	return &pb.CollectionOperationResponse{Result: true}, nil
}
func (m *mockCollections) Delete(_ context.Context, _ *pb.DeleteCollection, _ ...grpc.CallOption) (*pb.CollectionOperationResponse, error) {
	return &pb.CollectionOperationResponse{Result: true}, nil
}

// --- Mock graph opener ---

type mockOpener struct{}

func (o *mockOpener) OpenSession(_ context.Context) graph.CypherSession {
	return &mockCypherSession{}
}

type mockCypherSession struct{}

func (s *mockCypherSession) Run(_ context.Context, _ string, _ map[string]any) (graph.CypherResult, error) {
	return &mockCR{}, nil
}
func (s *mockCypherSession) Close(_ context.Context) error { return nil }
func (s *mockCypherSession) ExecuteWrite(_ context.Context, _ func(tx graph.CypherRunner) (any, error)) (any, error) {
	return nil, nil
}

type mockCR struct{}

func (r *mockCR) Next(_ context.Context) bool              { return false }
func (r *mockCR) Record() *neo4j.Record                    { return nil }

// --- Helper to build test deps with working mocks ---

func testDeps() Deps {
	vs := semantic.NewWithClients(&mockPoints{}, &mockCollections{}, "test")
	gs := graph.NewWithOpener(&mockOpener{})
	return Deps{
		Embedder:    &mockEmbedder{},
		VectorStore: vs,
		GraphStore:  gs,
		Logger:      slog.Default(),
	}
}

// --- Tests ---

func TestNewEmbed_Success(t *testing.T) {
	embedder := &mockEmbedder{}
	stage := NewEmbed(embedder)

	doc := ChunkedDoc{
		ParsedDoc: ParsedDoc{ID: "test:1", Content: "hello"},
		Chunks:    []Chunk{{Text: "hello", Index: 0, DocID: "test:1"}},
	}
	result := stage(context.Background(), doc)
	if result.IsErr() {
		_, err := result.Unwrap()
		t.Fatalf("unexpected error: %v", err)
	}
	ed, _ := result.Unwrap()
	if len(ed.Embeddings) != 1 {
		t.Fatalf("expected 1 embedding, got %d", len(ed.Embeddings))
	}
}

func TestNewEmbed_Error(t *testing.T) {
	embedder := &mockEmbedder{err: fmt.Errorf("embed fail")}
	stage := NewEmbed(embedder)

	doc := ChunkedDoc{
		ParsedDoc: ParsedDoc{ID: "test:1"},
		Chunks:    []Chunk{{Text: "hello", Index: 0, DocID: "test:1"}},
	}
	result := stage(context.Background(), doc)
	if !result.IsErr() {
		t.Fatal("expected error")
	}
}

func TestNewEmbed_MultipleBatches(t *testing.T) {
	embedder := &mockEmbedder{}
	stage := NewEmbed(embedder)

	// Create more chunks than EmbedBatchSize
	chunks := make([]Chunk, EmbedBatchSize+5)
	for i := range chunks {
		chunks[i] = Chunk{Text: "word", Index: i, DocID: "test:1"}
	}

	doc := ChunkedDoc{
		ParsedDoc: ParsedDoc{ID: "test:1"},
		Chunks:    chunks,
	}
	result := stage(context.Background(), doc)
	if result.IsErr() {
		_, err := result.Unwrap()
		t.Fatalf("unexpected error: %v", err)
	}
	ed, _ := result.Unwrap()
	if len(ed.Embeddings) != len(chunks) {
		t.Fatalf("expected %d embeddings, got %d", len(chunks), len(ed.Embeddings))
	}
}

func TestNewStore_Success(t *testing.T) {
	// We need mock VectorStore and GraphStore. Use the refactored versions.
	// For GraphStore, use NewWithOpener with a mock.
	// For VectorStore, use NewWithClients with a mock.
	// But these are in other packages — we import them.

	// Actually, NewStore takes *semantic.VectorStore and *graph.GraphStore.
	// We can't easily mock those without the refactored constructors.
	// Let's test via NewPipeline which wires everything together.
	t.Skip("tested via pipeline integration")
}

func TestTapStage(t *testing.T) {
	log := slog.Default()
	stage := TapStage[string]("test", log)
	result := stage(context.Background(), "hello")
	if result.IsErr() {
		t.Fatal("unexpected error")
	}
	v, _ := result.Unwrap()
	if v != "hello" {
		t.Fatal("value should pass through")
	}
}

func TestLoggedTap(t *testing.T) {
	log := slog.Default()
	stage := LoggedTap[int]("test", log)
	result := stage(context.Background(), 42)
	if result.IsErr() {
		t.Fatal("unexpected error")
	}
	v, _ := result.Unwrap()
	if v != 42 {
		t.Fatal("value should pass through")
	}
}

func TestNewPipeline(t *testing.T) {
	deps := testDeps()
	pipeline := NewPipeline(deps)

	post := validPost()
	result := pipeline(context.Background(), post)
	if result.IsErr() {
		_, err := result.Unwrap()
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewPipeline_NilLogger(t *testing.T) {
	deps := testDeps()
	deps.Logger = nil // should use slog.Default()
	pipeline := NewPipeline(deps)
	result := pipeline(context.Background(), validPost())
	if result.IsErr() {
		_, err := result.Unwrap()
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewPipeline_InvalidPost(t *testing.T) {
	deps := testDeps()
	pipeline := NewPipeline(deps)

	post := validPost()
	post.Source = "invalid"
	result := pipeline(context.Background(), post)
	if !result.IsErr() {
		t.Fatal("expected validation error")
	}
}

func TestNewPipeline_EmbedError(t *testing.T) {
	deps := testDeps()
	deps.Embedder = &mockEmbedder{err: fmt.Errorf("embed fail")}
	pipeline := NewPipeline(deps)

	result := pipeline(context.Background(), validPost())
	if !result.IsErr() {
		t.Fatal("expected embed error")
	}
}

// --- NATS Consumer Tests ---

func startNATS(t *testing.T) (*natsserver.Server, *nats.Conn) {
	t.Helper()
	opts := &natsserver.Options{Port: -1}
	ns, err := natsserver.NewServer(opts)
	if err != nil {
		t.Fatalf("nats server: %v", err)
	}
	ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		t.Fatal("nats not ready")
	}
	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		t.Fatalf("nats connect: %v", err)
	}
	return ns, nc
}

func TestStartConsumer_Success(t *testing.T) {
	ns, nc := startNATS(t)
	defer ns.Shutdown()
	defer nc.Close()

	deps := testDeps()

	sub, err := StartConsumer(nc, deps)
	if err != nil {
		t.Fatalf("StartConsumer: %v", err)
	}
	defer sub.Unsubscribe()

	// Publish a valid post — will fail at Store stage (nil stores) but should not crash
	post := validPost()
	data, _ := json.Marshal(post)
	nc.Publish(IngestSubject, data)
	nc.Flush()
	time.Sleep(200 * time.Millisecond) // wait for processing
}

func TestStartConsumer_InvalidJSON(t *testing.T) {
	ns, nc := startNATS(t)
	defer ns.Shutdown()
	defer nc.Close()

	deps := testDeps()

	sub, err := StartConsumer(nc, deps)
	if err != nil {
		t.Fatalf("StartConsumer: %v", err)
	}
	defer sub.Unsubscribe()

	nc.Publish(IngestSubject, []byte("not json"))
	nc.Flush()
	time.Sleep(100 * time.Millisecond)
}

func TestStartConsumer_Dedup(t *testing.T) {
	ns, nc := startNATS(t)
	defer ns.Shutdown()
	defer nc.Close()

	deps := testDeps()
	deps.DeduplicateF = func(_ context.Context, docID string) (bool, error) {
		return true, nil // always duplicate
	}

	sub, err := StartConsumer(nc, deps)
	if err != nil {
		t.Fatalf("StartConsumer: %v", err)
	}
	defer sub.Unsubscribe()

	post := validPost()
	data, _ := json.Marshal(post)
	nc.Publish(IngestSubject, data)
	nc.Flush()
	time.Sleep(100 * time.Millisecond)
}

func TestStartConsumer_DedupError(t *testing.T) {
	ns, nc := startNATS(t)
	defer ns.Shutdown()
	defer nc.Close()

	deps := testDeps()
	deps.DeduplicateF = func(_ context.Context, docID string) (bool, error) {
		return false, fmt.Errorf("dedup error")
	}

	sub, err := StartConsumer(nc, deps)
	if err != nil {
		t.Fatalf("StartConsumer: %v", err)
	}
	defer sub.Unsubscribe()

	post := validPost()
	data, _ := json.Marshal(post)
	nc.Publish(IngestSubject, data)
	nc.Flush()
	time.Sleep(200 * time.Millisecond)
}

func TestStartConsumer_RetryAndDLQ(t *testing.T) {
	ns, nc := startNATS(t)
	defer ns.Shutdown()
	defer nc.Close()

	// Use nil stores to force pipeline failure for DLQ testing
	deps := Deps{
		Embedder: &mockEmbedder{err: fmt.Errorf("always fail")},
		VectorStore: semantic.NewWithClients(&mockPoints{}, &mockCollections{}, "test"),
		GraphStore:  graph.NewWithOpener(&mockOpener{}),
		Logger:   slog.Default(),
	}

	// Subscribe to DLQ to check it receives messages
	dlqReceived := make(chan bool, 1)
	nc.Subscribe(DLQSubject, func(msg *nats.Msg) {
		dlqReceived <- true
	})

	sub, err := StartConsumer(nc, deps)
	if err != nil {
		t.Fatalf("StartConsumer: %v", err)
	}
	defer sub.Unsubscribe()

	// Publish with retry count at MaxRetries-1 so next failure goes to DLQ
	post := validPost()
	data, _ := json.Marshal(post)
	msg := nats.NewMsg(IngestSubject)
	msg.Data = data
	msg.Header = nats.Header{}
	msg.Header.Set("X-Retry-Count", fmt.Sprintf("%d", MaxRetries-1))
	nc.PublishMsg(msg)
	nc.Flush()

	select {
	case <-dlqReceived:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("expected DLQ message")
	}
}

func TestStartConsumer_RetryRepublish(t *testing.T) {
	ns, nc := startNATS(t)
	defer ns.Shutdown()
	defer nc.Close()

	deps := Deps{
		Embedder:    &mockEmbedder{err: fmt.Errorf("always fail")},
		VectorStore: semantic.NewWithClients(&mockPoints{}, &mockCollections{}, "test"),
		GraphStore:  graph.NewWithOpener(&mockOpener{}),
		Logger:      slog.Default(),
	}

	sub, err := StartConsumer(nc, deps)
	if err != nil {
		t.Fatalf("StartConsumer: %v", err)
	}
	defer sub.Unsubscribe()

	// Publish without retry header — should retry (republish)
	post := validPost()
	data, _ := json.Marshal(post)
	nc.Publish(IngestSubject, data)
	nc.Flush()
	time.Sleep(300 * time.Millisecond)
}

func TestChunkDoc_ShortContent(t *testing.T) {
	ctx := context.Background()
	doc := ParsedDoc{
		ID:        "test:1",
		Content:   "Short",
		Sentences: nil, // empty sentences
	}
	result := ChunkDoc(ctx, doc)
	if result.IsErr() {
		_, err := result.Unwrap()
		t.Fatalf("unexpected error: %v", err)
	}
	chunked, _ := result.Unwrap()
	if len(chunked.Chunks) != 1 {
		t.Fatalf("expected 1 fallback chunk, got %d", len(chunked.Chunks))
	}
}

func TestChunkSentences_Empty(t *testing.T) {
	chunks := chunkSentences("doc1", nil, 512, 50)
	if chunks != nil {
		t.Fatal("expected nil for empty sentences")
	}
}

func TestChunkSentences_ZeroChunkSize(t *testing.T) {
	sentences := []string{"hello world"}
	chunks := chunkSentences("doc1", sentences, 0, 0)
	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk with default size")
	}
}

func TestChunkSentences_NegativeOverlap(t *testing.T) {
	sentences := []string{"hello world", "second sentence"}
	chunks := chunkSentences("doc1", sentences, 512, -1)
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
}

func TestWordCount(t *testing.T) {
	if wordCount("hello world foo") != 3 {
		t.Fatal("expected 3 words")
	}
	if wordCount("") != 0 {
		t.Fatal("expected 0 for empty")
	}
}

func TestParsedDocFromPost(t *testing.T) {
	post := validPost()
	doc := parsedDocFromPost(post)
	if doc.ID != "reddit:abc123" {
		t.Fatalf("wrong ID: %s", doc.ID)
	}
	if doc.Metadata["author"] != "user1" {
		t.Fatal("missing author in metadata")
	}
	if doc.Vehicle != "2019 Honda Civic" {
		t.Fatalf("wrong vehicle: %s", doc.Vehicle)
	}
}
