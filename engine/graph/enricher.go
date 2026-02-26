package graph

import (
	"context"
	"fmt"
	"strings"
)

// ManualSection represents a parsed section from a vehicle manual.
type ManualSection struct {
	Title      string
	Content    string
	PageRange  string // e.g. "45-52"
	System     string // classified system (may be empty if unclassified)
	Subsystem  string // classified subsystem
	Components []ExtractedComponent
}

// ExtractedComponent is a component found in manual text.
type ExtractedComponent struct {
	Name        string
	PartNumber  string            // if found
	Description string
	Specs       map[string]string // voltage, amperage, resistance, etc.
}

// Enricher builds vehicle-specific knowledge graph nodes from manual content.
type Enricher struct {
	graph *GraphStore
}

// NewEnricher creates a new Enricher.
func NewEnricher(gs *GraphStore) *Enricher {
	return &Enricher{graph: gs}
}

// EnrichFromManual processes extracted manual sections and builds the vehicle-specific graph.
// It creates ONLY the System/Subsystem/Component nodes that are evidenced in the sections.
func (e *Enricher) EnrichFromManual(ctx context.Context, vi VehicleInfo, sections []ManualSection) error {
	if len(sections) == 0 {
		return nil
	}

	// Ensure the vehicle hierarchy exists.
	if err := e.graph.EnsureVehicleHierarchy(ctx, vi); err != nil {
		return fmt.Errorf("enricher: vehicle hierarchy: %w", err)
	}

	vehiclePrefix := vehicleScopePrefix(vi)
	myID := modelYearID(vi)

	sess := e.graph.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	_, err := sess.ExecuteWrite(ctx, func(tx CypherRunner) (any, error) {
		for _, sec := range sections {
			sys := sec.System
			sub := sec.Subsystem

			// Classify if not already done.
			if sys == "" {
				sys, sub = ClassifySection(sec.Title, sec.Content)
			}
			if sys == "" {
				continue // unclassifiable section — skip
			}

			sysID := vehiclePrefix + ":" + sanitizeID(sys)

			// Create vehicle-scoped System node and link to ModelYear.
			cypher := `MERGE (s:System {id: $id}) SET s.name = $name
			           WITH s
			           MATCH (my:ModelYear {id: $myID})
			           MERGE (my)-[:HAS_SYSTEM]->(s)`
			if _, err := tx.Run(ctx, cypher, map[string]any{
				"id": sysID, "name": sys, "myID": myID,
			}); err != nil {
				return nil, err
			}

			// Create Subsystem if classified.
			subID := ""
			if sub != "" {
				subID = sysID + ":" + sanitizeID(sub)
				cypher = `MERGE (ss:Subsystem {id: $id}) SET ss.name = $name, ss.system_id = $sysID
				          WITH ss
				          MATCH (s:System {id: $sysID})
				          MERGE (s)-[:HAS_SUBSYSTEM]->(ss)`
				if _, err := tx.Run(ctx, cypher, map[string]any{
					"id": subID, "name": sub, "sysID": sysID,
				}); err != nil {
					return nil, err
				}
			}

			// Create Components and link them.
			for _, comp := range sec.Components {
				compID := vehiclePrefix + ":" + sanitizeID(comp.Name)
				if comp.PartNumber != "" {
					compID = vehiclePrefix + ":" + sanitizeID(comp.PartNumber)
				}

				props := map[string]any{
					"id":   compID,
					"name": comp.Name,
					"type": "component",
				}
				if comp.PartNumber != "" {
					props["part_number"] = comp.PartNumber
				}
				if comp.Description != "" {
					props["description"] = comp.Description
				}
				for k, v := range comp.Specs {
					props["spec_"+sanitizeID(k)] = v
				}

				cypher = `MERGE (c:Component {id: $id}) SET c += $props`
				if _, err := tx.Run(ctx, cypher, map[string]any{
					"id": compID, "props": props,
				}); err != nil {
					return nil, err
				}

				// Link component to subsystem or system.
				if subID != "" {
					cypher = `MATCH (ss:Subsystem {id: $ssID}), (c:Component {id: $cID})
					          MERGE (ss)-[:HAS_COMPONENT]->(c)`
					if _, err := tx.Run(ctx, cypher, map[string]any{
						"ssID": subID, "cID": compID,
					}); err != nil {
						return nil, err
					}
				} else {
					cypher = `MATCH (s:System {id: $sID}), (c:Component {id: $cID})
					          MERGE (s)-[:HAS_COMPONENT]->(c)`
					if _, err := tx.Run(ctx, cypher, map[string]any{
						"sID": sysID, "cID": compID,
					}); err != nil {
						return nil, err
					}
				}
			}
		}
		return nil, nil
	})
	return err
}

// EnrichFromSource classifies a source's component string and creates vehicle-scoped
// system/subsystem nodes. Used for NHTSA complaints, iFixit guides, etc.
func (e *Enricher) EnrichFromSource(ctx context.Context, vi VehicleInfo, componentStr, docID string) error {
	sys, sub := ClassifyComponent(componentStr, "")
	if sys == "" {
		return nil
	}

	vehiclePrefix := vehicleScopePrefix(vi)
	myID := modelYearID(vi)
	sysID := vehiclePrefix + ":" + sanitizeID(sys)

	sess := e.graph.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	_, err := sess.ExecuteWrite(ctx, func(tx CypherRunner) (any, error) {
		// Create System → link to ModelYear.
		cypher := `MERGE (s:System {id: $id}) SET s.name = $name
		           WITH s
		           MATCH (my:ModelYear {id: $myID})
		           MERGE (my)-[:HAS_SYSTEM]->(s)`
		if _, err := tx.Run(ctx, cypher, map[string]any{
			"id": sysID, "name": sys, "myID": myID,
		}); err != nil {
			return nil, err
		}

		targetID := sysID
		if sub != "" {
			subID := sysID + ":" + sanitizeID(sub)
			cypher = `MERGE (ss:Subsystem {id: $id}) SET ss.name = $name, ss.system_id = $sysID
			          WITH ss
			          MATCH (s:System {id: $sysID})
			          MERGE (s)-[:HAS_SUBSYSTEM]->(ss)`
			if _, err := tx.Run(ctx, cypher, map[string]any{
				"id": subID, "name": sub, "sysID": sysID,
			}); err != nil {
				return nil, err
			}
			targetID = subID
		}

		// Link document to system/subsystem.
		if docID != "" {
			cypher = `MATCH (t {id: $tID}), (d:Component {id: $dID})
			          MERGE (d)-[:DOCUMENTED_IN]->(t)`
			if _, err := tx.Run(ctx, cypher, map[string]any{
				"tID": targetID, "dID": docID,
			}); err != nil {
				return nil, err
			}
		}

		return nil, nil
	})
	return err
}

// vehicleScopePrefix returns "make-model-year" for scoped node IDs.
func vehicleScopePrefix(vi VehicleInfo) string {
	return fmt.Sprintf("%s-%s-%d",
		strings.ToLower(vi.Make),
		strings.ToLower(strings.ReplaceAll(vi.Model, " ", "-")),
		vi.Year,
	)
}

// modelYearID returns the ModelYear node ID.
func modelYearID(vi VehicleInfo) string {
	return fmt.Sprintf("%s-%s-%d",
		strings.ToLower(vi.Make),
		strings.ToLower(strings.ReplaceAll(vi.Model, " ", "-")),
		vi.Year,
	)
}
