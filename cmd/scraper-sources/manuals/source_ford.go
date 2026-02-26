package manuals

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

// FordSource discovers owner manuals from Ford's website.
type FordSource struct {
	client *http.Client
}

func NewFordSource() *FordSource {
	return &FordSource{client: &http.Client{Timeout: 30 * time.Second}}
}

func (s *FordSource) Name() string { return "ford" }

func (s *FordSource) Discover(ctx context.Context, makes []string, years []int) ([]graph.ManualEntry, error) {
	if !containsIgnoreCase(makes, "Ford") {
		return nil, nil
	}

	models := []string{
		"f-150", "escape", "explorer", "mustang", "bronco",
		"ranger", "edge", "expedition", "maverick", "bronco-sport",
		"transit", "super-duty", "lightning",
	}

	var entries []graph.ManualEntry
	for _, year := range years {
		for _, model := range models {
			select {
			case <-ctx.Done():
				return entries, ctx.Err()
			default:
			}

			// Ford owner manual URL pattern
			url := fmt.Sprintf("https://www.ford.com/support/vehicle-information/owner-manuals/%d/%s/", year, model)
			entries = append(entries, graph.ManualEntry{
				ID:           graph.ManualEntryID(url),
				URL:          url,
				SourceSite:   "ford.com",
				Make:         "Ford",
				Model:        normModel(model),
				Year:         year,
				ManualType:   "owner",
				Language:     "en",
				Status:       "discovered",
				DiscoveredAt: time.Now(),
			})
		}

		// Crawl the main listing page for that year
		pageURL := fmt.Sprintf("https://www.ford.com/support/vehicle-information/owner-manuals/%d/", year)
		found, err := s.discoverFromPage(ctx, pageURL, year)
		if err != nil {
			log.Printf("ford: page crawl %d: %v", year, err)
		} else {
			entries = append(entries, found...)
		}

		time.Sleep(time.Second)
	}

	return entries, nil
}

func (s *FordSource) discoverFromPage(ctx context.Context, pageURL string, year int) ([]graph.ManualEntry, error) {
	result := fn.Retry(ctx, fn.RetryOpts{
		MaxAttempts: 2,
		InitialWait: time.Second,
		MaxWait:     5 * time.Second,
	}, func(ctx context.Context) fn.Result[[]byte] {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
		if err != nil {
			return fn.Err[[]byte](err)
		}
		req.Header.Set("User-Agent", "WessleyBot/1.0")
		resp, err := s.client.Do(req)
		if err != nil {
			return fn.Err[[]byte](err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fn.Errf[[]byte]("status %d", resp.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
		if err != nil {
			return fn.Err[[]byte](err)
		}
		return fn.Ok(body)
	})

	body, err := result.Unwrap()
	if err != nil {
		return nil, err
	}

	matches := toyotaPDFRegex.FindAllStringSubmatch(string(body), -1)
	var entries []graph.ManualEntry
	seen := make(map[string]bool)

	for _, m := range matches {
		url := m[1]
		if seen[url] {
			continue
		}
		seen[url] = true
		entries = append(entries, graph.ManualEntry{
			ID:           graph.ManualEntryID(url),
			URL:          url,
			SourceSite:   "ford.com",
			Make:         "Ford",
			Model:        inferModelFromURL(url, "Ford"),
			Year:         year,
			ManualType:   inferManualType(url),
			Language:     "en",
			Status:       "discovered",
			DiscoveredAt: time.Now(),
		})
	}
	return entries, nil
}
