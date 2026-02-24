package scraper

import "time"

// ScrapedPost represents a scraped and processed content item.
type ScrapedPost struct {
	Source      string    `json:"source"`
	SourceID    string    `json:"source_id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Author      string    `json:"author"`
	URL         string    `json:"url"`
	PublishedAt time.Time `json:"published_at"`
	ScrapedAt   time.Time `json:"scraped_at"`
	Metadata    Metadata  `json:"metadata"`
}

// Metadata holds extracted automotive context.
type Metadata struct {
	Vehicle  string   `json:"vehicle,omitempty"`
	Symptoms []string `json:"symptoms,omitempty"`
	Fixes    []string `json:"fixes,omitempty"`
	Keywords []string `json:"keywords,omitempty"`
}

// ScrapeOpts configures a scrape run.
type ScrapeOpts struct {
	Query      string
	MaxResults int
	ChannelIDs []string
}
