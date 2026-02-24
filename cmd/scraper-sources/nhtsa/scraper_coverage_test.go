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
	modelsResp := modelsResponse{
		Count:   1,
		Results: []modelEntry{{Model: "CAMRY"}},
	}
	complaintsResp := apiResponse{
		Count: 2,
		Results: []Complaint{
			{
				ODINumber: 11111, Components: "ENGINE",
				Summary: "Engine stalling with vibration and warning light",
				DateComplaintFiled: "01/15/2024",
				Products: []Product{{Type: "Vehicle", ProductYear: "2020", ProductMake: "TOYOTA", ProductModel: "CAMRY"}},
			},
			{
				ODINumber: 11112, Components: "BRAKES",
				Summary: "Brake failure and noise when stopping",
				DateComplaintFiled: "2024-02-20",
				Products: []Product{{Type: "Vehicle", ProductYear: "2020", ProductMake: "TOYOTA", ProductModel: "CAMRY"}},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "models") {
			json.NewEncoder(w).Encode(modelsResp)
		} else {
			json.NewEncoder(w).Encode(complaintsResp)
		}
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
	modelsResp := modelsResponse{Count: 1, Results: []modelEntry{{Model: "F150"}}}
	complaintsResp := apiResponse{
		Count: 3,
		Results: []Complaint{
			{ODINumber: 1, Components: "ENGINE", Summary: "noise", DateComplaintFiled: "01/01/2024",
				Products: []Product{{Type: "Vehicle", ProductYear: "2021", ProductMake: "FORD", ProductModel: "F150"}}},
			{ODINumber: 2, Components: "BRAKES", Summary: "leak", DateComplaintFiled: "01/02/2024",
				Products: []Product{{Type: "Vehicle", ProductYear: "2021", ProductMake: "FORD", ProductModel: "F150"}}},
			{ODINumber: 3, Components: "STEERING", Summary: "vibration", DateComplaintFiled: "01/03/2024",
				Products: []Product{{Type: "Vehicle", ProductYear: "2021", ProductMake: "FORD", ProductModel: "F150"}}},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "models") {
			json.NewEncoder(w).Encode(modelsResp)
		} else {
			json.NewEncoder(w).Encode(complaintsResp)
		}
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

func TestVehicleProduct(t *testing.T) {
	c := Complaint{
		Products: []Product{
			{Type: "Tire", ProductMake: "BRIDGESTONE"},
			{Type: "Vehicle", ProductYear: "2024", ProductMake: "TOYOTA", ProductModel: "CAMRY"},
		},
	}
	vp := c.VehicleProduct()
	if vp == nil {
		t.Fatal("expected vehicle product")
	}
	if vp.ProductMake != "TOYOTA" {
		t.Errorf("expected TOYOTA, got %s", vp.ProductMake)
	}

	c2 := Complaint{Products: []Product{{Type: "Tire"}}}
	if c2.VehicleProduct() != nil {
		t.Error("expected nil for no vehicle product")
	}
}

func TestFetchAll_PostMetadata(t *testing.T) {
	modelsResp := modelsResponse{Count: 1, Results: []modelEntry{{Model: "CIVIC"}}}
	complaintsResp := apiResponse{
		Count: 1,
		Results: []Complaint{
			{
				ODINumber: 99999, Components: "ELECTRICAL SYSTEM",
				Summary: "electrical issues and warning light",
				DateComplaintFiled: "03/15/2024",
				Products: []Product{{Type: "Vehicle", ProductYear: "2022", ProductMake: "HONDA", ProductModel: "CIVIC"}},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "models") {
			json.NewEncoder(w).Encode(modelsResp)
		} else {
			json.NewEncoder(w).Encode(complaintsResp)
		}
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
}
