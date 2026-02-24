package ifixit

import (
	"strings"
	"testing"
)

func TestBuildGuideContent(t *testing.T) {
	g := Guide{
		Summary: "How to replace brake pads",
		Steps: []Step{
			{OrderBy: 1, Title: "Remove wheel", Lines: []Line{{Text: "Use lug wrench to remove wheel nuts"}}},
			{OrderBy: 2, Title: "Remove caliper", Lines: []Line{{Text: "Disconnect brake caliper bolts"}}},
		},
	}
	content := buildGuideContent(g)
	if !strings.Contains(content, "brake pads") {
		t.Fatal("expected summary in content")
	}
	if !strings.Contains(content, "Step 1") {
		t.Fatal("expected step 1")
	}
	if !strings.Contains(content, "lug wrench") {
		t.Fatal("expected step content")
	}
}

func TestExtractFixes(t *testing.T) {
	fixes := extractFixes("Remove the old filter and install the new one. Tighten bolts to spec.")
	if len(fixes) < 3 {
		t.Fatalf("expected at least 3 fixes, got %v", fixes)
	}
}

func TestNewScraper(t *testing.T) {
	s := NewScraper(Config{
		Categories: []string{"Car"},
		MaxGuides:  10,
	})
	if s == nil {
		t.Fatal("expected non-nil scraper")
	}
}
