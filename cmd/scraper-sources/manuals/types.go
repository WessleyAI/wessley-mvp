// Package manuals provides a web crawler for vehicle owner's manuals (PDF).
package manuals

import "time"

// Config controls the manual crawler behavior.
type Config struct {
	// Directory is the local directory containing PDF manuals (legacy).
	Directory string
	// OutputDir is where to save downloaded PDFs.
	OutputDir string
	// RateLimit is the delay between processing files / HTTP requests.
	RateLimit time.Duration
	// MaxFiles is the maximum number of files to process (0 = unlimited).
	MaxFiles int
	// Makes is which makes to crawl (empty = all known).
	Makes []string
	// YearRange is the min/max year to crawl.
	YearRange [2]int
	// MaxPerSource limits discovered entries per source.
	MaxPerSource int
	// MaxFileSize is the max PDF size to download in bytes.
	MaxFileSize int64
	// Concurrency is the number of parallel downloads.
	Concurrency int
	// Sources lists which sources to enable: "toyota", "honda", "ford", "archive", "nhtsa", "search".
	Sources []string
	// UserAgent is the HTTP User-Agent string.
	UserAgent string
}

// DefaultConfig returns sensible defaults for the crawler.
func DefaultConfig() Config {
	return Config{
		OutputDir:    "./manuals",
		RateLimit:    2 * time.Second,
		MaxPerSource: 100,
		MaxFileSize:  200 * 1024 * 1024, // 200MB
		Concurrency:  3,
		YearRange:    [2]int{2015, 2026},
		UserAgent:    "WessleyBot/1.0 (automotive-manual-indexer)",
		Sources:      []string{"toyota", "honda", "ford", "archive", "nhtsa", "search"},
	}
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

// KnownMakes lists common vehicle manufacturers.
var KnownMakes = []string{
	"Toyota", "Honda", "Ford", "Chevrolet", "BMW", "Nissan",
	"Hyundai", "Kia", "Volkswagen", "Mercedes-Benz", "Audi",
	"Subaru", "Mazda", "Lexus", "Jeep", "Ram", "GMC",
	"Dodge", "Chrysler", "Buick", "Cadillac", "Acura",
	"Infiniti", "Volvo", "Land Rover", "Porsche", "Tesla",
}
