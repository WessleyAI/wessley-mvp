package rag

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/engine/semantic"
	mlpb "github.com/WessleyAI/wessley-mvp/ml/proto/wessley/ml/v1"
	"google.golang.org/grpc"
)

// --- Query error: embed fails ---

func TestQuery_EmbedError2(t *testing.T) {
	svc := &Service{
		embed:  &mockEmbedClient{err: errors.New("embed fail")},
		chat:   &mockChatClient{},
		search: &mockSearcher{},
		opts:   DefaultOptions(),
		logger: slog.Default(),
	}

	_, err := svc.Query(context.Background(), "test", "")
	if err == nil || err.Error() != "rag: embed query: embed fail" {
		t.Fatalf("expected embed error, got %v", err)
	}
}

// --- Query error: search fails ---

func TestQuery_SearchError2(t *testing.T) {
	svc := &Service{
		embed:  &mockEmbedClient{resp: &mlpb.EmbedResponse{Values: []float32{0.1}}},
		chat:   &mockChatClient{},
		search: &mockSearcher{err: errors.New("search fail")},
		opts:   DefaultOptions(),
		logger: slog.Default(),
	}

	_, err := svc.Query(context.Background(), "test", "")
	if err == nil {
		t.Fatal("expected search error")
	}
}

// --- Query error: chat fails ---

func TestQuery_ChatError(t *testing.T) {
	svc := &Service{
		embed:  &mockEmbedClient{resp: &mlpb.EmbedResponse{Values: []float32{0.1}}},
		chat:   &mockChatClient{err: errors.New("chat fail")},
		search: &mockSearcher{results: []semantic.SearchResult{{ID: "r1", Content: "test"}}},
		graph:  nil,
		opts:   DefaultOptions(),
		logger: slog.Default(),
	}
	svc.opts.UseGraph = false

	_, err := svc.Query(context.Background(), "test", "")
	if err == nil {
		t.Fatal("expected chat error")
	}
}

// --- Query: no graph enricher ---

func TestQuery_NoGraphEnricher(t *testing.T) {
	svc := &Service{
		embed:  &mockEmbedClient{resp: &mlpb.EmbedResponse{Values: []float32{0.1}}},
		chat:   &mockChatClient{resp: &mlpb.ChatResponse{Reply: "ok", Model: "m"}},
		search: &mockSearcher{results: []semantic.SearchResult{}},
		graph:  nil,
		opts:   DefaultOptions(),
		logger: slog.Default(),
	}

	ans, err := svc.Query(context.Background(), "test", "")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if ans.Text != "ok" {
		t.Fatal("wrong answer")
	}
}

// --- enrichWithGraph: graph error (logged, returns empty) ---

func TestEnrichWithGraph_Error(t *testing.T) {
	svc := &Service{
		graph:  &mockGraphEnricher{err: errors.New("graph fail")},
		logger: slog.Default(),
	}
	result := svc.enrichWithGraph(context.Background(), "alternator problem diagnosis", "corolla")
	if result != "" {
		t.Fatal("expected empty on error")
	}
}

// --- enrichWithGraph: no keywords ---

func TestEnrichWithGraph_NoKeywords(t *testing.T) {
	svc := &Service{
		graph:  &mockGraphEnricher{},
		logger: slog.Default(),
	}
	// All stop words
	result := svc.enrichWithGraph(context.Background(), "the a an is", "")
	if result != "" {
		t.Fatal("expected empty with no keywords")
	}
}

// --- enrichWithGraph: no components found ---

func TestEnrichWithGraph_NoComponents(t *testing.T) {
	svc := &Service{
		graph:  &mockGraphEnricher{components: nil, edges: nil},
		logger: slog.Default(),
	}
	result := svc.enrichWithGraph(context.Background(), "alternator test", "")
	if result != "" {
		t.Fatal("expected empty when no components")
	}
}

// --- enrichWithGraph: with components and edges ---

