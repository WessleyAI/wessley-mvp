// Package vehiclenlp extracts vehicle Make/Model/Year from unstructured text
// using regex patterns and a comprehensive vehicle database. No external dependencies.
package vehiclenlp

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// VehicleMatch represents an extracted vehicle mention.
type VehicleMatch struct {
	Make       string  // e.g. "Honda"
	Model      string  // e.g. "Civic"
	Year       int     // e.g. 2019 (0 if not found)
	Confidence float64 // 0.0-1.0
	Span       string  // the matched text fragment
}

// makeAliases maps abbreviations/nicknames to canonical make names.
var makeAliases = map[string]string{
	"chevy":         "Chevrolet",
	"chevrolet":     "Chevrolet",
	"merc":          "Mercedes-Benz",
	"benz":          "Mercedes-Benz",
	"mercedes":      "Mercedes-Benz",
	"mercedes-benz": "Mercedes-Benz",
	"vw":            "Volkswagen",
	"volkswagen":    "Volkswagen",
	"toyota":        "Toyota",
	"honda":         "Honda",
	"ford":          "Ford",
	"bmw":           "BMW",
	"audi":          "Audi",
	"nissan":        "Nissan",
	"hyundai":       "Hyundai",
	"kia":           "Kia",
	"subaru":        "Subaru",
	"mazda":         "Mazda",
	"jeep":          "Jeep",
	"ram":           "Ram",
	"gmc":           "GMC",
	"dodge":         "Dodge",
	"lexus":         "Lexus",
	"acura":         "Acura",
	"tesla":         "Tesla",
	"porsche":       "Porsche",
	"volvo":         "Volvo",
	"buick":         "Buick",
	"cadillac":      "Cadillac",
	"lincoln":       "Lincoln",
	"infiniti":      "Infiniti",
	"genesis":       "Genesis",
	"mitsubishi":    "Mitsubishi",
	"chrysler":      "Chrysler",
	"land rover":    "Land Rover",
	"jaguar":        "Jaguar",
	"alfa romeo":    "Alfa Romeo",
	"fiat":          "Fiat",
	"mini":          "Mini",
	"rivian":        "Rivian",
	"lucid":         "Lucid",
	"polestar":      "Polestar",
}

