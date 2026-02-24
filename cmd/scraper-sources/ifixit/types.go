// Package ifixit provides a scraper for iFixit automotive repair guides.
package ifixit

import "time"

// Guide represents an iFixit guide from the API.
type Guide struct {
	GuideID   int    `json:"guideid"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	URL       string `json:"url"`
	Category  string `json:"category"`
	Subject   string `json:"subject"`
	Type      string `json:"type"`
	Locale    string `json:"locale"`
	Difficulty string `json:"difficulty"`
	Author    struct {
		Username string `json:"username"`
	} `json:"author"`
	Steps []Step `json:"steps"`
	ModifiedDate int64 `json:"modified_date"`
}

// Step represents a single step in a guide.
type Step struct {
	OrderBy int    `json:"orderby"`
	Title   string `json:"title"`
	Lines   []Line `json:"lines"`
}

// Line represents a text line in a step.
type Line struct {
	Text  string `json:"text_raw"`
	Level int    `json:"level"`
}

// Config controls iFixit scraper behavior.
type Config struct {
	Categories []string
	MaxGuides  int
	RateLimit  time.Duration
}
