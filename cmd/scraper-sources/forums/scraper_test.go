package forums

import (
	"testing"
)

func TestParseSearchResults(t *testing.T) {
	html := `
<div class="results">
  <a href="/threads/brake-pad-replacement.12345/">Brake Pad Replacement Guide</a>
  <a href="/threads/oil-change-tips.12346/">Oil Change Tips for Beginners</a>
  <a href="/threads/brake-pad-replacement.12345/">Brake Pad Replacement Guide</a>
</div>`

	forum := ForumConfig{
		Name:    "TestForum",
		BaseURL: "https://example.com/forums",
	}

	posts := parseSearchResults(html, forum, "brakes")
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts (deduped), got %d", len(posts))
	}
	if posts[0].Source != "forum:TestForum" {
		t.Fatalf("unexpected source: %s", posts[0].Source)
	}
	if posts[0].Title != "Brake Pad Replacement Guide" {
		t.Fatalf("unexpected title: %s", posts[0].Title)
	}
}

func TestDefaultForums(t *testing.T) {
	forums := DefaultForums()
	if len(forums) == 0 {
		t.Fatal("expected at least one default forum")
	}
}

func TestNewScraper(t *testing.T) {
	s := NewScraper(Config{
		Forums:    DefaultForums(),
		Queries:   []string{"brakes"},
		MaxPerForum: 10,
	})
	if s == nil {
		t.Fatal("expected non-nil scraper")
	}
}
