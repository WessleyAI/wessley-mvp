package nhtsa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/scraper"
	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

const (
	complaintsURL = "https://api.nhtsa.gov/complaints/complaintsByVehicle"
	modelsURL     = "https://api.nhtsa.gov/products/vehicle/models"
)

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

	years := s.cfg.Years()

	for _, year := range years {
		for _, make_ := range s.cfg.Makes {
			select {
			case <-ctx.Done():
				return allPosts, ctx.Err()
			default:
			}

			// First, fetch available models for this make+year
			models, err := s.fetchModels(ctx, make_, year, limiter)
			if err != nil {
				log.Printf("warning: failed to fetch NHTSA models for %s %d: %v", make_, year, err)
				continue
			}

			if len(models) == 0 {
				log.Printf("warning: no models found for %s year %d", make_, year)
				continue
			}

			// Limit to top 5 popular models to avoid too many requests
			if len(models) > 5 {
				models = models[:5]
			}

			for _, model := range models {
				posts, err := s.fetchMakeModel(ctx, make_, model, year, limiter)
				if err != nil {
					log.Printf("warning: failed to fetch NHTSA complaints for %s %s %d: %v", make_, model, year, err)
					continue
				}
				allPosts = append(allPosts, posts...)

				if s.cfg.MaxPerMake > 0 && len(allPosts) >= s.cfg.MaxPerMake {
					break
				}
			}
		}
	}
	return allPosts, nil
}

type modelsResponse struct {
	Count   int           `json:"count"`
	Results []modelEntry  `json:"results"`
}

type modelEntry struct {
	Model string `json:"model"`
}

func (s *Scraper) fetchModels(ctx context.Context, make_ string, year int, limiter *time.Ticker) ([]string, error) {
	url := fmt.Sprintf("%s?modelYear=%d&make=%s&issueType=c", modelsURL, year, make_)

	<-limiter.C
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "wessley-scraper/1.0 (automotive repair data collection)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var mr modelsResponse
	if err := json.Unmarshal(body, &mr); err != nil {
		return nil, err
	}

	var models []string
	for _, m := range mr.Results {
		models = append(models, m.Model)
	}
	return models, nil
}

func (s *Scraper) fetchMakeModel(ctx context.Context, make_, model string, year int, limiter *time.Ticker) ([]scraper.ScrapedPost, error) {
	url := fmt.Sprintf("%s?make=%s&model=%s&modelYear=%d", complaintsURL, neturl.QueryEscape(make_), neturl.QueryEscape(model), year)

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
		return nil, fmt.Errorf("nhtsa %s %s: %w", make_, model, err)
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

		// Extract vehicle info from products array
		vehicleMake := make_
		vehicleModel := model
		vehicleYear := year
		if vp := c.VehicleProduct(); vp != nil {
			vehicleMake = vp.ProductMake
			vehicleModel = vp.ProductModel
			if vp.ProductYear != "" && vp.ProductYear != "9999" {
				fmt.Sscanf(vp.ProductYear, "%d", &vehicleYear)
			}
		}
		vehicle := fmt.Sprintf("%d %s %s", vehicleYear, vehicleMake, vehicleModel)

		posts = append(posts, scraper.ScrapedPost{
			Source:      "nhtsa",
			SourceID:    fmt.Sprintf("nhtsa-%d", c.ODINumber),
			Title:       fmt.Sprintf("NHTSA Complaint: %s - %s", vehicle, c.Components),
			Content:     c.Summary,
			Author:      "nhtsa-consumer",
			URL:         fmt.Sprintf("https://www.nhtsa.gov/vehicle/%d/%s/%s/complaints", vehicleYear, vehicleMake, vehicleModel),
			PublishedAt: published,
			ScrapedAt:   now,
			Metadata: scraper.Metadata{
				Vehicle:    vehicle,
				VehicleInfo: &scraper.VehicleInfo{
					Make:  vehicleMake,
					Model: vehicleModel,
					Year:  vehicleYear,
				},
				Symptoms:   extractSymptoms(c.Summary),
				Keywords:   []string{strings.ToLower(c.Components), "nhtsa", "complaint"},
				Components: c.Components,
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
