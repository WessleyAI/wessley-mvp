package forums

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"context"
)

func TestDoGet_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	s := NewScraper(Config{})
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

	s := NewScraper(Config{})
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

	s := NewScraper(Config{})
	s.client = srv.Client()
	result := s.doGet(context.Background(), srv.URL+"/test")
	if result.IsOk() {
		t.Fatal("expected error for 500")
	}
}

func TestDoGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html>hello</html>"))
	}))
	defer srv.Close()

	s := NewScraper(Config{})
	s.client = srv.Client()
	result := s.doGet(context.Background(), srv.URL+"/test")
	if result.IsErr() {
		_, err := result.Unwrap()
		t.Fatalf("unexpected error: %v", err)
	}
}
