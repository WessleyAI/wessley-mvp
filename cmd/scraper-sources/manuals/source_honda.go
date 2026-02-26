package manuals

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

// HondaSource discovers owner manuals from Honda's website.
type HondaSource struct {
	client *http.Client
}

func NewHondaSource() *HondaSource {
	return &HondaSource{client: &http.Client{Timeout: 30 * time.Second}}
}

func (s *HondaSource) Name() string { return "honda" }

func (s *HondaSource) Discover(ctx context.Context, makes []string, years []int) ([]graph.ManualEntry, error) {
	if !containsIgnoreCase(makes, "Honda") {
		return nil, nil
	}

	models := []string{
		"civic", "accord", "cr-v", "pilot", "odyssey",
		"hr-v", "ridgeline", "passport", "insight", "fit",
		"prologue",
	}

	var entries []graph.ManualEntry
	for _, year := range years {
		for _, model := range models {
			select {
			case <-ctx.Done():
				return entries, ctx.Err()
			default:
			}

			// Honda's owner manual URL pattern
			url := fmt.Sprintf("https://owners.honda.com/Documentconnection/VisualPDF?body=%s&year=%d", model, year)
			entries = append(entries, graph.ManualEntry{
				ID:           graph.ManualEntryID(url),
				URL:          url,
				SourceSite:   "owners.honda.com",
				Make:         "Honda",
				Model:        normModel(model),
				Year:         year,
				ManualType:   "owner",
				Language:     "en",
				Status:       "discovered",
				DiscoveredAt: time.Now(),
			})
		}

		// Also crawl the manuals listing page
		pageURL := fmt.Sprintf("https://owners.honda.com/vehicles/information/manuals?year=%d", year)
		found, err := s.discoverFromPage(ctx, pageURL, year)
		if err != nil {
			log.Printf("honda: page crawl %d: %v", year, err)
		} else {
			entries = append(entries, found...)
		}

		time.Sleep(time.Second)
	}

	return entries, nil
}

func (s *HondaSource) discoverFromPage(ctx context.Context, pageURL string, year int) ([]graph.ManualEntry, error) {
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

	return extractHondaPDFLinks(string(body), year), nil
}

func extractHondaPDFLinks(html string, year int) []graph.ManualEntry {
	matches := toyotaPDFRegex.FindAllStringSubmatch(html, -1) // reuse PDF regex
	var entries []graph.ManualEntry
	seen := make(map[string]bool)

	for _, m := range matches {
		url := m[1]
		if seen[url] || !strings.Contains(strings.ToLower(url), "honda") {
			continue
		}
		seen[url] = true

		entries = append(entries, graph.ManualEntry{
			ID:           graph.ManualEntryID(url),
			URL:          url,
			SourceSite:   "owners.honda.com",
			Make:         "Honda",
			Model:        inferModelFromURL(url, "Honda"),
			Year:         year,
			ManualType:   inferManualType(url),
			Language:     "en",
			Status:       "discovered",
			DiscoveredAt: time.Now(),
		})
	}
	return entries
}
