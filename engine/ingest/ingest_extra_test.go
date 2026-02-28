package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/scraper"
)

func TestWordCount(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"hello world", 2},
		{"", 0},
		{"single", 1},
		{"  multiple   spaces  ", 2},
	}
	for _, tt := range tests {
		if got := wordCount(tt.in); got != tt.want {
			t.Errorf("wordCount(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestChunkSentencesEmpty(t *testing.T) {
	chunks := chunkSentences("doc", nil, 100, 10)
	if chunks != nil {
		t.Fatal("empty sentences should return nil")
	}
}

func TestChunkSentencesSingleShort(t *testing.T) {
	chunks := chunkSentences("doc", []string{"Hello"}, 100, 10)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Text != "Hello" {
		t.Fatalf("expected Hello, got %q", chunks[0].Text)
	}
	if chunks[0].DocID != "doc" {
		t.Fatal("wrong docID")
	}
}

func TestChunkSentencesDefaultChunkSize(t *testing.T) {
	// chunkSize=0 should use default
	chunks := chunkSentences("doc", []string{"a", "b"}, 0, 0)
	if len(chunks) == 0 {
		t.Fatal("should produce chunks with default size")
	}
}

func TestChunkSentencesNegativeOverlap(t *testing.T) {
	// Negative overlap → treated as 0
	chunks := chunkSentences("doc", []string{"a", "b", "c"}, 100, -5)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk with large size, got %d", len(chunks))
	}
}

func TestParsedDocFromPost(t *testing.T) {
	post := scraper.ScrapedPost{
		Source:   "youtube",
		SourceID: "vid1",
		Title:    "Test Video",
		Content:  "First sentence. Second sentence.",
		Author:   "author1",
		URL:      "https://youtube.com/watch?v=vid1",
		Metadata: scraper.Metadata{
			Vehicle: "2020 Toyota Camry",
		},
	}

	doc := parsedDocFromPost(post)
	if doc.ID != "youtube:vid1" {
		t.Errorf("expected ID youtube:vid1, got %s", doc.ID)
	}
	if doc.Source != "youtube" {
		t.Errorf("expected source youtube, got %s", doc.Source)
	}
	if doc.Vehicle != "2020 Toyota Camry" {
		t.Errorf("expected vehicle, got %s", doc.Vehicle)
	}
	if doc.Metadata["author"] != "author1" {
		t.Error("expected author in metadata")
	}
	if len(doc.Sentences) == 0 {
		t.Error("expected sentences")
	}
}

func TestValidateStage_EmptySourceID(t *testing.T) {
	post := validPost()
	post.SourceID = ""
	result := Validate(context.Background(), post)
	if !result.IsErr() {
		t.Fatal("expected error for empty source_id")
	}
}

func TestValidateStage_EmptyTitle(t *testing.T) {
	post := validPost()
	post.Title = ""
	result := Validate(context.Background(), post)
	if !result.IsErr() {
		t.Fatal("expected error for empty title")
	}
}

func TestChunkDocFallbackSingleChunk(t *testing.T) {
	// Very short content that produces no sentences via splitSentences
	doc := ParsedDoc{
		ID:        "test:1",
		Content:   "short",
		Sentences: nil, // no sentences
	}
	result := ChunkDoc(context.Background(), doc)
	if result.IsErr() {
		_, err := result.Unwrap()
		t.Fatalf("unexpected error: %v", err)
	}
	chunked, _ := result.Unwrap()
	if len(chunked.Chunks) != 1 {
		t.Fatalf("expected fallback single chunk, got %d", len(chunked.Chunks))
	}
	if chunked.Chunks[0].Text != "short" {
		t.Errorf("expected content as chunk text, got %q", chunked.Chunks[0].Text)
	}
}

func TestSplitSentencesEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		minCount int
	}{
		{"dots no space", "a.b.c", 1}, // dots without spaces are not sentence breaks (except at end)
		{"multiple newlines", "a\n\nb\n", 2},
		{"exclamation", "Wow! Amazing!", 2},
		{"question", "What? How?", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitSentences(tt.input)
			if len(got) < tt.minCount {
				t.Errorf("splitSentences(%q) = %d sentences, want >= %d: %v", tt.input, len(got), tt.minCount, got)
			}
		})
	}
}

func TestValidPostHelper(t *testing.T) {
	post := validPost()
	if post.Source != "reddit" {
		t.Fatal("validPost should return reddit source")
	}
	if post.ScrapedAt.IsZero() {
		t.Fatal("ScrapedAt should be set")
	}
	if post.PublishedAt.After(time.Now().Add(time.Second)) {
		t.Fatal("PublishedAt should not be in the future")
	}
}
