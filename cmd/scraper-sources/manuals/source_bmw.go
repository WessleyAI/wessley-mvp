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

// BMWSource discovers owner manuals from BMW's website.
type BMWSource struct {
	client *http.Client
}

// NewBMWSource creates a new BMWSource.
func NewBMWSource() *BMWSource {
	return &BMWSource{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *BMWSource) Name() string { return "bmw" }

func (s *BMWSource) Discover(ctx context.Context, makes []string, years []int) ([]graph.ManualEntry, error) {
	if !containsIgnoreCase(makes, "BMW") {
		return nil, nil
	}

	var entries []graph.ManualEntry

	models := []string{
		"3-series", "5-series", "7-series", "x1", "x3", "x5", "x7", "m3", "m5", "ix", "i4",
	}

	for _, year := range years {
		for _, model := range models {
			select {
			case <-ctx.Done():
				return entries, ctx.Err()
			default:
			}

			url := fmt.Sprintf("https://www.bmwusa.com/content/dam/bmw/us/manuals/%d/%s-owners-manual.pdf", year, model)
			entry := graph.ManualEntry{
				ID:           graph.ManualEntryID(url),
				URL:          url,
				SourceSite:   "www.bmwusa.com",
				Make:         "BMW",
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
		pageURL := fmt.Sprintf("https://www.bmwusa.com/owners-manual/%d", year)
		found, err := s.discoverFromPage(ctx, pageURL, year)
		if err != nil {
			log.Printf("bmw: page crawl %d: %v", year, err)
		} else {
			entries = append(entries, found...)
		}

		time.Sleep(time.Second) // Rate limit between years
	}

	return entries, nil
}

var bmwPDFRegex = pdfLinkRegex

func (s *BMWSource) discoverFromPage(ctx context.Context, pageURL string, year int) ([]graph.ManualEntry, error) {
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

	return extractPDFLinks(string(body), "www.bmwusa.com", "BMW", year), nil
}
