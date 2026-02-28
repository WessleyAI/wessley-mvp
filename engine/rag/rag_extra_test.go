package rag

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/WessleyAI/wessley-mvp/engine/semantic"
	mlpb "github.com/WessleyAI/wessley-mvp/ml/proto/wessley/ml/v1"
	"google.golang.org/grpc"
)

func TestQuery_ChatError(t *testing.T) {
	svc := &Service{
		embed:  &mockEmbedClient{resp: &mlpb.EmbedResponse{Values: []float32{0.1}}},
		chat:   &mockChatClient{err: fmt.Errorf("chat service down")},
		search: &mockSearcher{results: []semantic.SearchResult{{ID: "c1", Content: "x"}}},
		opts:   DefaultOptions(),
		logger: slog.Default(),
	}
	svc.opts.UseGraph = false

	_, err := svc.Query(context.Background(), "question", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "rag: chat: chat service down" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestQuery_EmptySearchResults(t *testing.T) {
	chatClient := &mockChatClient{
		resp: &mlpb.ChatResponse{Reply: "I don't have enough information."},
	}
	svc := &Service{
		embed:  &mockEmbedClient{resp: &mlpb.EmbedResponse{Values: []float32{0.1}}},
		chat:   chatClient,
		search: &mockSearcher{results: nil},
		opts:   DefaultOptions(),
		logger: slog.Default(),
	}
	svc.opts.UseGraph = false

	ans, err := svc.Query(context.Background(), "unknown topic", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ans.Sources) != 0 {
		t.Errorf("expected 0 sources, got %d", len(ans.Sources))
	}
	if len(chatClient.lastReq.Context) != 0 {
		t.Errorf("expected 0 context parts, got %d", len(chatClient.lastReq.Context))
	}
}

func TestQuery_GraphEmptyComponents(t *testing.T) {
	chatClient := &mockChatClient{
		resp: &mlpb.ChatResponse{Reply: "answer"},
	}
	svc := &Service{
		embed:  &mockEmbedClient{resp: &mlpb.EmbedResponse{Values: []float32{0.1}}},
		chat:   chatClient,
		search: &mockSearcher{results: []semantic.SearchResult{{ID: "c1", Content: "x"}}},
		graph:  &mockGraphEnricher{components: nil, edges: nil},
		opts:   DefaultOptions(),
		logger: slog.Default(),
	}

	ans, err := svc.Query(context.Background(), "How does the ECU work?", "")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if ans.Text != "answer" {
		t.Error("wrong answer")
	}
	// No graph context added since components are empty
	if len(chatClient.lastReq.Context) != 1 {
		t.Errorf("expected 1 context part, got %d", len(chatClient.lastReq.Context))
	}
}

func TestExtractKeywords_Empty(t *testing.T) {
	kw := extractKeywords("")
	if len(kw) != 0 {
		t.Errorf("expected no keywords from empty string, got %v", kw)
	}
}

func TestExtractKeywords_AllStopWords(t *testing.T) {
	kw := extractKeywords("the is a an of in for on")
	if len(kw) != 0 {
		t.Errorf("expected no keywords from all stop words, got %v", kw)
	}
}

func TestExtractKeywords_WithPunctuation(t *testing.T) {
	kw := extractKeywords("What about the alternator? And the battery!")
	kwMap := map[string]bool{}
	for _, k := range kw {
		kwMap[k] = true
	}
	if !kwMap["alternator"] || !kwMap["battery"] {
		t.Errorf("expected alternator and battery, got %v", kw)
	}
}

func TestBuildContextParts_Empty(t *testing.T) {
	parts := buildContextParts(nil, "")
	if len(parts) != 0 {
		t.Errorf("expected 0 parts, got %d", len(parts))
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.TopK <= 0 {
		t.Error("TopK should be positive")
	}
	if opts.SearchTimeout <= 0 {
		t.Error("SearchTimeout should be positive")
	}
	if opts.SystemPrompt == "" {
		t.Error("SystemPrompt should not be empty")
	}
	if opts.UseGraph != true {
		t.Error("UseGraph should default to true")
	}
}

// Test that Query correctly passes vehicle filter
func TestQuery_VehicleFilter(t *testing.T) {
	var capturedFilter map[string]string
	searcher := &filterCapturingSearcher{
		results: []semantic.SearchResult{{ID: "c1", Content: "x"}},
		capture: func(f map[string]string) { capturedFilter = f },
	}
	chatClient := &mockChatClient{
		resp: &mlpb.ChatResponse{Reply: "ok"},
	}
	svc := &Service{
		embed:  &mockEmbedClient{resp: &mlpb.EmbedResponse{Values: []float32{0.1}}},
		chat:   chatClient,
		search: searcher,
		opts:   DefaultOptions(),
		logger: slog.Default(),
	}
	svc.opts.UseGraph = false

	_, err := svc.Query(context.Background(), "test question with words", "2020-toyota-camry")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if capturedFilter["vehicle"] != "2020-toyota-camry" {
		t.Errorf("expected vehicle filter, got %v", capturedFilter)
	}
}

type filterCapturingSearcher struct {
	results []semantic.SearchResult
	capture func(map[string]string)
}

func (s *filterCapturingSearcher) Search(_ context.Context, _ []float32, _ int, filter map[string]string) ([]semantic.SearchResult, error) {
	s.capture(filter)
	return s.results, nil
}

// Test enrichWithGraph directly
func TestEnrichWithGraph_NoKeywords(t *testing.T) {
	svc := &Service{
		graph:  &mockGraphEnricher{},
		logger: slog.Default(),
	}
	result := svc.enrichWithGraph(context.Background(), "the is a", "") // all stop words
	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

// mockEmbedBatchClient for embed batch testing
type mockEmbedBatchClient struct {
	mlpb.EmbedServiceClient
	embedResp *mlpb.EmbedResponse
	batchResp *mlpb.EmbedBatchResponse
	err       error
}

func (m *mockEmbedBatchClient) Embed(_ context.Context, _ *mlpb.EmbedRequest, _ ...grpc.CallOption) (*mlpb.EmbedResponse, error) {
	return m.embedResp, m.err
}

func (m *mockEmbedBatchClient) EmbedBatch(_ context.Context, _ *mlpb.EmbedBatchRequest, _ ...grpc.CallOption) (*mlpb.EmbedBatchResponse, error) {
	return m.batchResp, m.err
}
