package rag

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/engine/semantic"
	mlpb "github.com/WessleyAI/wessley-mvp/ml/proto/wessley/ml/v1"
	"google.golang.org/grpc"
)

// --- mocks ---

type mockEmbedClient struct {
	mlpb.EmbedServiceClient
	resp *mlpb.EmbedResponse
	err  error
}

func (m *mockEmbedClient) Embed(_ context.Context, _ *mlpb.EmbedRequest, _ ...grpc.CallOption) (*mlpb.EmbedResponse, error) {
	return m.resp, m.err
}

type mockChatClient struct {
	mlpb.ChatServiceClient
	resp    *mlpb.ChatResponse
	err     error
	lastReq *mlpb.ChatRequest
}

func (m *mockChatClient) Chat(_ context.Context, req *mlpb.ChatRequest, _ ...grpc.CallOption) (*mlpb.ChatResponse, error) {
	m.lastReq = req
	return m.resp, m.err
}

type mockSearcher struct {
	results []semantic.SearchResult
	err     error
}

func (m *mockSearcher) Search(_ context.Context, _ []float32, _ int, _ map[string]string) ([]semantic.SearchResult, error) {
	return m.results, m.err
}

type mockGraphEnricher struct {
	components []graph.Component
	edges      []graph.Edge
	err        error
}

func (m *mockGraphEnricher) FindRelatedComponents(_ context.Context, _ []string, _ string) ([]graph.Component, []graph.Edge, error) {
	return m.components, m.edges, m.err
}

// --- tests ---

func TestQuery_Success(t *testing.T) {
	embedClient := &mockEmbedClient{
		resp: &mlpb.EmbedResponse{Values: []float32{0.1, 0.2, 0.3}, Dimensions: 3},
	}
	chatClient := &mockChatClient{
		resp: &mlpb.ChatResponse{Reply: "The ECU controls the fuel injection.", TokensUsed: 42, Model: "test-model"},
	}
	searcher := &mockSearcher{
		results: []semantic.SearchResult{
			{ID: "chunk-1", Score: 0.95, Content: "ECU controls fuel injection", DocID: "doc-1", Source: "manual.pdf"},
			{ID: "chunk-2", Score: 0.80, Content: "Wiring diagram for ECU", DocID: "doc-2", Source: "diagram.pdf"},
		},
	}
	graphE := &mockGraphEnricher{
		components: []graph.Component{
			{ID: "ecu-1", Name: "Main ECU", Type: "ecu"},
		},
		edges: []graph.Edge{
			{From: "ecu-1", To: "sensor-1", Type: "connects_to"},
		},
	}

	svc := &Service{
		embed:  embedClient,
		chat:   chatClient,
		search: searcher,
		graph:  graphE,
		opts:   DefaultOptions(),
		logger: slog.Default(),
	}

	ans, err := svc.Query(context.Background(), "How does the ECU work?", "corolla-2020")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ans.Text != "The ECU controls the fuel injection." {
		t.Errorf("unexpected text: %s", ans.Text)
	}
	if len(ans.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(ans.Sources))
	}
	if ans.TokensUsed != 42 {
		t.Errorf("expected 42 tokens, got %d", ans.TokensUsed)
	}

	// Verify chat received context parts (2 search results + 1 graph context)
	if len(chatClient.lastReq.Context) != 3 {
		t.Errorf("expected 3 context parts, got %d", len(chatClient.lastReq.Context))
	}
}

func TestQuery_WithoutGraph(t *testing.T) {
	embedClient := &mockEmbedClient{
		resp: &mlpb.EmbedResponse{Values: []float32{0.1}, Dimensions: 1},
	}
	chatClient := &mockChatClient{
		resp: &mlpb.ChatResponse{Reply: "answer"},
	}
	searcher := &mockSearcher{
		results: []semantic.SearchResult{
			{ID: "c1", Score: 0.9, Content: "content"},
		},
	}

	opts := DefaultOptions()
	opts.UseGraph = false

	svc := &Service{
		embed:  embedClient,
		chat:   chatClient,
		search: searcher,
		graph:  nil,
		opts:   opts,
		logger: slog.Default(),
	}

	ans, err := svc.Query(context.Background(), "test question", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ans.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(ans.Sources))
	}
	// No graph context → only 1 context part
	if len(chatClient.lastReq.Context) != 1 {
		t.Errorf("expected 1 context part, got %d", len(chatClient.lastReq.Context))
	}
}

func TestQuery_EmbedError(t *testing.T) {
	svc := &Service{
		embed:  &mockEmbedClient{err: fmt.Errorf("embed down")},
		search: &mockSearcher{},
		opts:   DefaultOptions(),
		logger: slog.Default(),
	}

	_, err := svc.Query(context.Background(), "question", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "rag: embed query: embed down" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestQuery_SearchError(t *testing.T) {
	svc := &Service{
		embed:  &mockEmbedClient{resp: &mlpb.EmbedResponse{Values: []float32{0.1}}},
		search: &mockSearcher{err: fmt.Errorf("qdrant timeout")},
		opts:   DefaultOptions(),
		logger: slog.Default(),
	}

	_, err := svc.Query(context.Background(), "question", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestQuery_GraphFailureGraceful(t *testing.T) {
	embedClient := &mockEmbedClient{
		resp: &mlpb.EmbedResponse{Values: []float32{0.1}},
	}
	chatClient := &mockChatClient{
		resp: &mlpb.ChatResponse{Reply: "answer"},
	}
	searcher := &mockSearcher{
		results: []semantic.SearchResult{{ID: "c1", Score: 0.9, Content: "content"}},
	}
	graphE := &mockGraphEnricher{err: fmt.Errorf("neo4j down")}

	svc := &Service{
		embed:  embedClient,
		chat:   chatClient,
		search: searcher,
		graph:  graphE,
		opts:   DefaultOptions(),
		logger: slog.Default(),
	}

	ans, err := svc.Query(context.Background(), "How does the ECU work?", "")
	if err != nil {
		t.Fatalf("graph failure should not cause query failure: %v", err)
	}
	if ans.Text != "answer" {
		t.Errorf("unexpected text: %s", ans.Text)
	}
	// Graph failed → only search results in context
	if len(chatClient.lastReq.Context) != 1 {
		t.Errorf("expected 1 context part (graph failed), got %d", len(chatClient.lastReq.Context))
	}
}

func TestExtractKeywords(t *testing.T) {
	kw := extractKeywords("What is the ECU wiring for Toyota Corolla?")
	if len(kw) == 0 {
		t.Fatal("expected keywords")
	}
	// Should contain ecu, wiring, toyota, corolla but not "what", "is", "the", "for"
	kwMap := map[string]bool{}
	for _, k := range kw {
		kwMap[k] = true
	}
	for _, expected := range []string{"ecu", "wiring", "toyota", "corolla"} {
		if !kwMap[expected] {
			t.Errorf("expected keyword %q not found in %v", expected, kw)
		}
	}
	for _, stop := range []string{"what", "the", "for"} {
		if kwMap[stop] {
			t.Errorf("stop word %q should not be in keywords", stop)
		}
	}
}

func TestBuildContextParts(t *testing.T) {
	results := []semantic.SearchResult{
		{ID: "a", Source: "s1", Score: 0.9, Content: "content1"},
	}
	parts := buildContextParts(results, "graph info")
	if len(parts) != 2 {
		t.Errorf("expected 2 parts, got %d", len(parts))
	}

	parts = buildContextParts(results, "")
	if len(parts) != 1 {
		t.Errorf("expected 1 part without graph, got %d", len(parts))
	}
}
