// Command backfill links orphaned Component nodes to the vehicle hierarchy.
// It queries Neo4j for Component nodes with no relationships, parses vehicle
// info from their properties, and calls the enricher to create proper links.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	neo4jURL := envOr("NEO4J_URL", "neo4j://localhost:7687")
	neo4jUser := envOr("NEO4J_USER", "neo4j")
	neo4jPass := envOr("NEO4J_PASS", "wessley123")

	driver, err := neo4j.NewDriverWithContext(neo4jURL, neo4j.BasicAuth(neo4jUser, neo4jPass, ""))
	if err != nil {
		log.Fatalf("neo4j connect: %v", err)
	}
	defer driver.Close(ctx)

	gs := graph.New(driver)
	enricher := graph.NewEnricher(gs)

	// Fetch all orphaned Component nodes.
	sess := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer sess.Close(ctx)

	result, err := sess.Run(ctx,
		`MATCH (c:Component) WHERE NOT (c)--() RETURN c.id AS id, c.name AS name, c.vehicle AS vehicle, c.type AS type, c.prop_source AS source`,
		nil,
	)
	if err != nil {
		log.Fatalf("query orphans: %v", err)
	}

	type orphan struct {
		id, name, vehicle, typ, source string
	}
	var orphans []orphan
	for result.Next(ctx) {
		rec := result.Record()
		o := orphan{}
		if v, ok := rec.Get("id"); ok && v != nil {
			o.id = fmt.Sprint(v)
		}
		if v, ok := rec.Get("name"); ok && v != nil {
			o.name = fmt.Sprint(v)
		}
		if v, ok := rec.Get("vehicle"); ok && v != nil {
			o.vehicle = fmt.Sprint(v)
		}
		if v, ok := rec.Get("type"); ok && v != nil {
			o.typ = fmt.Sprint(v)
		}
		if v, ok := rec.Get("source"); ok && v != nil {
			o.source = fmt.Sprint(v)
		}
		orphans = append(orphans, o)
	}

	log.Printf("Found %d orphaned Component nodes", len(orphans))

	var linked, skipped, errors int

	for i, o := range orphans {
		vi, componentStr := parseVehicleInfo(o)
		if vi.Make == "" || vi.Model == "" || vi.Year == 0 {
			skipped++
			continue
		}

		// Create vehicle hierarchy.
		if err := gs.EnsureVehicleHierarchy(ctx, vi); err != nil {
			log.Printf("[%d] EnsureVehicleHierarchy error for %s: %v", i, o.id, err)
			errors++
			continue
		}

		// Enrich: classify component and link to system/subsystem.
		if err := enricher.EnrichFromSource(ctx, vi, componentStr, o.id); err != nil {
			log.Printf("[%d] EnrichFromSource error for %s: %v", i, o.id, err)
			errors++
			continue
		}

		linked++
		if linked%100 == 0 {
			log.Printf("Progress: %d linked, %d skipped, %d errors (of %d)", linked, skipped, errors, len(orphans))
		}
	}

	log.Printf("Done! Linked: %d, Skipped: %d, Errors: %d, Total: %d", linked, skipped, errors, len(orphans))

	// Verify new relationship count.
	r2, err := sess.Run(ctx, `MATCH ()-[r]->() RETURN count(r) AS cnt`, nil)
	if err == nil && r2.Next(ctx) {
		if v, ok := r2.Record().Get("cnt"); ok {
			log.Printf("Total relationships now: %v", v)
		}
	}

	r3, err := sess.Run(ctx, `MATCH (c:Component) WHERE NOT (c)--() RETURN count(c) AS cnt`, nil)
	if err == nil && r3.Next(ctx) {
		if v, ok := r3.Record().Get("cnt"); ok {
			log.Printf("Remaining orphaned components: %v", v)
		}
	}
}

// yearPattern matches "YYYY MAKE MODEL" in vehicle strings like "2024 TOYOTA VENZA".
var yearPattern = regexp.MustCompile(`^(\d{4})\s+(\S+)\s+(.+)$`)

// nhtsaNamePattern extracts component from NHTSA complaint names like
// "NHTSA Complaint: 2024 TOYOTA VENZA - AIR BAGS"
var nhtsaNamePattern = regexp.MustCompile(`-\s*(.+)$`)

// titleYearPattern finds year + make in free-text titles like "my 2018 Chevy equinox..."
var titleYearPattern = regexp.MustCompile(`(?i)\b(19[89]\d|20[0-2]\d)\b\s+(` + strings.Join(knownMakes, "|") + `)\s+(\w+(?:\s+\w+)?)`)

var knownMakes = []string{
	"toyota", "honda", "ford", "chevrolet", "chevy", "gmc", "ram", "dodge",
	"chrysler", "jeep", "nissan", "hyundai", "kia", "subaru", "mazda",
	"volkswagen", "vw", "bmw", "mercedes", "audi", "tesla", "volvo",
	"lexus", "acura", "infiniti", "genesis", "porsche", "mitsubishi",
	"lincoln", "buick", "cadillac", "renault", "fiat", "mini", "saturn",
	"pontiac", "oldsmobile", "mercury", "saab", "suzuki", "scion", "isuzu",
}

var makeNormalize = map[string]string{
	"chevy": "Chevrolet",
	"vw":    "Volkswagen",
}

func parseVehicleInfo(o struct {
	id, name, vehicle, typ, source string
}) (graph.VehicleInfo, string) {
	var vi graph.VehicleInfo
	componentStr := ""

	// Try structured vehicle field first (NHTSA format: "2024 TOYOTA VENZA").
	if o.vehicle != "" {
		if m := yearPattern.FindStringSubmatch(strings.TrimSpace(o.vehicle)); m != nil {
			year, _ := strconv.Atoi(m[1])
			vi = graph.VehicleInfo{
				Make:  titleCase(m[2]),
				Model: titleCase(m[3]),
				Year:  year,
			}
		}
	}

	// Extract component string from name.
	if o.source == "nhtsa" {
		if m := nhtsaNamePattern.FindStringSubmatch(o.name); m != nil {
			componentStr = strings.TrimSpace(m[1])
		}
	}

	// If no vehicle from structured field, try parsing from title (reddit posts).
	if vi.Make == "" && o.name != "" {
		if m := titleYearPattern.FindStringSubmatch(o.name); m != nil {
			year, _ := strconv.Atoi(m[1])
			make := titleCase(m[2])
			if normalized, ok := makeNormalize[strings.ToLower(m[2])]; ok {
				make = normalized
			}
			model := titleCase(m[3])
			// Clean trailing common words from model
			model = strings.TrimRight(model, " ")
			for _, suffix := range []string{" And", " With", " Has", " Is", " The", " My", " A"} {
				model = strings.TrimSuffix(model, suffix)
			}
			if model != "" {
				vi = graph.VehicleInfo{Make: make, Model: model, Year: year}
			}
		}
	}

	if componentStr == "" {
		componentStr = o.name
	}

	return vi, componentStr
}

func titleCase(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return s
	}
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
