package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/engine/scraper"
	"github.com/WessleyAI/wessley-mvp/engine/semantic"
	mlpb "github.com/WessleyAI/wessley-mvp/ml/proto/wessley/ml/v1"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
)

// --- Mocks for error paths ---

type failGraphSession struct{}

func (s *failGraphSession) Run(_ context.Context, _ string, _ map[string]any) (graph.CypherResult, error) {
	return nil, fmt.Errorf("graph run fail")
}
func (s *failGraphSession) Close(_ context.Context) error { return nil }
func (s *failGraphSession) ExecuteWrite(_ context.Context, _ func(tx graph.CypherRunner) (any, error)) (any, error) {
	return nil, nil
}

type failGraphOpener struct{}

func (o *failGraphOpener) OpenSession(_ context.Context) graph.CypherSession {
	return &failGraphSession{}
}

type failUpsertPoints struct{}

func (m *failUpsertPoints) Upsert(_ context.Context, _ *pb.UpsertPoints, _ ...grpc.CallOption) (*pb.PointsOperationResponse, error) {
	return nil, fmt.Errorf("upsert fail")
}
func (m *failUpsertPoints) Delete(_ context.Context, _ *pb.DeletePoints, _ ...grpc.CallOption) (*pb.PointsOperationResponse, error) {
	return &pb.PointsOperationResponse{}, nil
}
func (m *failUpsertPoints) Search(_ context.Context, _ *pb.SearchPoints, _ ...grpc.CallOption) (*pb.SearchResponse, error) {
	return &pb.SearchResponse{}, nil
}

// --- NewStore error: graph save fails ---

func TestNewStore_GraphSaveError(t *testing.T) {
	gs := graph.NewWithOpener(&failGraphOpener{})
	vs := semantic.NewWithClients(&mockPoints{}, &mockCollections{}, "test")

	stage := NewStore(vs, gs)
	doc := EmbeddedDoc{
		ChunkedDoc: ChunkedDoc{
			ParsedDoc: ParsedDoc{ID: "d1", Title: "T", Source: "s"},
			Chunks:    []Chunk{{Index: 0, Text: "hello"}},
		},
		Embeddings: [][]float32{{0.1}},
	}

	r := stage(context.Background(), doc)
	if r.IsOk() {
		t.Fatal("expected graph save error")
	}
	_, err := r.Unwrap()
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- NewStore error: vector upsert fails ---

func TestNewStore_VectorUpsertError(t *testing.T) {
	gs := graph.NewWithOpener(&mockOpener{})
	vs := semantic.NewWithClients(&failUpsertPoints{}, &mockCollections{}, "test")

	stage := NewStore(vs, gs)
	doc := EmbeddedDoc{
		ChunkedDoc: ChunkedDoc{
			ParsedDoc: ParsedDoc{ID: "d1", Title: "T", Source: "s"},
			Chunks:    []Chunk{{Index: 0, Text: "hello"}},
		},
		Embeddings: [][]float32{{0.1}},
	}

	r := stage(context.Background(), doc)
	if r.IsOk() {
		t.Fatal("expected vector upsert error")
	}
}

// --- StartConsumer: message with retry header ---

func startNATS2(t *testing.T) (*natsserver.Server, *nats.Conn) {
	t.Helper()
	opts := &natsserver.Options{Port: -1}
	ns, err := natsserver.NewServer(opts)
	if err != nil {
		t.Fatal(err)
	}
	ns.Start()
	if !ns.ReadyForConnections(2 * time.Second) {
		t.Fatal("nats not ready")
	}
	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	return ns, nc
}

func TestStartConsumer_WithRetryHeader(t *testing.T) {
	ns, nc := startNATS2(t)
	defer ns.Shutdown()
	defer nc.Close()

	deps := testDeps()
	// Make embedder fail to trigger retry path
	deps.Embedder = &mockEmbedder{err: fmt.Errorf("embed error")}

	sub, err := StartConsumer(nc, deps)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	post := scraper.ScrapedPost{
		Title: "Test", Content: "Some content about cars and brakes testing",
		Source: "test", SourceID: "s1",
	}
	data, _ := json.Marshal(post)

	// Publish with retry count near max
	msg := nats.NewMsg(IngestSubject)
	msg.Data = data
	msg.Header = nats.Header{}
	msg.Header.Set("X-Retry-Count", fmt.Sprintf("%d", MaxRetries-1))
	nc.PublishMsg(msg)
	nc.Flush()

	// Wait for DLQ message
	dlqSub, _ := nc.SubscribeSync(DLQSubject)
	dlqMsg, err := dlqSub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("expected DLQ message, got error: %v", err)
	}
	if dlqMsg == nil {
		t.Fatal("expected DLQ message")
	}
}

func TestStartConsumer_NilLogger(t *testing.T) {
	ns, nc := startNATS2(t)
	defer ns.Shutdown()
	defer nc.Close()

	deps := testDeps()
	deps.Logger = nil

	sub, err := StartConsumer(nc, deps)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()
}

// --- Validate edge cases ---

func TestValidate_Success(t *testing.T) {
	post := scraper.ScrapedPost{
		Title: "Test", Content: "Some valid content here about cars",
		Source: "reddit", SourceID: "s1",
	}
	r := Validate(context.Background(), post)
	if r.IsErr() {
		_, err := r.Unwrap()
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Parse ---

func TestParse_WithMetadata(t *testing.T) {
	post := scraper.ScrapedPost{
		Title: "Test", Content: "Content here", Source: "reddit",
		SourceID: "abc", URL: "http://example.com",
		Metadata: scraper.Metadata{Vehicle: "corolla"},
	}
	r := Parse(context.Background(), post)
	if r.IsErr() {
		_, err := r.Unwrap()
		t.Fatalf("unexpected error: %v", err)
	}
	v, _ := r.Unwrap()
	if v.Vehicle != "corolla" {
		t.Fatalf("expected vehicle corolla, got %s", v.Vehicle)
	}
}

// Ensure mock types satisfy interfaces
var _ mlpb.EmbedServiceClient = (*mockEmbedder)(nil)
var _ graph.CypherResult = (*mockCR)(nil)
var _ graph.CypherSession = (*mockCypherSession)(nil)
var _ graph.SessionOpener = (*mockOpener)(nil)
var _ neo4j.Record // import usage
