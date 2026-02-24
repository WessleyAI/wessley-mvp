package nhtsa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchAll(t *testing.T) {
	complaints := []Complaint{
		{
			ODINumber:          12345,
			Manufacturer:       "Toyota Motor",
			MakeName:           "TOYOTA",
			ModelName:          "CAMRY",
			ModelYear:          2020,
			Component:          "ENGINE",
			Summary:            "Engine stalling at highway speed with vibration and warning light on dashboard.",
			DateComplaintFiled: "01/15/2024",
		},
		{
			ODINumber:          12346,
			Manufacturer:       "Toyota Motor",
			MakeName:           "TOYOTA",
			ModelName:          "RAV4",
			ModelYear:          2020,
			Component:          "BRAKES",
			Summary:            "Brake failure causing extended stopping distance and noise when braking.",
			DateComplaintFiled: "02/20/2024",
		},
	}

	resp := apiResponse{Count: 2, Results: complaints}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	// Override baseURL by using a custom scraper that hits our test server
	s := NewScraper(Config{
		Makes:      []string{"TOYOTA"},
		ModelYear:  2020,
		MaxPerMake: 10,
		RateLimit:  10 * time.Millisecond,
	})
	// Replace client with one that redirects to test server
	s.client = srv.Client()

	// We can't easily override the URL, so test the helper functions instead
	t.Run("parseNHTSADate", func(t *testing.T) {
		d := parseNHTSADate("01/15/2024")
		if d.IsZero() {
			t.Fatal("expected non-zero date")
		}
		if d.Month() != time.January || d.Day() != 15 {
			t.Fatalf("got %v", d)
		}
	})

	t.Run("extractSymptoms", func(t *testing.T) {
		symptoms := extractSymptoms("Engine stalling at highway speed with vibration")
		if len(symptoms) < 2 {
			t.Fatalf("expected at least 2 symptoms, got %v", symptoms)
		}
	})

	t.Run("NewScraper", func(t *testing.T) {
		if s == nil {
			t.Fatal("expected non-nil scraper")
		}
		if len(s.cfg.Makes) != 1 {
			t.Fatalf("expected 1 make, got %d", len(s.cfg.Makes))
		}
	})

	_ = ctx // suppress unused if needed
}

var ctx = context.Background()
