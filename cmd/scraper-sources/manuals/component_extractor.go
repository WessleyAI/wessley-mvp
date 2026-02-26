package manuals

import (
	"regexp"
	"strings"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
)

// Part number patterns: alphanumeric with dashes (e.g., "28100-0V030", "YL3Z-13008-AA").
var partNumberPattern = regexp.MustCompile(`\b([A-Z0-9]{2,6}[-][A-Z0-9]{3,6}(?:[-][A-Z0-9]{2,4})?)\b`)

// Spec patterns.
var specPatterns = map[string]*regexp.Regexp{
	"voltage":    regexp.MustCompile(`(\d+(?:\.\d+)?)\s*[Vv](?:olts?)?\b`),
	"amperage":   regexp.MustCompile(`(\d+(?:\.\d+)?)\s*[Aa](?:mps?|mperes?)?\b`),
	"resistance": regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:[Oo]hms?|Ω)\b`),
	"gap":        regexp.MustCompile(`(\d+(?:\.\d+)?(?:\s*[-–]\s*\d+(?:\.\d+)?)?)\s*mm\s+gap\b`),
	"torque":     regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:ft[.-]?lbs?|Nm|lb[.-]?ft)\b`),
	"pressure":   regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:psi|kPa|bar)\b`),
}

// Component name patterns.
var componentNamePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(starter\s+motor)\b`),
	regexp.MustCompile(`(?i)\b(alternator)\b`),
	regexp.MustCompile(`(?i)\b(water\s+pump)\b`),
	regexp.MustCompile(`(?i)\b(fuel\s+pump)\b`),
	regexp.MustCompile(`(?i)\b(fuel\s+filter)\b`),
	regexp.MustCompile(`(?i)\b(fuel\s+injector)\b`),
	regexp.MustCompile(`(?i)\b(oil\s+pump)\b`),
	regexp.MustCompile(`(?i)\b(oil\s+filter)\b`),
	regexp.MustCompile(`(?i)\b(spark\s+plug)\b`),
	regexp.MustCompile(`(?i)\b(ignition\s+coil)\b`),
	regexp.MustCompile(`(?i)\b(catalytic\s+converter)\b`),
	regexp.MustCompile(`(?i)\b(oxygen\s+sensor)\b`),
	regexp.MustCompile(`(?i)\b(o2\s+sensor)\b`),
	regexp.MustCompile(`(?i)\b(mass\s+air\s+flow\s+sensor)\b`),
	regexp.MustCompile(`(?i)\b(throttle\s+body)\b`),
	regexp.MustCompile(`(?i)\b(idle\s+air\s+control)\b`),
	regexp.MustCompile(`(?i)\b(power\s+steering\s+pump)\b`),
	regexp.MustCompile(`(?i)\b(brake\s+caliper)\b`),
	regexp.MustCompile(`(?i)\b(brake\s+rotor)\b`),
	regexp.MustCompile(`(?i)\b(master\s+cylinder)\b`),
	regexp.MustCompile(`(?i)\b(wheel\s+bearing)\b`),
	regexp.MustCompile(`(?i)\b(control\s+arm)\b`),
	regexp.MustCompile(`(?i)\b(tie\s+rod)\b`),
	regexp.MustCompile(`(?i)\b(ball\s+joint)\b`),
	regexp.MustCompile(`(?i)\b(cv\s+joint)\b`),
	regexp.MustCompile(`(?i)\b(drive\s+shaft)\b`),
	regexp.MustCompile(`(?i)\b(blower\s+motor)\b`),
	regexp.MustCompile(`(?i)\b(heater\s+core)\b`),
	regexp.MustCompile(`(?i)\b(compressor)\b`),
	regexp.MustCompile(`(?i)\b(radiator)\b`),
	regexp.MustCompile(`(?i)\b(thermostat)\b`),
}

// Fuse/relay patterns.
var fusePattern = regexp.MustCompile(`(?i)\b(fuse\s*#?\d+)\b`)
var relayPattern = regexp.MustCompile(`(?i)\b(relay\s*[A-Z]?\d+)\b`)

// Wire/connector patterns.
var connectorPattern = regexp.MustCompile(`(?i)\b(connector\s+[A-Z]?\d+)\b`)
var wirePattern = regexp.MustCompile(`(?i)\b(wire\s+[A-Z]{2,3}(?:/[A-Z]{2,3})?)\b`)
var pinPattern = regexp.MustCompile(`(?i)\b(pin\s+\d+)\b`)

// ExtractComponents extracts components from section text.
func ExtractComponents(text string) []graph.ExtractedComponent {
	if text == "" {
		return nil
	}

	seen := make(map[string]bool)
	var components []graph.ExtractedComponent

	addComponent := func(name, partNum, desc string) {
		key := strings.ToLower(name)
		if seen[key] {
			return
		}
		seen[key] = true
		specs := extractSpecs(text)
		components = append(components, graph.ExtractedComponent{
			Name:        normalizeComponentName(name),
			PartNumber:  partNum,
			Description: desc,
			Specs:       specs,
		})
	}

	// Extract named components.
	for _, pat := range componentNamePatterns {
		matches := pat.FindAllString(text, -1)
		for _, m := range matches {
			// Look for an associated part number nearby.
			pn := findNearbyPartNumber(text, m)
			addComponent(m, pn, "")
		}
	}

	// Extract fuses.
	for _, m := range fusePattern.FindAllString(text, -1) {
		addComponent(m, "", "")
	}

	// Extract relays.
	for _, m := range relayPattern.FindAllString(text, -1) {
		addComponent(m, "", "")
	}

	// Extract connectors.
	for _, m := range connectorPattern.FindAllString(text, -1) {
		addComponent(m, "", "")
	}

	return components
}

// extractSpecs finds spec values in text.
func extractSpecs(text string) map[string]string {
	specs := make(map[string]string)
	for name, pat := range specPatterns {
		if m := pat.FindStringSubmatch(text); len(m) > 1 {
			specs[name] = m[0]
		}
	}
	return specs
}

// findNearbyPartNumber looks for a part number near a component mention.
func findNearbyPartNumber(text, component string) string {
	idx := strings.Index(strings.ToLower(text), strings.ToLower(component))
	if idx < 0 {
		return ""
	}
	// Search within 200 chars after the component name.
	end := idx + len(component) + 200
	if end > len(text) {
		end = len(text)
	}
	nearby := text[idx:end]
	if m := partNumberPattern.FindString(nearby); m != "" {
		return m
	}
	return ""
}

// normalizeComponentName cleans up and title-cases a component name.
func normalizeComponentName(name string) string {
	name = strings.TrimSpace(name)
	words := strings.Fields(strings.ToLower(name))
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
