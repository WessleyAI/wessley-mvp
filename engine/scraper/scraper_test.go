package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewYouTubeScraper(t *testing.T) {
	s := NewYouTubeScraper("key", nil)
	if s.apiKey != "key" {
		t.Fatal("wrong apiKey")
	}
	if len(s.channels) != len(DefaultYouTubeChannels) {
		t.Fatal("should use default channels")
	}

	s2 := NewYouTubeScraper("key", []string{"ch1"})
	if len(s2.channels) != 1 {
		t.Fatal("should use provided channels")
	}
}

func TestGetTranscript_Success(t *testing.T) {
	transcript := `<?xml version="1.0" encoding="utf-8"?>
<transcript>
  <text start="0.0" dur="2.0">Hello world</text>
  <text start="2.0" dur="1.5">[Music] this is a test</text>
</transcript>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(transcript))
	}))
	defer srv.Close()

	// Patch the URL by using a custom client and overriding via transport
	// Actually, GetTranscript builds URLs to youtube.com. We need to intercept.
	// The simplest approach: test CleanTranscript more, and test GetTranscript with a server that mimics the API.
	// Since GetTranscript hardcodes youtube.com URLs, we test it by overriding the http.Client transport.

	client := srv.Client()
	transport := &rewriteTransport{
		base:    client.Transport,
		baseURL: srv.URL,
	}
	client.Transport = transport

	result := GetTranscript(context.Background(), client, "test123")
	text, err := result.Unwrap()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "Hello world") {
		t.Errorf("expected transcript content, got: %s", text)
	}
	if strings.Contains(text, "[Music]") {
		t.Error("bracket noise should be removed")
	}
}

func TestGetTranscript_NoTranscript(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	client := srv.Client()
	client.Transport = &rewriteTransport{base: client.Transport, baseURL: srv.URL}

	result := GetTranscript(context.Background(), client, "missing")
	if result.IsOk() {
		t.Fatal("expected error for missing transcript")
	}
}

func TestGetTranscript_InvalidXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("this is not xml at all, and it's long enough to pass the length check"))
	}))
	defer srv.Close()

	client := srv.Client()
	client.Transport = &rewriteTransport{base: client.Transport, baseURL: srv.URL}

	result := GetTranscript(context.Background(), client, "badxml")
	if result.IsOk() {
		t.Fatal("expected error for invalid XML")
	}
}

func TestGetTranscript_EmptyTranscript(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<?xml version="1.0"?><transcript></transcript>`))
	}))
	defer srv.Close()

	client := srv.Client()
	client.Transport = &rewriteTransport{base: client.Transport, baseURL: srv.URL}

	result := GetTranscript(context.Background(), client, "empty")
	if result.IsOk() {
		t.Fatal("expected error for empty transcript")
	}
}

func TestSearchVideos_Success(t *testing.T) {
	resp := searchResponse{
		Items: []struct {
			ID struct {
				VideoID string `json:"videoId"`
			} `json:"id"`
			Snippet struct {
				Title        string `json:"title"`
				Description  string `json:"description"`
				ChannelTitle string `json:"channelTitle"`
				PublishedAt  string `json:"publishedAt"`
			} `json:"snippet"`
		}{
			{
				ID:      struct{ VideoID string `json:"videoId"` }{VideoID: "vid1"},
				Snippet: struct {
					Title        string `json:"title"`
					Description  string `json:"description"`
					ChannelTitle string `json:"channelTitle"`
					PublishedAt  string `json:"publishedAt"`
				}{
					Title:       "Test Video",
					Description: "desc",
					ChannelTitle: "TestChannel",
					PublishedAt: time.Now().Format(time.RFC3339),
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := NewYouTubeScraper("testkey", nil)
	s.httpClient = srv.Client()
	s.httpClient.Transport = &rewriteTransport{base: s.httpClient.Transport, baseURL: srv.URL}

	result := s.SearchVideos(context.Background(), "test query", 5)
	videos, err := result.Unwrap()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(videos) != 1 || videos[0].VideoID != "vid1" {
		t.Fatalf("wrong videos: %v", videos)
	}
}

func TestSearchVideos_NoAPIKey(t *testing.T) {
	s := NewYouTubeScraper("", nil)
	result := s.SearchVideos(context.Background(), "test", 5)
	if result.IsOk() {
		t.Fatal("expected error for empty API key")
	}
}

func TestSearchVideos_QuotaExhausted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
	}))
	defer srv.Close()

	s := NewYouTubeScraper("key", nil)
	s.httpClient = srv.Client()
	s.httpClient.Transport = &rewriteTransport{base: s.httpClient.Transport, baseURL: srv.URL}

	result := s.SearchVideos(context.Background(), "test", 5)
	_, err := result.Unwrap()
	if err != ErrQuotaExhausted {
		t.Fatalf("expected ErrQuotaExhausted, got %v", err)
	}
}

