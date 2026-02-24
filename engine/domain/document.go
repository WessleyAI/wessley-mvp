package domain

import (
	"fmt"

	"github.com/WessleyAI/wessley-mvp/engine/scraper"
)

// ValidSources enumerates accepted scrape sources.
var ValidSources = map[string]bool{
	"reddit":  true,
	"youtube": true,
	"forum":   true,
}

// ValidateScrapedPost checks a ScrapedPost before ingestion.
func ValidateScrapedPost(post scraper.ScrapedPost) error {
	if post.Content == "" {
		return fmt.Errorf("validate: content is empty")
	}
	if !ValidSources[post.Source] {
		return fmt.Errorf("validate: unknown source %q", post.Source)
	}
	if post.SourceID == "" {
		return fmt.Errorf("validate: source_id is empty")
	}
	if post.Title == "" {
		return fmt.Errorf("validate: title is empty")
	}
	return nil
}
