package ifixit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type redirectTransport struct {
	server *httptest.Server
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.server.URL, "http://")
	return http.DefaultTransport.RoundTrip(req)
}

func TestFetchAll_Success(t *testing.T) {
	resp := searchResponse{
		TotalResults: 2,
		Results: []searchResult{
			{
				DataType:     "guide",
				GuideID:      1001,
				Title:        "Replace Brake Pads",
				DisplayTitle: "How to Replace Brake Pads",
				Summary:      "Remove old pads and install new ones",
				URL:          "https://ifixit.com/Guide/1001",
				ModifiedDate: 1700000000,
			},
			{
				DataType:     "guide",
				GuideID:      1002,
				Title:        "Car Battery Replacement",
				Summary:      "Replace your car battery",
				URL:          "https://ifixit.com/Guide/1002",
				ModifiedDate: 1700000000,
			},
			{
				DataType: "wiki",
				Title:    "Car and Truck",
				URL:      "https://ifixit.com/Device/Car",
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := NewScraper(Config{Categories: []string{"Car"}, MaxGuides: 50, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	// Should only include guides, not wiki entries
	if len(posts) < 2 {
		t.Fatalf("expected at least 2 guide posts, got %d", len(posts))
	}
	if posts[0].Source != "ifixit" {
		t.Errorf("unexpected source: %s", posts[0].Source)
	}
	// DisplayTitle should be used when available
	if posts[0].Title != "How to Replace Brake Pads" {
		t.Errorf("expected display title, got: %s", posts[0].Title)
	}
}

func TestFetchAll_MaxGuides(t *testing.T) {
	resp := searchResponse{
		TotalResults: 3,
		Results: []searchResult{
			{DataType: "guide", GuideID: 1, Title: "Guide 1", URL: "https://ifixit.com/1", ModifiedDate: 1700000000},
			{DataType: "guide", GuideID: 2, Title: "Guide 2", URL: "https://ifixit.com/2", ModifiedDate: 1700000000},
			{DataType: "guide", GuideID: 3, Title: "Guide 3", URL: "https://ifixit.com/3", ModifiedDate: 1700000000},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := NewScraper(Config{Categories: []string{"Car"}, MaxGuides: 2, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(posts) > 2 {
		t.Fatalf("expected at most 2 posts (MaxGuides=2), got %d", len(posts))
	}
}

func TestFetchAll_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	s := NewScraper(Config{Categories: []string{"Car"}, MaxGuides: 10, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("expected 0 posts, got %d", len(posts))
	}
}

func TestFetchAll_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	s := NewScraper(Config{Categories: []string{"Car"}, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	posts, _ := s.FetchAll(ctx)
	if len(posts) != 0 {
		t.Fatalf("expected 0 posts, got %d", len(posts))
	}
}

func TestFetchAll_Dedup(t *testing.T) {
	resp := searchResponse{
		TotalResults: 2,
		Results: []searchResult{
			{DataType: "guide", GuideID: 1, Title: "Same Guide", URL: "https://ifixit.com/1", ModifiedDate: 1700000000},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := NewScraper(Config{Categories: []string{"Car"}, MaxGuides: 50, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, _ := s.FetchAll(context.Background())
	// Same guide returned from multiple queries should be deduped
	guideCount := 0
	for _, p := range posts {
		if p.SourceID == "ifixit-1" {
			guideCount++
		}
	}
	if guideCount > 1 {
		t.Errorf("expected deduped guide, got %d copies", guideCount)
	}
}

func TestExtractFixes(t *testing.T) {
	tests := []struct {
		input    string
		minCount int
	}{
		{"Remove the old part and install the new one", 2},
		{"Clean and inspect the component, then tighten", 3},
		{"nothing relevant here", 0},
	}
	for _, tt := range tests {
		fixes := extractFixes(tt.input)
		if len(fixes) < tt.minCount {
			t.Errorf("input %q: expected at least %d fixes, got %v", tt.input, tt.minCount, fixes)
		}
	}
}
