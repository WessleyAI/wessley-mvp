package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/scraper"
)

func validPost() scraper.ScrapedPost {
	return scraper.ScrapedPost{
		Source:      "reddit",
		SourceID:    "abc123",
		Title:       "2019 Honda Civic fuse box issue",
		Content:     "The main fuse keeps blowing. I checked the alternator and it seems fine. The wiring harness near the battery has some corrosion.",
		Author:      "user1",
		URL:         "https://reddit.com/r/MechanicAdvice/abc123",
		PublishedAt: time.Now(),
		ScrapedAt:   time.Now(),
		Metadata: scraper.Metadata{
			Vehicle:  "2019 Honda Civic",
			Symptoms: []string{"fuse blowing"},
			Fixes:    []string{"check alternator"},
			Keywords: []string{"fuse", "alternator", "wiring"},
		},
	}
}

func TestValidateStage_Valid(t *testing.T) {
	ctx := context.Background()
	result := Validate(ctx, validPost())
	if result.IsErr() {
		_, err := result.Unwrap()
		t.Fatalf("expected ok, got error: %v", err)
	}
}

func TestValidateStage_InvalidSource(t *testing.T) {
	ctx := context.Background()
	post := validPost()
	post.Source = "unknown"
	result := Validate(ctx, post)
	if !result.IsErr() {
		t.Fatal("expected error for invalid source")
	}
}

func TestValidateStage_EmptyContent(t *testing.T) {
	ctx := context.Background()
	post := validPost()
	post.Content = ""
	result := Validate(ctx, post)
	if !result.IsErr() {
		t.Fatal("expected error for empty content")
	}
}

func TestParseStage(t *testing.T) {
	ctx := context.Background()
	post := validPost()
	result := Parse(ctx, post)
	if result.IsErr() {
		_, err := result.Unwrap()
		t.Fatalf("parse failed: %v", err)
	}
	doc, _ := result.Unwrap()
	if doc.ID != "reddit:abc123" {
		t.Errorf("expected ID reddit:abc123, got %s", doc.ID)
	}
	if doc.Title != post.Title {
		t.Errorf("title mismatch")
	}
	if len(doc.Sentences) == 0 {
		t.Error("expected sentences")
	}
}

func TestChunkDocStage(t *testing.T) {
	ctx := context.Background()
	doc := ParsedDoc{
		ID:        "test:1",
		Content:   "Sentence one. Sentence two. Sentence three.",
		Sentences: splitSentences("Sentence one. Sentence two. Sentence three."),
	}
	result := ChunkDoc(ctx, doc)
	if result.IsErr() {
		_, err := result.Unwrap()
		t.Fatalf("chunk failed: %v", err)
	}
	chunked, _ := result.Unwrap()
	if len(chunked.Chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	for _, c := range chunked.Chunks {
		if c.DocID != "test:1" {
			t.Errorf("chunk docID mismatch: %s", c.DocID)
		}
	}
}

func TestSplitSentences(t *testing.T) {
	tests := []struct {
		input    string
		minCount int
	}{
		{"Hello world. This is a test. Third sentence!", 3},
		{"Single sentence", 1},
		{"Line one\nLine two\nLine three", 3},
		{"", 0},
	}
	for _, tt := range tests {
		got := splitSentences(tt.input)
		if len(got) < tt.minCount {
			t.Errorf("splitSentences(%q) = %d sentences, want >= %d", tt.input, len(got), tt.minCount)
		}
	}
}

func TestChunkSentences_Overlap(t *testing.T) {
	// Create many sentences to force multiple chunks.
	sentences := make([]string, 100)
	for i := range sentences {
		sentences[i] = "This is a test sentence with several words in it to count as multiple tokens."
	}
	chunks := chunkSentences("doc1", sentences, 50, 10)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	// Verify indices are sequential.
	for i, c := range chunks {
		if c.Index != i {
			t.Errorf("chunk %d has index %d", i, c.Index)
		}
		if c.DocID != "doc1" {
			t.Errorf("chunk docID mismatch")
		}
	}
}

func TestPipelineComposition(t *testing.T) {
	// Test that Validate → Parse → ChunkDoc composes correctly (no embed/store).
	ctx := context.Background()
	post := validPost()

	// Validate
	vResult := Validate(ctx, post)
	if vResult.IsErr() {
		_, err := vResult.Unwrap()
		t.Fatalf("validate: %v", err)
	}
	vPost, _ := vResult.Unwrap()

	// Parse
	pResult := Parse(ctx, vPost)
	if pResult.IsErr() {
		_, err := pResult.Unwrap()
		t.Fatalf("parse: %v", err)
	}
	doc, _ := pResult.Unwrap()

	// Chunk
	cResult := ChunkDoc(ctx, doc)
	if cResult.IsErr() {
		_, err := cResult.Unwrap()
		t.Fatalf("chunk: %v", err)
	}
	chunked, _ := cResult.Unwrap()
	if len(chunked.Chunks) == 0 {
		t.Fatal("expected chunks")
	}
}
