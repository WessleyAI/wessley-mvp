package manuals

import (
	"regexp"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
)

var pdfLinkRegex = regexp.MustCompile(`href="(https?://[^"]*\.pdf)"`)

func extractPDFLinks(html, sourceSite, make_ string, year int) []graph.ManualEntry {
	matches := pdfLinkRegex.FindAllStringSubmatch(html, -1)
	var entries []graph.ManualEntry
	seen := make(map[string]bool)

	for _, m := range matches {
		url := m[1]
		if seen[url] {
			continue
		}
		seen[url] = true

		model := inferModelFromURL(url, make_)
		entries = append(entries, graph.ManualEntry{
			ID:           graph.ManualEntryID(url),
			URL:          url,
			SourceSite:   sourceSite,
			Make:         make_,
			Model:        model,
			Year:         year,
			ManualType:   inferManualType(url),
			Language:     "en",
			Status:       "discovered",
			DiscoveredAt: time.Now(),
		})
	}
	return entries
}
