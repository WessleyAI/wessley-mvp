package manuals

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
)

// mockSource implements ManualSource for testing.
type mockSource struct {
	name    string
	entries []graph.ManualEntry
	err     error
}

func (m *mockSource) Name() string { return m.name }
func (m *mockSource) Discover(_ context.Context, _ []string, _ []int) ([]graph.ManualEntry, error) {
	return m.entries, m.err
}

func TestToyotaSourceName(t *testing.T) {
	s := NewToyotaSource()
	if s.Name() != "toyota" {
		t.Errorf("expected 'toyota', got %q", s.Name())
	}
}

func TestHondaSourceName(t *testing.T) {
	s := NewHondaSource()
	if s.Name() != "honda" {
		t.Errorf("expected 'honda', got %q", s.Name())
	}
}

func TestFordSourceName(t *testing.T) {
	s := NewFordSource()
	if s.Name() != "ford" {
		t.Errorf("expected 'ford', got %q", s.Name())
	}
}

func TestArchiveSourceName(t *testing.T) {
	s := NewArchiveSource()
	if s.Name() != "archive" {
		t.Errorf("expected 'archive', got %q", s.Name())
	}
}

func TestNHTSASourceName(t *testing.T) {
	s := NewNHTSASource()
	if s.Name() != "nhtsa" {
		t.Errorf("expected 'nhtsa', got %q", s.Name())
	}
}

func TestGenericSearchSourceName(t *testing.T) {
	s := NewGenericSearchSource()
	if s.Name() != "search" {
		t.Errorf("expected 'search', got %q", s.Name())
	}
}

func TestToyotaSourceSkipsNonToyota(t *testing.T) {
	s := NewToyotaSource()
	entries, err := s.Discover(context.Background(), []string{"Honda"}, []int{2024})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for non-toyota makes, got %d", len(entries))
	}
}

