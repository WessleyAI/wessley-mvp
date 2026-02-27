package manuals

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

// ArchiveSource discovers vehicle manuals from Archive.org.
type ArchiveSource struct {
	client *http.Client
}

func NewArchiveSource() *ArchiveSource {
	return &ArchiveSource{client: &http.Client{Timeout: 60 * time.Second}}
}

func (s *ArchiveSource) Name() string { return "archive" }

func (s *ArchiveSource) Discover(ctx context.Context, makes []string, years []int) ([]graph.ManualEntry, error) {
	var entries []graph.ManualEntry

	var queries []string
	// Per-make queries that actually return results on Archive.org
	for _, make_ := range makes {
		queries = append(queries,
			fmt.Sprintf(`"%s" "repair manual" AND mediatype:texts`, make_),
			fmt.Sprintf(`"%s" "service manual" AND mediatype:texts`, make_),
			fmt.Sprintf(`"%s" "owner manual" AND mediatype:texts`, make_),
		)
	}

	for _, q := range queries {
		select {
		case <-ctx.Done():
			return entries, ctx.Err()
		default:
		}

		found, err := s.searchArchive(ctx, q, makes)
		if err != nil {
			continue
		}
		entries = append(entries, found...)
		time.Sleep(2 * time.Second) // Rate limit
	}

	return dedup(entries), nil
}

type archiveResponse struct {
	Response struct {
		Docs []archiveDoc `json:"docs"`
	} `json:"response"`
}

type archiveDoc struct {
	Identifier string      `json:"identifier"`
	Title      string      `json:"title"`
	Subject    interface{} `json:"subject"` // can be string or []string
	Year       interface{} `json:"year"`
}

// archiveMetadataResponse represents the Archive.org metadata API response.
type archiveMetadataResponse struct {
	Files []archiveFile `json:"files"`
}

type archiveFile struct {
	Name   string `json:"name"`
	Format string `json:"format"`
	Size   string `json:"size"`
}

