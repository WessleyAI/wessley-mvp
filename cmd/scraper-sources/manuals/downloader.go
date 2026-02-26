package manuals

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
)

// Downloader handles downloading and saving manual PDFs.
type Downloader struct {
	client      *http.Client
	outputDir   string
	maxFileSize int64
	userAgent   string
}

// NewDownloader creates a new Downloader.
func NewDownloader(client *http.Client, outputDir string, maxFileSize int64, userAgent string) *Downloader {
	return &Downloader{
		client:      client,
		outputDir:   outputDir,
		maxFileSize: maxFileSize,
		userAgent:   userAgent,
	}
}

// Download fetches a manual PDF and saves it to the organized directory structure.
// Returns the local file path.
func (d *Downloader) Download(ctx context.Context, entry graph.ManualEntry) (string, error) {
	// Build target path: {outputDir}/{make}/{model}/{year}/{manual_type}.pdf
	make_ := sanitizePath(entry.Make)
	model := sanitizePath(entry.Model)
	year := fmt.Sprintf("%d", entry.Year)
	manualType := entry.ManualType
	if manualType == "" {
		manualType = "owner"
	}

	dir := filepath.Join(d.outputDir, make_, model, year)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	// Use manual type + hash suffix to avoid collisions
	filename := fmt.Sprintf("%s_%s.pdf", manualType, entry.ID[:8])
	localPath := filepath.Join(dir, filename)

	// Check if already downloaded
	if _, err := os.Stat(localPath); err == nil {
		return localPath, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, entry.URL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", d.userAgent)

	// Support resume via Range header
	var existingSize int64
	tmpPath := localPath + ".tmp"
	if info, err := os.Stat(tmpPath); err == nil {
		existingSize = info.Size()
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existingSize))
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return "", fmt.Errorf("http status %d", resp.StatusCode)
	}

	// Check content length
	if d.maxFileSize > 0 && resp.ContentLength > d.maxFileSize {
		return "", fmt.Errorf("file too large: %d bytes", resp.ContentLength)
	}

	flags := os.O_CREATE | os.O_WRONLY
	if resp.StatusCode == http.StatusPartialContent {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	f, err := os.OpenFile(tmpPath, flags, 0o644)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}

	reader := io.Reader(resp.Body)
	if d.maxFileSize > 0 {
		reader = io.LimitReader(resp.Body, d.maxFileSize-existingSize)
	}

	n, err := io.Copy(f, reader)
	f.Close()
	if err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	// Verify it's a PDF (check magic bytes)
	if err := verifyPDF(tmpPath); err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	_ = n // total bytes written

	// Move tmp to final
	if err := os.Rename(tmpPath, localPath); err != nil {
		return "", fmt.Errorf("rename: %w", err)
	}

	return localPath, nil
}

// verifyPDF checks that the file starts with %PDF.
func verifyPDF(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	header := make([]byte, 5)
	n, err := f.Read(header)
	if err != nil || n < 4 {
		return fmt.Errorf("cannot read PDF header")
	}
	if string(header[:4]) != "%PDF" {
		return fmt.Errorf("not a valid PDF file")
	}
	return nil
}

func sanitizePath(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return -1
	}, s)
	return s
}
