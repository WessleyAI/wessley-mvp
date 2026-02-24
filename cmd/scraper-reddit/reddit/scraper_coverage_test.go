package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// redirectTransport intercepts HTTP requests and redirects them to a test server.
type redirectTransport struct {
	server *httptest.Server
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the URL to point to our test server
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.server.URL, "http://")
	return http.DefaultTransport.RoundTrip(req)
}

func newTestScraper(handler http.Handler, cfg Config) (*Scraper, *httptest.Server) {
	srv := httptest.NewServer(handler)
	s := NewScraper(cfg)
	s.client = &http.Client{
		Transport: &redirectTransport{server: srv},
		Timeout:   5 * time.Second,
	}
	return s, srv
}

func TestFetchAll_Success(t *testing.T) {
	listing := listingResponse{}
	listing.Data.Children = []listingChild{
		{
			Kind: "t3",
			Data: listingData{
				ID:            "abc123",
				Subreddit:     "MechanicAdvice",
				Title:         "Brake squeal",
				Author:        "user1",
				SelfText:      "My brakes squeal",
				Score:         42,
				NumComments:   2,
				CreatedUTC:    1700000000,
				Permalink:     "/r/MechanicAdvice/comments/abc123/brake/",
				LinkFlairText: "Brakes",
				URL:           "https://reddit.com/r/MechanicAdvice/abc123",
			},
		},
	}

	commentListing := listingResponse{}
	commentListing.Data.Children = []listingChild{
		{
			Kind: "t1",
			Data: listingData{
				ID:         "com1",
				Author:     "mechanic1",
				Body:       "Check pads",
				Score:      10,
				CreatedUTC: 1700001000,
				ParentID:   "t3_abc123",
				Depth:      0,
			},
		},
		{
			Kind: "more", // non-comment kind should be skipped
			Data: listingData{ID: "more1"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/new.json") {
			json.NewEncoder(w).Encode(listing)
		} else {
			json.NewEncoder(w).Encode([]listingResponse{listing, commentListing})
		}
	}))
	defer srv.Close()

	s := NewScraper(Config{
		Subreddits:      []string{"MechanicAdvice"},
		PostsPerSub:     10,
		CommentsPerPost: 10,
		RateLimit:       1 * time.Millisecond,
	})
	s.client = &http.Client{
		Transport: &redirectTransport{server: srv},
		Timeout:   5 * time.Second,
	}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].ID != "abc123" {
		t.Errorf("expected ID abc123, got %s", posts[0].ID)
	}
	if posts[0].Flair != "Brakes" {
		t.Errorf("expected flair Brakes, got %s", posts[0].Flair)
	}
	if len(posts[0].Comments) != 1 {
		t.Errorf("expected 1 comment, got %d", len(posts[0].Comments))
	}
	if posts[0].Comments[0].Body != "Check pads" {
		t.Errorf("unexpected comment body: %s", posts[0].Comments[0].Body)
	}
	if posts[0].Permalink != "https://www.reddit.com/r/MechanicAdvice/comments/abc123/brake/" {
		t.Errorf("unexpected permalink: %s", posts[0].Permalink)
	}
}

func TestFetchAll_MultipleSubreddits(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/new.json") {
			listing := listingResponse{}
			listing.Data.Children = []listingChild{
				{Kind: "t3", Data: listingData{ID: fmt.Sprintf("post%d", callCount), CreatedUTC: 1700000000, Permalink: fmt.Sprintf("/r/sub/comments/post%d/t/", callCount)}},
			}
			json.NewEncoder(w).Encode(listing)
		} else {
			// Return comments with only 1 listing (less than 2)
			json.NewEncoder(w).Encode([]listingResponse{{}})
		}
	}))
	defer srv.Close()

	s := NewScraper(Config{
		Subreddits:      []string{"Sub1", "Sub2"},
		PostsPerSub:     5,
		CommentsPerPost: 5,
		RateLimit:       1 * time.Millisecond,
	})
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
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	s := NewScraper(Config{
		Subreddits:  []string{"Sub1", "Sub2"},
		PostsPerSub: 5,
		RateLimit:   1 * time.Millisecond,
	})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	posts, _ := s.FetchAll(ctx)
	if len(posts) != 0 {
		t.Fatalf("expected 0 posts with cancelled ctx, got %d", len(posts))
	}
}

