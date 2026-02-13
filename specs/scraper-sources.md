# Spec: scraper-sources — Forums, NHTSA, iFixit Scrapers

**Branch:** `spec/scraper-sources`  
**Effort:** 4-5 days (1 dev)  
**Priority:** P1 — Phase 2

---

## Scope

Three additional scrapers: configurable forum scraper (ported from Python), NHTSA recalls API client, iFixit guide scraper.

### Files

```
engine/scraper/forum.go, forum_test.go
engine/scraper/nhtsa.go, nhtsa_test.go
engine/scraper/ifixit.go, ifixit_test.go
engine/scraper/source.go   # Scraper interface + ScrapeOpts
```

---

## 1. Forum Scraper

### Configs (from Python `FORUM_CONFIGS`)

```go
type ForumConfig struct {
    Name, BaseURL, SearchPath                          string
    ThreadSelector, TitleSelector, ContentSelector     string
    AuthorSelector, DateSelector, NextPageSelector     string
}

var DefaultForumConfigs = map[string]ForumConfig{
    "bimmerpost":   {Name: "BimmerPost",    BaseURL: "https://www.bimmerpost.com",  SearchPath: "/forums/search.php", ThreadSelector: "li.searchResult",  TitleSelector: "h3.title a", ContentSelector: "div.snippet"},
    "toyotanation": {Name: "ToyotaNation",  BaseURL: "https://www.toyotanation.com", SearchPath: "/threads/search",   ThreadSelector: "div.structItem",  TitleSelector: "div.structItem-title a", ContentSelector: "div.structItem-snippet"},
    "hondatech":    {Name: "Honda-Tech",    BaseURL: "https://honda-tech.com",      SearchPath: "/forums/search/",   ThreadSelector: "li.searchResult",  TitleSelector: "h3.title a", ContentSelector: "div.snippet"},
    "jeepforum":    {Name: "JeepForum",     BaseURL: "https://www.jeepforum.com",   SearchPath: "/forum/search.php", ThreadSelector: "li.searchResult",  TitleSelector: "h3.title a", ContentSelector: "div.snippet"},
    "mbworld":      {Name: "MBWorld",       BaseURL: "https://mbworld.org",         SearchPath: "/forums/search.php", ThreadSelector: "li.searchResult", TitleSelector: "h3.title a", ContentSelector: "div.snippet"},
}
```

- Uses `goquery` for CSS selector parsing
- Pagination via `NextPageSelector` links
- Resolves relative URLs against BaseURL
- User-Agent: `"Mozilla/5.0 (compatible; WessleyBot/1.0; +https://wessley.ai)"`

## 2. NHTSA Recalls API

```go
type NHTSAScraper struct { httpClient *http.Client; rateLimiter *rate.Limiter }

func (s *NHTSAScraper) ScrapeRecalls(ctx context.Context, make, model string, year int) fn.Result[[]ScrapedPost]
func (s *NHTSAScraper) ScrapeAllRecalls(ctx context.Context, limit int) <-chan fn.Result[ScrapedPost]
```

- Endpoint: `https://api.nhtsa.gov/recalls/recallsByVehicle?make={}&model={}&modelYear={}`
- No API key required; returns JSON `results` array
- Fields: NHTSACampaignNumber, Component, Summary, Consequence, Remedy

## 3. iFixit Guide Scraper

```go
type IFixitScraper struct { httpClient *http.Client; rateLimiter *rate.Limiter }

func (s *IFixitScraper) ScrapeGuides(ctx context.Context, query string, limit int) <-chan fn.Result[ScrapedPost]
```

- Search: `GET https://www.ifixit.com/api/2.0/search/{query}?filter=guide&limit=20`
- Detail: `GET https://www.ifixit.com/api/2.0/guides/{id}`
- No API key; extract step-by-step text

## Shared Interface

```go
type Scraper interface {
    Scrape(ctx context.Context, opts ScrapeOpts) <-chan fn.Result[ScrapedPost]
}
type ScrapeOpts struct { Query string; Limit int }
```

## Acceptance Criteria

- [ ] Forum scraper with all 5 configs, pagination, relative URL resolution
- [ ] Forum configs pluggable (new forums = config only)
- [ ] NHTSA fetches recalls by vehicle, converts to ScrapedPost
- [ ] iFixit fetches guides, extracts step text
- [ ] All implement `Scraper` interface
- [ ] All rate-limited, all extract vehicle signatures
- [ ] Context cancellation, unit tests with mocked HTTP

## Dependencies

- `pkg/fn`, `golang.org/x/time/rate`, `github.com/PuerkitoBio/goquery`

## Reference

- Python: `services/knowledge-scraper/src/scrapers/forum.py`
