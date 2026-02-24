// Package forums provides scrapers for automotive repair forums.
package forums

import "time"

// ForumThread represents a scraped forum thread.
type ForumThread struct {
	ID        string    `json:"id"`
	ForumName string    `json:"forum_name"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	URL       string    `json:"url"`
	Replies   []Reply   `json:"replies"`
	PostedAt  time.Time `json:"posted_at"`
}

// Reply represents a reply in a forum thread.
type Reply struct {
	Author  string    `json:"author"`
	Content string    `json:"content"`
	PostedAt time.Time `json:"posted_at"`
}

// ForumConfig describes a single forum to scrape.
type ForumConfig struct {
	Name    string
	BaseURL string
	// SearchPath is the URL path template for search (uses %s for query).
	SearchPath string
}

// Config controls forum scraper behavior.
type Config struct {
	Forums    []ForumConfig
	Queries   []string
	MaxPerForum int
	RateLimit time.Duration
}
