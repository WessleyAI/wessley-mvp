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

	"github.com/WessleyAI/wessley-mvp/pkg/metrics"

	"github.com/WessleyAI/wessley-mvp/cmd/scraper-sources/forums"
	"github.com/WessleyAI/wessley-mvp/cmd/scraper-sources/ifixit"
	"github.com/WessleyAI/wessley-mvp/cmd/scraper-sources/manuals"
	"github.com/WessleyAI/wessley-mvp/cmd/scraper-sources/nhtsa"
	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/engine/scraper"
	"github.com/WessleyAI/wessley-mvp/pkg/natsutil"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	natsURL := flag.String("nats", "", "NATS URL (if empty, output JSON to stdout)")
	subject := flag.String("subject", "wessley.scraper.sources.posts", "NATS subject to publish to")
	outputDir := flag.String("output-dir", "", "directory to write JSON files for ingest pipeline (e.g. /tmp/wessley-data)")
	interval := flag.Duration("interval", 30*time.Minute, "polling interval (0 = one-shot)")
	sources := flag.String("sources", "nhtsa,ifixit,forums", "comma-separated sources to scrape")
	nhtsaMakes := flag.String("nhtsa-makes", "TOYOTA,HONDA,FORD,CHEVROLET,BMW,NISSAN", "comma-separated vehicle makes for NHTSA")
	nhtsaYear := flag.Int("nhtsa-year", 2024, "model year for NHTSA queries")
	manualsDir := flag.String("manuals-dir", "", "directory containing PDF vehicle manuals (legacy) / output dir for crawler")
	manualsMax := flag.Int("manuals-max", 0, "max manual files to process (0 = unlimited)")
	manualsDiscover := flag.Bool("manuals-discover", false, "crawl sources and build manual index only")
	manualsDownload := flag.Bool("manuals-download", false, "download pending manuals from index")
	manualsProcess := flag.Bool("manuals-process", false, "full pipeline: discover + download + ingest")
	manualsStatus := flag.Bool("manuals-status", false, "print manual registry stats")
	manualsMakes := flag.String("manuals-makes", "", "comma-separated makes to target")
	manualsYears := flag.String("manuals-years", "2015-2026", "year range (e.g. 2015-2026)")
	manualsSources := flag.String("manuals-sources", "toyota,honda,ford,chevrolet,gmc,ram,jeep,dodge,chrysler,nissan,hyundai,kia,subaru,mazda,volkswagen,bmw,mercedes,audi,tesla,volvo,lexus,acura,infiniti,genesis,porsche,mitsubishi,lincoln,buick,cadillac,archive,nhtsa,search", "comma-separated sources")
	neo4jURL := flag.String("neo4j-url", "", "Neo4j URL for manual registry")
	neo4jUser := flag.String("neo4j-user", "neo4j", "Neo4j username")
	neo4jPass := flag.String("neo4j-pass", "password", "Neo4j password")
	flag.Parse()

	met := metrics.New()
	mDocsTotal := func(source string) *metrics.Counter {
		return met.Counter(metrics.WithLabels("wessley_scraper_sources_docs_total", "source", source), "Docs scraped by source")
	}
	mErrorsTotal := func(source string) *metrics.Counter {
		return met.Counter(metrics.WithLabels("wessley_scraper_sources_errors_total", "source", source), "Scraper errors by source")
	}
	mScrapeDur := func(source string) *metrics.Histogram {
		return met.Histogram(metrics.WithLabels("wessley_scraper_sources_scrape_duration_seconds", "source", source), "Scrape duration by source", nil)
	}
	mLastScrape := met.Gauge("wessley_scraper_sources_last_scrape_timestamp", "Epoch of last scrape")
	mManualsDiscovered := met.Counter("wessley_scraper_manuals_discovered_total", "Manuals discovered")
	mManualsDownloaded := met.Counter("wessley_scraper_manuals_downloaded_total", "Manuals downloaded")
	mManualsFailed := met.Counter("wessley_scraper_manuals_failed_total", "Manuals failed")
	mBytesTotal := func(source string) *metrics.Counter {
		return met.Counter(metrics.WithLabels("wessley_scraper_sources_bytes_total", "source", source), "Bytes scraped by source")
	}
	mManualsDownloadBytes := met.Counter("wessley_scraper_manuals_download_bytes_total", "Total manual download bytes")
	mManualsDownloadDur := met.Histogram("wessley_scraper_manuals_download_duration_seconds", "Manual download duration", nil)
	mManualsPdfPages := met.Counter("wessley_scraper_manuals_pdf_pages_total", "Total PDF pages processed")

	// Suppress unused warnings for manual-mode metrics
	_ = mManualsDiscovered
	_ = mManualsDownloaded
	_ = mManualsFailed
	_ = mBytesTotal
	_ = mManualsDownloadBytes
	_ = mManualsDownloadDur
	_ = mManualsPdfPages

	met.CollectRuntime("wessley_scraper_sources", 15*time.Second)
	met.ServeAsync(9092)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// --- Manual crawler mode ---
	if *manualsDiscover || *manualsDownload || *manualsProcess || *manualsStatus {
		if *neo4jURL == "" {
			*neo4jURL = os.Getenv("NEO4J_URL")
			if *neo4jURL == "" {
				*neo4jURL = "neo4j://localhost:7687"
			}
		}
		if u := os.Getenv("NEO4J_USER"); u != "" && *neo4jUser == "neo4j" {
			*neo4jUser = u
		}
		if p := os.Getenv("NEO4J_PASS"); p != "" && *neo4jPass == "password" {
			*neo4jPass = p
		}

		driver, err := neo4j.NewDriverWithContext(*neo4jURL, neo4j.BasicAuth(*neo4jUser, *neo4jPass, ""))
		if err != nil {
			log.Fatalf("neo4j connect: %v", err)
		}
		defer driver.Close(ctx)

		graphStore := graph.New(driver)

		outputDir := *manualsDir
		if outputDir == "" {
			outputDir = "./manuals"
		}

		yearRange := parseYearRange(*manualsYears)
		var makes []string
		if *manualsMakes != "" {
			makes = strings.Split(*manualsMakes, ",")
		}

		crawlerCfg := manuals.CrawlerConfig{
			OutputDir:    outputDir,
			MaxPerSource: 100,
			RateLimit:    2 * time.Second,
			UserAgent:    "WessleyBot/1.0",
			MaxFileSize:  200 * 1024 * 1024,
			Concurrency:  3,
			Makes:        makes,
			YearRange:    yearRange,
		}

		srcs := buildManualSources(*manualsSources)
		crawler := manuals.NewCrawler(graphStore, crawlerCfg, srcs...)

		switch {
		case *manualsStatus:
			stats, err := graphStore.ManualStats(ctx)
			if err != nil {
				log.Fatalf("manual stats: %v", err)
			}
			fmt.Printf("Total: %d\n", stats.Total)
			fmt.Println("By Status:")
			for k, v := range stats.ByStatus {
				fmt.Printf("  %s: %d\n", k, v)
			}
			fmt.Println("By Source:")
			for k, v := range stats.BySource {
				fmt.Printf("  %s: %d\n", k, v)
			}
		case *manualsDiscover:
			n, err := crawler.Discover(ctx)
			if err != nil {
				log.Fatalf("discover: %v", err)
			}
			fmt.Printf("Discovered %d manuals\n", n)
		case *manualsDownload:
			n, err := crawler.Download(ctx, 0)
			if err != nil {
				log.Fatalf("download: %v", err)
			}
			fmt.Printf("Downloaded %d manuals\n", n)
			// Auto-ingest downloaded PDFs into JSON
			ni, err := crawler.Ingest(ctx, outputDir, 0)
			if err != nil {
				log.Printf("ingest: %v", err)
			} else if ni > 0 {
				fmt.Printf("Ingested %d manuals into JSON\n", ni)
			}
		case *manualsProcess:
			if err := crawler.Process(ctx); err != nil {
				log.Fatalf("process: %v", err)
			}
			fmt.Println("Manual processing complete")
		}
		return
	}

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

	var manualScraper *manuals.Scraper
	if enabledSources["manuals"] && *manualsDir != "" {
		manualScraper = manuals.NewScraper(manuals.Config{
			Directory: *manualsDir,
			MaxFiles:  *manualsMax,
			RateLimit: 500 * time.Millisecond,
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

	// Ensure output dir exists if specified
	if *outputDir != "" {
		os.MkdirAll(*outputDir, 0o755)
		log.Printf("writing JSON files to %s", *outputDir)
	}

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
		// Write posts to output dir as JSON files for ingest pipeline
		if *outputDir != "" && len(posts) > 0 {
			source := "unknown"
			if len(posts) > 0 {
				source = posts[0].Source
			}
			// Replace colons in source (e.g. "reddit:MechanicAdvice" → "reddit-MechanicAdvice")
			source = strings.ReplaceAll(source, ":", "-")
			filename := fmt.Sprintf("%s/%s-%d.json", *outputDir, source, time.Now().UnixNano())
			f, err := os.Create(filename)
			if err != nil {
				log.Printf("output-dir write error: %v", err)
			} else {
				fenc := json.NewEncoder(f)
				for _, p := range posts {
					fenc.Encode(p)
				}
				f.Close()
				log.Printf("wrote %d posts to %s", len(posts), filename)
			}
		}
		return nil
	}

	run := func() error {
		var total int

		if nhtsaScraper != nil {
			start := time.Now()
			posts, err := nhtsaScraper.FetchAll(ctx)
			if err != nil {
				mErrorsTotal("nhtsa").Inc()
				log.Printf("nhtsa error: %v", err)
			} else {
				mScrapeDur("nhtsa").Since(start)
				mDocsTotal("nhtsa").Add(int64(len(posts)))
				log.Printf("nhtsa: fetched %d posts", len(posts))
				total += len(posts)
				if err := emit(posts); err != nil {
					return err
				}
			}
		}

		if ifixitScraper != nil {
			start := time.Now()
			posts, err := ifixitScraper.FetchAll(ctx)
			if err != nil {
				mErrorsTotal("ifixit").Inc()
				log.Printf("ifixit error: %v", err)
			} else {
				mScrapeDur("ifixit").Since(start)
				mDocsTotal("ifixit").Add(int64(len(posts)))
				log.Printf("ifixit: fetched %d posts", len(posts))
				total += len(posts)
				if err := emit(posts); err != nil {
					return err
				}
			}
		}

		if manualScraper != nil {
			start := time.Now()
			posts, err := manualScraper.FetchAll(ctx)
			if err != nil {
				mErrorsTotal("manuals").Inc()
				log.Printf("manuals error: %v", err)
			} else {
				mScrapeDur("manuals").Since(start)
				mDocsTotal("manuals").Add(int64(len(posts)))
				log.Printf("manuals: fetched %d posts", len(posts))
				total += len(posts)
				if err := emit(posts); err != nil {
					return err
				}
			}
		}

		if forumScraper != nil {
			start := time.Now()
			posts, err := forumScraper.FetchAll(ctx)
			if err != nil {
				mErrorsTotal("forums").Inc()
				log.Printf("forums error: %v", err)
			} else {
				mScrapeDur("forums").Since(start)
				mDocsTotal("forums").Add(int64(len(posts)))
				log.Printf("forums: fetched %d posts", len(posts))
				total += len(posts)
				if err := emit(posts); err != nil {
					return err
				}
			}
		}

		mLastScrape.Set(time.Now().Unix())
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

func parseYearRange(s string) [2]int {
	parts := strings.SplitN(s, "-", 2)
	var yr [2]int
	if len(parts) == 2 {
		fmt.Sscanf(parts[0], "%d", &yr[0])
		fmt.Sscanf(parts[1], "%d", &yr[1])
	}
	if yr[0] == 0 {
		yr[0] = 2015
	}
	if yr[1] == 0 {
		yr[1] = 2026
	}
	return yr
}

func buildManualSources(sourcesList string) []manuals.ManualSource {
	enabled := make(map[string]bool)
	for _, s := range strings.Split(sourcesList, ",") {
		enabled[strings.TrimSpace(s)] = true
	}

	var srcs []manuals.ManualSource
	if enabled["toyota"] {
		srcs = append(srcs, manuals.NewToyotaSource())
	}
	if enabled["honda"] {
		srcs = append(srcs, manuals.NewHondaSource())
	}
	if enabled["ford"] {
		srcs = append(srcs, manuals.NewFordSource())
	}
	if enabled["archive"] {
		srcs = append(srcs, manuals.NewArchiveSource())
	}
	if enabled["nhtsa"] {
		srcs = append(srcs, manuals.NewNHTSASource())
	}
	if enabled["chevrolet"] {
		srcs = append(srcs, manuals.NewChevroletSource())
	}
	if enabled["gmc"] {
		srcs = append(srcs, manuals.NewGMCSource())
	}
	if enabled["ram"] {
		srcs = append(srcs, manuals.NewRamSource())
	}
	if enabled["jeep"] {
		srcs = append(srcs, manuals.NewJeepSource())
	}
	if enabled["dodge"] {
		srcs = append(srcs, manuals.NewDodgeSource())
	}
	if enabled["chrysler"] {
		srcs = append(srcs, manuals.NewChryslerSource())
	}
	if enabled["nissan"] {
		srcs = append(srcs, manuals.NewNissanSource())
	}
	if enabled["hyundai"] {
		srcs = append(srcs, manuals.NewHyundaiSource())
	}
	if enabled["kia"] {
		srcs = append(srcs, manuals.NewKiaSource())
	}
	if enabled["subaru"] {
		srcs = append(srcs, manuals.NewSubaruSource())
	}
	if enabled["mazda"] {
		srcs = append(srcs, manuals.NewMazdaSource())
	}
	if enabled["volkswagen"] {
		srcs = append(srcs, manuals.NewVolkswagenSource())
	}
	if enabled["bmw"] {
		srcs = append(srcs, manuals.NewBMWSource())
	}
	if enabled["mercedes"] {
		srcs = append(srcs, manuals.NewMercedesBenzSource())
	}
	if enabled["audi"] {
		srcs = append(srcs, manuals.NewAudiSource())
	}
	if enabled["tesla"] {
		srcs = append(srcs, manuals.NewTeslaSource())
	}
	if enabled["volvo"] {
		srcs = append(srcs, manuals.NewVolvoSource())
	}
	if enabled["lexus"] {
		srcs = append(srcs, manuals.NewLexusSource())
	}
	if enabled["acura"] {
		srcs = append(srcs, manuals.NewAcuraSource())
	}
	if enabled["infiniti"] {
		srcs = append(srcs, manuals.NewInfinitiSource())
	}
	if enabled["genesis"] {
		srcs = append(srcs, manuals.NewGenesisSource())
	}
	if enabled["porsche"] {
		srcs = append(srcs, manuals.NewPorscheSource())
	}
	if enabled["mitsubishi"] {
		srcs = append(srcs, manuals.NewMitsubishiSource())
	}
	if enabled["lincoln"] {
		srcs = append(srcs, manuals.NewLincolnSource())
	}
	if enabled["buick"] {
		srcs = append(srcs, manuals.NewBuickSource())
	}
	if enabled["cadillac"] {
		srcs = append(srcs, manuals.NewCadillacSource())
	}
	if enabled["search"] {
		srcs = append(srcs, manuals.NewGenericSearchSource())
	}
	return srcs
}
