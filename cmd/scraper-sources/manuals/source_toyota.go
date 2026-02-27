package manuals

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

// ToyotaSource discovers owner manuals from Toyota's website.
type ToyotaSource struct {
	client *http.Client
}

// NewToyotaSource creates a new ToyotaSource.
func NewToyotaSource() *ToyotaSource {
	return &ToyotaSource{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *ToyotaSource) Name() string { return "toyota" }

func (s *ToyotaSource) Discover(ctx context.Context, makes []string, years []int) ([]graph.ManualEntry, error) {
	// Only process if Toyota is in the makes list
	if !containsIgnoreCase(makes, "Toyota") {
		return nil, nil
	}

	var entries []graph.ManualEntry

	// Toyota models commonly available
	models := []string{
		"camry", "corolla", "rav4", "highlander", "tacoma",
		"tundra", "4runner", "prius", "sienna", "avalon",
		"venza", "supra", "gr86", "corolla-cross", "bz4x",
		"crown", "sequoia", "land-cruiser",
	}

	for _, year := range years {
		for _, model := range models {
			select {
			case <-ctx.Done():
				return entries, ctx.Err()
			default:
			}

			// Use Toyota's current owners manual portal URL pattern
			url := fmt.Sprintf("https://www.toyota.com/owners/resources/warranty-owners-manuals/%s/%d", model, year)
			entry := graph.ManualEntry{
				ID:           graph.ManualEntryID(url),
				URL:          url,
				SourceSite:   "toyota.com",
				Make:         "Toyota",
				Model:        normModel(model),
				Year:         year,
				ManualType:   "owner",
				Language:     "en",
				Status:       "discovered",
				DiscoveredAt: time.Now(),
			}
			entries = append(entries, entry)
		}

		// Also check the owners page for additional links
		pageURL := fmt.Sprintf("https://www.toyota.com/owners/resources/warranty-owners-manuals/%d", year)
		found, err := s.discoverFromPage(ctx, pageURL, year)
		if err != nil {
			log.Printf("toyota: page crawl %d: %v", year, err)
		} else {
			entries = append(entries, found...)
		}

		time.Sleep(time.Second) // Rate limit between years
	}

	return entries, nil
}

func (s *ToyotaSource) discoverFromPage(ctx context.Context, pageURL string, year int) ([]graph.ManualEntry, error) {
	result := fn.Retry(ctx, fn.RetryOpts{
		MaxAttempts: 2,
		InitialWait: time.Second,
		MaxWait:     5 * time.Second,
	}, func(ctx context.Context) fn.Result[[]byte] {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
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
		body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
		if err != nil {
			return fn.Err[[]byte](err)
		}
		return fn.Ok(body)
	})

	body, err := result.Unwrap()
	if err != nil {
		return nil, err
	}

	return extractToyotaPDFLinks(string(body), year), nil
}

var toyotaPDFRegex = regexp.MustCompile(`href="(https?://[^"]*\.pdf)"`)

func extractToyotaPDFLinks(html string, year int) []graph.ManualEntry {
	matches := toyotaPDFRegex.FindAllStringSubmatch(html, -1)
	var entries []graph.ManualEntry
	seen := make(map[string]bool)

	for _, m := range matches {
		url := m[1]
		if seen[url] {
			continue
		}
		seen[url] = true

		model := inferModelFromURL(url, "Toyota")
		entries = append(entries, graph.ManualEntry{
			ID:           graph.ManualEntryID(url),
			URL:          url,
			SourceSite:   "toyota.com",
			Make:         "Toyota",
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

func normModel(s string) string {
	s = strings.ReplaceAll(s, "-", " ")
	return strings.Title(s)
}

func containsIgnoreCase(list []string, target string) bool {
	for _, s := range list {
		if strings.EqualFold(s, target) {
			return true
		}
	}
	return false
}

func inferModelFromURL(url, make_ string) string {
	url = strings.ToLower(url)
	models := map[string][]string{
		"Toyota":        {"camry", "corolla", "rav4", "highlander", "tacoma", "tundra", "4runner", "prius", "sienna", "avalon", "venza", "supra", "gr86", "corolla-cross", "bz4x", "crown", "sequoia", "land-cruiser"},
		"Honda":         {"civic", "accord", "cr-v", "pilot", "odyssey", "hr-v", "ridgeline", "passport"},
		"Ford":          {"f-150", "escape", "explorer", "mustang", "bronco", "ranger", "edge", "expedition"},
		"Chevrolet":     {"silverado", "equinox", "traverse", "tahoe", "suburban", "colorado", "blazer", "trax", "malibu", "camaro", "bolt-ev", "bolt-euv", "trailblazer", "corvette"},
		"GMC":           {"sierra", "yukon", "canyon", "acadia", "terrain", "hummer-ev"},
		"Ram":           {"1500", "2500", "3500", "promaster"},
		"Jeep":          {"wrangler", "grand-cherokee", "cherokee", "gladiator", "compass", "renegade", "wagoneer"},
		"Dodge":         {"charger", "durango", "hornet"},
		"Chrysler":      {"pacifica", "300"},
		"Nissan":        {"altima", "sentra", "rogue", "pathfinder", "frontier", "titan", "maxima", "kicks", "ariya", "leaf", "versa", "murano"},
		"Hyundai":       {"elantra", "sonata", "tucson", "santa-fe", "kona", "palisade", "ioniq-5", "ioniq-6", "venue"},
		"Kia":           {"forte", "k5", "sportage", "telluride", "sorento", "carnival", "ev6", "ev9", "seltos", "soul"},
		"Subaru":        {"outback", "forester", "crosstrek", "wrx", "impreza", "ascent", "brz", "solterra"},
		"Mazda":         {"mazda3", "cx-5", "cx-30", "cx-50", "cx-90", "mx-5"},
		"Volkswagen":    {"golf", "jetta", "tiguan", "atlas", "id-4", "taos"},
		"BMW":           {"3-series", "5-series", "7-series", "x1", "x3", "x5", "x7", "m3", "m5", "ix", "i4"},
		"Mercedes-Benz": {"c-class", "e-class", "s-class", "glc", "gle", "gls", "eqs", "eqe"},
		"Audi":          {"a3", "a4", "a6", "a8", "q3", "q5", "q7", "q8", "e-tron"},
		"Tesla":         {"model-3", "model-y", "model-s", "model-x", "cybertruck"},
		"Volvo":         {"xc40", "xc60", "xc90", "s60", "v60", "ex30", "ex90"},
		"Lexus":         {"es", "is", "rx", "nx", "tx", "gx", "lx", "ux", "lc", "rz"},
		"Acura":         {"integra", "tlx", "mdx", "rdx", "zdx"},
		"Infiniti":      {"q50", "qx50", "qx55", "qx60", "qx80"},
		"Genesis":       {"g70", "g80", "g90", "gv60", "gv70", "gv80"},
		"Porsche":       {"911", "cayenne", "macan", "taycan", "panamera", "718"},
		"Mitsubishi":    {"outlander", "eclipse-cross", "mirage"},
		"Lincoln":       {"aviator", "corsair", "nautilus", "navigator"},
		"Buick":         {"enclave", "encore-gx", "envista", "envision"},
		"Cadillac":      {"escalade", "ct4", "ct5", "xt4", "xt5", "xt6", "lyriq"},
	}
	for _, m := range models[make_] {
		if strings.Contains(url, m) {
			return normModel(m)
		}
	}
	return ""
}

func inferManualType(url string) string {
	url = strings.ToLower(url)
	switch {
	case strings.Contains(url, "service"):
		return "service"
	case strings.Contains(url, "electric"):
		return "electrical"
	case strings.Contains(url, "body"):
		return "body_repair"
	case strings.Contains(url, "quick"):
		return "quick_reference"
	default:
		return "owner"
	}
}
