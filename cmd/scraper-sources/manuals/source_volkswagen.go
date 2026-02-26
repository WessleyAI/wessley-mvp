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

// VolkswagenSource discovers owner manuals from Volkswagen's website.
type VolkswagenSource struct {
	client *http.Client
}

// NewVolkswagenSource creates a new VolkswagenSource.
func NewVolkswagenSource() *VolkswagenSource {
	return &VolkswagenSource{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *VolkswagenSource) Name() string { return "volkswagen" }

func (s *VolkswagenSource) Discover(ctx context.Context, makes []string, years []int) ([]graph.ManualEntry, error) {
	if !containsIgnoreCase(makes, "Volkswagen") {
		return nil, nil
	}

	var entries []graph.ManualEntry

	models := []string{
		"golf", "jetta", "tiguan", "atlas", "id-4", "taos",
	}

	for _, year := range years {
		for _, model := range models {
			select {
			case <-ctx.Done():
				return entries, ctx.Err()
			default:
			}

			url := fmt.Sprintf("https://www.vw.com/content/dam/vw/us/manuals/%d/%s-owners-manual.pdf", year, model)
			entry := graph.ManualEntry{
				ID:           graph.ManualEntryID(url),
				URL:          url,
				SourceSite:   "www.vw.com",
				Make:         "Volkswagen",
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
		pageURL := fmt.Sprintf("https://www.vw.com/en/owners/manuals/%d", year)
		found, err := s.discoverFromPage(ctx, pageURL, year)
		if err != nil {
			log.Printf("volkswagen: page crawl %d: %v", year, err)
		} else {
			entries = append(entries, found...)
		}

		time.Sleep(time.Second) // Rate limit between years
	}

	return entries, nil
}

var volkswagenPDFRegex = pdfLinkRegex

func (s *VolkswagenSource) discoverFromPage(ctx context.Context, pageURL string, year int) ([]graph.ManualEntry, error) {
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

	return extractPDFLinks(string(body), "www.vw.com", "Volkswagen", year), nil
}