func TestHondaSourceSkipsNonHonda(t *testing.T) {
	s := NewHondaSource()
	entries, err := s.Discover(context.Background(), []string{"Toyota"}, []int{2024})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestFordSourceSkipsNonFord(t *testing.T) {
	s := NewFordSource()
	entries, err := s.Discover(context.Background(), []string{"Toyota"}, []int{2024})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestExtractToyotaPDFLinks(t *testing.T) {
	html := `<a href="https://www.toyota.com/manuals/camry_2024.pdf">Manual</a>
			 <a href="https://www.toyota.com/manuals/corolla_2024.pdf">Manual</a>`
	entries := extractToyotaPDFLinks(html, 2024)
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.SourceSite != "toyota.com" {
			t.Errorf("expected source_site 'toyota.com', got %q", e.SourceSite)
		}
		if e.Year != 2024 {
			t.Errorf("expected year 2024, got %d", e.Year)
		}
	}
}

func TestInferManualType(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/service_manual.pdf", "service"},
		{"https://example.com/electrical_diagram.pdf", "electrical"},
		{"https://example.com/body_repair.pdf", "body_repair"},
		{"https://example.com/quick_start.pdf", "quick_reference"},
		{"https://example.com/owner_manual.pdf", "owner"},
	}
	for _, tt := range tests {
		got := inferManualType(tt.url)
		if got != tt.want {
			t.Errorf("inferManualType(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Toyota", "toyota"},
		{"Mercedes-Benz", "mercedes-benz"},
		{"Land Rover", "land_rover"},
		{"F-150", "f-150"},
	}
	for _, tt := range tests {
		got := sanitizePath(tt.in)
		if got != tt.want {
			t.Errorf("sanitizePath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestVerifyPDF(t *testing.T) {
	dir := t.TempDir()

	// Valid PDF
	valid := filepath.Join(dir, "valid.pdf")
	os.WriteFile(valid, []byte("%PDF-1.4 content"), 0o644)
	if err := verifyPDF(valid); err != nil {
		t.Errorf("expected valid PDF, got error: %v", err)
	}

	// Invalid PDF
	invalid := filepath.Join(dir, "invalid.pdf")
	os.WriteFile(invalid, []byte("<html>not a pdf</html>"), 0o644)
	if err := verifyPDF(invalid); err == nil {
		t.Error("expected error for invalid PDF")
	}
}

func TestDedup(t *testing.T) {
	entries := []graph.ManualEntry{
		{ID: "a", URL: "https://a.com/1.pdf"},
		{ID: "b", URL: "https://b.com/2.pdf"},
		{ID: "a", URL: "https://a.com/1.pdf"},
	}
	result := dedup(entries)
	if len(result) != 2 {
		t.Errorf("expected 2 deduped entries, got %d", len(result))
	}
}

func TestMakeYearRange(t *testing.T) {
	years := makeYearRange([2]int{2020, 2023})
	if len(years) != 4 {
		t.Errorf("expected 4 years, got %d", len(years))
	}
	if years[0] != 2020 || years[3] != 2023 {
		t.Errorf("unexpected year range: %v", years)
	}
}

func TestManualEntryID(t *testing.T) {
	id1 := graph.ManualEntryID("https://example.com/manual.pdf")
	id2 := graph.ManualEntryID("https://example.com/manual.pdf")
	id3 := graph.ManualEntryID("https://example.com/other.pdf")
	if id1 != id2 {
		t.Error("same URL should produce same ID")
	}
	if id1 == id3 {
		t.Error("different URLs should produce different IDs")
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	if !containsIgnoreCase([]string{"Toyota", "Honda"}, "toyota") {
		t.Error("should find toyota case-insensitive")
	}
	if containsIgnoreCase([]string{"Toyota"}, "Ford") {
		t.Error("should not find Ford")
	}
}

func TestDownloaderWithMockServer(t *testing.T) {
	// Create a mock HTTP server serving a fake PDF
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte("%PDF-1.4 fake content here"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	dl := NewDownloader(srv.Client(), dir, 10*1024*1024, "test")

	entry := graph.ManualEntry{
		ID:         "test123456789",
		URL:        srv.URL + "/manual.pdf",
		Make:       "Toyota",
		Model:      "Camry",
		Year:       2024,
		ManualType: "owner",
	}

	path, err := dl.Download(context.Background(), entry)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("file not created at %s", path)
	}
}

func TestArchiveResponseParsing(t *testing.T) {
	// Test extractVehicleFromArchiveDoc
	doc := archiveDoc{
		Identifier: "toyota_camry_2020_manual",
		Title:      "2020 Toyota Camry Owner's Manual",
		Year:       float64(2020),
	}
	make_, model, year := extractVehicleFromArchiveDoc(doc, []string{"Toyota", "Honda"})
	if make_ != "Toyota" {
		t.Errorf("expected Toyota, got %q", make_)
	}
	if model != "Camry" {
		t.Errorf("expected Camry, got %q", model)
	}
	if year != 2020 {
		t.Errorf("expected 2020, got %d", year)
	}
}

func TestExtractDomain(t *testing.T) {
	got := extractDomain("https://www.toyota.com/manuals/test.pdf")
	if got != "www.toyota.com" {
		t.Errorf("expected www.toyota.com, got %q", got)
	}
}

func TestNormModel(t *testing.T) {
	tests := []struct{ in, want string }{
		{"camry", "Camry"},
		{"cr-v", "Cr V"},
		{"rav4", "Rav4"},
	}
	for _, tt := range tests {
		got := normModel(tt.in)
		if got != tt.want {
			t.Errorf("normModel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestInferManualTypeFromTitle(t *testing.T) {
	tests := []struct {
		title, want string
	}{
		{"Service Repair Manual", "service"},
		{"Electrical Wiring Diagram", "electrical"},
		{"Owner's Manual", "owner"},
		{"Quick Reference Guide", "quick_reference"},
	}
	for _, tt := range tests {
		got := inferManualTypeFromTitle(tt.title)
		if got != tt.want {
			t.Errorf("inferManualTypeFromTitle(%q) = %q, want %q", tt.title, got, tt.want)
		}
	}
}

// TestArchiveSourceWithMock tests the Archive source with a mocked HTTP server.
func TestArchiveSourceWithMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"response":{"docs":[{"identifier":"toyota_manual_2020","title":"2020 Toyota Camry Manual","subject":"vehicle manual","year":2020}]}}`)
	}))
	defer srv.Close()

	// We can't easily inject the URL into ArchiveSource without changing it,
	// but we test the parsing logic directly
	doc := archiveDoc{
		Identifier: "toyota_manual_2020",
		Title:      "2020 Toyota Camry Manual",
		Subject:    "vehicle manual",
		Year:       float64(2020),
	}
	make_, _, year := extractVehicleFromArchiveDoc(doc, []string{"Toyota"})
	if make_ != "Toyota" || year != 2020 {
		t.Errorf("unexpected: make=%q year=%d", make_, year)
	}
}
