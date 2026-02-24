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
	guides := []Guide{
		{
			GuideID:      1001,
			Title:        "Replace Brake Pads",
			Summary:      "How to replace brake pads on a 2018 Honda Civic",
			URL:          "https://ifixit.com/Guide/1001",
			Category:     "Car_Brakes",
			Subject:      "2018 Honda Civic",
			Difficulty:   "Moderate",
			Author:       struct{ Username string `json:"username"` }{Username: "mechanic1"},
			ModifiedDate: 1700000000,
			Steps: []Step{
				{OrderBy: 1, Title: "Remove wheel", Lines: []Line{{Text: "Remove the lug nuts and take off the wheel"}}},
				{OrderBy: 2, Title: "", Lines: []Line{{Text: "Remove caliper bolts"}}},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(guides)
	}))
	defer srv.Close()

	s := NewScraper(Config{Categories: []string{"Car_Brakes"}, MaxGuides: 10, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	p := posts[0]
	if p.Source != "ifixit" {
		t.Errorf("unexpected source: %s", p.Source)
	}
	if p.SourceID != "ifixit-1001" {
		t.Errorf("unexpected sourceID: %s", p.SourceID)
	}
	if p.Author != "mechanic1" {
		t.Errorf("unexpected author: %s", p.Author)
	}
	if !strings.Contains(p.Content, "Remove the lug nuts") {
		t.Errorf("expected content to have step text")
	}
	if p.Metadata.Vehicle != "2018 Honda Civic" {
		t.Errorf("unexpected vehicle: %s", p.Metadata.Vehicle)
	}
	if len(p.Metadata.Fixes) == 0 {
		t.Error("expected fixes extracted")
	}
	if len(p.Metadata.Keywords) < 3 {
		t.Errorf("expected at least 3 keywords, got %v", p.Metadata.Keywords)
	}
}

func TestFetchAll_MultipleCategories(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Guide{{GuideID: 1, Title: "Test", ModifiedDate: 1700000000, Author: struct{ Username string `json:"username"` }{Username: "u"}}})
	}))
	defer srv.Close()

	s := NewScraper(Config{Categories: []string{"A", "B"}, MaxGuides: 10, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}
}

func TestFetchAll_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	s := NewScraper(Config{Categories: []string{"A", "B"}, MaxGuides: 10, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	posts, _ := s.FetchAll(ctx)
	if len(posts) != 0 {
		t.Fatalf("expected 0, got %d", len(posts))
	}
}

func TestFetchAll_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	s := NewScraper(Config{Categories: []string{"A"}, MaxGuides: 10, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("expected 0, got %d", len(posts))
	}
}

func TestFetchAll_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	s := NewScraper(Config{Categories: []string{"A"}, MaxGuides: 10, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, _ := s.FetchAll(context.Background())
	if len(posts) != 0 {
		t.Fatalf("expected 0, got %d", len(posts))
	}
}

func TestFetchAll_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
	}))
	defer srv.Close()

	s := NewScraper(Config{Categories: []string{"A"}, MaxGuides: 10, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, _ := s.FetchAll(context.Background())
	if len(posts) != 0 {
		t.Fatalf("expected 0, got %d", len(posts))
	}
}

func TestFetchAll_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	s := NewScraper(Config{Categories: []string{"A"}, MaxGuides: 10, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, _ := s.FetchAll(context.Background())
	if len(posts) != 0 {
		t.Fatalf("expected 0, got %d", len(posts))
	}
}

func TestDoGet_UserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		json.NewEncoder(w).Encode([]Guide{})
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	s.doGet(context.Background(), srv.URL+"/test")
	if !strings.Contains(gotUA, "wessley-scraper") {
		t.Errorf("expected wessley-scraper UA, got %s", gotUA)
	}
}

func TestDoGet_CancelledContext(t *testing.T) {
	s := NewScraper(Config{RateLimit: time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := s.doGet(ctx, "http://localhost:1/test")
	if result.IsOk() {
		t.Fatal("expected error")
	}
}

func TestBuildGuideContent_EmptySummary(t *testing.T) {
	g := Guide{
		Steps: []Step{
			{OrderBy: 1, Title: "Step one", Lines: []Line{{Text: "Do something"}}},
		},
	}
	content := buildGuideContent(g)
	if strings.HasPrefix(content, "\n\n") {
		t.Error("should not start with double newline when summary empty")
	}
	if !strings.Contains(content, "Step 1: Step one") {
		t.Errorf("unexpected content: %s", content)
	}
}

func TestBuildGuideContent_StepWithoutTitle(t *testing.T) {
	g := Guide{
		Steps: []Step{
			{OrderBy: 1, Title: "", Lines: []Line{{Text: "Do something"}}},
		},
	}
	content := buildGuideContent(g)
	if !strings.Contains(content, "Step 1:\n") {
		t.Errorf("expected untitled step format, got: %s", content)
	}
}

func TestBuildGuideContent_MultipleLines(t *testing.T) {
	g := Guide{
		Summary: "Summary text",
		Steps: []Step{
			{OrderBy: 1, Title: "First", Lines: []Line{{Text: "Line A"}, {Text: "Line B"}}},
			{OrderBy: 2, Title: "Second", Lines: []Line{{Text: "Line C"}}},
		},
	}
	content := buildGuideContent(g)
	if !strings.Contains(content, "Line A") || !strings.Contains(content, "Line B") || !strings.Contains(content, "Line C") {
		t.Errorf("missing lines in content: %s", content)
	}
}

func TestExtractFixes_All(t *testing.T) {
	text := "replace remove install tighten disconnect reconnect drain refill bleed adjust align lubricate clean inspect"
	fixes := extractFixes(text)
	if len(fixes) != 14 {
		t.Errorf("expected 14 fixes, got %d: %v", len(fixes), fixes)
	}
}

func TestExtractFixes_None(t *testing.T) {
	fixes := extractFixes("nothing relevant here at all")
	if len(fixes) != 0 {
		t.Errorf("expected 0 fixes, got %v", fixes)
	}
}
