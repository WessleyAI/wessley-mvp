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

// VehicleInfo holds structured vehicle identification.
type VehicleInfo struct {
	Make  string `json:"make,omitempty"`
	Model string `json:"model,omitempty"`
	Year  int    `json:"year,omitempty"`
	Trim  string `json:"trim,omitempty"`
}

// Metadata holds extracted automotive context.
type Metadata struct {
	Vehicle     string       `json:"vehicle,omitempty"`
	VehicleInfo *VehicleInfo `json:"vehicle_info,omitempty"`
	Symptoms    []string     `json:"symptoms,omitempty"`
	Fixes       []string     `json:"fixes,omitempty"`
	Keywords    []string     `json:"keywords,omitempty"`
	Section     string       `json:"section,omitempty"`     // system/subsystem classification
	Components  string       `json:"components,omitempty"`  // raw component string (e.g. from NHTSA)
}

// ScrapeOpts configures a scrape run.
type ScrapeOpts struct {
	Query      string
	MaxResults int
	ChannelIDs []string
}
