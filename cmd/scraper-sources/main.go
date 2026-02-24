// Command scraper-sources scrapes automotive data from NHTSA complaints,
// iFixit repair guides, and automotive forums, outputting structured JSON
// to stdout or publishing to NATS.
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
	"time"

	"github.com/nats-io/nats.go"

	"github.com/WessleyAI/wessley-mvp/cmd/scraper-sources/forums"
	"github.com/WessleyAI/wessley-mvp/cmd/scraper-sources/ifixit"
	"github.com/WessleyAI/wessley-mvp/cmd/scraper-sources/nhtsa"
	"github.com/WessleyAI/wessley-mvp/engine/scraper"
	"github.com/WessleyAI/wessley-mvp/pkg/natsutil"
)

func main() {
	natsURL := flag.String("nats", "", "NATS URL (if empty, output JSON to stdout)")
	subject := flag.String("subject", "wessley.scraper.sources.posts", "NATS subject to publish to")
	interval := flag.Duration("interval", 30*time.Minute, "polling interval (0 = one-shot)")
	sources := flag.String("sources", "nhtsa,ifixit,forums", "comma-separated sources to scrape")
	nhtsaMakes := flag.String("nhtsa-makes", "TOYOTA,HONDA,FORD,CHEVROLET,BMW,NISSAN", "comma-separated vehicle makes for NHTSA")
	nhtsaYear := flag.Int("nhtsa-year", 2024, "model year for NHTSA queries")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	enabledSources := make(map[string]bool)
	for _, s := range strings.Split(*sources, ",") {
		enabledSources[strings.TrimSpace(s)] = true
	}

	// Initialize scrapers
	var nhtsaScraper *nhtsa.Scraper
	if enabledSources["nhtsa"] {
		makes := strings.Split(*nhtsaMakes, ",")
		nhtsaScraper = nhtsa.NewScraper(nhtsa.Config{
			Makes:      makes,
			ModelYear:  *nhtsaYear,
			MaxPerMake: 50,
			RateLimit:  2 * time.Second,
		})
	}

	var ifixitScraper *ifixit.Scraper
	if enabledSources["ifixit"] {
		ifixitScraper = ifixit.NewScraper(ifixit.Config{
			Categories: []string{
				"Car and Truck",
				"Car",
			},
			MaxGuides: 50,
			RateLimit: 1 * time.Second,
		})
	}

	var forumScraper *forums.Scraper
	if enabledSources["forums"] {
		forumScraper = forums.NewScraper(forums.Config{
			Forums: forums.DefaultForums(),
			Queries: []string{
				"engine repair",
				"brake problem",
				"transmission issue",
				"check engine light",
				"oil leak",
			},
			MaxPerForum: 25,
			RateLimit:   3 * time.Second,
		})
	}

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

	emit := func(posts []scraper.ScrapedPost) error {
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

	run := func() error {
		var total int

		if nhtsaScraper != nil {
			posts, err := nhtsaScraper.FetchAll(ctx)
			if err != nil {
				log.Printf("nhtsa error: %v", err)
			} else {
				log.Printf("nhtsa: fetched %d posts", len(posts))
				total += len(posts)
				if err := emit(posts); err != nil {
					return err
				}
			}
		}

		if ifixitScraper != nil {
			posts, err := ifixitScraper.FetchAll(ctx)
			if err != nil {
				log.Printf("ifixit error: %v", err)
			} else {
				log.Printf("ifixit: fetched %d posts", len(posts))
				total += len(posts)
				if err := emit(posts); err != nil {
					return err
				}
			}
		}

		if forumScraper != nil {
			posts, err := forumScraper.FetchAll(ctx)
			if err != nil {
				log.Printf("forums error: %v", err)
			} else {
				log.Printf("forums: fetched %d posts", len(posts))
				total += len(posts)
				if err := emit(posts); err != nil {
					return err
				}
			}
		}

		log.Printf("total: %d posts scraped", total)
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
