package nhtsa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/scraper"
	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

const baseURL = "https://api.nhtsa.gov/complaints/complaintsByVehicle"

// Scraper fetches complaints from the NHTSA complaints API.
type Scraper struct {
	cfg    Config
	client *http.Client
}

// NewScraper creates a Scraper with the given config.
func NewScraper(cfg Config) *Scraper {
	return &Scraper{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchAll scrapes NHTSA complaints for all configured makes and returns ScrapedPosts.
func (s *Scraper) FetchAll(ctx context.Context) ([]scraper.ScrapedPost, error) {
	var allPosts []scraper.ScrapedPost
	limiter := time.NewTicker(s.cfg.RateLimit)
	defer limiter.Stop()

	for _, make_ := range s.cfg.Makes {
		select {
		case <-ctx.Done():
			return allPosts, ctx.Err()
		default:
		}

		posts, err := s.fetchMake(ctx, make_, limiter)
		if err != nil {
			log.Printf("warning: failed to fetch NHTSA complaints for %s: %v", make_, err)
			continue
		}
		allPosts = append(allPosts, posts...)
	}
	return allPosts, nil
}

func (s *Scraper) fetchMake(ctx context.Context, make_ string, limiter *time.Ticker) ([]scraper.ScrapedPost, error) {
	url := fmt.Sprintf("%s?make=%s&modelYear=%d", baseURL, make_, s.cfg.ModelYear)

	result := fn.Retry(ctx, fn.RetryOpts{
		MaxAttempts: 3,
		InitialWait: 5 * time.Second,
		MaxWait:     30 * time.Second,
		Jitter:      true,
	}, func(ctx context.Context) fn.Result[*apiResponse] {
		<-limiter.C
		return s.doGet(ctx, url)
	})

	resp, err := result.Unwrap()
	if err != nil {
		return nil, fmt.Errorf("nhtsa %s: %w", make_, err)
	}

	now := time.Now().UTC()
	count := len(resp.Results)
	if s.cfg.MaxPerMake > 0 && count > s.cfg.MaxPerMake {
		count = s.cfg.MaxPerMake
	}

	posts := make([]scraper.ScrapedPost, 0, count)
	for i, c := range resp.Results {
		if i >= count {
			break
		}
		published := parseNHTSADate(c.DateComplaintFiled)
		vehicle := fmt.Sprintf("%d %s %s", c.ModelYear, c.MakeName, c.ModelName)

		posts = append(posts, scraper.ScrapedPost{
			Source:      "nhtsa",
			SourceID:    fmt.Sprintf("nhtsa-%d", c.ODINumber),
			Title:       fmt.Sprintf("NHTSA Complaint: %s - %s", vehicle, c.Component),
			Content:     c.Summary,
			Author:      "nhtsa-consumer",
			URL:         fmt.Sprintf("https://www.nhtsa.gov/vehicle/%d/%s/%s/complaints", c.ModelYear, c.MakeName, c.ModelName),
			PublishedAt: published,
			ScrapedAt:   now,
			Metadata: scraper.Metadata{
				Vehicle:  vehicle,
				Symptoms: extractSymptoms(c.Summary),
				Keywords: []string{strings.ToLower(c.Component), "nhtsa", "complaint"},
			},
		})
	}
	return posts, nil
}

type apiResponse struct {
	Count   int         `json:"count"`
	Results []Complaint `json:"results"`
}

func (s *Scraper) doGet(ctx context.Context, url string) fn.Result[*apiResponse] {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fn.Err[*apiResponse](err)
	}
	req.Header.Set("User-Agent", "wessley-scraper/1.0 (automotive repair data collection)")

	resp, err := s.client.Do(req)
	if err != nil {
		return fn.Err[*apiResponse](err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return fn.Err[*apiResponse](fmt.Errorf("http %d from %s", resp.StatusCode, url))
	}
	if resp.StatusCode != http.StatusOK {
		return fn.Err[*apiResponse](fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fn.Err[*apiResponse](fmt.Errorf("read body: %w", err))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fn.Err[*apiResponse](fmt.Errorf("decode: %w", err))
	}
	return fn.Ok(&apiResp)
}

func parseNHTSADate(s string) time.Time {
	for _, layout := range []string{"01/02/2006", "2006-01-02", time.RFC3339} {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func extractSymptoms(summary string) []string {
	lower := strings.ToLower(summary)
	knownSymptoms := []string{
		"stalling", "vibration", "noise", "leak", "overheating",
		"warning light", "brake failure", "acceleration", "steering",
		"transmission", "engine", "airbag", "fire", "electrical",
	}
	var found []string
	for _, s := range knownSymptoms {
		if strings.Contains(lower, s) {
			found = append(found, s)
		}
	}
	return found
}
