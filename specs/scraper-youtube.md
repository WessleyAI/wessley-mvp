# Spec: scraper-youtube — Go YouTube Transcript Scraper

**Branch:** `spec/scraper-youtube`  
**Effort:** 4-5 days (1 dev)  
**Priority:** P1 — Phase 2

---

## Scope

Port the Python YouTube scraper to Go. Discover automotive repair videos via YouTube Data API v3, extract transcripts (manual captions preferred, auto-generated fallback), clean text, extract vehicle patterns.

### Files

```
engine/scraper/youtube.go       # YouTubeScraper
engine/scraper/youtube_test.go
engine/scraper/transcript.go    # Transcript fetch + clean
engine/scraper/transcript_test.go
```

---

## Channels (7 — from Python)

```go
var DefaultYouTubeChannels = []string{
    "ChrisFix", "ScottyKilmer", "SouthMainAutoRepairAvoca",
    "EricTheCarGuy", "RainmanRaysRepairs", "1AAuto", "BleepinJeep",
}
```

## Automotive Keywords

```go
var AutomotiveKeywords = []string{
    "car repair", "auto repair", "mechanic", "how to fix",
    "engine", "transmission", "brakes", "electrical",
    "wiring", "fuse", "relay", "alternator", "starter", "battery",
}
```

## Key Types

```go
type YouTubeScraper struct {
    apiKey      string
    channels    []string
    rateLimiter *rate.Limiter
    httpClient  *http.Client
}

type VideoMeta struct {
    VideoID, Title, Description, Channel string
    PublishedAt time.Time
}

func (s *YouTubeScraper) Scrape(ctx context.Context, opts ScrapeOpts) <-chan fn.Result[ScrapedPost]
func (s *YouTubeScraper) ScrapeVideo(ctx context.Context, videoID, title string) fn.Result[ScrapedPost]
func (s *YouTubeScraper) ScrapeVideoIDs(ctx context.Context, ids []string) <-chan fn.Result[ScrapedPost]
func (s *YouTubeScraper) SearchVideos(ctx context.Context, query string, max int) fn.Result[[]VideoMeta]
```

## Transcript Extraction

```go
func GetTranscript(ctx context.Context, videoID string) fn.Result[string]
func CleanTranscript(text string) string
```

**Cleaning (from Python):** Remove `[Music]`/`[Applause]`/`[Laughter]` bracket noise, collapse whitespace, trim.

**Fetch strategy:** Parse `timedtext` XML endpoint or extract caption track URLs from `ytInitialPlayerResponse`. Prefer manual English → fallback to auto-generated.

## YouTube Data API v3

- Search: `GET youtube/v3/search` — `part=snippet, type=video, videoDuration=medium, relevanceLanguage=en`
- Stats: `GET youtube/v3/videos` — `part=statistics`, batch up to 50 IDs

## Scraping Strategy (from Python)

1. Query-based search (if query provided)
2. Else: first 6 keywords + "tutorial" as queries
3. Then: known channel discovery
4. Deduplicate by video ID
5. On HTTP 403 (quota) → stop all scraping

## Acceptance Criteria

- [ ] Discovers videos via Data API v3
- [ ] Extracts transcripts (manual preferred, auto fallback)
- [ ] Cleans transcript text
- [ ] Vehicle pattern extraction from title + transcript
- [ ] Rate limiting, deduplication, quota handling (403 → stop)
- [ ] `ScrapeVideoIDs` works without API key
- [ ] Context cancellation, unit tests with mocked HTTP
- [ ] Integrates as `fn.Stage`

## Dependencies

- `pkg/fn`, `golang.org/x/time/rate`, `net/http`, `encoding/xml`

## Reference

- Python: `services/knowledge-scraper/src/scrapers/youtube.py`
- FINAL_ARCHITECTURE.md §8.2


## Feb 15 Refinement: Scraper Operational Concerns

### Incremental Scraping

Track `last_scraped` timestamp per channel in Redis to avoid re-processing old videos:

```go
lastScraped, _ := redis.Get(ctx, "scraper:youtube:last_scraped:"+channel).Time()

// Use publishedAfter parameter in YouTube API search to only get new videos
// After successful scrape, update timestamp
redis.Set(ctx, "scraper:youtube:last_scraped:"+channel, time.Now(), 0)
```

### Session/Token Refresh

YouTube Data API uses API keys (no OAuth2 for public data), but quota resets daily. Track quota usage:

```go
// Track daily quota usage in Redis
quotaKey := "scraper:youtube:quota:" + time.Now().Format("2006-01-02")
used, _ := redis.Incr(ctx, quotaKey).Result()
redis.Expire(ctx, quotaKey, 25*time.Hour) // auto-cleanup

if used > quotaLimit {
    return fn.Err[[]ScrapedPost](ErrQuotaExhausted)
}
```

### Content-Hash Dedup

Deduplicate scraped videos using content hash in Redis:

```go
hash := sha256.Sum256([]byte("youtube" + videoID + transcript))
key := "scraper:dedup:" + hex.EncodeToString(hash[:])

if !redis.SetNX(ctx, key, 1, 30*24*time.Hour).Val() {
    return // skip duplicate
}
```

### Additional acceptance criteria
- [ ] Incremental scraping via `last_scraped` per channel in Redis
- [ ] Daily quota tracking in Redis with auto-expiry
- [ ] Content-hash dedup in Redis (SHA-256, 30-day TTL)
- [ ] New dependency: Redis client
