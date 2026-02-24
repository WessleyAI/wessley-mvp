package graph

import (
	"context"
	"fmt"

	"github.com/WessleyAI/wessley-mvp/pkg/repo"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

// GraphStore provides graph operations on top of the generic Neo4j repository.
type GraphStore struct {
	driver     neo4j.DriverWithContext
	components *repo.Neo4jRepo[Component, string]
}

// New creates a new GraphStore.
func New(driver neo4j.DriverWithContext) *GraphStore {
	return &GraphStore{
		driver:     driver,
		components: newComponentRepo(driver),
	}
}

// GetComponent returns a component by ID.
func (g *GraphStore) GetComponent(ctx context.Context, id string) (Component, error) {
	return g.components.Get(ctx, id)
}

// SaveComponent creates or updates a component node.
func (g *GraphStore) SaveComponent(ctx context.Context, c Component) error {
	sess := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer sess.Close(ctx)

	cypher := `MERGE (n:Component {id: $id}) SET n += $props`
	_, err := sess.Run(ctx, cypher, map[string]any{
		"id":    c.ID,
		"props": componentToMap(c),
	})
	return err
}

// SaveEdge creates or updates an edge between two components.
func (g *GraphStore) SaveEdge(ctx context.Context, e Edge) error {
	sess := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer sess.Close(ctx)

	cypher := fmt.Sprintf(
		`MATCH (a:Component {id: $from}), (b:Component {id: $to})
		 MERGE (a)-[r:%s {id: $id}]->(b)
		 SET r.wire = $wire`,
		sanitizeRelType(e.Type),
	)
	_, err := sess.Run(ctx, cypher, map[string]any{
		"from": e.From,
		"to":   e.To,
		"id":   e.ID,
		"wire": e.Wire,
	})
	return err
}

// Neighbors returns components within the given traversal depth from a node.
func (g *GraphStore) Neighbors(ctx context.Context, nodeID string, depth int) ([]Component, error) {
	if depth <= 0 {
		depth = 1
	}
	sess := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer sess.Close(ctx)

	cypher := fmt.Sprintf(
		`MATCH (start:Component {id: $id})-[*1..%d]-(n:Component)
		 WHERE n.id <> $id
		 RETURN DISTINCT n`, depth)
	result, err := sess.Run(ctx, cypher, map[string]any{"id": nodeID})
	if err != nil {
		return nil, err
	}
	return collectComponents(ctx, result)
}

// FindByVehicle returns all components for a specific vehicle.
func (g *GraphStore) FindByVehicle(ctx context.Context, year int, make, model string) ([]Component, error) {
	sess := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer sess.Close(ctx)

	vehicleKey := fmt.Sprintf("%d-%s-%s", year, make, model)
	cypher := `MATCH (n:Component {vehicle: $vehicle}) RETURN n`
	result, err := sess.Run(ctx, cypher, map[string]any{"vehicle": vehicleKey})
	if err != nil {
		return nil, err
	}
	return collectComponents(ctx, result)
}

// FindByType returns all components of a given type.
func (g *GraphStore) FindByType(ctx context.Context, componentType string) ([]Component, error) {
	sess := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer sess.Close(ctx)

	cypher := `MATCH (n:Component {type: $type}) RETURN n`
	result, err := sess.Run(ctx, cypher, map[string]any{"type": componentType})
	if err != nil {
		return nil, err
	}
	return collectComponents(ctx, result)
}

// TracePath finds the shortest path between two components.
func (g *GraphStore) TracePath(ctx context.Context, fromID, toID string) ([]Component, error) {
	sess := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer sess.Close(ctx)

	cypher := `MATCH p = shortestPath((a:Component {id: $from})-[*]-(b:Component {id: $to}))
				RETURN nodes(p) AS nodes`
	result, err := sess.Run(ctx, cypher, map[string]any{"from": fromID, "to": toID})
	if err != nil {
		return nil, err
	}
	if !result.Next(ctx) {
		return nil, fmt.Errorf("no path from %s to %s", fromID, toID)
	}

	nodesVal, ok := result.Record().Get("nodes")
	if !ok {
		return nil, fmt.Errorf("no nodes in path result")
	}
	nodeList, ok := nodesVal.([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected nodes type")
	}

	var components []Component
	for _, raw := range nodeList {
		node, ok := raw.(dbtype.Node)
		if !ok {
			continue
		}
		components = append(components, componentFromProps(node.Props))
	}
	return components, nil
}

// SaveBatch saves multiple components and edges in a single transaction.
func (g *GraphStore) SaveBatch(ctx context.Context, components []Component, edges []Edge) error {
	sess := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer sess.Close(ctx)

	_, err := sess.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Batch create/merge components
		for _, c := range components {
			cypher := `MERGE (n:Component {id: $id}) SET n += $props`
			if _, err := tx.Run(ctx, cypher, map[string]any{
				"id":    c.ID,
				"props": componentToMap(c),
			}); err != nil {
				return nil, err
			}
		}
		// Batch create/merge edges
		for _, e := range edges {
			cypher := fmt.Sprintf(
				`MATCH (a:Component {id: $from}), (b:Component {id: $to})
				 MERGE (a)-[r:%s {id: $id}]->(b)
				 SET r.wire = $wire`,
				sanitizeRelType(e.Type),
			)
			if _, err := tx.Run(ctx, cypher, map[string]any{
				"from": e.From,
				"to":   e.To,
				"id":   e.ID,
				"wire": e.Wire,
			}); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return err
}

// collectComponents reads all Component nodes from a result set.
func collectComponents(ctx context.Context, result neo4j.ResultWithContext) ([]Component, error) {
	var items []Component
	for result.Next(ctx) {
		node, _, err := neo4j.GetRecordValue[dbtype.Node](result.Record(), "n")
		if err != nil {
			return nil, err
		}
		items = append(items, componentFromProps(node.Props))
	}
	return items, nil
}

// componentFromProps constructs a Component from Neo4j node properties.
func componentFromProps(props map[string]any) Component {
	c := Component{
		ID:         strProp(props, "id"),
		Name:       strProp(props, "name"),
		Type:       strProp(props, "type"),
		Vehicle:    strProp(props, "vehicle"),
		Properties: make(map[string]string),
	}
	for k, v := range props {
		if len(k) > 5 && k[:5] == "prop_" {
			if s, ok := v.(string); ok {
				c.Properties[k[5:]] = s
			}
		}
	}
	return c
}

// sanitizeRelType ensures the relationship type is a valid Cypher identifier.
func sanitizeRelType(t string) string {
	safe := make([]byte, 0, len(t))
	for i := range t {
		c := t[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			safe = append(safe, c)
		}
	}
	if len(safe) == 0 {
		return "RELATED_TO"
	}
	// Uppercase for Neo4j convention
	for i := range safe {
		if safe[i] >= 'a' && safe[i] <= 'z' {
			safe[i] -= 32
		}
	}
	return string(safe)
}
