package manuals

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/scraper"
)

// Scraper processes PDF vehicle manuals from a directory.
type Scraper struct {
	cfg Config
}

// NewScraper creates a new manual scraper with the given config.
func NewScraper(cfg Config) *Scraper {
	return &Scraper{cfg: cfg}
}

// FetchAll processes all PDF files in the configured directory.
func (s *Scraper) FetchAll(ctx context.Context) ([]scraper.ScrapedPost, error) {
	if s.cfg.Directory == "" {
		return nil, fmt.Errorf("manuals: directory not configured")
	}

	entries, err := os.ReadDir(s.cfg.Directory)
	if err != nil {
		return nil, fmt.Errorf("manuals: read dir: %w", err)
	}

	var posts []scraper.ScrapedPost
	count := 0

	for _, entry := range entries {
		if ctx.Err() != nil {
			break
		}
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".pdf") {
			continue
		}
		if s.cfg.MaxFiles > 0 && count >= s.cfg.MaxFiles {
			break
		}

		fullPath := filepath.Join(s.cfg.Directory, name)
		post, err := s.processFile(fullPath, name)
		if err != nil {
			log.Printf("manuals: skip %s: %v", name, err)
			continue
		}

		posts = append(posts, post)
		count++

		if s.cfg.RateLimit > 0 {
			time.Sleep(s.cfg.RateLimit)
		}
	}

	return posts, nil
}

func (s *Scraper) processFile(path, filename string) (scraper.ScrapedPost, error) {
	content, err := ExtractTextFromPDF(path)
	if err != nil {
		return scraper.ScrapedPost{}, err
	}
	if content == "" {
		return scraper.ScrapedPost{}, fmt.Errorf("no text extracted")
	}

	make, model, year := TagVehicleInfo(filename, content)

	var vi *scraper.VehicleInfo
	if make != "" {
		vi = &scraper.VehicleInfo{
			Make:  make,
			Model: model,
			Year:  year,
		}
	}

	vehicle := ""
	if make != "" && year > 0 {
		vehicle = fmt.Sprintf("%d-%s-%s", year, make, model)
	}

	return scraper.ScrapedPost{
		Source:   "manual",
		SourceID: filename,
		Title:    strings.TrimSuffix(filename, filepath.Ext(filename)),
		Content:  content,
		URL:      "file://" + path,
		ScrapedAt: time.Now(),
		Metadata: scraper.Metadata{
			Vehicle:     vehicle,
			VehicleInfo: vi,
			Keywords:    []string{"manual", "owner's manual"},
		},
	}, nil
}

// processFileMulti processes a PDF file and returns one ScrapedPost per detected section.
func (s *Scraper) processFileMulti(path, filename string) ([]scraper.ScrapedPost, error) {
	content, err := ExtractTextFromPDF(path)
	if err != nil {
		return nil, err
	}
	if content == "" {
		return nil, fmt.Errorf("no text extracted")
	}

	mk, mdl, year := TagVehicleInfo(filename, content)

	var vi *scraper.VehicleInfo
	if mk != "" {
		vi = &scraper.VehicleInfo{
			Make:  mk,
			Model: mdl,
			Year:  year,
		}
	}

	vehicle := ""
	if mk != "" && year > 0 {
		vehicle = fmt.Sprintf("%d-%s-%s", year, mk, mdl)
	}

	sections := ParseSections(content)
	if len(sections) == 0 {
		// Fallback to single post.
		post, err := s.processFile(path, filename)
		if err != nil {
			return nil, err
		}
		return []scraper.ScrapedPost{post}, nil
	}

	now := time.Now()
	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
	var posts []scraper.ScrapedPost

	for i, sec := range sections {
		keywords := []string{"manual", "owner's manual"}
		if sec.System != "" {
			keywords = append(keywords, strings.ToLower(sec.System))
		}
		if sec.Subsystem != "" {
			keywords = append(keywords, strings.ToLower(sec.Subsystem))
		}

		meta := scraper.Metadata{
			Vehicle:     vehicle,
			VehicleInfo: vi,
			Keywords:    keywords,
		}
		if sec.System != "" {
			meta.Section = sec.System
			if sec.Subsystem != "" {
				meta.Section = sec.System + "/" + sec.Subsystem
			}
		}

		posts = append(posts, scraper.ScrapedPost{
			Source:    "manual",
			SourceID:  fmt.Sprintf("%s-sec-%d", filename, i),
			Title:     fmt.Sprintf("%s - %s", baseName, sec.Title),
			Content:   sec.Content,
			URL:       "file://" + path,
			ScrapedAt: now,
			Metadata:  meta,
		})
	}

	return posts, nil
}

// FetchAllSections processes all PDFs and returns section-level posts.
func (s *Scraper) FetchAllSections(ctx context.Context) ([]scraper.ScrapedPost, error) {
	if s.cfg.Directory == "" {
		return nil, fmt.Errorf("manuals: directory not configured")
	}

	entries, err := os.ReadDir(s.cfg.Directory)
	if err != nil {
		return nil, fmt.Errorf("manuals: read dir: %w", err)
	}

	var posts []scraper.ScrapedPost
	count := 0

	for _, entry := range entries {
		if ctx.Err() != nil {
			break
		}
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".pdf") {
			continue
		}
		if s.cfg.MaxFiles > 0 && count >= s.cfg.MaxFiles {
			break
		}

		fullPath := filepath.Join(s.cfg.Directory, name)
		sectionPosts, err := s.processFileMulti(fullPath, name)
		if err != nil {
			log.Printf("manuals: skip %s: %v", name, err)
			continue
		}

		posts = append(posts, sectionPosts...)
		count++

		if s.cfg.RateLimit > 0 {
			time.Sleep(s.cfg.RateLimit)
		}
	}

	return posts, nil
}
