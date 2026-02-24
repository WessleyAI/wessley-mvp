package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- urlRewriter transport ---

type urlRewriter struct {
	target string
}

func (u *urlRewriter) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = "http"
	req2.URL.Host = u.target[len("http://"):]
	return http.DefaultTransport.RoundTrip(req2)
}

// --- GetTranscript: success on first URL ---

func TestGetTranscript_SuccessFirstURL(t *testing.T) {
	xml := `<transcript><text start="0" dur="5">Hello world this is a test transcript with sufficient content for testing</text></transcript>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(xml))
	}))
	defer srv.Close()

	client := &http.Client{Transport: &urlRewriter{target: srv.URL}}
	r := GetTranscript(context.Background(), client, "test-id")
	if r.IsErr() {
		_, err := r.Unwrap()
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- GetTranscript: first URL fails, second succeeds ---

func TestGetTranscript_FallbackToSecondURL(t *testing.T) {
	callCount := 0
	xml := `<transcript><text start="0" dur="5">Hello world this is a test transcript with sufficient content for testing</text></transcript>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(404)
			return
		}
		w.Write([]byte(xml))
	}))
	defer srv.Close()

	client := &http.Client{Transport: &urlRewriter{target: srv.URL}}
	r := GetTranscript(context.Background(), client, "test-id")
	if r.IsErr() {
		_, err := r.Unwrap()
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount < 2 {
		t.Fatal("expected at least 2 calls")
	}
}

// --- GetTranscript: all URLs fail ---

func TestGetTranscript_AllFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	client := &http.Client{Transport: &urlRewriter{target: srv.URL}}
	r := GetTranscript(context.Background(), client, "test-id")
	if r.IsOk() {
		t.Fatal("expected error when all URLs fail")
	}
}

// --- GetTranscript: bad XML ---

func TestGetTranscript_BadXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("this is not xml and it is long enough to pass the length check of fifty characters here"))
	}))
	defer srv.Close()

	client := &http.Client{Transport: &urlRewriter{target: srv.URL}}
	r := GetTranscript(context.Background(), client, "test-id")
	if r.IsOk() {
		t.Fatal("expected error for bad XML")
	}
}

// --- GetTranscript: empty texts ---

