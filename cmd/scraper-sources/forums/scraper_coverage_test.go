package forums

import (
	"context"
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
	html := `<div>
		<a href="/threads/brake-issue.123/">Brake Issue Discussion</a>
		<a href="/threads/oil-change.456/">Oil Change Tips</a>
	</div>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(html))
	}))
	defer srv.Close()

	s := NewScraper(Config{
		Forums:      []ForumConfig{{Name: "TestForum", BaseURL: "https://example.com", SearchPath: "/search/?q=%s"}},
		Queries:     []string{"brakes"},
		MaxPerForum: 10,
		RateLimit:   time.Millisecond,
	})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}
	if posts[0].Source != "forum:TestForum" {
		t.Errorf("unexpected source: %s", posts[0].Source)
	}
}

func TestFetchAll_MultipleForumsAndQueries(t *testing.T) {
	html := `<a href="/threads/test.1/">Test Thread</a>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(html))
	}))
	defer srv.Close()

	s := NewScraper(Config{
		Forums: []ForumConfig{
			{Name: "Forum1", BaseURL: "https://f1.com", SearchPath: "/search/?q=%s"},
			{Name: "Forum2", BaseURL: "https://f2.com", SearchPath: "/search/?q=%s"},
		},
		Queries:     []string{"brakes", "engine"},
		MaxPerForum: 10,
		RateLimit:   time.Millisecond,
	})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	// 2 forums x 2 queries = 4 fetches, 1 result each
	if len(posts) != 4 {
		t.Fatalf("expected 4 posts, got %d", len(posts))
	}
}

func TestFetchAll_MaxPerForum(t *testing.T) {
	html := `<div>
		<a href="/threads/a.1/">Thread A</a>
		<a href="/threads/b.2/">Thread B</a>
		<a href="/threads/c.3/">Thread C</a>
	</div>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(html))
	}))
	defer srv.Close()

	s := NewScraper(Config{
		Forums:      []ForumConfig{{Name: "F", BaseURL: "https://f.com", SearchPath: "/search/?q=%s"}},
		Queries:     []string{"test"},
		MaxPerForum: 2,
		RateLimit:   time.Millisecond,
	})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, _ := s.FetchAll(context.Background())
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts (MaxPerForum=2), got %d", len(posts))
	}
}

func TestFetchAll_MaxPerForumZero(t *testing.T) {
	html := `<a href="/threads/a.1/">Thread A</a><a href="/threads/b.2/">Thread B</a>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(html))
	}))
	defer srv.Close()

	s := NewScraper(Config{
		Forums:      []ForumConfig{{Name: "F", BaseURL: "https://f.com", SearchPath: "/search/?q=%s"}},
		Queries:     []string{"test"},
		MaxPerForum: 0,
		RateLimit:   time.Millisecond,
	})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, _ := s.FetchAll(context.Background())
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}
}

func TestFetchAll_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	s := NewScraper(Config{
		Forums:  []ForumConfig{{Name: "F", BaseURL: "https://f.com", SearchPath: "/search/?q=%s"}},
		Queries: []string{"test"},
		RateLimit: time.Millisecond,
	})
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

	s := NewScraper(Config{
		Forums:  []ForumConfig{{Name: "F", BaseURL: "https://f.com", SearchPath: "/search/?q=%s"}},
		Queries: []string{"test"},
		RateLimit: time.Millisecond,
	})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("expected 0, got %d", len(posts))
	}
}

func TestFetchAll_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
	}))
	defer srv.Close()

	s := NewScraper(Config{
		Forums:  []ForumConfig{{Name: "F", BaseURL: "https://f.com", SearchPath: "/search/?q=%s"}},
		Queries: []string{"test"},
		RateLimit: time.Millisecond,
	})
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

	s := NewScraper(Config{
		Forums:  []ForumConfig{{Name: "F", BaseURL: "https://f.com", SearchPath: "/search/?q=%s"}},
		Queries: []string{"test"},
		RateLimit: time.Millisecond,
	})
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
		w.Write([]byte("ok"))
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

func TestParseSearchResults_AbsoluteURLs(t *testing.T) {
	html := `<a href="https://other.com/threads/test.1/">Absolute Thread</a>`
	forum := ForumConfig{Name: "F", BaseURL: "https://example.com"}
	posts := parseSearchResults(html, forum, "q")
	if len(posts) != 1 {
		t.Fatalf("expected 1, got %d", len(posts))
	}
	if posts[0].URL != "https://other.com/threads/test.1/" {
		t.Errorf("expected absolute URL preserved, got %s", posts[0].URL)
	}
}

func TestParseSearchResults_EmptyTitle(t *testing.T) {
	html := `<a href="/threads/test.1/">   </a>`
	forum := ForumConfig{Name: "F", BaseURL: "https://example.com"}
	posts := parseSearchResults(html, forum, "q")
	if len(posts) != 0 {
		t.Fatalf("expected 0 posts for empty title, got %d", len(posts))
	}
}

func TestParseSearchResults_NoMatches(t *testing.T) {
	html := `<div>No links here</div>`
	forum := ForumConfig{Name: "F", BaseURL: "https://example.com"}
	posts := parseSearchResults(html, forum, "q")
	if len(posts) != 0 {
		t.Fatalf("expected 0, got %d", len(posts))
	}
}

func TestParseSearchResults_TopicLinks(t *testing.T) {
	html := `<a href="/topic/brake-pads-42/">Brake Pads Guide</a>`
	forum := ForumConfig{Name: "F", BaseURL: "https://example.com"}
	posts := parseSearchResults(html, forum, "brakes")
	if len(posts) != 1 {
		t.Fatalf("expected 1, got %d", len(posts))
	}
}

func TestParseSearchResults_ShowthreadLinks(t *testing.T) {
	html := `<a href="/showthread.php?t=12345">Oil Change Help</a>`
	forum := ForumConfig{Name: "F", BaseURL: "https://example.com"}
	posts := parseSearchResults(html, forum, "oil")
	if len(posts) != 1 {
		t.Fatalf("expected 1, got %d", len(posts))
	}
	if !strings.HasPrefix(posts[0].URL, "https://example.com") {
		t.Errorf("expected prefixed URL, got %s", posts[0].URL)
	}
}

func TestParseSearchResults_Metadata(t *testing.T) {
	html := `<a href="/threads/t.1/">Thread Title</a>`
	forum := ForumConfig{Name: "MyForum", BaseURL: "https://example.com"}
	posts := parseSearchResults(html, forum, "brakes")
	if len(posts) != 1 {
		t.Fatalf("expected 1, got %d", len(posts))
	}
	kw := posts[0].Metadata.Keywords
	if len(kw) != 3 {
		t.Errorf("expected 3 keywords, got %v", kw)
	}
	if posts[0].SourceID == "" {
		t.Error("expected non-empty SourceID")
	}
}
