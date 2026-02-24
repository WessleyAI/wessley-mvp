package forums

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/scraper"
	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

// Scraper fetches threads from automotive forums via HTML scraping.
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

// DefaultForums returns the standard list of automotive forums to scrape.
func DefaultForums() []ForumConfig {
	return []ForumConfig{
		{Name: "BobIsTheOilGuy", BaseURL: "https://bobistheoilguy.com/forums", SearchPath: "/search/?q=%s&o=date"},
		{Name: "AutoRepairForum", BaseURL: "https://www.autorepairforum.com", SearchPath: "/search/?q=%s"},
		{Name: "MechanicsForum", BaseURL: "https://www.mechanicsforum.com", SearchPath: "/search/?q=%s"},
	}
}

// FetchAll scrapes all configured forums and returns ScrapedPosts.
func (s *Scraper) FetchAll(ctx context.Context) ([]scraper.ScrapedPost, error) {
	var allPosts []scraper.ScrapedPost
	limiter := time.NewTicker(s.cfg.RateLimit)
	defer limiter.Stop()

	for _, forum := range s.cfg.Forums {
		for _, query := range s.cfg.Queries {
			select {
			case <-ctx.Done():
				return allPosts, ctx.Err()
			default:
			}

			posts, err := s.fetchForum(ctx, forum, query, limiter)
			if err != nil {
				log.Printf("warning: failed to fetch %s for %q: %v", forum.Name, query, err)
				continue
			}
			allPosts = append(allPosts, posts...)
		}
	}
	return allPosts, nil
}

func (s *Scraper) fetchForum(ctx context.Context, forum ForumConfig, query string, limiter *time.Ticker) ([]scraper.ScrapedPost, error) {
	searchURL := forum.BaseURL + fmt.Sprintf(forum.SearchPath, query)

	result := fn.Retry(ctx, fn.RetryOpts{
		MaxAttempts: 3,
		InitialWait: 5 * time.Second,
		MaxWait:     30 * time.Second,
		Jitter:      true,
	}, func(ctx context.Context) fn.Result[string] {
		<-limiter.C
		return s.doGet(ctx, searchURL)
	})

	html, err := result.Unwrap()
	if err != nil {
		return nil, fmt.Errorf("%s search %q: %w", forum.Name, query, err)
	}

	posts := parseSearchResults(html, forum, query)
	if s.cfg.MaxPerForum > 0 && len(posts) > s.cfg.MaxPerForum {
		posts = posts[:s.cfg.MaxPerForum]
	}
	return posts, nil
}

func (s *Scraper) doGet(ctx context.Context, url string) fn.Result[string] {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fn.Err[string](err)
	}
	req.Header.Set("User-Agent", "wessley-scraper/1.0 (automotive repair data collection)")

	resp, err := s.client.Do(req)
	if err != nil {
		return fn.Err[string](err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return fn.Err[string](fmt.Errorf("http %d from %s", resp.StatusCode, url))
	}
	if resp.StatusCode != http.StatusOK {
		return fn.Err[string](fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fn.Err[string](fmt.Errorf("read body: %w", err))
	}
	return fn.Ok(string(body))
}

// parseSearchResults extracts thread info from forum search HTML.
// This uses simple regex patterns — robust enough for the major forum platforms
// (vBulletin, XenForo, phpBB) that power most automotive forums.
func parseSearchResults(html string, forum ForumConfig, query string) []scraper.ScrapedPost {
	now := time.Now().UTC()
	var posts []scraper.ScrapedPost

	// Extract thread links and titles from common forum HTML patterns
	// Matches: <a href="/threads/some-title.12345/" ...>Title Text</a>
	threadRe := regexp.MustCompile(`<a\s+href="([^"]*(?:thread|topic|showthread)[^"]*)"[^>]*>([^<]+)</a>`)
	matches := threadRe.FindAllStringSubmatch(html, -1)

	seen := make(map[string]bool)
	for _, m := range matches {
		href, title := m[1], strings.TrimSpace(m[2])
		if seen[href] || title == "" {
			continue
		}
		seen[href] = true

		url := href
		if !strings.HasPrefix(href, "http") {
			url = forum.BaseURL + href
		}

		posts = append(posts, scraper.ScrapedPost{
			Source:      "forum:" + forum.Name,
			SourceID:    fmt.Sprintf("forum-%s-%s", forum.Name, href),
			Title:       title,
			Content:     "", // content fetched in a follow-up pass if needed
			Author:      "",
			URL:         url,
			PublishedAt: time.Time{},
			ScrapedAt:   now,
			Metadata: scraper.Metadata{
				Keywords: []string{strings.ToLower(forum.Name), "forum", query},
			},
		})
	}
	return posts
}