func TestFetchAll_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	s := NewScraper(Config{
		Subreddits:  []string{"Sub1"},
		PostsPerSub: 5,
		RateLimit:   1 * time.Millisecond,
	})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	// Should not fail hard, just log warning and return empty
	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("expected 0 posts, got %d", len(posts))
	}
}

func TestFetchAll_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
	}))
	defer srv.Close()

	s := NewScraper(Config{
		Subreddits:  []string{"Sub1"},
		PostsPerSub: 5,
		RateLimit:   1 * time.Millisecond,
	})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("expected 0 posts on rate limit, got %d", len(posts))
	}
}

func TestFetchAll_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer srv.Close()

	s := NewScraper(Config{
		Subreddits:  []string{"Sub1"},
		PostsPerSub: 5,
		RateLimit:   1 * time.Millisecond,
	})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("expected 0 posts, got %d", len(posts))
	}
}

func TestFetchAll_CommentError(t *testing.T) {
	listing := listingResponse{}
	listing.Data.Children = []listingChild{
		{Kind: "t3", Data: listingData{ID: "p1", CreatedUTC: 1700000000, Permalink: "/r/sub/comments/p1/t/"}},
	}

	callNum := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callNum++
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/new.json") {
			json.NewEncoder(w).Encode(listing)
		} else {
			// Return invalid JSON for comments
			w.Write([]byte("bad json"))
		}
	}))
	defer srv.Close()

	s := NewScraper(Config{
		Subreddits:      []string{"Sub1"},
		PostsPerSub:     5,
		CommentsPerPost: 5,
		RateLimit:       1 * time.Millisecond,
	})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	// Post should still be returned even with comment error
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if len(posts[0].Comments) != 0 {
		t.Errorf("expected 0 comments on error, got %d", len(posts[0].Comments))
	}
}

