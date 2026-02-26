package manuals

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChevroletSourceDiscover(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<html><a href="https://my.chevrolet.com/manuals/silverado-2024.pdf">manual</a></html>`)
	}))
	defer srv.Close()

	s := NewChevroletSource()
	entries, err := s.Discover(context.Background(), []string{"Chevrolet"}, []int{2024})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected entries from Chevrolet source")
	}
	found := false
	for _, e := range entries {
		if e.Make == "Chevrolet" && e.Year == 2024 {
			found = true
			break
		}
	}
	if !found {
		t.Error("no Chevrolet 2024 entry found")
	}
}

func TestChevroletSourceSkipsOtherMakes(t *testing.T) {
	s := NewChevroletSource()
	entries, err := s.Discover(context.Background(), []string{"Toyota"}, []int{2024})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no entries for Toyota, got %d", len(entries))
	}
}

func TestTeslaSourceDiscover(t *testing.T) {
	s := NewTeslaSource()
	entries, err := s.Discover(context.Background(), []string{"Tesla"}, []int{2024})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected entries from Tesla source")
	}
	for _, e := range entries {
		if e.Make != "Tesla" {
			t.Errorf("expected make Tesla, got %s", e.Make)
		}
		if e.SourceSite != "www.tesla.com" {
			t.Errorf("expected source site www.tesla.com, got %s", e.SourceSite)
		}
	}
}

func TestNissanSourceDiscover(t *testing.T) {
	s := NewNissanSource()
	entries, err := s.Discover(context.Background(), []string{"Nissan"}, []int{2024})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected entries from Nissan source")
	}
	modelFound := map[string]bool{}
	for _, e := range entries {
		if e.Make != "Nissan" {
			t.Errorf("expected make Nissan, got %s", e.Make)
		}
		modelFound[e.Model] = true
	}
	for _, expected := range []string{"Altima", "Rogue", "Frontier"} {
		if !modelFound[expected] {
			t.Errorf("expected model %s not found", expected)
		}
	}
}

func TestExtractPDFLinks(t *testing.T) {
	html := `<a href="https://example.com/manual.pdf">link</a>
<a href="https://example.com/camry-service.pdf">service</a>
<a href="https://example.com/manual.pdf">dup</a>`

	entries := extractPDFLinks(html, "example.com", "Toyota", 2024)
	if len(entries) != 2 {
		t.Errorf("expected 2 entries (deduped), got %d", len(entries))
	}
}

func TestInferModelFromURLNewMakes(t *testing.T) {
	tests := []struct {
		url, make_, expected string
	}{
		{"https://example.com/silverado-manual.pdf", "Chevrolet", "Silverado"},
		{"https://example.com/model-3-owners.pdf", "Tesla", "Model 3"},
		{"https://example.com/cx-5-manual.pdf", "Mazda", "Cx 5"},
		{"https://example.com/escalade-2024.pdf", "Cadillac", "Escalade"},
		{"https://example.com/unknown.pdf", "Chevrolet", ""},
	}
	for _, tt := range tests {
		got := inferModelFromURL(tt.url, tt.make_)
		if got != tt.expected {
			t.Errorf("inferModelFromURL(%q, %q) = %q, want %q", tt.url, tt.make_, got, tt.expected)
		}
	}
}
