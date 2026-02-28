package scraper

import "testing"

func TestCleanTranscriptEmpty(t *testing.T) {
	if got := CleanTranscript(""); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestCleanTranscriptAllBrackets(t *testing.T) {
	got := CleanTranscript("[Music] [Applause] [Laughter] [Cheering] [Inaudible]")
	if got != "" {
		t.Errorf("expected empty after bracket removal, got %q", got)
	}
}

func TestCleanTranscriptHTMLEntities(t *testing.T) {
	got := CleanTranscript("a &lt; b &gt; c &amp; d &quot;e&quot;")
	if got != `a < b > c & d "e"` {
		t.Errorf("unexpected: %q", got)
	}
}

func TestExtractMetadata_MultipleSymptoms(t *testing.T) {
	title := "2019 Honda Civic troubleshooting"
	transcript := "the car won't start, we're getting a misfire and there's overheating plus check engine light"
	m := extractMetadata(title, transcript)

	if len(m.Symptoms) < 3 {
		t.Errorf("expected at least 3 symptoms, got %d: %v", len(m.Symptoms), m.Symptoms)
	}
}

func TestExtractMetadata_MultipleFixes(t *testing.T) {
	m := extractMetadata("repair", "we need to replace the part, then clean and adjust")
	if len(m.Fixes) < 3 {
		t.Errorf("expected at least 3 fixes, got %d: %v", len(m.Fixes), m.Fixes)
	}
}

func TestExtractMetadata_VehiclePattern(t *testing.T) {
	tests := []struct {
		text    string
		hasVeh  bool
	}{
		{"2020 Toyota Camry", true},
		{"1998 Honda Civic", true},
		{"some random text", false},
		{"car from 2025 Ford Mustang era", true},
	}
	for _, tt := range tests {
		m := extractMetadata(tt.text, "")
		if (m.Vehicle != "") != tt.hasVeh {
			t.Errorf("extractMetadata(%q): vehicle=%q, expected hasVeh=%v", tt.text, m.Vehicle, tt.hasVeh)
		}
	}
}

func TestNewYouTubeScraperDefaults(t *testing.T) {
	s := NewYouTubeScraper("key", nil)
	if len(s.channels) == 0 {
		t.Fatal("expected default channels")
	}
	if s.apiKey != "key" {
		t.Fatal("wrong api key")
	}
}

func TestNewYouTubeScraperCustomChannels(t *testing.T) {
	s := NewYouTubeScraper("key", []string{"ch1", "ch2"})
	if len(s.channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(s.channels))
	}
}
