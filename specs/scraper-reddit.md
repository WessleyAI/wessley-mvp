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
    rateLimiter    *rate.Limiter
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

## Reddit API

- OAuth2: POST `https://www.reddit.com/api/v1/access_token` → bearer token
- Hot: `GET /r/{sub}/hot?limit=100` on `oauth.reddit.com`
- Search: `GET /r/{sub}/search?q={q}&limit=100&restrict_sr=true`
- Rate limit: 60 req/min via `golang.org/x/time/rate`
- Skip selftext-empty posts (link-only)
- Retry on 429, 500, 503

## Acceptance Criteria

- [ ] Scrapes all 7 subreddits
- [ ] OAuth2 token acquisition and refresh
- [ ] Rate limiting (configurable RPM, default 60)
- [ ] Vehicle pattern extraction matches Python behavior
- [ ] Skips deleted/removed/link-only posts
- [ ] Channel-based async output
- [ ] Context cancellation stops scraping
- [ ] Retry transient HTTP errors
- [ ] Comment scraping for enrichment
- [ ] Unit tests with mocked HTTP
- [ ] Integrates as `fn.Stage`

## Dependencies

- `pkg/fn`, `golang.org/x/time/rate`, `net/http` (no external Reddit lib)

## Reference

- Python: `services/knowledge-scraper/src/scrapers/reddit.py`
- FINAL_ARCHITECTURE.md §8.2
