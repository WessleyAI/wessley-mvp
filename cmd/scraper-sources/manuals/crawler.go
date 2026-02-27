package manuals

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/engine/scraper"
)

// Crawler orchestrates discovery and downloading of vehicle manual PDFs.
type Crawler struct {
	sources []ManualSource
	graph   *graph.GraphStore
	cfg     CrawlerConfig
	client  *http.Client
}

// CrawlerConfig controls crawler behavior.
type CrawlerConfig struct {
	OutputDir    string
	MaxPerSource int
	RateLimit    time.Duration
	UserAgent    string
	MaxFileSize  int64
	Concurrency  int
	Makes        []string
	YearRange    [2]int
}

// NewCrawler creates a new Crawler with the given sources and config.
func NewCrawler(g *graph.GraphStore, cfg CrawlerConfig, sources ...ManualSource) *Crawler {
	return &Crawler{
		sources: sources,
		graph:   g,
		cfg:     cfg,
		client: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       cfg.Concurrency,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression: false,
			},
		},
	}
}

// Discover crawls all sources and registers found manuals in the graph.
// Does NOT download — just builds the index.
func (c *Crawler) Discover(ctx context.Context) (int, error) {
	years := makeYearRange(c.cfg.YearRange)
	makes := c.cfg.Makes
	if len(makes) == 0 {
		makes = KnownMakes
	}

	var totalDiscovered int
	for _, src := range c.sources {
		select {
		case <-ctx.Done():
			return totalDiscovered, ctx.Err()
		default:
		}

		log.Printf("manuals: discovering from %s...", src.Name())
		entries, err := src.Discover(ctx, makes, years)
		if err != nil {
			log.Printf("manuals: %s discover error: %v", src.Name(), err)
			continue
		}

		// Cap per source
		if c.cfg.MaxPerSource > 0 && len(entries) > c.cfg.MaxPerSource {
			entries = entries[:c.cfg.MaxPerSource]
		}

		saved := 0
		for _, entry := range entries {
			if entry.ID == "" {
				entry.ID = graph.ManualEntryID(entry.URL)
			}
			if entry.Status == "" {
				entry.Status = "discovered"
			}
			if entry.DiscoveredAt.IsZero() {
				entry.DiscoveredAt = time.Now()
			}
			if err := c.graph.SaveManualEntry(ctx, entry); err != nil {
				log.Printf("manuals: save entry error: %v", err)
				continue
			}
			saved++
		}
		log.Printf("manuals: %s discovered %d, saved %d", src.Name(), len(entries), saved)
		totalDiscovered += saved
	}
	return totalDiscovered, nil
}

// Download fetches pending manuals from the registry.
func (c *Crawler) Download(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 100
	}
	pending, err := c.graph.GetPendingDownloads(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("get pending downloads: %w", err)
	}

	dl := NewDownloader(c.client, c.cfg.OutputDir, c.cfg.MaxFileSize, c.cfg.UserAgent)

	var (
		downloaded int
		mu         sync.Mutex
		sem        = make(chan struct{}, c.cfg.Concurrency)
		wg         sync.WaitGroup
	)

	for _, entry := range pending {
		select {
		case <-ctx.Done():
			break
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(e graph.ManualEntry) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := c.graph.UpdateManualStatus(ctx, e.ID, "downloading", ""); err != nil {
				log.Printf("manuals: status update error: %v", err)
				return
			}

			localPath, err := dl.Download(ctx, e)
			if err != nil {
				log.Printf("manuals: download %s failed: %v", e.URL, err)
				_ = c.graph.UpdateManualStatus(ctx, e.ID, "failed", err.Error())
				return
			}

			now := time.Now()
			e.LocalPath = localPath
			e.DownloadedAt = &now
			e.Status = "downloaded"
			if err := c.graph.SaveManualEntry(ctx, e); err != nil {
				log.Printf("manuals: save downloaded entry error: %v", err)
				return
			}

			mu.Lock()
			downloaded++
			mu.Unlock()

			// Rate limit between downloads
			time.Sleep(c.cfg.RateLimit)
		}(entry)
	}

	wg.Wait()
	return downloaded, nil
}

