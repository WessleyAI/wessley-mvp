package domain

import (
	"strings"
	"testing"

	"github.com/WessleyAI/wessley-mvp/engine/scraper"
)

func TestValidateScrapedPost(t *testing.T) {
	valid := scraper.ScrapedPost{
		Source:   "reddit",
		SourceID: "abc123",
		Title:    "Test Post",
		Content:  "Some content here",
	}

	if err := ValidateScrapedPost(valid); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidateScrapedPost_EmptyContent(t *testing.T) {
	post := scraper.ScrapedPost{Source: "reddit", SourceID: "abc", Title: "T"}
	err := ValidateScrapedPost(post)
	if err == nil || !strings.Contains(err.Error(), "content is empty") {
		t.Fatalf("expected content error, got %v", err)
	}
}

func TestValidateScrapedPost_UnknownSource(t *testing.T) {
	post := scraper.ScrapedPost{Source: "tiktok", SourceID: "abc", Title: "T", Content: "C"}
	err := ValidateScrapedPost(post)
	if err == nil || !strings.Contains(err.Error(), "unknown source") {
		t.Fatalf("expected source error, got %v", err)
	}
}

func TestValidateScrapedPost_EmptySourceID(t *testing.T) {
	post := scraper.ScrapedPost{Source: "reddit", Title: "T", Content: "C"}
	err := ValidateScrapedPost(post)
	if err == nil || !strings.Contains(err.Error(), "source_id is empty") {
		t.Fatalf("expected source_id error, got %v", err)
	}
}

func TestValidateScrapedPost_EmptyTitle(t *testing.T) {
	post := scraper.ScrapedPost{Source: "reddit", SourceID: "abc", Content: "C"}
	err := ValidateScrapedPost(post)
	if err == nil || !strings.Contains(err.Error(), "title is empty") {
		t.Fatalf("expected title error, got %v", err)
	}
}

func TestValidateScrapedPost_AllSources(t *testing.T) {
	for _, src := range []string{"reddit", "youtube", "forum"} {
		post := scraper.ScrapedPost{Source: src, SourceID: "x", Title: "T", Content: "C"}
		if err := ValidateScrapedPost(post); err != nil {
			t.Errorf("source %q should be valid: %v", src, err)
		}
	}
}

func TestValidationError_Error(t *testing.T) {
	ve := NewValidationError("make", "Lada", ErrUnsupportedMake)
	s := ve.Error()
	if !strings.Contains(s, "make") || !strings.Contains(s, "Lada") || !strings.Contains(s, "unsupported make") {
		t.Fatalf("unexpected error string: %s", s)
	}
}
