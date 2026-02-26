package manuals

import (
	"regexp"
	"strings"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
)

// sectionHeaderPatterns detects common manual section headers.
var sectionHeaderPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?im)^(?:chapter|section)\s+\d+[.:]\s*(.+)$`),
	regexp.MustCompile(`(?im)^\d+\.\d+\s+(.+)$`),
	regexp.MustCompile(`(?im)^\d+\s+[-–—]\s+(.+)$`),
	regexp.MustCompile(`(?im)^([A-Z][A-Z\s/&]{3,})$`), // ALL CAPS headers
}

// knownSectionHeaders are common manual section titles (uppercase for matching).
var knownSectionHeaders = []string{
	"ENGINE", "ELECTRICAL SYSTEM", "ELECTRICAL", "BRAKES", "BRAKE SYSTEM",
	"SUSPENSION", "STEERING", "TRANSMISSION", "FUEL SYSTEM",
	"EXHAUST SYSTEM", "EXHAUST", "COOLING SYSTEM", "COOLING",
	"HVAC", "AIR CONDITIONING", "CLIMATE CONTROL",
	"BODY", "INTERIOR", "EXTERIOR",
	"SAFETY", "AIRBAGS", "RESTRAINT SYSTEM",
	"MAINTENANCE SCHEDULE", "MAINTENANCE", "SCHEDULED MAINTENANCE",
	"SPECIFICATIONS", "GENERAL INFORMATION",
	"STARTING AND DRIVING", "INSTRUMENTS AND CONTROLS",
	"LIGHTING", "AUDIO SYSTEM", "NAVIGATION",
	"TIRES AND WHEELS", "TOWING",
}

// pageBreakPattern detects page breaks in extracted PDF text.
var pageBreakPattern = regexp.MustCompile(`(?m)\f|(?:^-{3,}$)|(?:Page\s+\d+)`)

// ParseSections splits manual text into classified sections.
func ParseSections(text string) []graph.ManualSection {
	if text == "" {
		return nil
	}

	lines := strings.Split(text, "\n")
	var sections []graph.ManualSection
	var currentTitle string
	var currentLines []string
	currentPage := ""

	flush := func() {
		if currentTitle == "" && len(currentLines) == 0 {
			return
		}
		content := strings.TrimSpace(strings.Join(currentLines, "\n"))
		if content == "" && currentTitle == "" {
			return
		}
		title := currentTitle
		if title == "" {
			title = "Untitled Section"
		}
		sys, sub := graph.ClassifySection(title, content)
		components := ExtractComponents(content)
		sections = append(sections, graph.ManualSection{
			Title:      title,
			Content:    content,
			PageRange:  currentPage,
			System:     sys,
			Subsystem:  sub,
			Components: components,
		})
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for page breaks — update page tracking.
		if pageBreakPattern.MatchString(trimmed) {
			if m := regexp.MustCompile(`Page\s+(\d+)`).FindStringSubmatch(trimmed); len(m) > 1 {
				if currentPage == "" {
					currentPage = m[1]
				} else {
					currentPage = strings.Split(currentPage, "-")[0] + "-" + m[1]
				}
			}
			continue
		}

		// Check if this line is a section header.
		if isHeader := detectSectionHeader(trimmed); isHeader {
			flush()
			currentTitle = cleanHeaderTitle(trimmed)
			currentLines = nil
			currentPage = ""
			continue
		}

		currentLines = append(currentLines, line)
	}

	flush()

	// If no sections detected, return whole text as single section.
	if len(sections) == 0 && text != "" {
		sys, sub := graph.ClassifySection("", text)
		components := ExtractComponents(text)
		sections = []graph.ManualSection{{
			Title:      "Full Document",
			Content:    text,
			System:     sys,
			Subsystem:  sub,
			Components: components,
		}}
	}

	return sections
}

// detectSectionHeader checks if a line looks like a section header.
func detectSectionHeader(line string) bool {
	if line == "" {
		return false
	}

	// Check against patterns.
	for _, pat := range sectionHeaderPatterns {
		if pat.MatchString(line) {
			return true
		}
	}

	// Check against known section headers.
	upper := strings.ToUpper(strings.TrimSpace(line))
	for _, h := range knownSectionHeaders {
		if upper == h {
			return true
		}
	}

	return false
}

// cleanHeaderTitle normalizes a header title.
func cleanHeaderTitle(raw string) string {
	// Remove chapter/section prefixes.
	cleaned := regexp.MustCompile(`(?i)^(?:chapter|section)\s+\d+[.:]\s*`).ReplaceAllString(raw, "")
	cleaned = regexp.MustCompile(`^\d+\.\d+\s+`).ReplaceAllString(cleaned, "")
	cleaned = regexp.MustCompile(`^\d+\s+[-–—]\s+`).ReplaceAllString(cleaned, "")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return raw
	}
	// Title case if all caps.
	if cleaned == strings.ToUpper(cleaned) && len(cleaned) > 3 {
		words := strings.Fields(strings.ToLower(cleaned))
		for i, w := range words {
			if len(w) > 0 {
				words[i] = strings.ToUpper(w[:1]) + w[1:]
			}
		}
		return strings.Join(words, " ")
	}
	return cleaned
}
