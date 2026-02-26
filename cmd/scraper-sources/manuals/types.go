// Package manuals provides a scraper for vehicle owner's manuals (PDF).
package manuals

import "time"

// Config controls the manual scraper behavior.
type Config struct {
	// Directory is the local directory containing PDF manuals.
	Directory string
	// RateLimit is the delay between processing files.
	RateLimit time.Duration
	// MaxFiles is the maximum number of files to process (0 = unlimited).
	MaxFiles int
}

// ManualDoc represents a parsed vehicle manual document.
type ManualDoc struct {
	FilePath string
	FileName string
	Content  string
	Make     string
	Model    string
	Year     int
	Trim     string
}