func TestEnrichWithGraph_WithEdges(t *testing.T) {
	svc := &Service{
		graph: &mockGraphEnricher{
			components: []graph.Component{{ID: "c1", Name: "Alt", Type: "alternator"}},
			edges:      []graph.Edge{{From: "c1", To: "c2", Type: "powers"}},
		},
		logger: slog.Default(),
	}
	result := svc.enrichWithGraph(context.Background(), "alternator problem", "")
	if result == "" {
		t.Fatal("expected non-empty graph context")
	}
}

// --- enrichWithGraph: with components, no edges ---

func TestEnrichWithGraph_NoEdges(t *testing.T) {
	svc := &Service{
		graph: &mockGraphEnricher{
			components: []graph.Component{{ID: "c1", Name: "Alt", Type: "alternator"}},
			edges:      nil,
		},
		logger: slog.Default(),
	}
	result := svc.enrichWithGraph(context.Background(), "alternator problem", "")
	if result == "" {
		t.Fatal("expected non-empty graph context")
	}
}

// --- Query with vehicle filter ---

func TestQuery_WithVehicle(t *testing.T) {
	svc := &Service{
		embed:  &mockEmbedClient{resp: &mlpb.EmbedResponse{Values: []float32{0.1}}},
		chat:   &mockChatClient{resp: &mlpb.ChatResponse{Reply: "answer", Model: "m"}},
		search: &mockSearcher{results: []semantic.SearchResult{{ID: "r1", Content: "c", Score: 0.9, DocID: "d1", Source: "s1"}}},
		graph:  &mockGraphEnricher{components: []graph.Component{{ID: "c1", Name: "E", Type: "ecu"}}},
		opts:   DefaultOptions(),
		logger: slog.Default(),
	}

	ans, err := svc.Query(context.Background(), "How does ECU work?", "corolla-2020")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(ans.Sources) != 1 || ans.Sources[0].ID != "r1" {
		t.Fatal("wrong sources")
	}
}

// --- extractKeywords ---

func TestExtractKeywords_Alternator(t *testing.T) {
	kw := extractKeywords("What is the alternator doing?")
	found := false
	for _, k := range kw {
		if k == "alternator" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected alternator keyword")
	}
}

func TestExtractKeywords_AllStopWords(t *testing.T) {
	kw := extractKeywords("the is a an")
	if len(kw) != 0 {
		t.Fatalf("expected empty, got %v", kw)
	}
}

func TestExtractKeywords_ShortWords(t *testing.T) {
	kw := extractKeywords("go is ok")
	// "go" and "is" and "ok" are <=2 chars or stop words
	for _, k := range kw {
		if k == "is" {
			t.Fatal("stop word should be filtered")
		}
	}
}

// --- buildContextParts ---

func TestBuildContextParts_WithGraph(t *testing.T) {
	results := []semantic.SearchResult{
		{ID: "r1", Source: "s1", Score: 0.9, Content: "content1"},
	}
	parts := buildContextParts(results, "graph stuff")
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
}

func TestBuildContextParts_NoGraph(t *testing.T) {
	results := []semantic.SearchResult{
		{ID: "r1", Source: "s1", Score: 0.9, Content: "content1"},
	}
	parts := buildContextParts(results, "")
	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}
}

// --- DefaultOptions ---

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.TopK != 5 {
		t.Fatal("wrong default TopK")
	}
	if !opts.UseGraph {
		t.Fatal("UseGraph should be true by default")
	}
}

// --- New with nil logger ---

func TestNew_NilLogger(t *testing.T) {
	// New with nil conn is fine for unit testing the logger path
	conn, _ := grpc.NewClient("passthrough:///localhost:0", grpc.WithTransportCredentials(nil))
	if conn == nil {
		// Skip if we can't create a client
		t.Skip("can't create grpc client")
	}
	svc := New(conn, &mockSearcher{}, &mockGraphEnricher{}, DefaultOptions(), nil)
	if svc.logger == nil {
		t.Fatal("logger should default to slog.Default()")
	}
}
