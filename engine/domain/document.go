package domain

import (
	"fmt"
	"strings"

	"github.com/WessleyAI/wessley-mvp/engine/scraper"
)

// ValidSources enumerates accepted scrape sources.
var ValidSources = map[string]bool{
	"reddit":  true,
	"youtube": true,
	"forum":   true,
	"nhtsa":   true,
	"ifixit":  true,
}

// validSource returns true if the source is known.
// Sources with prefixes like "reddit:", "forum:", "youtube:" are accepted.
func validSource(src string) bool {
	if ValidSources[src] {
		return true
	}
	// Accept prefixed sources (e.g., "reddit:MechanicAdvice", "forum:toyotanation")
	for base := range ValidSources {
		if strings.HasPrefix(src, base+":") {
			return true
		}
	}
	return false
}

// ValidateScrapedPost checks a ScrapedPost before ingestion.
func ValidateScrapedPost(post scraper.ScrapedPost) error {
	if post.Content == "" {
		return fmt.Errorf("validate: content is empty")
	}
	if !validSource(post.Source) {
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