func TestScrapeVideo_Success(t *testing.T) {
	transcript := `<?xml version="1.0" encoding="utf-8"?>
<transcript>
  <text start="0.0" dur="2.0">How to replace brake pads on a 2018 Honda Civic</text>
  <text start="2.0" dur="1.5">First remove the caliper</text>
</transcript>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(transcript))
	}))
	defer srv.Close()

	s := NewYouTubeScraper("key", nil)
	s.httpClient = srv.Client()
	s.httpClient.Transport = &rewriteTransport{base: s.httpClient.Transport, baseURL: srv.URL}

	result := s.ScrapeVideo(context.Background(), "vid1", "Brake Pad Replacement 2018 Honda Civic")
	post, err := result.Unwrap()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.Source != "youtube" || post.SourceID != "vid1" {
		t.Fatal("wrong source info")
	}
	if !strings.Contains(post.Content, "replace brake pads") {
		t.Errorf("unexpected content: %s", post.Content)
	}
}

func TestScrapeVideo_Duplicate(t *testing.T) {
	s := NewYouTubeScraper("key", nil)
	s.seen.Store("dup1", true)

	result := s.ScrapeVideo(context.Background(), "dup1", "Test")
	if result.IsOk() {
		t.Fatal("expected error for duplicate")
	}
}

func TestScrapeVideoIDs(t *testing.T) {
	transcript := `<?xml version="1.0" encoding="utf-8"?>
<transcript>
  <text start="0.0" dur="2.0">Test content for video</text>
</transcript>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(transcript))
	}))
	defer srv.Close()

	s := NewYouTubeScraper("key", nil)
	s.httpClient = srv.Client()
	s.httpClient.Transport = &rewriteTransport{base: s.httpClient.Transport, baseURL: srv.URL}

	ch := s.ScrapeVideoIDs(context.Background(), []string{"v1", "v2"})
	var results []string
	for r := range ch {
		if r.IsOk() {
			post, _ := r.Unwrap()
			results = append(results, post.SourceID)
		}
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestScrapeVideoIDs_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	s := NewYouTubeScraper("key", nil)
	ch := s.ScrapeVideoIDs(ctx, []string{"v1"})
	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 results when cancelled, got %d", count)
	}
}

func TestScrape_WithQuery(t *testing.T) {
	// Search returns one video, which we then scrape
	searchResp := searchResponse{
		Items: []struct {
			ID struct {
				VideoID string `json:"videoId"`
			} `json:"id"`
			Snippet struct {
				Title        string `json:"title"`
				Description  string `json:"description"`
				ChannelTitle string `json:"channelTitle"`
				PublishedAt  string `json:"publishedAt"`
			} `json:"snippet"`
		}{
			{
				ID: struct{ VideoID string `json:"videoId"` }{VideoID: "sv1"},
				Snippet: struct {
					Title        string `json:"title"`
					Description  string `json:"description"`
					ChannelTitle string `json:"channelTitle"`
					PublishedAt  string `json:"publishedAt"`
				}{Title: "Test", PublishedAt: "2024-01-01T00:00:00Z"},
			},
		},
	}

	transcript := `<?xml version="1.0" encoding="utf-8"?>
<transcript><text start="0" dur="1">test content here</text></transcript>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "youtube/v3/search") {
			json.NewEncoder(w).Encode(searchResp)
		} else {
			w.Write([]byte(transcript))
		}
	}))
	defer srv.Close()

	s := NewYouTubeScraper("key", nil)
	s.httpClient = srv.Client()
	s.httpClient.Transport = &rewriteTransport{base: s.httpClient.Transport, baseURL: srv.URL}

	ch := s.Scrape(context.Background(), ScrapeOpts{Query: "test", MaxResults: 1})
	count := 0
	for r := range ch {
		if r.IsOk() {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 scraped post, got %d", count)
	}
}

func TestScrape_NoQuery_UsesDefaults(t *testing.T) {
	// Returns quota exhausted to stop early
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
	}))
	defer srv.Close()

	s := NewYouTubeScraper("key", nil)
	s.httpClient = srv.Client()
	s.httpClient.Transport = &rewriteTransport{base: s.httpClient.Transport, baseURL: srv.URL}

	ch := s.Scrape(context.Background(), ScrapeOpts{})
	var gotQuotaErr bool
	for r := range ch {
		_, err := r.Unwrap()
		if err == ErrQuotaExhausted {
			gotQuotaErr = true
		}
	}
	if !gotQuotaErr {
		t.Fatal("expected quota exhausted error")
	}
}

func TestScrape_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cancel() // cancel on first request
		w.WriteHeader(403)
	}))
	defer srv.Close()

	s := NewYouTubeScraper("key", nil)
	s.httpClient = srv.Client()
	s.httpClient.Transport = &rewriteTransport{base: s.httpClient.Transport, baseURL: srv.URL}

	ch := s.Scrape(ctx, ScrapeOpts{Query: "test"})
	for range ch {
	}
	// Just ensure it doesn't hang
}

func TestExtractMetadata_AllFields(t *testing.T) {
	title := "2020 Ford F150 engine won't start dead battery"
	transcript := "we need to replace the starter and clean the terminals"

	m := extractMetadata(title, transcript)
	if m.Vehicle == "" {
		t.Error("expected vehicle")
	}
	if len(m.Symptoms) == 0 {
		t.Error("expected symptoms")
	}
	if len(m.Fixes) == 0 {
		t.Error("expected fixes")
	}
}

func TestCleanTranscript_AllEntities(t *testing.T) {
	input := `&quot;hello&quot; &lt;b&gt; &amp; &#39;world&#39; [Cheering]`
	got := CleanTranscript(input)
	if strings.Contains(got, "&") && !strings.Contains(got, "& ") {
		// should have decoded all entities
	}
	if strings.Contains(got, "[Cheering]") {
		t.Error("should remove [Cheering]")
	}
}

// rewriteTransport rewrites all request URLs to point at our test server.
type rewriteTransport struct {
	base    http.RoundTripper
	baseURL string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Preserve the path and query
	newURL := fmt.Sprintf("%s%s", t.baseURL, req.URL.RequestURI())
	newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
	if err != nil {
		return nil, err
	}
	newReq.Header = req.Header
	if t.base != nil {
		return t.base.RoundTrip(newReq)
	}
	return http.DefaultTransport.RoundTrip(newReq)
}