// makeModels maps canonical make to list of models.
var makeModels = map[string][]string{
	"Toyota":       {"Camry", "Corolla", "RAV4", "Highlander", "Tacoma", "Tundra", "Prius", "4Runner", "Sienna", "Supra", "GR86", "Venza", "C-HR", "Sequoia", "Land Cruiser"},
	"Honda":        {"Civic", "Accord", "CR-V", "Pilot", "Odyssey", "HR-V", "Ridgeline", "Fit", "Passport", "Insight"},
	"Ford":         {"F-150", "F-250", "F-350", "Mustang", "Explorer", "Escape", "Ranger", "Bronco", "Edge", "Expedition", "Maverick", "Focus", "Fusion", "Fiesta", "Transit"},
	"Chevrolet":    {"Silverado", "Equinox", "Malibu", "Tahoe", "Suburban", "Camaro", "Colorado", "Traverse", "Blazer", "Bolt", "Impala", "Trax", "Cruze", "Spark"},
	"BMW":          {"3 Series", "5 Series", "7 Series", "X3", "X5", "X1", "X7", "M3", "M5", "i4", "iX", "4 Series", "2 Series", "X6"},
	"Mercedes-Benz": {"C-Class", "E-Class", "S-Class", "GLC", "GLE", "A-Class", "CLA", "GLA", "GLB", "GLS", "AMG GT", "EQS", "EQE"},
	"Audi":         {"A4", "A6", "A3", "Q5", "Q7", "Q3", "A5", "A8", "Q8", "e-tron", "RS5", "RS7", "S4", "TT"},
	"Nissan":       {"Altima", "Sentra", "Rogue", "Pathfinder", "Frontier", "Maxima", "Murano", "Titan", "Z", "Kicks", "Versa", "Armada", "Leaf"},
	"Hyundai":      {"Elantra", "Sonata", "Tucson", "Santa Fe", "Kona", "Palisade", "Ioniq 5", "Ioniq 6", "Venue", "Accent", "Santa Cruz"},
	"Kia":          {"Forte", "K5", "Sportage", "Telluride", "Sorento", "Seltos", "EV6", "EV9", "Soul", "Stinger", "Carnival", "Rio", "Niro"},
	"Volkswagen":   {"Golf", "Jetta", "Tiguan", "Atlas", "Passat", "Taos", "ID.4", "GTI", "Arteon", "Beetle"},
	"Subaru":       {"Outback", "Forester", "Crosstrek", "Impreza", "WRX", "Legacy", "Ascent", "BRZ", "Solterra"},
	"Mazda":        {"Mazda3", "Mazda6", "CX-5", "CX-9", "CX-30", "CX-50", "MX-5", "CX-90"},
	"Jeep":         {"Wrangler", "Grand Cherokee", "Cherokee", "Compass", "Renegade", "Gladiator", "Grand Wagoneer", "Wagoneer"},
	"Ram":          {"1500", "2500", "3500", "ProMaster"},
	"GMC":          {"Sierra", "Terrain", "Acadia", "Yukon", "Canyon", "Hummer EV"},
	"Dodge":        {"Charger", "Challenger", "Durango", "Hornet"},
	"Lexus":        {"RX", "ES", "NX", "IS", "GX", "LX", "UX", "LC", "LS", "RC"},
	"Acura":        {"TLX", "MDX", "RDX", "Integra", "ILX", "NSX"},
	"Tesla":        {"Model 3", "Model Y", "Model S", "Model X", "Cybertruck"},
	"Porsche":      {"911", "Cayenne", "Macan", "Taycan", "Panamera", "Boxster", "Cayman"},
	"Volvo":        {"XC90", "XC60", "XC40", "S60", "S90", "V60", "V90", "C40"},
	"Buick":        {"Enclave", "Encore", "Envision", "Regal", "LaCrosse"},
	"Cadillac":     {"Escalade", "CT5", "CT4", "XT5", "XT4", "XT6", "Lyriq"},
	"Lincoln":      {"Navigator", "Aviator", "Corsair", "Nautilus"},
	"Infiniti":     {"Q50", "Q60", "QX50", "QX60", "QX80"},
	"Genesis":      {"G70", "G80", "G90", "GV70", "GV80", "GV60"},
	"Mitsubishi":   {"Outlander", "Eclipse Cross", "Mirage", "Outlander Sport"},
	"Chrysler":     {"Pacifica", "300"},
	"Land Rover":   {"Range Rover", "Defender", "Discovery", "Range Rover Sport", "Evoque"},
	"Jaguar":       {"F-Pace", "E-Pace", "XF", "XE", "F-Type", "I-Pace"},
	"Alfa Romeo":   {"Giulia", "Stelvio", "Tonale"},
	"Fiat":         {"500", "500X"},
	"Mini":         {"Cooper", "Countryman", "Clubman"},
	"Rivian":       {"R1T", "R1S"},
	"Lucid":        {"Air"},
	"Polestar":     {"Polestar 2", "Polestar 3"},
}

// uniqueModels maps models that are distinctive enough to identify a make on their own.
var uniqueModels map[string]string // model_lower -> canonical make

// modelByMake maps make_lower -> model_lower -> canonical model
var modelByMake map[string]map[string]string

// allMakePatterns is a regex alternation of all make names/aliases, longest first.
var makeRe *regexp.Regexp

// yearRe matches 4-digit years or 'YY abbreviated years.
var yearFullRe = regexp.MustCompile(`\b((?:19|20)\d{2})\b`)
var yearAbbrRe = regexp.MustCompile(`'(\d{2})\b`)

