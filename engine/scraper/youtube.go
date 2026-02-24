package scraper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WessleyAI/wessley-mvp/pkg/fn"
	"golang.org/x/time/rate"
)

// DefaultYouTubeChannels lists known automotive repair channels.
var DefaultYouTubeChannels = []string{
	"ChrisFix", "ScottyKilmer", "SouthMainAutoRepairAvoca",
	"EricTheCarGuy", "RainmanRaysRepairs", "1AAuto", "BleepinJeep",
}

// AutomotiveKeywords used for search queries.
var AutomotiveKeywords = []string{
	"car repair", "auto repair", "mechanic", "how to fix",
	"engine", "transmission", "brakes", "electrical",
	"wiring", "fuse", "relay", "alternator", "starter", "battery",
}

// vehiclePattern extracts year/make/model from text.
var vehiclePattern = regexp.MustCompile(`(?i)((?:19|20)\d{2})\s+([\w-]+)\s+([\w-]+)`)

// YouTubeScraper discovers and scrapes automotive YouTube videos.
type YouTubeScraper struct {
	apiKey      string
	channels    []string
	rateLimiter *rate.Limiter
	httpClient  *http.Client
	seen        sync.Map // dedup by video ID
}

// NewYouTubeScraper creates a scraper with the given API key.
func NewYouTubeScraper(apiKey string, channels []string) *YouTubeScraper {
	if len(channels) == 0 {
		channels = DefaultYouTubeChannels
	}
	return &YouTubeScraper{
		apiKey:      apiKey,
		channels:    channels,
		rateLimiter: rate.NewLimiter(rate.Every(200*time.Millisecond), 5),
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// VideoMeta holds video metadata from search results.
type VideoMeta struct {
	VideoID     string    `json:"video_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Channel     string    `json:"channel"`
	PublishedAt time.Time `json:"published_at"`
}

// searchResponse is the YouTube Data API v3 search response.
type searchResponse struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
		Snippet struct {
			Title        string `json:"title"`
			Description  string `json:"description"`
			ChannelTitle string `json:"channelTitle"`
			PublishedAt  string `json:"publishedAt"`
		} `json:"snippet"`
	} `json:"items"`
	Error *struct {
		Code int `json:"code"`
	} `json:"error"`
}

// SearchVideos searches YouTube for videos matching a query.
func (s *YouTubeScraper) SearchVideos(ctx context.Context, query string, max int) fn.Result[[]VideoMeta] {
	if s.apiKey == "" {
		return fn.Err[[]VideoMeta](fmt.Errorf("YouTube API key required for search"))
	}

	if err := s.rateLimiter.Wait(ctx); err != nil {
		return fn.Err[[]VideoMeta](err)
	}

	params := url.Values{
		"part":              {"snippet"},
		"q":                 {query},
		"type":              {"video"},
		"videoDuration":     {"medium"},
		"relevanceLanguage": {"en"},
		"maxResults":        {strconv.Itoa(max)},
		"key":               {s.apiKey},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://www.googleapis.com/youtube/v3/search?"+params.Encode(), nil)
	if err != nil {
		return fn.Err[[]VideoMeta](err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fn.Err[[]VideoMeta](err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return fn.Err[[]VideoMeta](ErrQuotaExhausted)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fn.Err[[]VideoMeta](err)
	}

	var sr searchResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return fn.Err[[]VideoMeta](err)
	}

	var videos []VideoMeta
	for _, item := range sr.Items {
		pub, _ := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
		videos = append(videos, VideoMeta{
			VideoID:     item.ID.VideoID,
			Title:       item.Snippet.Title,
			Description: item.Snippet.Description,
			Channel:     item.Snippet.ChannelTitle,
			PublishedAt: pub,
		})
	}
	return fn.Ok(videos)
}

// ErrQuotaExhausted is returned when YouTube API quota is exceeded.
var ErrQuotaExhausted = fmt.Errorf("youtube API quota exhausted")

// Scrape runs a full scrape based on options.
func (s *YouTubeScraper) Scrape(ctx context.Context, opts ScrapeOpts) <-chan fn.Result[ScrapedPost] {
	ch := make(chan fn.Result[ScrapedPost], 32)

	go func() {
		defer close(ch)

		var queries []string
		if opts.Query != "" {
			queries = []string{opts.Query}
		} else {
			// First 6 keywords + "tutorial"
			for i, kw := range AutomotiveKeywords {
				if i >= 6 {
					break
				}
				queries = append(queries, kw+" tutorial")
			}
		}

		maxPerQuery := opts.MaxResults
		if maxPerQuery <= 0 {
			maxPerQuery = 10
		}

		// Search-based discovery
		for _, q := range queries {
			if ctx.Err() != nil {
				return
			}
			result := s.SearchVideos(ctx, q, maxPerQuery)
			videos, err := result.Unwrap()
			if err != nil {
				if err == ErrQuotaExhausted {
					ch <- fn.Err[ScrapedPost](ErrQuotaExhausted)
					return
				}
				continue
			}

			for _, v := range videos {
				if ctx.Err() != nil {
					return
				}
				r := s.ScrapeVideo(ctx, v.VideoID, v.Title)
				if r.IsOk() {
					ch <- r
				}
			}
		}
	}()

	return ch
}

// ScrapeVideo scrapes a single video by ID.
func (s *YouTubeScraper) ScrapeVideo(ctx context.Context, videoID, title string) fn.Result[ScrapedPost] {
	// Dedup
	if _, loaded := s.seen.LoadOrStore(videoID, true); loaded {
		return fn.Err[ScrapedPost](fmt.Errorf("duplicate video %s", videoID))
	}

	transcriptResult := GetTranscript(ctx, s.httpClient, videoID)
	transcript, err := transcriptResult.Unwrap()
	if err != nil {
		return fn.Err[ScrapedPost](err)
	}

	// Content hash dedup
	h := sha256.Sum256([]byte("youtube" + videoID + transcript))
	_ = hex.EncodeToString(h[:]) // available for Redis dedup

	meta := extractMetadata(title, transcript)

	return fn.Ok(ScrapedPost{
		Source:    "youtube",
		SourceID:  videoID,
		Title:     title,
		Content:   transcript,
		Author:    "",
		URL:       "https://www.youtube.com/watch?v=" + videoID,
		ScrapedAt: time.Now(),
		Metadata:  meta,
	})
}

// ScrapeVideoIDs scrapes specific video IDs (no API key needed).
func (s *YouTubeScraper) ScrapeVideoIDs(ctx context.Context, ids []string) <-chan fn.Result[ScrapedPost] {
	ch := make(chan fn.Result[ScrapedPost], len(ids))

	go func() {
		defer close(ch)
		for _, id := range ids {
			if ctx.Err() != nil {
				return
			}
			if err := s.rateLimiter.Wait(ctx); err != nil {
				return
			}
			ch <- s.ScrapeVideo(ctx, id, "")
		}
	}()

	return ch
}

// extractMetadata parses vehicle info, symptoms, and fixes from text.
func extractMetadata(title, transcript string) Metadata {
	combined := title + " " + transcript
	m := Metadata{}

	// Vehicle pattern
	if matches := vehiclePattern.FindStringSubmatch(combined); len(matches) >= 4 {
		m.Vehicle = matches[1] + " " + matches[2] + " " + matches[3]
	}

	// Symptom keywords
	symptoms := []string{
		"won't start", "no crank", "no start", "overheating", "misfire",
		"rough idle", "stalling", "vibration", "noise", "leak",
		"check engine", "warning light", "dead battery", "won't turn over",
	}
	for _, s := range symptoms {
		if strings.Contains(strings.ToLower(combined), s) {
			m.Symptoms = append(m.Symptoms, s)
		}
	}

	// Fix keywords
	fixes := []string{
		"replace", "install", "remove", "tighten", "adjust",
		"clean", "flush", "bleed", "recharge", "reset",
	}
	for _, f := range fixes {
		if strings.Contains(strings.ToLower(combined), f) {
			m.Fixes = append(m.Fixes, f)
		}
	}

	// Matching automotive keywords
	for _, kw := range AutomotiveKeywords {
		if strings.Contains(strings.ToLower(combined), kw) {
			m.Keywords = append(m.Keywords, kw)
		}
	}

	return m
}
