// Command scraper-reddit scrapes automotive repair subreddits for posts and
// comments, outputting structured JSON to stdout or publishing to NATS.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/WessleyAI/wessley-mvp/cmd/scraper-reddit/reddit"
	"github.com/WessleyAI/wessley-mvp/pkg/natsutil"
)

func main() {
	natsURL := flag.String("nats", "", "NATS URL (if empty, output JSON to stdout)")
	subject := flag.String("subject", "wessley.scraper.reddit.posts", "NATS subject to publish to")
	limit := flag.Int("limit", 25, "posts per subreddit per fetch")
	interval := flag.Duration("interval", 5*time.Minute, "polling interval (0 = one-shot)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	subreddits := []string{
		"MechanicAdvice",
		"CarRepair",
		"AskMechanic",
		"AutoRepair",
		"Cartalk",
	}

	scraper := reddit.NewScraper(reddit.Config{
		Subreddits:      subreddits,
		PostsPerSub:     *limit,
		CommentsPerPost: 50,
		RateLimit:       2 * time.Second, // 1 request per 2s to stay under Reddit limits
	})

	var nc *nats.Conn
	if *natsURL != "" {
		var err error
		nc, err = nats.Connect(*natsURL)
		if err != nil {
			log.Fatalf("nats connect: %v", err)
		}
		defer nc.Close()
		log.Printf("publishing to NATS subject %s", *subject)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	run := func() error {
		posts, err := scraper.FetchAll(ctx)
		if err != nil {
			return fmt.Errorf("fetch: %w", err)
		}
		log.Printf("fetched %d posts", len(posts))

		for _, p := range posts {
			if nc != nil {
				if err := natsutil.Publish(ctx, nc, *subject, p); err != nil {
					log.Printf("nats publish error: %v", err)
				}
			} else {
				if err := enc.Encode(p); err != nil {
					return fmt.Errorf("encode: %w", err)
				}
			}
		}
		return nil
	}

	// First run
	if err := run(); err != nil {
		log.Fatalf("scrape: %v", err)
	}

	if *interval <= 0 {
		return
	}

	// Poll loop
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("shutting down")
			return
		case <-ticker.C:
			if err := run(); err != nil {
				log.Printf("scrape error: %v", err)
			}
		}
	}
}
