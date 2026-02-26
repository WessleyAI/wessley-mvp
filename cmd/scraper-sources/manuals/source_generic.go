package manuals

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

// GenericSearchSource discovers manuals via web search for PDF links.
type GenericSearchSource struct {
	client *http.Client
}

func NewGenericSearchSource() *GenericSearchSource {
	return &GenericSearchSource{client: &http.Client{Timeout: 30 * time.Second}}
}

func (s *GenericSearchSource) Name() string { return "search" }

func (s *GenericSearchSource) Discover(ctx context.Context, makes []string, years []int) ([]graph.ManualEntry, error) {
	var entries []graph.ManualEntry

	// Build search queries for make/year combinations
	for _, make_ := range makes {
		for _, year := range years {
			select {
			case <-ctx.Done():
				return entries, ctx.Err()
			default:
			}

			query := fmt.Sprintf(`"%d %s owner manual" filetype:pdf`, year, make_)
			found, err := s.searchForPDFs(ctx, query, make_, year)
			if err != nil {
				continue
			}
			entries = append(entries, found...)
			time.Sleep(3 * time.Second) // Aggressive rate limiting for search
		}
	}

	return dedup(entries), nil
}

var pdfURLRegex = regexp.MustCompile(`(https?://[^\s"'<>]+\.pdf)`)

func (s *GenericSearchSource) searchForPDFs(ctx context.Context, query, make_ string, year int) ([]graph.ManualEntry, error) {
	// Use a simple approach: fetch search results page and extract PDF links
	// In production, this would use a proper search API
	searchURL := "https://html.duckduckgo.com/html/?" + url.Values{
		"q": {query},
	}.Encode()

	result := fn.Retry(ctx, fn.RetryOpts{
		MaxAttempts: 2,
		InitialWait: 2 * time.Second,
		MaxWait:     10 * time.Second,
	}, func(ctx context.Context) fn.Result[[]byte] {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
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

	matches := pdfURLRegex.FindAllString(string(body), 20)

	var entries []graph.ManualEntry
	seen := make(map[string]bool)

	for _, pdfURL := range matches {
		if seen[pdfURL] {
			continue
		}
		seen[pdfURL] = true

		entries = append(entries, graph.ManualEntry{
			ID:           graph.ManualEntryID(pdfURL),
			URL:          pdfURL,
			SourceSite:   extractDomain(pdfURL),
			Make:         make_,
			Model:        inferModelFromURL(pdfURL, make_),
			Year:         year,
			ManualType:   inferManualType(pdfURL),
			Language:     "en",
			Status:       "discovered",
			DiscoveredAt: time.Now(),
		})
	}

	return entries, nil
}

func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Host
}