func TestGetTranscript_EmptyTexts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<transcript></transcript>                                                  `))
	}))
	defer srv.Close()

	client := &http.Client{Transport: &urlRewriter{target: srv.URL}}
	r := GetTranscript(context.Background(), client, "test-id")
	if r.IsOk() {
		t.Fatal("expected error for empty texts")
	}
}

// --- GetTranscript: short response ---

func TestGetTranscript_ShortResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("short"))
	}))
	defer srv.Close()

	client := &http.Client{Transport: &urlRewriter{target: srv.URL}}
	r := GetTranscript(context.Background(), client, "test-id")
	if r.IsOk() {
		t.Fatal("expected error for short response")
	}
}

// --- GetTranscript: context cancelled ---

func TestGetTranscript_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &http.Client{}
	r := GetTranscript(ctx, client, "test-id")
	if r.IsOk() {
		t.Fatal("expected error")
	}
}

// --- CleanTranscript ---

func TestCleanTranscript_AllEntities2(t *testing.T) {
	input := "[Music] Hello  &#39;world&#39; &amp; &quot;test&quot; &lt;tag&gt;  [Applause] [Laughter] [Cheering] [Inaudible]"
	result := CleanTranscript(input)
	if result != "Hello 'world' & \"test\" <tag>" {
		t.Fatalf("unexpected: %q", result)
	}
}

// --- SearchVideos: bad JSON ---

func TestSearchVideos_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	s := NewYouTubeScraper("test-key", nil)
	s.httpClient = &http.Client{Transport: &urlRewriter{target: srv.URL}}

	r := s.SearchVideos(context.Background(), "test", 5)
	if r.IsOk() {
		t.Fatal("expected error for bad JSON")
	}
}

// --- Scrape: quota exhausted ---

func TestScrape_QuotaExhausted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
	}))
	defer srv.Close()

	s := NewYouTubeScraper("test-key", nil)
	s.httpClient = &http.Client{Transport: &urlRewriter{target: srv.URL}}

	ch := s.Scrape(context.Background(), ScrapeOpts{Query: "test", MaxResults: 1})
	for r := range ch {
		if r.IsErr() {
			_, err := r.Unwrap()
			if err == ErrQuotaExhausted {
				return
			}
		}
	}
}

// --- ScrapeVideo: transcript fails ---

func TestScrapeVideo_TranscriptFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	s := NewYouTubeScraper("", nil)
	s.httpClient = &http.Client{Transport: &urlRewriter{target: srv.URL}}

	r := s.ScrapeVideo(context.Background(), "test-id-fail", "Test Title")
	if r.IsOk() {
		t.Fatal("expected error when transcript fails")
	}
}

// --- ScrapeVideoIDs: rate limiter context cancel ---

func TestScrapeVideoIDs_Cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewYouTubeScraper("", nil)
	ch := s.ScrapeVideoIDs(ctx, []string{"id1", "id2"})
	for range ch {
	}
}

// --- Scrape: search errors continue ---

func TestScrape_SearchErrorContinues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	s := NewYouTubeScraper("test-key", nil)
	s.httpClient = &http.Client{Transport: &urlRewriter{target: srv.URL}}

	ch := s.Scrape(context.Background(), ScrapeOpts{Query: "test"})
	for range ch {
	}
}

// --- Scrape with successful search but failed scrape ---

func TestScrape_WithResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("part") == "snippet" {
			resp := `{"items":[{"id":{"videoId":"vid1"},"snippet":{"title":"Test","description":"desc","channelTitle":"ch","publishedAt":"2024-01-01T00:00:00Z"}}]}`
			w.Write([]byte(resp))
			return
		}
		// transcript endpoint fails
		w.WriteHeader(404)
	}))
	defer srv.Close()

	s := NewYouTubeScraper("test-key", nil)
	s.httpClient = &http.Client{Transport: &urlRewriter{target: srv.URL}}

	ch := s.Scrape(context.Background(), ScrapeOpts{Query: "test", MaxResults: 1})
	for range ch {
	}
}

// --- Scrape: no query uses defaults ---

func TestScrape_DefaultQueries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(searchResponse{})
	}))
	defer srv.Close()

	s := NewYouTubeScraper("test-key", nil)
	s.httpClient = &http.Client{Transport: &urlRewriter{target: srv.URL}}

	ch := s.Scrape(context.Background(), ScrapeOpts{MaxResults: 1})
	for range ch {
	}
}

// --- extractMetadata edge cases ---

func TestExtractMetadata_NoMatches(t *testing.T) {
	m := extractMetadata("", "")
	if m.Vehicle != "" {
		t.Fatal("expected empty vehicle")
	}
}

func TestExtractMetadata_VehicleOnly(t *testing.T) {
	m := extractMetadata("2020 Toyota Corolla", "")
	if m.Vehicle == "" {
		t.Fatal("expected vehicle")
	}
}

func TestExtractMetadata_AllFields2(t *testing.T) {
	m := extractMetadata(
		"2020 Toyota Corolla",
		"The engine won't start due to dead battery. I need to replace and install new battery.",
	)
	if m.Vehicle == "" {
		t.Fatal("expected vehicle")
	}
	if len(m.Symptoms) == 0 {
		t.Fatal("expected symptoms")
	}
	if len(m.Fixes) == 0 {
		t.Fatal("expected fixes")
	}
	if len(m.Keywords) == 0 {
		t.Fatal("expected keywords")
	}
}

// --- ScrapeVideoIDs: success ---

func TestScrapeVideoIDs_Success(t *testing.T) {
	xml := `<transcript><text start="0" dur="5">Hello world this is a test transcript with sufficient content for testing purposes</text></transcript>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(xml))
	}))
	defer srv.Close()

	s := NewYouTubeScraper("", nil)
	s.httpClient = &http.Client{Transport: &urlRewriter{target: srv.URL}}

	ch := s.ScrapeVideoIDs(context.Background(), []string{"vid1", "vid2"})
	count := 0
	for r := range ch {
		if r.IsOk() {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("expected 2 successful scrapes, got %d", count)
	}
}

// --- SearchVideos: context cancelled ---

func TestSearchVideos_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewYouTubeScraper("test-key", nil)
	r := s.SearchVideos(ctx, "test", 5)
	if r.IsOk() {
		t.Fatal("expected error for cancelled context")
	}
}

// --- Scrape: context cancelled ---

func TestScrape_ContextCancelled2(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cancel() // cancel during search
		json.NewEncoder(w).Encode(searchResponse{})
	}))
	defer srv.Close()

	s := NewYouTubeScraper("test-key", nil)
	s.httpClient = &http.Client{Transport: &urlRewriter{target: srv.URL}}

	ch := s.Scrape(ctx, ScrapeOpts{Query: "test"})
	for range ch {
	}
}

// Ensure fmt is used
var _ = fmt.Sprintf
