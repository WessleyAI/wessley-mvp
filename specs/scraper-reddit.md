# Spec: scraper-reddit — Go Reddit Scraper

**Branch:** `spec/scraper-reddit`  
**Effort:** 3-4 days (1 dev)  
**Priority:** P1 — Phase 2

---

## Scope

Port the existing Python Reddit scraper to Go. Scrape automotive subreddits for posts, extract vehicle patterns (year/make/model), and feed into the ingestion pipeline.

### Files

```
engine/scraper/reddit.go       # RedditScraper
engine/scraper/reddit_test.go
engine/scraper/vehicle.go      # Shared VehicleSignature + extraction (used by all scrapers)
```

---

## Subreddits (7 — from Python scraper)

```go
var DefaultSubreddits = []string{
    "MechanicAdvice", "cartalk", "AskMechanics",
    "Justrolledintotheshop", "autorepair", "CarHelp", "MechanicAdviceEurope",
}
```

## Vehicle Pattern (from Python)

```go
var VehiclePattern = regexp.MustCompile(
    `(\d{4})(?:\s*-\s*(\d{4}))?\s+([A-Za-z]+)\s+([A-Za-z0-9]+(?:\s+[A-Za-z0-9]+)?)`)

type VehicleSignature struct {
    Make, Model         string
    YearStart, YearEnd  int
}
func ExtractVehicle(text string) *VehicleSignature
```

## Key Types

```go
type RedditScraper struct {
    subreddits     []string
    clientID       string
    clientSecret   string
    userAgent      string
    rateLimiter    *resilience.Limiter  // shared pkg/resilience rate limiter (not custom)
}

type ScrapedPost struct {
    Source, SourceID, URL, Title, Content, Author string
    CreatedAt  time.Time
    Vehicle    *VehicleSignature
    Metadata   map[string]string  // subreddit, upvotes, comments_count
}

func (s *RedditScraper) Scrape(ctx context.Context, opts ScrapeOpts) <-chan fn.Result[ScrapedPost]
func (s *RedditScraper) ScrapeComments(ctx context.Context, id string, limit int) fn.Result[[]string]
func RedditStage(scraper *RedditScraper) fn.Stage[ScrapeOpts, []ScrapedPost]
```

## Rate Limiting

Uses the shared `pkg/resilience.Limiter` (token bucket) — NOT a custom rate limiter. Configured per-source:

```go
redditLimiter := resilience.NewLimiter(resilience.LimiterConfig{
    Rate:  1.0,  // 60 req/min
    Burst: 5,
})
scraper := NewRedditScraper(cfg, redditLimiter)
```

## Bulkhead Pattern

Each subreddit gets its own goroutine pool to prevent one slow/failing subreddit from blocking others:

```go
// Each subreddit scrapes independently with bounded concurrency
func (s *RedditScraper) Scrape(ctx context.Context, opts ScrapeOpts) <-chan fn.Result[ScrapedPost] {
    out := make(chan fn.Result[ScrapedPost])
    for _, sub := range s.subreddits {
        go func(sub string) {
            // own goroutine pool per subreddit (bounded)
            sem := make(chan struct{}, opts.MaxConcurrentPerSub) // default: 3
            // scrape sub with semaphore-bounded workers
        }(sub)
    }
    return out
}
```

## Reddit API

- OAuth2: POST `https://www.reddit.com/api/v1/access_token` → bearer token
- Hot: `GET /r/{sub}/hot?limit=100` on `oauth.reddit.com`
- Search: `GET /r/{sub}/search?q={q}&limit=100&restrict_sr=true`
- Rate limit: via shared pkg/resilience.Limiter
- Skip selftext-empty posts (link-only)
- Retry on 429, 500, 503

## Acceptance Criteria

- [ ] Scrapes all 7 subreddits
- [ ] OAuth2 token acquisition and refresh
- [ ] Uses shared pkg/resilience rate limiter (not custom)
- [ ] Bulkhead pattern: each subreddit gets its own goroutine pool
- [ ] Vehicle pattern extraction matches Python behavior
- [ ] Skips deleted/removed/link-only posts
- [ ] Channel-based async output
- [ ] Context cancellation stops scraping
- [ ] Retry transient HTTP errors
- [ ] Comment scraping for enrichment
- [ ] Unit tests with mocked HTTP
- [ ] Integrates as `fn.Stage`

## Dependencies

- `pkg/fn`, `pkg/resilience`, `net/http` (no external Reddit lib)

## Reference

- Python: `services/knowledge-scraper/src/scrapers/reddit.py`
- FINAL_ARCHITECTURE.md §8.2


## Feb 15 Refinement: Scraper Operational Concerns

### Incremental Scraping

Track `last_scraped` timestamp per subreddit in Redis to avoid re-scraping old content:

```go
// Before scraping, check last_scraped timestamp
lastScraped, _ := redis.Get(ctx, "scraper:reddit:last_scraped:"+subreddit).Time()

// Only fetch posts newer than lastScraped
// After successful scrape, update timestamp
redis.Set(ctx, "scraper:reddit:last_scraped:"+subreddit, time.Now(), 0)
```

### Session/Token Refresh

Reddit OAuth2 tokens expire after 1 hour. Implement proactive refresh:

```go
type tokenManager struct {
    token     string
    expiresAt time.Time
    mu        sync.Mutex
}

func (t *tokenManager) Token(ctx context.Context) (string, error) {
    t.mu.Lock()
    defer t.mu.Unlock()
    if time.Now().After(t.expiresAt.Add(-5 * time.Minute)) {
        // Refresh token proactively 5 min before expiry
    }
    return t.token, nil
}
```

### Content-Hash Dedup

Deduplicate scraped posts using content hash stored in Redis:

```go
// SHA-256 hash of (source + sourceID + content)
hash := sha256.Sum256([]byte(post.Source + post.SourceID + post.Content))
key := "scraper:dedup:" + hex.EncodeToString(hash[:])

// SETNX — if key exists, post is a duplicate
if !redis.SetNX(ctx, key, 1, 30*24*time.Hour).Val() {
    // Skip duplicate
    return
}
```

### Additional acceptance criteria
- [ ] Incremental scraping via `last_scraped` per subreddit in Redis
- [ ] Proactive OAuth2 token refresh (5 min before expiry)
- [ ] Content-hash dedup in Redis (SHA-256, 30-day TTL)
- [ ] New dependency: Redis client