func init() {
	uniqueModels = make(map[string]string)
	modelByMake = make(map[string]map[string]string)

	// Build model lookup maps
	modelCount := make(map[string]int) // how many makes have this model
	for make_, models := range makeModels {
		lower := strings.ToLower(make_)
		modelByMake[lower] = make(map[string]string)
		for _, m := range models {
			ml := strings.ToLower(m)
			modelByMake[lower][ml] = m
			modelCount[ml]++
		}
	}
	// Models unique to one make
	for make_, models := range makeModels {
		for _, m := range models {
			ml := strings.ToLower(m)
			if modelCount[ml] == 1 {
				uniqueModels[ml] = make_
			}
		}
	}

	// Build make regex (all aliases + canonical names), sorted longest first for greedy match
	var makeNames []string
	seen := make(map[string]bool)
	for alias := range makeAliases {
		if !seen[alias] {
			makeNames = append(makeNames, regexp.QuoteMeta(alias))
			seen[alias] = true
		}
	}
	// Sort by length descending for longest match
	for i := 0; i < len(makeNames); i++ {
		for j := i + 1; j < len(makeNames); j++ {
			if len(makeNames[j]) > len(makeNames[i]) {
				makeNames[i], makeNames[j] = makeNames[j], makeNames[i]
			}
		}
	}
	makeRe = regexp.MustCompile(`(?i)\b(` + strings.Join(makeNames, "|") + `)(?:'s)?\b`)
}

