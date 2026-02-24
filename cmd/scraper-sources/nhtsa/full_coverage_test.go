package nhtsa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type rewriteTransport struct {
	server *httptest.Server
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.server.URL, "http://")
	return http.DefaultTransport.RoundTrip(req)
}

func TestDoGet_Success(t *testing.T) {
	resp := apiResponse{Count: 1, Results: []Complaint{{
		ODINumber: 1, Summary: "test", Component: "ENGINE",
		MakeName: "TOYOTA", ModelName: "CAMRY", ModelYear: 2020,
	}}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	result := s.doGet(context.Background(), srv.URL+"/test")
	if result.IsErr() {
		_, err := result.Unwrap()
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDoGet_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	result := s.doGet(context.Background(), srv.URL+"/test")
	if result.IsOk() {
		t.Fatal("expected error")
	}
}

func TestDoGet_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	result := s.doGet(context.Background(), srv.URL+"/test")
	if result.IsOk() {
		t.Fatal("expected error for 404")
	}
}

func TestDoGet_429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	result := s.doGet(context.Background(), srv.URL+"/test")
	if result.IsOk() {
		t.Fatal("expected error for 429")
	}
}

func TestDoGet_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	result := s.doGet(context.Background(), srv.URL+"/test")
	if result.IsOk() {
		t.Fatal("expected error for 500")
	}
}

func TestDoGet_CancelledContext2(t *testing.T) {
	s := NewScraper(Config{RateLimit: time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := s.doGet(ctx, "http://localhost:1/test")
	if result.IsOk() {
		t.Fatal("expected error")
	}
}

func TestFetchAll_ContextCancelled2(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewScraper(Config{Makes: []string{"TOYOTA"}, RateLimit: time.Millisecond})
	posts, _ := s.FetchAll(ctx)
	if len(posts) != 0 {
		t.Fatalf("expected 0, got %d", len(posts))
	}
}

func TestFetchMake_MaxPerMake(t *testing.T) {
	resp := apiResponse{Count: 3, Results: []Complaint{
		{ODINumber: 1, MakeName: "TOYOTA", ModelName: "CAMRY", ModelYear: 2020, Component: "ENGINE", Summary: "s"},
		{ODINumber: 2, MakeName: "TOYOTA", ModelName: "CAMRY", ModelYear: 2020, Component: "BRAKES", Summary: "s"},
		{ODINumber: 3, MakeName: "TOYOTA", ModelName: "CAMRY", ModelYear: 2020, Component: "LIGHTS", Summary: "s"},
	}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := NewScraper(Config{Makes: []string{"TOYOTA"}, ModelYear: 2020, MaxPerMake: 2, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &rewriteTransport{server: srv}, Timeout: 5 * time.Second}

	posts, _ := s.FetchAll(context.Background())
	if len(posts) > 2 {
		t.Fatalf("expected max 2, got %d", len(posts))
	}
}

func TestParseNHTSADate_Invalid(t *testing.T) {
	d := parseNHTSADate("not-a-date")
	if !d.IsZero() {
		t.Fatal("expected zero time")
	}
}

func TestParseNHTSADate_RFC3339(t *testing.T) {
	d := parseNHTSADate("2024-01-15T00:00:00Z")
	if d.IsZero() {
		t.Fatal("expected non-zero")
	}
}

func TestParseNHTSADate_ISO(t *testing.T) {
	d := parseNHTSADate("2024-01-15")
	if d.IsZero() {
		t.Fatal("expected non-zero")
	}
}

func TestExtractSymptoms_Empty(t *testing.T) {
	s := extractSymptoms("nothing here")
	if len(s) != 0 {
		t.Fatalf("expected 0, got %v", s)
	}
}

func TestExtractSymptoms_Multiple(t *testing.T) {
	s := extractSymptoms("The engine was stalling with vibration and noise")
	if len(s) < 3 {
		t.Fatalf("expected at least 3, got %v", s)
	}
}
