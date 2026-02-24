package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

const baseURL = "https://www.reddit.com"

// Scraper fetches posts and comments from Reddit's public JSON API.
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

// FetchAll scrapes all configured subreddits and returns posts with comments.
func (s *Scraper) FetchAll(ctx context.Context) ([]Post, error) {
	var allPosts []Post
	limiter := time.NewTicker(s.cfg.RateLimit)
	defer limiter.Stop()

	for _, sub := range s.cfg.Subreddits {
		select {
		case <-ctx.Done():
			return allPosts, ctx.Err()
		default:
		}

		posts, err := s.fetchSubreddit(ctx, sub, limiter)
		if err != nil {
			log.Printf("warning: failed to fetch r/%s: %v", sub, err)
			continue
		}
		allPosts = append(allPosts, posts...)
	}
	return allPosts, nil
}

func (s *Scraper) fetchSubreddit(ctx context.Context, sub string, limiter *time.Ticker) ([]Post, error) {
	url := fmt.Sprintf("%s/r/%s/new.json?limit=%d&raw_json=1", baseURL, sub, s.cfg.PostsPerSub)

	result := fn.Retry(ctx, fn.RetryOpts{
		MaxAttempts: 3,
		InitialWait: 5 * time.Second,
		MaxWait:     30 * time.Second,
		Jitter:      true,
	}, func(ctx context.Context) fn.Result[*listingResponse] {
		<-limiter.C
		return s.doGet(ctx, url)
	})

	resp, err := result.Unwrap()
	if err != nil {
		return nil, fmt.Errorf("r/%s listing: %w", sub, err)
	}

	now := time.Now().UTC()
	posts := make([]Post, 0, len(resp.Data.Children))

	for _, child := range resp.Data.Children {
		d := child.Data
		post := Post{
			ID:          d.ID,
			Subreddit:   d.Subreddit,
			Title:       d.Title,
			Author:      d.Author,
			SelfText:    d.SelfText,
			URL:         d.URL,
			Permalink:   "https://www.reddit.com" + d.Permalink,
			Score:       d.Score,
			NumComments: d.NumComments,
			CreatedUTC:  time.Unix(int64(d.CreatedUTC), 0).UTC(),
			Flair:       d.LinkFlairText,
			ScrapedAt:   now,
		}

		// Fetch comments with rate limiting + retry
		comments, err := s.fetchComments(ctx, d.Permalink, limiter)
		if err != nil {
			log.Printf("warning: comments for %s: %v", d.ID, err)
		} else {
			post.Comments = comments
		}

		posts = append(posts, post)
	}

	return posts, nil
}

func (s *Scraper) fetchComments(ctx context.Context, permalink string, limiter *time.Ticker) ([]Comment, error) {
	url := fmt.Sprintf("%s%s.json?limit=%d&raw_json=1&sort=top", baseURL, permalink, s.cfg.CommentsPerPost)

	result := fn.Retry(ctx, fn.RetryOpts{
		MaxAttempts: 2,
		InitialWait: 3 * time.Second,
		MaxWait:     15 * time.Second,
		Jitter:      true,
	}, func(ctx context.Context) fn.Result[[]Comment] {
		<-limiter.C
		return s.doGetComments(ctx, url)
	})

	comments, err := result.Unwrap()
	if err != nil {
		return nil, err
	}
	return comments, nil
}

func (s *Scraper) doGet(ctx context.Context, url string) fn.Result[*listingResponse] {
	body, err := s.httpGet(ctx, url)
	if err != nil {
		return fn.Err[*listingResponse](err)
	}
	defer body.Close()

	var resp listingResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return fn.Err[*listingResponse](fmt.Errorf("decode listing: %w", err))
	}
	return fn.Ok(&resp)
}

func (s *Scraper) doGetComments(ctx context.Context, url string) fn.Result[[]Comment] {
	body, err := s.httpGet(ctx, url)
	if err != nil {
		return fn.Err[[]Comment](err)
	}
	defer body.Close()

	// Reddit returns [postListing, commentListing]
	var listings []listingResponse
	if err := json.NewDecoder(body).Decode(&listings); err != nil {
		return fn.Err[[]Comment](fmt.Errorf("decode comments: %w", err))
	}
	if len(listings) < 2 {
		return fn.Ok([]Comment(nil))
	}

	var comments []Comment
	for _, child := range listings[1].Data.Children {
		if child.Kind != "t1" {
			continue
		}
		d := child.Data
		comments = append(comments, Comment{
			ID:         d.ID,
			Author:     d.Author,
			Body:       d.Body,
			Score:      d.Score,
			CreatedUTC: time.Unix(int64(d.CreatedUTC), 0).UTC(),
			ParentID:   d.ParentID,
			Depth:      d.Depth,
		})
	}
	return fn.Ok(comments)
}

func (s *Scraper) httpGet(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "wessley-scraper/1.0 (automotive repair data collection)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		resp.Body.Close()
		return nil, fmt.Errorf("http %d from %s", resp.StatusCode, url)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
	return resp.Body, nil
}

// Reddit JSON API response types

type listingResponse struct {
	Data struct {
		Children []listingChild `json:"children"`
		After    string         `json:"after"`
	} `json:"data"`
}

type listingChild struct {
	Kind string      `json:"kind"`
	Data listingData `json:"data"`
}

type listingData struct {
	ID            string  `json:"id"`
	Subreddit     string  `json:"subreddit"`
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	SelfText      string  `json:"selftext"`
	Body          string  `json:"body"`
	URL           string  `json:"url"`
	Permalink     string  `json:"permalink"`
	Score         int     `json:"score"`
	NumComments   int     `json:"num_comments"`
	CreatedUTC    float64 `json:"created_utc"`
	LinkFlairText string  `json:"link_flair_text"`
	ParentID      string  `json:"parent_id"`
	Depth         int     `json:"depth"`
}
