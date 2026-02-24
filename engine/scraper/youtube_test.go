package scraper

import "testing"

func TestExtractMetadata(t *testing.T) {
	title := "How to Replace Brake Pads on a 2018 Honda Civic"
	transcript := "today we're going to replace the brake pads on this car that won't stop properly check engine light is on"

	m := extractMetadata(title, transcript)

	if m.Vehicle == "" {
		t.Error("expected vehicle to be extracted")
	}
	if len(m.Symptoms) == 0 {
		t.Error("expected symptoms to be extracted")
	}
	if len(m.Fixes) == 0 {
		t.Error("expected fixes to be extracted")
	}
	if len(m.Keywords) == 0 {
		t.Error("expected keywords to be extracted")
	}
}

func TestExtractMetadata_NoMatch(t *testing.T) {
	m := extractMetadata("random video", "nothing relevant here")
	if m.Vehicle != "" {
		t.Error("expected no vehicle")
	}
}
