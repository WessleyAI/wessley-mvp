package ifixit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

// FetchAll scrapes iFixit guides for all configured categories and returns ScrapedPosts.
func (s *Scraper) FetchAll(ctx context.Context) ([]scraper.ScrapedPost, error) {
	var allPosts []scraper.ScrapedPost
	limiter := time.NewTicker(s.cfg.RateLimit)
	defer limiter.Stop()

	for _, category := range s.cfg.Categories {
		select {
		case <-ctx.Done():
			return allPosts, ctx.Err()
		default:
		}

		posts, err := s.fetchCategory(ctx, category, limiter)
		if err != nil {
			log.Printf("warning: failed to fetch iFixit category %s: %v", category, err)
			continue
		}
		allPosts = append(allPosts, posts...)
	}
	return allPosts, nil
}

func (s *Scraper) fetchCategory(ctx context.Context, category string, limiter *time.Ticker) ([]scraper.ScrapedPost, error) {
	url := fmt.Sprintf("%s/guides?filter=category:%s&limit=%d&order=DESC", baseURL, category, s.cfg.MaxGuides)

	result := fn.Retry(ctx, fn.RetryOpts{
		MaxAttempts: 3,
		InitialWait: 5 * time.Second,
		MaxWait:     30 * time.Second,
		Jitter:      true,
	}, func(ctx context.Context) fn.Result[[]Guide] {
		<-limiter.C
		return s.doGet(ctx, url)
	})

	guides, err := result.Unwrap()
	if err != nil {
		return nil, fmt.Errorf("ifixit %s: %w", category, err)
	}

	now := time.Now().UTC()
	posts := make([]scraper.ScrapedPost, 0, len(guides))

	for _, g := range guides {
		content := buildGuideContent(g)
		published := time.Unix(g.ModifiedDate, 0).UTC()

		posts = append(posts, scraper.ScrapedPost{
			Source:      "ifixit",
			SourceID:    fmt.Sprintf("ifixit-%d", g.GuideID),
			Title:       g.Title,
			Content:     content,
			Author:      g.Author.Username,
			URL:         g.URL,
			PublishedAt: published,
			ScrapedAt:   now,
			Metadata: scraper.Metadata{
				Vehicle:  g.Subject,
				Fixes:    extractFixes(content),
				Keywords: []string{strings.ToLower(g.Category), "ifixit", "repair-guide", strings.ToLower(g.Difficulty)},
			},
		})
	}
	return posts, nil
}

func (s *Scraper) doGet(ctx context.Context, url string) fn.Result[[]Guide] {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fn.Err[[]Guide](err)
	}
	req.Header.Set("User-Agent", "wessley-scraper/1.0 (automotive repair data collection)")

	resp, err := s.client.Do(req)
	if err != nil {
		return fn.Err[[]Guide](err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return fn.Err[[]Guide](fmt.Errorf("http %d from %s", resp.StatusCode, url))
	}
	if resp.StatusCode != http.StatusOK {
		return fn.Err[[]Guide](fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fn.Err[[]Guide](fmt.Errorf("read body: %w", err))
	}

	var guides []Guide
	if err := json.Unmarshal(body, &guides); err != nil {
		return fn.Err[[]Guide](fmt.Errorf("decode: %w", err))
	}
	return fn.Ok(guides)
}

func buildGuideContent(g Guide) string {
	var sb strings.Builder
	if g.Summary != "" {
		sb.WriteString(g.Summary)
		sb.WriteString("\n\n")
	}
	for _, step := range g.Steps {
		if step.Title != "" {
			sb.WriteString(fmt.Sprintf("Step %d: %s\n", step.OrderBy, step.Title))
		} else {
			sb.WriteString(fmt.Sprintf("Step %d:\n", step.OrderBy))
		}
		for _, line := range step.Lines {
			sb.WriteString("  " + line.Text + "\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
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
