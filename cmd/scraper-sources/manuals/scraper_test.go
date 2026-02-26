package manuals

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestTagVehicleInfo_Filename(t *testing.T) {
	make, model, year := TagVehicleInfo("2020_Toyota_Camry_Manual.pdf", "")
	if make != "Toyota" {
		t.Fatalf("expected Toyota, got %s", make)
	}
	if year != 2020 {
		t.Fatalf("expected 2020, got %d", year)
	}
	if model != "Camry" {
		t.Fatalf("expected Camry, got %s", model)
	}
}

func TestTagVehicleInfo_Content(t *testing.T) {
	make, _, year := TagVehicleInfo("manual.pdf", "This is the 2019 Honda Civic owner's manual")
	if make != "Honda" {
		t.Fatalf("expected Honda, got %s", make)
	}
	if year != 2019 {
		t.Fatalf("expected 2019, got %d", year)
	}
}

func TestTagVehicleInfo_NoMatch(t *testing.T) {
	make, _, year := TagVehicleInfo("random.pdf", "no vehicle info here")
	if make != "" {
		t.Fatalf("expected empty make, got %s", make)
	}
	if year != 0 {
		t.Fatalf("expected 0 year, got %d", year)
	}
}

func TestExtractPDFText(t *testing.T) {
	// Minimal PDF-like content with BT/ET text blocks
	data := []byte("BT (Hello World) ET")
	got := extractPDFText(data)
	if got != "Hello World" {
		t.Fatalf("expected 'Hello World', got %q", got)
	}
}

func TestExtractPDFText_Empty(t *testing.T) {
	got := extractPDFText([]byte("no pdf content"))
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestCleanPDFText(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"hello\\nworld", "hello\nworld"},
		{"paren\\(test\\)", "paren(test)"},
		{"back\\\\slash", "back\\slash"},
		{"  spaces  ", "spaces"},
	}
	for _, tt := range tests {
		got := cleanPDFText(tt.input)
		if got != tt.want {
			t.Errorf("cleanPDFText(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestScraper_FetchAll_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	s := NewScraper(Config{Directory: dir})
	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("expected 0 posts, got %d", len(posts))
	}
}

func TestScraper_FetchAll_NoDir(t *testing.T) {
	s := NewScraper(Config{Directory: ""})
	_, err := s.FetchAll(context.Background())
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestScraper_FetchAll_WithPDF(t *testing.T) {
	dir := t.TempDir()
	// Create a minimal "PDF" with extractable text
	content := []byte("%PDF-1.4\nBT (2020 Toyota Camry Manual Content) ET\n%%EOF")
	if err := os.WriteFile(filepath.Join(dir, "2020_Toyota_Camry.pdf"), content, 0644); err != nil {
		t.Fatal(err)
	}
	// Non-PDF file should be ignored
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignore"), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewScraper(Config{Directory: dir})
	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].Source != "manual" {
		t.Fatalf("expected source=manual, got %s", posts[0].Source)
	}
	if posts[0].Metadata.VehicleInfo == nil {
		t.Fatal("expected VehicleInfo to be set")
	}
	if posts[0].Metadata.VehicleInfo.Make != "Toyota" {
		t.Fatalf("expected Toyota, got %s", posts[0].Metadata.VehicleInfo.Make)
	}
}

func TestScraper_FetchAll_MaxFiles(t *testing.T) {
	dir := t.TempDir()
	content := []byte("%PDF-1.4\nBT (Test content) ET\n%%EOF")
	for i := 0; i < 5; i++ {
		name := filepath.Join(dir, "manual"+string(rune('A'+i))+".pdf")
		os.WriteFile(name, content, 0644)
	}

	s := NewScraper(Config{Directory: dir, MaxFiles: 2})
	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}
}

func TestTitleCase(t *testing.T) {
	tests := []struct{ input, want string }{
		{"CAMRY", "Camry"},
		{"GRAND CHEROKEE", "Grand Cherokee"},
		{"", ""},
		{"f-150", "F-150"},
	}
	for _, tt := range tests {
		got := titleCase(tt.input)
		if got != tt.want {
			t.Errorf("titleCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractModel(t *testing.T) {
	got := extractModel("TOYOTA CAMRY 2020", "TOYOTA")
	if got != "Camry" {
		t.Fatalf("expected Camry, got %q", got)
	}
}

func TestExtractModel_NoMatch(t *testing.T) {
	got := extractModel("SOMETHING ELSE", "TOYOTA")
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestMin(t *testing.T) {
	if min(3, 5) != 3 {
		t.Fatal("min(3,5) should be 3")
	}
	if min(5, 3) != 3 {
		t.Fatal("min(5,3) should be 3")
	}
}
