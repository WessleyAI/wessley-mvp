// Command scraper-youtube discovers automotive repair videos on YouTube,
// extracts transcripts, and outputs structured JSON.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/WessleyAI/wessley-mvp/engine/scraper"
)

func main() {
	var (
		apiKey   = flag.String("api-key", os.Getenv("YOUTUBE_API_KEY"), "YouTube Data API v3 key")
		query    = flag.String("query", "", "search query (default: use automotive keywords)")
		videoIDs = flag.String("video-ids", "", "comma-separated video IDs to scrape directly (no API key needed)")
		maxRes   = flag.Int("max", 10, "max results per query")
	)
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	s := scraper.NewYouTubeScraper(*apiKey, nil)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	var ch <-chan func() (scraper.ScrapedPost, error)

	if *videoIDs != "" {
		ids := strings.Split(*videoIDs, ",")
		results := s.ScrapeVideoIDs(ctx, ids)
		ch = unwrapChan(results)
	} else {
		if *apiKey == "" {
			fmt.Fprintln(os.Stderr, "error: YouTube API key required (set YOUTUBE_API_KEY or use -api-key)")
			fmt.Fprintln(os.Stderr, "       use -video-ids for direct scraping without API key")
			os.Exit(1)
		}
		results := s.Scrape(ctx, scraper.ScrapeOpts{
			Query:      *query,
			MaxResults: *maxRes,
		})
		ch = unwrapChan(results)
	}

	count := 0
	for unwrap := range ch {
		post, err := unwrap()
		if err != nil {
			log.Printf("scrape error: %v", err)
			continue
		}
		if err := enc.Encode(post); err != nil {
			log.Printf("encode error: %v", err)
		}
		count++
	}

	log.Printf("scraped %d videos", count)
}

// unwrapChan converts fn.Result channel to a channel of unwrap functions
// to avoid importing fn in main's hot path.
func unwrapChan[T any](in <-chan interface{ Unwrap() (T, error) }) <-chan func() (T, error) {
	out := make(chan func() (T, error))
	go func() {
		defer close(out)
		for r := range in {
			r := r
			out <- r.Unwrap
		}
	}()
	return out
}
