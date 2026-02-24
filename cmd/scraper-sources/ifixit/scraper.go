package ifixit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/scraper"
	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

const baseURL = "https://www.ifixit.com/api/2.0"

// Scraper fetches automotive repair guides from iFixit's API.
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

// FetchAll scrapes iFixit guides using search queries and returns ScrapedPosts.
func (s *Scraper) FetchAll(ctx context.Context) ([]scraper.ScrapedPost, error) {
	var allPosts []scraper.ScrapedPost
	limiter := time.NewTicker(s.cfg.RateLimit)
	defer limiter.Stop()

	// Use search queries based on categories to find guides
	queries := []string{
		"car engine repair",
		"car brake repair",
		"car transmission",
		"truck repair",
		"car battery replacement",
		"car alternator",
		"car starter motor",
	}

	seen := make(map[string]bool)

	for _, query := range queries {
		select {
		case <-ctx.Done():
			return allPosts, ctx.Err()
		default:
		}

		posts, err := s.searchGuides(ctx, query, limiter)
		if err != nil {
			log.Printf("warning: failed to search iFixit for %q: %v", query, err)
			continue
		}

		for _, p := range posts {
			if !seen[p.SourceID] {
				seen[p.SourceID] = true
				allPosts = append(allPosts, p)
			}
		}

		if s.cfg.MaxGuides > 0 && len(allPosts) >= s.cfg.MaxGuides {
			allPosts = allPosts[:s.cfg.MaxGuides]
			break
		}
	}
	return allPosts, nil
}

type searchResponse struct {
	TotalResults int            `json:"totalResults"`
	Results      []searchResult `json:"results"`
}

type searchResult struct {
	DataType     string `json:"dataType"`
	GuideID      int    `json:"guideid"`
	Title        string `json:"title"`
	DisplayTitle string `json:"display_title"`
	URL          string `json:"url"`
	Summary      string `json:"summary"`
	Text         string `json:"text"`
	Namespace    string `json:"namespace"`
	ModifiedDate int64  `json:"modified_date"`
}

func (s *Scraper) searchGuides(ctx context.Context, query string, limiter *time.Ticker) ([]scraper.ScrapedPost, error) {
	searchURL := fmt.Sprintf("%s/search/%s?limit=%d", baseURL, url.PathEscape(query), s.cfg.MaxGuides)

	result := fn.Retry(ctx, fn.RetryOpts{
		MaxAttempts: 3,
		InitialWait: 5 * time.Second,
		MaxWait:     30 * time.Second,
		Jitter:      true,
	}, func(ctx context.Context) fn.Result[*searchResponse] {
		<-limiter.C
		return s.doSearch(ctx, searchURL)
	})

	resp, err := result.Unwrap()
	if err != nil {
		return nil, fmt.Errorf("ifixit search %q: %w", query, err)
	}

	now := time.Now().UTC()
	var posts []scraper.ScrapedPost

	for _, r := range resp.Results {
		// Only include guides (skip wiki pages, questions, etc.)
		if r.DataType != "guide" {
			continue
		}

		title := r.Title
		if r.DisplayTitle != "" {
			title = r.DisplayTitle
		}

		content := r.Summary
		if content == "" {
			content = r.Text
		}

		published := time.Unix(r.ModifiedDate, 0).UTC()

		posts = append(posts, scraper.ScrapedPost{
			Source:      "ifixit",
			SourceID:    fmt.Sprintf("ifixit-%d", r.GuideID),
			Title:       title,
			Content:     content,
			Author:      "",
			URL:         r.URL,
			PublishedAt: published,
			ScrapedAt:   now,
			Metadata: scraper.Metadata{
				Fixes:    extractFixes(content),
				Keywords: []string{"ifixit", "repair-guide", query},
			},
		})
	}
	return posts, nil
}

func (s *Scraper) doSearch(ctx context.Context, searchURL string) fn.Result[*searchResponse] {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return fn.Err[*searchResponse](err)
	}
	req.Header.Set("User-Agent", "wessley-scraper/1.0 (automotive repair data collection)")

	resp, err := s.client.Do(req)
	if err != nil {
		return fn.Err[*searchResponse](err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return fn.Err[*searchResponse](fmt.Errorf("http %d from %s", resp.StatusCode, searchURL))
	}
	if resp.StatusCode != http.StatusOK {
		return fn.Err[*searchResponse](fmt.Errorf("unexpected status %d from %s", resp.StatusCode, searchURL))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fn.Err[*searchResponse](fmt.Errorf("read body: %w", err))
	}

	var sr searchResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return fn.Err[*searchResponse](fmt.Errorf("decode: %w", err))
	}
	return fn.Ok(&sr)
}

// buildGuideContent builds a text representation of a Guide for ingestion.
func buildGuideContent(g Guide) string {
	var sb strings.Builder
	if g.Summary != "" {
		sb.WriteString(g.Summary)
		sb.WriteString("\n\n")
	}
	for _, step := range g.Steps {
		if step.Title != "" {
			fmt.Fprintf(&sb, "Step %d: %s\n", step.OrderBy, step.Title)
		} else {
			fmt.Fprintf(&sb, "Step %d:\n", step.OrderBy)
		}
		for _, line := range step.Lines {
			sb.WriteString("- ")
			sb.WriteString(line.Text)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String())
}

func extractFixes(content string) []string {
	lower := strings.ToLower(content)
	knownFixes := []string{
		"replace", "remove", "install", "tighten", "disconnect",
		"reconnect", "drain", "refill", "bleed", "adjust",
		"align", "lubricate", "clean", "inspect",
	}
	var found []string
	for _, f := range knownFixes {
		if strings.Contains(lower, f) {
			found = append(found, f)
		}
	}
	return found
}