// Ingest processes downloaded PDFs into JSON files for the ingest pipeline.
func (c *Crawler) Ingest(ctx context.Context, outputDir string, limit int) (int, error) {
	if limit <= 0 {
		limit = 100
	}
	pending, err := c.graph.GetPendingIngestion(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("get pending ingestion: %w", err)
	}

	if outputDir == "" {
		outputDir = c.cfg.OutputDir
	}

	var ingested int
	for _, entry := range pending {
		select {
		case <-ctx.Done():
			return ingested, ctx.Err()
		default:
		}

		if entry.LocalPath == "" {
			_ = c.graph.UpdateManualStatus(ctx, entry.ID, "failed", "no local path")
			continue
		}

		content, err := ExtractTextFromPDF(entry.LocalPath)
		if err != nil {
			log.Printf("manuals: extract text from %s: %v", entry.LocalPath, err)
			_ = c.graph.UpdateManualStatus(ctx, entry.ID, "failed", err.Error())
			continue
		}
		if strings.TrimSpace(content) == "" {
			_ = c.graph.UpdateManualStatus(ctx, entry.ID, "failed", "no text extracted")
			continue
		}

		sections := ParseSections(content)

		// Write JSON files for ingest pipeline
		baseName := strings.TrimSuffix(filepath.Base(entry.LocalPath), ".pdf")
		jsonDir := filepath.Join(outputDir, "json")
		if err := os.MkdirAll(jsonDir, 0o755); err != nil {
			log.Printf("manuals: mkdir %s: %v", jsonDir, err)
			continue
		}

		vehicle := ""
		if entry.Make != "" && entry.Year > 0 {
			vehicle = fmt.Sprintf("%d-%s-%s", entry.Year, entry.Make, entry.Model)
		}

		for i, sec := range sections {
			post := scraper.ScrapedPost{
				Source:    "manual",
				SourceID:  fmt.Sprintf("%s-sec-%d", baseName, i),
				Title:     fmt.Sprintf("%s - %s", baseName, sec.Title),
				Content:   sec.Content,
				URL:       entry.URL,
				ScrapedAt: time.Now(),
				Metadata: scraper.Metadata{
					Vehicle: vehicle,
					Keywords: []string{"manual", "owner's manual"},
				},
			}
			if entry.Make != "" {
				post.Metadata.VehicleInfo = &scraper.VehicleInfo{
					Make:  entry.Make,
					Model: entry.Model,
					Year:  entry.Year,
				}
			}
			if sec.System != "" {
				post.Metadata.Section = sec.System
				if sec.Subsystem != "" {
					post.Metadata.Section = sec.System + "/" + sec.Subsystem
				}
				post.Metadata.Keywords = append(post.Metadata.Keywords, strings.ToLower(sec.System))
			}

			jsonPath := filepath.Join(jsonDir, fmt.Sprintf("%s-sec-%d.json", baseName, i))
			data, err := json.Marshal(post)
			if err != nil {
				continue
			}
			if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
				log.Printf("manuals: write json %s: %v", jsonPath, err)
				continue
			}
		}

		now := time.Now()
		entry.IngestedAt = &now
		entry.Status = "ingested"
		entry.PageCount = len(sections)
		if err := c.graph.SaveManualEntry(ctx, entry); err != nil {
			log.Printf("manuals: save ingested entry error: %v", err)
			continue
		}
		ingested++
		log.Printf("manuals: ingested %s → %d sections", entry.LocalPath, len(sections))
	}
	return ingested, nil
}

// Process runs the full pipeline: discover → download → ingest.
func (c *Crawler) Process(ctx context.Context) error {
	discovered, err := c.Discover(ctx)
	if err != nil {
		return fmt.Errorf("discover: %w", err)
	}
	log.Printf("manuals: discovered %d total", discovered)

	downloaded, err := c.Download(ctx, 0)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	log.Printf("manuals: downloaded %d", downloaded)

	ingested, err := c.Ingest(ctx, "", 0)
	if err != nil {
		return fmt.Errorf("ingest: %w", err)
	}
	log.Printf("manuals: ingested %d", ingested)

	return nil
}

func makeYearRange(yr [2]int) []int {
	if yr[0] == 0 || yr[1] == 0 {
		yr = [2]int{2015, 2026}
	}
	var years []int
	for y := yr[0]; y <= yr[1]; y++ {
		years = append(years, y)
	}
	return years
}
