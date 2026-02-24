// Package rag orchestrates the Retrieval-Augmented Generation pipeline.
// It accepts a user question, embeds it, searches for relevant chunks,
// optionally enriches with graph context, builds a prompt, and calls the
// ML worker's ChatService for the final answer.
package rag

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/engine/semantic"
	mlpb "github.com/WessleyAI/wessley-mvp/ml/proto/wessley/ml/v1"
	"google.golang.org/grpc"
)

// Service is the RAG orchestration service.
type Service struct {
	embed   mlpb.EmbedServiceClient
	chat    mlpb.ChatServiceClient
	search  SemanticSearcher
	graph   GraphEnricher
	opts    Options
	logger  *slog.Logger
}

// SemanticSearcher abstracts Qdrant vector search.
type SemanticSearcher interface {
	Search(ctx context.Context, embedding []float32, topK int, filter map[string]string) ([]semantic.SearchResult, error)
}

// GraphEnricher optionally enriches a query with knowledge-graph context.
type GraphEnricher interface {
	FindRelatedComponents(ctx context.Context, keywords []string, vehicle string) ([]graph.Component, []graph.Edge, error)
}

// Options configures the RAG pipeline behaviour.
type Options struct {
	TopK          int
	Temperature   float32
	MaxTokens     int32
	Model         string
	SystemPrompt  string
	UseGraph      bool
	SearchTimeout time.Duration
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{
		TopK:          5,
		Temperature:   0.3,
		MaxTokens:     1024,
		Model:         "",
		SystemPrompt:  defaultSystemPrompt,
		UseGraph:      true,
		SearchTimeout: 5 * time.Second,
	}
}

const defaultSystemPrompt = `You are Wessley, an expert automotive electrical assistant.
Answer the user's question using ONLY the provided context. If the context
does not contain enough information, say so. Cite sources using [source_id].`

// New creates a new RAG Service.
func New(conn *grpc.ClientConn, search SemanticSearcher, graphEnricher GraphEnricher, opts Options, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		embed:  mlpb.NewEmbedServiceClient(conn),
		chat:   mlpb.NewChatServiceClient(conn),
		search: search,
		graph:  graphEnricher,
		opts:   opts,
		logger: logger,
	}
}

// Answer represents the structured response from the RAG pipeline.
type Answer struct {
	Text       string   `json:"text"`
	Sources    []Source `json:"sources"`
	TokensUsed int32    `json:"tokens_used"`
	Model      string   `json:"model"`
}

// Source represents a citation backing the answer.
type Source struct {
	ID      string  `json:"id"`
	Content string  `json:"content"`
	DocID   string  `json:"doc_id"`
	Source  string  `json:"source"`
	Score   float32 `json:"score"`
}

// Query runs the full RAG pipeline for a user question.
func (s *Service) Query(ctx context.Context, question string, vehicle string) (*Answer, error) {
	s.logger.Info("rag query start", "question_len", len(question), "vehicle", vehicle)

	// 1. Embed the query.
	embedResp, err := s.embed.Embed(ctx, &mlpb.EmbedRequest{Text: question})
	if err != nil {
		return nil, fmt.Errorf("rag: embed query: %w", err)
	}

	// 2. Semantic search.
	searchCtx, cancel := context.WithTimeout(ctx, s.opts.SearchTimeout)
	defer cancel()

	filter := map[string]string{}
	if vehicle != "" {
		filter["vehicle"] = vehicle
	}

	results, err := s.search.Search(searchCtx, embedResp.GetValues(), s.opts.TopK, filter)
	if err != nil {
		return nil, fmt.Errorf("rag: semantic search: %w", err)
	}
	s.logger.Info("rag semantic search done", "results", len(results))

	// 3. Optionally enrich with graph context.
	var graphContext string
	if s.opts.UseGraph && s.graph != nil {
		graphContext = s.enrichWithGraph(ctx, question, vehicle)
	}

	// 4. Build prompt with retrieved context.
	contextParts := buildContextParts(results, graphContext)

	// 5. Call ChatService.
	chatResp, err := s.chat.Chat(ctx, &mlpb.ChatRequest{
		Message:      question,
		Context:      contextParts,
		SystemPrompt: s.opts.SystemPrompt,
		Temperature:  s.opts.Temperature,
		Model:        s.opts.Model,
		MaxTokens:    s.opts.MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("rag: chat: %w", err)
	}

	// 6. Build structured response.
	sources := make([]Source, len(results))
	for i, r := range results {
		sources[i] = Source{
			ID:      r.ID,
			Content: r.Content,
			DocID:   r.DocID,
			Source:  r.Source,
			Score:   r.Score,
		}
	}

	return &Answer{
		Text:       chatResp.GetReply(),
		Sources:    sources,
		TokensUsed: chatResp.GetTokensUsed(),
		Model:      chatResp.GetModel(),
	}, nil
}

// enrichWithGraph attempts to get graph context; failures are logged and skipped.
func (s *Service) enrichWithGraph(ctx context.Context, question, vehicle string) string {
	keywords := extractKeywords(question)
	if len(keywords) == 0 {
		return ""
	}

	components, edges, err := s.graph.FindRelatedComponents(ctx, keywords, vehicle)
	if err != nil {
		s.logger.Warn("rag: graph enrichment failed, continuing without", "err", err)
		return ""
	}

	if len(components) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Related components from knowledge graph:\n")
	for _, c := range components {
		fmt.Fprintf(&b, "- %s (%s): %s\n", c.Name, c.Type, c.ID)
	}
	if len(edges) > 0 {
		b.WriteString("Relationships:\n")
		for _, e := range edges {
			fmt.Fprintf(&b, "- %s -[%s]-> %s\n", e.From, e.Type, e.To)
		}
	}
	return b.String()
}

// buildContextParts formats search results and graph context into context strings.
func buildContextParts(results []semantic.SearchResult, graphContext string) []string {
	parts := make([]string, 0, len(results)+1)
	for _, r := range results {
		part := fmt.Sprintf("[%s] (source: %s, score: %.3f)\n%s", r.ID, r.Source, r.Score, r.Content)
		parts = append(parts, part)
	}
	if graphContext != "" {
		parts = append(parts, graphContext)
	}
	return parts
}

// extractKeywords does simple keyword extraction from a question.
func extractKeywords(question string) []string {
	// Simple approach: split on spaces, filter short/stop words.
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "can": true, "shall": true, "to": true,
		"of": true, "in": true, "for": true, "on": true, "with": true,
		"at": true, "by": true, "from": true, "as": true, "into": true,
		"through": true, "during": true, "before": true, "after": true,
		"what": true, "where": true, "when": true, "how": true, "which": true,
		"who": true, "whom": true, "this": true, "that": true, "these": true,
		"those": true, "i": true, "me": true, "my": true, "it": true,
		"its": true, "and": true, "but": true, "or": true, "not": true,
	}

	words := strings.Fields(strings.ToLower(question))
	var keywords []string
	for _, w := range words {
		w = strings.Trim(w, "?.,!;:'\"")
		if len(w) > 2 && !stopWords[w] {
			keywords = append(keywords, w)
		}
	}
	return keywords
}
