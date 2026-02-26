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

// NissanSource discovers owner manuals from Nissan's website.
type NissanSource struct {
	client *http.Client
}

// NewNissanSource creates a new NissanSource.
func NewNissanSource() *NissanSource {
	return &NissanSource{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *NissanSource) Name() string { return "nissan" }

func (s *NissanSource) Discover(ctx context.Context, makes []string, years []int) ([]graph.ManualEntry, error) {
	if !containsIgnoreCase(makes, "Nissan") {
		return nil, nil
	}

	var entries []graph.ManualEntry

	models := []string{
		"altima", "sentra", "rogue", "pathfinder", "frontier", "titan", "maxima", "kicks", "ariya", "leaf", "z", "versa", "murano",
	}

	for _, year := range years {
		for _, model := range models {
			select {
			case <-ctx.Done():
				return entries, ctx.Err()
			default:
			}

			url := fmt.Sprintf("https://www.nissanusa.com/content/dam/nissan/us/manuals/%d/%s-owner-manual.pdf", year, model)
			entry := graph.ManualEntry{
				ID:           graph.ManualEntryID(url),
				URL:          url,
				SourceSite:   "www.nissanusa.com",
				Make:         "Nissan",
				Model:        normModel(model),
				Year:         year,
				ManualType:   "owner",
				Language:     "en",
				Status:       "discovered",
				DiscoveredAt: time.Now(),
			}
			entries = append(entries, entry)
		}

		// Crawl the owners manual page for additional PDF links
		pageURL := fmt.Sprintf("https://www.nissanusa.com/owners/manuals/%d", year)
		found, err := s.discoverFromPage(ctx, pageURL, year)
		if err != nil {
			log.Printf("nissan: page crawl %d: %v", year, err)
		} else {
			entries = append(entries, found...)
		}

		time.Sleep(time.Second) // Rate limit between years
	}

	return entries, nil
}

var nissanPDFRegex = pdfLinkRegex

func (s *NissanSource) discoverFromPage(ctx context.Context, pageURL string, year int) ([]graph.ManualEntry, error) {
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

	return extractPDFLinks(string(body), "www.nissanusa.com", "Nissan", year), nil
}
