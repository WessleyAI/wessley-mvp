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

type redirectTransport struct {
	server *httptest.Server
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.server.URL, "http://")
	return http.DefaultTransport.RoundTrip(req)
}

func TestFetchAll_Success(t *testing.T) {
	resp := apiResponse{
		Count: 2,
		Results: []Complaint{
			{
				ODINumber: 11111, MakeName: "TOYOTA", ModelName: "CAMRY", ModelYear: 2020,
				Component: "ENGINE", Summary: "Engine stalling with vibration and warning light",
				DateComplaintFiled: "01/15/2024",
			},
			{
				ODINumber: 11112, MakeName: "TOYOTA", ModelName: "RAV4", ModelYear: 2020,
				Component: "BRAKES", Summary: "Brake failure and noise when stopping",
				DateComplaintFiled: "2024-02-20",
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := NewScraper(Config{Makes: []string{"TOYOTA"}, ModelYear: 2020, MaxPerMake: 10, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}
	if posts[0].Source != "nhtsa" {
		t.Errorf("unexpected source: %s", posts[0].Source)
	}
	if !strings.Contains(posts[0].Title, "ENGINE") {
		t.Errorf("expected ENGINE in title: %s", posts[0].Title)
	}
	if len(posts[0].Metadata.Symptoms) == 0 {
		t.Error("expected symptoms extracted")
	}
}

func TestFetchAll_MaxPerMake(t *testing.T) {
	resp := apiResponse{
		Count: 3,
		Results: []Complaint{
			{ODINumber: 1, MakeName: "FORD", ModelName: "F150", ModelYear: 2021, Component: "ENGINE", Summary: "noise", DateComplaintFiled: "01/01/2024"},
			{ODINumber: 2, MakeName: "FORD", ModelName: "F150", ModelYear: 2021, Component: "BRAKES", Summary: "leak", DateComplaintFiled: "01/02/2024"},
			{ODINumber: 3, MakeName: "FORD", ModelName: "F150", ModelYear: 2021, Component: "STEERING", Summary: "vibration", DateComplaintFiled: "01/03/2024"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := NewScraper(Config{Makes: []string{"FORD"}, ModelYear: 2021, MaxPerMake: 2, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts (MaxPerMake=2), got %d", len(posts))
	}
}

func TestFetchAll_MaxPerMakeZero(t *testing.T) {
	resp := apiResponse{
		Count: 2,
		Results: []Complaint{
			{ODINumber: 1, MakeName: "FORD", ModelName: "F150", ModelYear: 2021, Component: "ENGINE", Summary: "noise", DateComplaintFiled: "01/01/2024"},
			{ODINumber: 2, MakeName: "FORD", ModelName: "F150", ModelYear: 2021, Component: "BRAKES", Summary: "leak", DateComplaintFiled: "01/02/2024"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := NewScraper(Config{Makes: []string{"FORD"}, ModelYear: 2021, MaxPerMake: 0, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	// MaxPerMake=0 means no limit
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}
}

func TestFetchAll_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	s := NewScraper(Config{Makes: []string{"A", "B"}, ModelYear: 2020, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	posts, _ := s.FetchAll(ctx)
	if len(posts) != 0 {
		t.Fatalf("expected 0 posts, got %d", len(posts))
	}
}

func TestFetchAll_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	s := NewScraper(Config{Makes: []string{"TOYOTA"}, ModelYear: 2020, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("expected 0 posts, got %d", len(posts))
	}
}

func TestFetchAll_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("bad json"))
	}))
	defer srv.Close()

	s := NewScraper(Config{Makes: []string{"TOYOTA"}, ModelYear: 2020, RateLimit: time.Millisecond})
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

	s := NewScraper(Config{Makes: []string{"TOYOTA"}, ModelYear: 2020, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("expected 0, got %d", len(posts))
	}
}

func TestFetchAll_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	s := NewScraper(Config{Makes: []string{"TOYOTA"}, ModelYear: 2020, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("expected 0, got %d", len(posts))
	}
}

func TestDoGet_Directly(t *testing.T) {
	resp := apiResponse{Count: 1, Results: []Complaint{{ODINumber: 1, Summary: "test"}}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("expected User-Agent header")
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := NewScraper(Config{RateLimit: time.Millisecond})
	s.client = srv.Client()

	result := s.doGet(context.Background(), srv.URL+"/test")
	if !result.IsOk() {
		_, err := result.Unwrap()
		t.Fatalf("expected ok: %v", err)
	}
	r, _ := result.Unwrap()
	if r.Count != 1 {
		t.Errorf("expected count 1, got %d", r.Count)
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

func TestParseNHTSADate_Formats(t *testing.T) {
	tests := []struct {
		input string
		zero  bool
	}{
		{"01/15/2024", false},
		{"2024-01-15", false},
		{"2024-01-15T00:00:00Z", false},
		{"invalid", true},
		{"", true},
	}
	for _, tt := range tests {
		d := parseNHTSADate(tt.input)
		if tt.zero && !d.IsZero() {
			t.Errorf("expected zero for %q, got %v", tt.input, d)
		}
		if !tt.zero && d.IsZero() {
			t.Errorf("expected non-zero for %q", tt.input)
		}
	}
}

func TestExtractSymptoms_Various(t *testing.T) {
	tests := []struct {
		input    string
		minCount int
	}{
		{"Engine stalling with vibration and noise", 3},
		{"brake failure and leak", 2},
		{"warning light on dashboard", 1},
		{"nothing relevant here", 0},
		{"overheating and fire risk with electrical issues", 3},
		{"transmission slipping and steering problems with airbag light and acceleration issues", 4},
	}
	for _, tt := range tests {
		symptoms := extractSymptoms(tt.input)
		if len(symptoms) < tt.minCount {
			t.Errorf("input %q: expected at least %d symptoms, got %v", tt.input, tt.minCount, symptoms)
		}
	}
}

func TestFetchAll_MultipleMakes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			Count: 1,
			Results: []Complaint{
				{ODINumber: 1, MakeName: "TEST", ModelName: "CAR", ModelYear: 2020, Component: "ENGINE", Summary: "noise", DateComplaintFiled: "01/01/2024"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := NewScraper(Config{Makes: []string{"TOYOTA", "FORD"}, ModelYear: 2020, MaxPerMake: 10, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, err := s.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts (one per make), got %d", len(posts))
	}
}

func TestFetchMake_PostMetadata(t *testing.T) {
	resp := apiResponse{
		Count: 1,
		Results: []Complaint{
			{
				ODINumber: 99999, MakeName: "HONDA", ModelName: "CIVIC", ModelYear: 2022,
				Component: "ELECTRICAL SYSTEM", Summary: "electrical issues and warning light",
				DateComplaintFiled: "03/15/2024",
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := NewScraper(Config{Makes: []string{"HONDA"}, ModelYear: 2022, MaxPerMake: 10, RateLimit: time.Millisecond})
	s.client = &http.Client{Transport: &redirectTransport{server: srv}, Timeout: 5 * time.Second}

	posts, _ := s.FetchAll(context.Background())
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	p := posts[0]
	if p.SourceID != "nhtsa-99999" {
		t.Errorf("unexpected source ID: %s", p.SourceID)
	}
	if p.Metadata.Vehicle != "2022 HONDA CIVIC" {
		t.Errorf("unexpected vehicle: %s", p.Metadata.Vehicle)
	}
	if !strings.Contains(p.URL, "HONDA") {
		t.Errorf("URL should contain HONDA: %s", p.URL)
	}
	if len(p.Metadata.Keywords) != 3 {
		t.Errorf("expected 3 keywords, got %v", p.Metadata.Keywords)
	}
}