func (s *ArchiveSource) searchArchive(ctx context.Context, query string, makes []string) ([]graph.ManualEntry, error) {
	u := "https://archive.org/advancedsearch.php?" + url.Values{
		"q":         {query},
		"fl[]":      {"identifier,title,subject,year"},
		"output":    {"json"},
		"rows":      {"100"},
		"page":      {"1"},
		"sort[]":    {"downloads desc"},
	}.Encode()

	result := fn.Retry(ctx, fn.RetryOpts{
		MaxAttempts: 2,
		InitialWait: 2 * time.Second,
		MaxWait:     10 * time.Second,
	}, func(ctx context.Context) fn.Result[[]byte] {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return fn.Err[[]byte](err)
		}
		req.Header.Set("User-Agent", "WessleyBot/1.0")
		resp, err := s.client.Do(req)
		if err != nil {
			return fn.Err[[]byte](err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fn.Errf[[]byte]("status %d", resp.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
		if err != nil {
			return fn.Err[[]byte](err)
		}
		return fn.Ok(body)
	})

	body, err := result.Unwrap()
	if err != nil {
		return nil, err
	}

	var resp archiveResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse archive response: %w", err)
	}

	var entries []graph.ManualEntry
	for _, doc := range resp.Response.Docs {
		select {
		case <-ctx.Done():
			return entries, ctx.Err()
		default:
		}

		// Fetch metadata to find actual PDF files in this item
		pdfURLs, err := s.findPDFsInItem(ctx, doc.Identifier)
		if err != nil || len(pdfURLs) == 0 {
			// Fallback: try the identifier-named PDF
			pdfURLs = []string{fmt.Sprintf("https://archive.org/download/%s/%s.pdf", doc.Identifier, doc.Identifier)}
		}

		make_, model, year := extractVehicleFromArchiveDoc(doc, makes)
		for _, pdfURL := range pdfURLs {
			entries = append(entries, graph.ManualEntry{
				ID:           graph.ManualEntryID(pdfURL),
				URL:          pdfURL,
				SourceSite:   "archive.org",
				Make:         make_,
				Model:        model,
				Year:         year,
				ManualType:   inferManualTypeFromTitle(doc.Title),
				Language:     "en",
				Status:       "discovered",
				DiscoveredAt: time.Now(),
			})
		}

		time.Sleep(1 * time.Second) // Rate limit metadata requests
	}
	return entries, nil
}

// findPDFsInItem fetches the Archive.org metadata API to find actual PDF files.
func (s *ArchiveSource) findPDFsInItem(ctx context.Context, identifier string) ([]string, error) {
	u := fmt.Sprintf("https://archive.org/metadata/%s/files", identifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "WessleyBot/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}

	var metaResp struct {
		Result []archiveFile `json:"result"`
	}
	if err := json.Unmarshal(body, &metaResp); err != nil {
		return nil, err
	}

	var urls []string
	for _, f := range metaResp.Result {
		if strings.HasSuffix(strings.ToLower(f.Name), ".pdf") {
			dlURL := fmt.Sprintf("https://archive.org/download/%s/%s",
				identifier, url.PathEscape(f.Name))
			urls = append(urls, dlURL)
		}
	}
	return urls, nil
}

var yearRegex = regexp.MustCompile(`\b(19[6-9]\d|20[0-2]\d)\b`)

func extractVehicleFromArchiveDoc(doc archiveDoc, makes []string) (make_, model string, year int) {
	title := strings.ToLower(doc.Title)
	subject := ""
	switch s := doc.Subject.(type) {
	case string:
		subject = strings.ToLower(s)
	case []interface{}:
		parts := make([]string, 0, len(s))
		for _, v := range s {
			if str, ok := v.(string); ok {
				parts = append(parts, str)
			}
		}
		subject = strings.ToLower(strings.Join(parts, " "))
	}
	combined := title + " " + subject

	// Extract make
	for _, m := range makes {
		if strings.Contains(combined, strings.ToLower(m)) {
			make_ = m
			break
		}
	}

	// Extract year from doc.Year or title
	switch y := doc.Year.(type) {
	case float64:
		year = int(y)
	case string:
		if m := yearRegex.FindString(y); m != "" {
			year, _ = strconv.Atoi(m)
		}
	}
	if year == 0 {
		if m := yearRegex.FindString(combined); m != "" {
			year, _ = strconv.Atoi(m)
		}
	}

	// Try to extract model from title
	if make_ != "" {
		model = extractModelFromText(combined, make_)
	}

	return
}

func extractModelFromText(text, make_ string) string {
	models := map[string][]string{
		"Toyota":    {"camry", "corolla", "rav4", "highlander", "tacoma", "tundra", "4runner", "prius", "sienna", "supra"},
		"Honda":     {"civic", "accord", "cr-v", "pilot", "odyssey", "fit", "hr-v", "ridgeline"},
		"Ford":      {"f-150", "escape", "explorer", "mustang", "bronco", "ranger", "edge", "expedition"},
		"Chevrolet": {"silverado", "equinox", "malibu", "traverse", "tahoe", "suburban", "colorado", "blazer"},
		"BMW":       {"3 series", "5 series", "x3", "x5", "x1", "7 series"},
		"Nissan":    {"altima", "sentra", "rogue", "pathfinder", "frontier", "titan", "maxima"},
	}

	for _, m := range models[make_] {
		if strings.Contains(text, m) {
			return normModel(m)
		}
	}
	return ""
}

func inferManualTypeFromTitle(title string) string {
	t := strings.ToLower(title)
	switch {
	case strings.Contains(t, "service") || strings.Contains(t, "repair"):
		return "service"
	case strings.Contains(t, "electrical") || strings.Contains(t, "wiring"):
		return "electrical"
	case strings.Contains(t, "body"):
		return "body_repair"
	case strings.Contains(t, "quick"):
		return "quick_reference"
	default:
		return "owner"
	}
}

func dedup(entries []graph.ManualEntry) []graph.ManualEntry {
	seen := make(map[string]bool, len(entries))
	result := make([]graph.ManualEntry, 0, len(entries))
	for _, e := range entries {
		if seen[e.ID] {
			continue
		}
		seen[e.ID] = true
		result = append(result, e)
	}
	return result
}