// Extract finds all vehicle mentions in text. Returns matches sorted by confidence.
func Extract(text string) []VehicleMatch {
	if text == "" {
		return nil
	}
	var matches []VehicleMatch

	// Strategy: find all make mentions, then look for adjacent year and model.
	makeLocs := makeRe.FindAllStringSubmatchIndex(text, -1)
	used := make(map[string]bool) // dedup by make+model+year

	for _, loc := range makeLocs {
		makeStr := text[loc[2]:loc[3]]
		canonical := makeAliases[strings.ToLower(makeStr)]
		if canonical == "" {
			continue
		}

		// Look for model after make (within ~40 chars)
		afterStart := loc[1]
		afterEnd := min(afterStart+40, len(text))
		after := text[afterStart:afterEnd]

		model, modelSpan := findModel(canonical, after)

		// Look for year before make or after model
		year := 0
		// Check before make (up to 10 chars back)
		beforeStart := max(0, loc[0]-10)
		before := text[beforeStart:loc[0]]
		year = findYear(before)

		// Check after model/make if no year found before
		if year == 0 {
			searchAfter := after
			if modelSpan > 0 {
				searchAfter = after[modelSpan:]
			}
			year = findYear(searchAfter)
		}

		// Check for abbreviated year before make
		if year == 0 {
			year = findAbbrYear(before)
		}

		conf := 0.0
		switch {
		case year > 0 && model != "":
			conf = 0.95
		case model != "" && year == 0:
			conf = 0.80
		case year > 0 && model == "":
			conf = 0.70
		default:
			conf = 0.60
		}

		// Build span
		spanStart := loc[0]
		if year > 0 {
			// Try to include year in span
			if beforeStart < spanStart {
				if idx := strings.Index(before, fmt.Sprintf("%d", year)); idx >= 0 {
					spanStart = beforeStart + idx
				}
			}
		}
		spanEnd := loc[1]
		if model != "" {
			spanEnd = afterStart + modelSpan
		}
		span := strings.TrimSpace(text[spanStart:min(spanEnd, len(text))])

		key := fmt.Sprintf("%s|%s|%d", canonical, model, year)
		if used[key] {
			continue
		}
		used[key] = true

		matches = append(matches, VehicleMatch{
			Make:       canonical,
			Model:      model,
			Year:       year,
			Confidence: conf,
			Span:       span,
		})
	}

	// Also check for standalone unique models not preceded by a make
	matches = append(matches, findStandaloneModels(text, used)...)

	// Sort by confidence descending
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Confidence > matches[i].Confidence {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	return matches
}

// ExtractBest returns the single highest-confidence match, or nil.
func ExtractBest(text string) *VehicleMatch {
	matches := Extract(text)
	if len(matches) == 0 {
		return nil
	}
	return &matches[0]
}

// findModel looks for a known model of the given make in the text fragment after the make name.
func findModel(make_, after string) (model string, spanEnd int) {
	makeLower := strings.ToLower(make_)
	models, ok := modelByMake[makeLower]
	if !ok {
		return "", 0
	}

	// Trim leading whitespace/punctuation
	trimmed := strings.TrimLeftFunc(after, func(r rune) bool {
		return unicode.IsSpace(r) || r == '\'' || r == 0x2019
	})
	offset := len(after) - len(trimmed)

	// Try longest models first (e.g. "Grand Cherokee" before "Cherokee")
	type modelEntry struct {
		lower, canonical string
	}
	var sorted []modelEntry
	for ml, mc := range models {
		sorted = append(sorted, modelEntry{ml, mc})
	}
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if len(sorted[j].lower) > len(sorted[i].lower) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	lowerTrimmed := strings.ToLower(trimmed)
	for _, entry := range sorted {
		if strings.HasPrefix(lowerTrimmed, entry.lower) {
			// Make sure it's a word boundary after the model
			endIdx := len(entry.lower)
			if endIdx < len(lowerTrimmed) {
				next := rune(lowerTrimmed[endIdx])
				if unicode.IsLetter(next) || unicode.IsDigit(next) {
					// Could be part of a longer word — check if it's a known continuation
					// Allow for trim levels like "Civic Si" but don't match partial words
					continue
				}
			}
			return entry.canonical, offset + endIdx
		}
	}
	return "", 0
}

func findYear(s string) int {
	m := yearFullRe.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	y, _ := strconv.Atoi(m[1])
	if y >= 1980 && y <= 2030 {
		return y
	}
	return 0
}

func findAbbrYear(s string) int {
	m := yearAbbrRe.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	yy, _ := strconv.Atoi(m[1])
	if yy >= 0 && yy <= 30 {
		return 2000 + yy
	}
	if yy >= 80 && yy <= 99 {
		return 1900 + yy
	}
	return 0
}

func findStandaloneModels(text string, used map[string]bool) []VehicleMatch {
	var matches []VehicleMatch
	lower := strings.ToLower(text)

	// Only check distinctive models (unique to one make)
	type candidate struct {
		model, make_ string
		idx          int
	}
	var candidates []candidate

	for modelLower, make_ := range uniqueModels {
		// Skip very short/ambiguous model names that would cause false positives
		if len(modelLower) < 2 {
			continue
		}
		// Skip "Z", "500" etc — too ambiguous standalone
		if len(modelLower) <= 2 && !strings.Contains(modelLower, "-") {
			continue
		}

		idx := strings.Index(lower, modelLower)
		if idx < 0 {
			continue
		}

		// Word boundary check
		if idx > 0 {
			prev := rune(lower[idx-1])
			if unicode.IsLetter(prev) || unicode.IsDigit(prev) {
				continue
			}
		}
		end := idx + len(modelLower)
		if end < len(lower) {
			next := rune(lower[end])
			if unicode.IsLetter(next) || unicode.IsDigit(next) {
				continue
			}
		}

		// Check not already matched with a make
		canonical := makeModels[make_]
		var modelCanonical string
		for _, cm := range canonical {
			if strings.EqualFold(cm, modelLower) {
				modelCanonical = cm
				break
			}
		}
		if modelCanonical == "" {
			modelCanonical = modelLower // fallback
		}

		key := fmt.Sprintf("%s|%s|%d", make_, modelCanonical, 0)
		if used[key] {
			continue
		}

		// Check for year near the model
		nearStart := max(0, idx-12)
		nearEnd := min(end+12, len(text))
		year := findYear(text[nearStart:nearEnd])
		if year == 0 {
			year = findAbbrYear(text[nearStart:idx])
		}

		conf := 0.50
		if year > 0 {
			conf = 0.75
			key = fmt.Sprintf("%s|%s|%d", make_, modelCanonical, year)
			if used[key] {
				continue
			}
		}

		used[key] = true
		span := strings.TrimSpace(text[nearStart:nearEnd])
		candidates = append(candidates, candidate{modelCanonical, make_, idx})
		matches = append(matches, VehicleMatch{
			Make:       make_,
			Model:      modelCanonical,
			Year:       year,
			Confidence: conf,
			Span:       span,
		})
	}

	return matches
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
