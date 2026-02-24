package scraper

import "testing"

func TestCleanTranscript(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"[Music] hello  world [Applause]", "hello world"},
		{"it&#39;s a &amp; b", "it's a & b"},
		{"  lots   of   spaces  ", "lots of spaces"},
		{"[Laughter] test [Inaudible] end", "test end"},
	}
	for _, tt := range tests {
		got := CleanTranscript(tt.in)
		if got != tt.want {
			t.Errorf("CleanTranscript(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
