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

// AcuraSource discovers owner manuals from Acura's website.
type AcuraSource struct {
	client *http.Client
}

// NewAcuraSource creates a new AcuraSource.
func NewAcuraSource() *AcuraSource {
	return &AcuraSource{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *AcuraSource) Name() string { return "acura" }

func (s *AcuraSource) Discover(ctx context.Context, makes []string, years []int) ([]graph.ManualEntry, error) {
	if !containsIgnoreCase(makes, "Acura") {
		return nil, nil
	}

	var entries []graph.ManualEntry

	models := []string{
		"integra", "tlx", "mdx", "rdx", "zdx",
	}

	for _, year := range years {
		for _, model := range models {
			select {
			case <-ctx.Done():
				return entries, ctx.Err()
			default:
			}

			url := fmt.Sprintf("https://owners.acura.com/content/dam/honda/owners/manuals/%d/%s-owners-manual.pdf", year, model)
			entry := graph.ManualEntry{
				ID:           graph.ManualEntryID(url),
				URL:          url,
				SourceSite:   "owners.acura.com",
				Make:         "Acura",
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
		pageURL := fmt.Sprintf("https://owners.acura.com/vehicles/manuals/%d", year)
		found, err := s.discoverFromPage(ctx, pageURL, year)
		if err != nil {
			log.Printf("acura: page crawl %d: %v", year, err)
		} else {
			entries = append(entries, found...)
		}

		time.Sleep(time.Second) // Rate limit between years
	}

	return entries, nil
}

var acuraPDFRegex = pdfLinkRegex

func (s *AcuraSource) discoverFromPage(ctx context.Context, pageURL string, year int) ([]graph.ManualEntry, error) {
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

	return extractPDFLinks(string(body), "owners.acura.com", "Acura", year), nil
}
