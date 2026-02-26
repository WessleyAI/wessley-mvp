package manuals

import (
	"regexp"
	"strconv"
	"strings"
)

// knownMakes for regex matching.
var knownMakes = []string{
	"Toyota", "Honda", "Ford", "Chevrolet", "BMW", "Mercedes",
	"Audi", "Nissan", "Hyundai", "Kia", "Volkswagen", "Subaru",
	"Mazda", "Jeep", "Ram", "GMC", "Dodge", "Lexus", "Acura",
	"Tesla", "Porsche", "Volvo", "Buick", "Cadillac", "Lincoln",
	"Infiniti", "Genesis", "Mitsubishi", "Chrysler",
}

// yearPattern matches 4-digit years in reasonable vehicle range.
var yearPattern = regexp.MustCompile(`(?:^|[\s_\-/.,;:()])((19[89]\d|20[0-2]\d))(?:$|[\s_\-/.,;:()])`)


// TagVehicleInfo extracts vehicle information from filename and content.
func TagVehicleInfo(filename, content string) (make, model string, year int) {
	// Try filename first (usually more reliable)
	make, model, year = extractFromText(filename)
	if make != "" && year > 0 {
		return
	}

	// Fall back to content (first 2000 chars)
	preview := content
	if len(preview) > 2000 {
		preview = preview[:2000]
	}
	m2, mod2, y2 := extractFromText(preview)
	if make == "" {
		make = m2
	}
	if model == "" {
		model = mod2
	}
	if year == 0 {
		year = y2
	}
	return
}

func extractFromText(text string) (make, model string, year int) {
	upper := strings.ToUpper(text)

	// Find make
	for _, m := range knownMakes {
		if strings.Contains(upper, strings.ToUpper(m)) {
			make = m
			break
		}
	}

	// Find year
	if matches := yearPattern.FindStringSubmatch(text); len(matches) > 1 {
		if y, err := strconv.Atoi(matches[1]); err == nil {
			year = y
		}
	}

	// Try to find model after make
	if make != "" {
		model = extractModel(upper, strings.ToUpper(make))
	}

	return
}

// extractModel tries to find a model name following the make name.
func extractModel(upper, upperMake string) string {
	idx := strings.Index(upper, upperMake)
	if idx < 0 {
		return ""
	}
	after := strings.TrimSpace(upper[idx+len(upperMake):])

	// Common models to look for
	models := []string{
		"CAMRY", "COROLLA", "RAV4", "HIGHLANDER", "TACOMA", "PRIUS",
		"CIVIC", "ACCORD", "CR-V", "PILOT", "ODYSSEY",
		"F-150", "MUSTANG", "EXPLORER", "ESCAPE", "RANGER", "BRONCO",
		"SILVERADO", "EQUINOX", "MALIBU", "TAHOE", "SUBURBAN",
		"3 SERIES", "5 SERIES", "X3", "X5", "M3",
		"ALTIMA", "SENTRA", "ROGUE", "PATHFINDER",
		"ELANTRA", "SONATA", "TUCSON", "SANTA FE",
		"FORTE", "K5", "SPORTAGE", "TELLURIDE",
		"GOLF", "JETTA", "TIGUAN", "ATLAS",
		"OUTBACK", "FORESTER", "CROSSTREK", "WRX",
		"WRANGLER", "GRAND CHEROKEE", "CHEROKEE",
		"MODEL 3", "MODEL Y", "MODEL S", "MODEL X",
	}

	for _, m := range models {
		if strings.HasPrefix(after, m) || strings.Contains(after[:min(len(after), 30)], m) {
			return titleCase(m)
		}
	}

	// Take first word after make as model guess
	fields := strings.Fields(after)
	if len(fields) > 0 && len(fields[0]) > 1 {
		return titleCase(fields[0])
	}

	return ""
}

func titleCase(s string) string {
	if s == "" {
		return ""
	}
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
