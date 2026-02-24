//go:build integration

package ingest

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/engine/scraper"
	"github.com/WessleyAI/wessley-mvp/engine/semantic"
	mlpb "github.com/WessleyAI/wessley-mvp/ml/proto/wessley/ml/v1"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"google.golang.org/grpc"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// mockEmbedClient returns fixed embeddings for testing.
type mockEmbedClient struct{}

func (m *mockEmbedClient) Embed(ctx context.Context, in *mlpb.EmbedRequest, opts ...grpc.CallOption) (*mlpb.EmbedResponse, error) {
	return &mlpb.EmbedResponse{
		Values: []float32{0.1, 0.2, 0.3, 0.4},
	}, nil
}

func (m *mockEmbedClient) EmbedBatch(ctx context.Context, in *mlpb.EmbedBatchRequest, opts ...grpc.CallOption) (*mlpb.EmbedBatchResponse, error) {
	embeddings := make([]*mlpb.EmbedResponse, len(in.GetTexts()))
	for i := range embeddings {
		embeddings[i] = &mlpb.EmbedResponse{Values: []float32{float32(i)*0.1 + 0.1, 0.2, 0.3, 0.4}}
	}
	return &mlpb.EmbedBatchResponse{Embeddings: embeddings}, nil
}

func TestIngestPipeline_EndToEnd(t *testing.T) {
	ctx := context.Background()

	// Connect Neo4j
	neo4jURL := envOr("NEO4J_URL", "neo4j://localhost:7687")
	driver, err := neo4j.NewDriverWithContext(neo4jURL, neo4j.NoAuth())
	if err != nil {
		t.Fatalf("neo4j connect: %v", err)
	}
	defer func() {
		sess := driver.NewSession(ctx, neo4j.SessionConfig{})
		sess.Run(ctx, "MATCH (n) DETACH DELETE n", nil)
		sess.Close(ctx)
		driver.Close(ctx)
	}()

	if err := driver.VerifyConnectivity(ctx); err != nil {
		t.Fatalf("neo4j verify: %v", err)
	}

	// Connect Qdrant
	qdrantURL := envOr("QDRANT_URL", "localhost:6334")
	vs, err := semantic.New(qdrantURL, "test_ingest_e2e")
	if err != nil {
		t.Fatalf("qdrant connect: %v", err)
	}
	defer func() {
		vs.DeleteCollection(ctx)
		vs.Close()
	}()

	if err := vs.EnsureCollection(ctx, 4); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}

	gs := graph.New(driver)

	// Build pipeline with mock embedder
	deps := Deps{
		Embedder:    &mockEmbedClient{},
		VectorStore: vs,
		GraphStore:  gs,
	}
	pipeline := NewPipeline(deps)

	// Run pipeline with mock scraped post
	post := scraper.ScrapedPost{
		Source:   "reddit",
		SourceID: "integ-test-001",
		Title:    "How to change oil on 2020 Toyota Camry",
		Content:  "Step one: drain the old oil. Step two: replace the filter. Step three: add new oil. Make sure to use the correct weight.",
		Author:   "test-user",
		URL:      "https://reddit.com/r/cars/test",
		Metadata: scraper.Metadata{
			Vehicle:  "2020-Toyota-Camry",
			Keywords: []string{"oil change", "maintenance"},
		},
		PublishedAt: time.Now(),
		ScrapedAt:   time.Now(),
	}

	result := pipeline(ctx, post)
	if result.IsErr() {
		_, err := result.Unwrap()
		t.Fatalf("pipeline failed: %v", err)
	}

	docID, _ := result.Unwrap()
	if docID == "" {
		t.Fatal("expected non-empty doc ID")
	}

	// Verify component saved in Neo4j
	comp, err := gs.GetComponent(ctx, docID)
	if err != nil {
		t.Fatalf("GetComponent: %v", err)
	}
	if comp.Name != post.Title {
		t.Fatalf("expected title %q, got %q", post.Title, comp.Name)
	}

	// Verify vectors saved in Qdrant
	results, err := vs.Search(ctx, []float32{0.1, 0.2, 0.3, 0.4}, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected vector results, got 0")
	}
}
