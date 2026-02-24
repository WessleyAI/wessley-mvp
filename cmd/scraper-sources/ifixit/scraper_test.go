package ifixit

import (
	"testing"
)

func TestNewScraper(t *testing.T) {
	s := NewScraper(Config{
		Categories: []string{"Car"},
		MaxGuides:  10,
	})
	if s == nil {
		t.Fatal("expected non-nil scraper")
	}
}