func TestHttpGet_StatusCodes(t *testing.T) {
	tests := []struct {
		name   string
		status int
		wantErr bool
	}{
		{"200 OK", 200, false},
		{"404 Not Found", 404, true},
		{"429 Rate Limit", 429, true},
		{"500 Server Error", 500, true},
		{"503 Unavailable", 503, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.status == 200 {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{}`))
				} else {
					w.WriteHeader(tt.status)
				}
			}))
			defer srv.Close()

			s := NewScraper(Config{RateLimit: time.Millisecond})
			s.client = srv.Client()

			body, err := s.httpGet(context.Background(), srv.URL+"/test")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				body.Close()
			}
		})
	}
}

func TestDoGet_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	result := s.doGet(context.Background(), srv.URL+"/test")
	if result.IsOk() {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestDoGet_ValidJSON(t *testing.T) {
	listing := listingResponse{}
	listing.Data.Children = []listingChild{{Kind: "t3", Data: listingData{ID: "abc"}}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(listing)
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	result := s.doGet(context.Background(), srv.URL+"/test")
	if !result.IsOk() {
		_, err := result.Unwrap()
		t.Fatalf("expected ok, got error: %v", err)
	}
	resp, _ := result.Unwrap()
	if len(resp.Data.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(resp.Data.Children))
	}
}

func TestDoGetComments_ValidJSON(t *testing.T) {
	postListing := listingResponse{}
	commentListing := listingResponse{}
	commentListing.Data.Children = []listingChild{
		{Kind: "t1", Data: listingData{ID: "c1", Author: "user", Body: "test body", CreatedUTC: 1700000000}},
		{Kind: "more", Data: listingData{ID: "more1"}},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]listingResponse{postListing, commentListing})
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	result := s.doGetComments(context.Background(), srv.URL+"/test")
	if !result.IsOk() {
		_, err := result.Unwrap()
		t.Fatalf("expected ok, got: %v", err)
	}
	comments, _ := result.Unwrap()
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
}

func TestDoGetComments_SingleListing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]listingResponse{{}})
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	result := s.doGetComments(context.Background(), srv.URL+"/test")
	if !result.IsOk() {
		_, err := result.Unwrap()
		t.Fatalf("expected ok, got: %v", err)
	}
	comments, _ := result.Unwrap()
	if len(comments) != 0 {
		t.Fatalf("expected 0 comments, got %d", len(comments))
	}
}

func TestDoGetComments_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("bad"))
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	result := s.doGetComments(context.Background(), srv.URL+"/test")
	if result.IsOk() {
		t.Fatal("expected error")
	}
}

func TestDoGetComments_HttpError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	result := s.doGetComments(context.Background(), srv.URL+"/test")
	if result.IsOk() {
		t.Fatal("expected error")
	}
}

func TestDoGet_HttpError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	result := s.doGet(context.Background(), srv.URL+"/test")
	if result.IsOk() {
		t.Fatal("expected error")
	}
}

func TestHttpGet_UserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	body, err := s.httpGet(context.Background(), srv.URL+"/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body.Close()
	if !strings.Contains(gotUA, "wessley-scraper") {
		t.Errorf("expected wessley-scraper user agent, got %s", gotUA)
	}
}

func TestHttpGet_BadURL(t *testing.T) {
	s := NewScraper(Config{RateLimit: time.Millisecond})
	_, err := s.httpGet(context.Background(), "http://[::1]:namedport")
	if err == nil {
		t.Fatal("expected error for bad URL")
	}
}

func TestHttpGet_CancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second)
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.httpGet(ctx, srv.URL+"/test")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestFetchSubreddit_Directly(t *testing.T) {
	listing := listingResponse{}
	listing.Data.Children = []listingChild{
		{Kind: "t3", Data: listingData{
			ID: "p1", Subreddit: "TestSub", Title: "Test Post",
			Author: "user", SelfText: "body", Score: 5,
			CreatedUTC: 1700000000, Permalink: "/r/TestSub/comments/p1/t/",
		}},
	}

	commentListing := listingResponse{}
	commentListing.Data.Children = []listingChild{
		{Kind: "t1", Data: listingData{ID: "c1", Author: "commenter", Body: "reply", Score: 3, CreatedUTC: 1700001000}},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/new.json") {
			json.NewEncoder(w).Encode(listing)
		} else {
			json.NewEncoder(w).Encode([]listingResponse{listing, commentListing})
		}
	}))
	defer srv.Close()

	s := NewScraper(Config{PostsPerSub: 10, CommentsPerPost: 10, RateLimit: 1 * time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	limiter := time.NewTicker(1 * time.Millisecond)
	defer limiter.Stop()

	posts, err := s.fetchSubreddit(context.Background(), "TestSub", limiter)
	if err != nil {
		t.Fatalf("fetchSubreddit: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].Title != "Test Post" {
		t.Errorf("unexpected title: %s", posts[0].Title)
	}
}

func TestFetchComments_Directly(t *testing.T) {
	commentListing := listingResponse{}
	commentListing.Data.Children = []listingChild{
		{Kind: "t1", Data: listingData{ID: "c1", Author: "user1", Body: "reply1", Score: 5, CreatedUTC: 1700000000, ParentID: "t3_p1", Depth: 0}},
		{Kind: "t1", Data: listingData{ID: "c2", Author: "user2", Body: "reply2", Score: 3, CreatedUTC: 1700001000, ParentID: "t1_c1", Depth: 1}},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]listingResponse{{}, commentListing})
	}))
	defer srv.Close()

	s := NewScraper(Config{CommentsPerPost: 10, RateLimit: 1 * time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	limiter := time.NewTicker(1 * time.Millisecond)
	defer limiter.Stop()

	comments, err := s.fetchComments(context.Background(), "/r/sub/comments/p1/t/", limiter)
	if err != nil {
		t.Fatalf("fetchComments: %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
	if comments[1].Depth != 1 {
		t.Errorf("expected depth 1, got %d", comments[1].Depth)
	}
	if comments[0].ParentID != "t3_p1" {
		t.Errorf("unexpected parent: %s", comments[0].ParentID)
	}
}

func TestNewScraper_Fields(t *testing.T) {
	cfg := Config{
		Subreddits:      []string{"a", "b"},
		PostsPerSub:     25,
		CommentsPerPost: 50,
		RateLimit:       2 * time.Second,
	}
	s := NewScraper(cfg)
	if s.cfg.PostsPerSub != 25 {
		t.Errorf("expected 25, got %d", s.cfg.PostsPerSub)
	}
	if s.client.Timeout != 30*time.Second {
		t.Errorf("unexpected timeout: %v", s.client.Timeout)
	}
}

func TestHttpGet_ReadBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	body, err := s.httpGet(context.Background(), srv.URL+"/test")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	defer body.Close()
	data, _ := io.ReadAll(body)
	if string(data) != "hello world" {
		t.Errorf("unexpected body: %s", data)
	}
}
