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

// AudiSource discovers owner manuals from Audi's website.
type AudiSource struct {
	client *http.Client
}

// NewAudiSource creates a new AudiSource.
func NewAudiSource() *AudiSource {
	return &AudiSource{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *AudiSource) Name() string { return "audi" }

func (s *AudiSource) Discover(ctx context.Context, makes []string, years []int) ([]graph.ManualEntry, error) {
	if !containsIgnoreCase(makes, "Audi") {
		return nil, nil
	}

	var entries []graph.ManualEntry

	models := []string{
		"a3", "a4", "a6", "a8", "q3", "q5", "q7", "q8", "e-tron",
	}

	for _, year := range years {
		for _, model := range models {
			select {
			case <-ctx.Done():
				return entries, ctx.Err()
			default:
			}

			url := fmt.Sprintf("https://www.audiusa.com/content/dam/nemo/us/manuals/%d/audi-%s-owners-manual.pdf", year, model)
			entry := graph.ManualEntry{
				ID:           graph.ManualEntryID(url),
				URL:          url,
				SourceSite:   "www.audiusa.com",
				Make:         "Audi",
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
		pageURL := fmt.Sprintf("https://www.audiusa.com/us/web/en/owners/manuals/%d", year)
		found, err := s.discoverFromPage(ctx, pageURL, year)
		if err != nil {
			log.Printf("audi: page crawl %d: %v", year, err)
		} else {
			entries = append(entries, found...)
		}

		time.Sleep(time.Second) // Rate limit between years
	}

	return entries, nil
}

var audiPDFRegex = pdfLinkRegex

func (s *AudiSource) discoverFromPage(ctx context.Context, pageURL string, year int) ([]graph.ManualEntry, error) {
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

	return extractPDFLinks(string(body), "www.audiusa.com", "Audi", year), nil
}
