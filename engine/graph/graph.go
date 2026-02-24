package graph

import (
	"context"
	"fmt"

	"github.com/WessleyAI/wessley-mvp/pkg/repo"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

// cypher runner abstractions for testability.

// CypherResult abstracts iterating over query results.
type CypherResult interface {
	Next(ctx context.Context) bool
	Record() *neo4j.Record
}

// CypherRunner can execute a Cypher query.
type CypherRunner interface {
	Run(ctx context.Context, cypher string, params map[string]any) (CypherResult, error)
}

// CypherSession extends CypherRunner with Close and ExecuteWrite.
type CypherSession interface {
	CypherRunner
	Close(ctx context.Context) error
	ExecuteWrite(ctx context.Context, work func(tx CypherRunner) (any, error)) (any, error)
}

// SessionOpener creates sessions.
type SessionOpener interface {
	OpenSession(ctx context.Context) CypherSession
}

// neo4jSessionAdapter wraps neo4j.SessionWithContext.
type neo4jSessionAdapter struct {
	sess neo4j.SessionWithContext
}

func (a *neo4jSessionAdapter) Run(ctx context.Context, cypher string, params map[string]any) (CypherResult, error) {
	return a.sess.Run(ctx, cypher, params)
}
func (a *neo4jSessionAdapter) Close(ctx context.Context) error {
	return a.sess.Close(ctx)
}
func (a *neo4jSessionAdapter) ExecuteWrite(ctx context.Context, work func(tx CypherRunner) (any, error)) (any, error) {
	return a.sess.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return work(&neo4jTxAdapter{tx: tx})
	})
}

type neo4jTxAdapter struct {
	tx neo4j.ManagedTransaction
}

func (a *neo4jTxAdapter) Run(ctx context.Context, cypher string, params map[string]any) (CypherResult, error) {
	return a.tx.Run(ctx, cypher, params)
}

// neo4jDriverAdapter wraps neo4j.DriverWithContext as SessionOpener.
type neo4jDriverAdapter struct {
	driver neo4j.DriverWithContext
}

func (a *neo4jDriverAdapter) OpenSession(ctx context.Context) CypherSession {
	return &neo4jSessionAdapter{sess: a.driver.NewSession(ctx, neo4j.SessionConfig{})}
}

// GraphStore provides graph operations on top of the generic Neo4j repository.
type GraphStore struct {
	opener     SessionOpener
	components *repo.Neo4jRepo[Component, string]
}

// New creates a new GraphStore.
func New(driver neo4j.DriverWithContext) *GraphStore {
	return &GraphStore{
		opener:     &neo4jDriverAdapter{driver: driver},
		components: newComponentRepo(driver),
	}
}

// NewWithOpener creates a GraphStore with a custom SessionOpener (for testing).
func NewWithOpener(opener SessionOpener) *GraphStore {
	return &GraphStore{
		opener: opener,
	}
}

// GetComponent returns a component by ID.
func (g *GraphStore) GetComponent(ctx context.Context, id string) (Component, error) {
	if g.components != nil {
		return g.components.Get(ctx, id)
	}
	// Fallback for stores created with NewWithOpener (no repo).
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MATCH (n:Component {id: $id}) RETURN n`
	result, err := sess.Run(ctx, cypher, map[string]any{"id": id})
	if err != nil {
		return Component{}, err
	}
	if !result.Next(ctx) {
		return Component{}, fmt.Errorf("component %s not found", id)
	}
	nVal, ok := result.Record().Get("n")
	if !ok {
		return Component{}, fmt.Errorf("no n field in record")
	}
	node, ok := nVal.(dbtype.Node)
	if !ok {
		return Component{}, fmt.Errorf("unexpected type for n")
	}
	return componentFromProps(node.Props), nil
}

// SaveComponent creates or updates a component node.
func (g *GraphStore) SaveComponent(ctx context.Context, c Component) error {
	sess := g.opener.OpenSession(ctx)
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
	sess := g.opener.OpenSession(ctx)
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
	sess := g.opener.OpenSession(ctx)
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
	sess := g.opener.OpenSession(ctx)
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
	sess := g.opener.OpenSession(ctx)
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
	sess := g.opener.OpenSession(ctx)
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
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	_, err := sess.ExecuteWrite(ctx, func(tx CypherRunner) (any, error) {
		for _, c := range components {
			cypher := `MERGE (n:Component {id: $id}) SET n += $props`
			if _, err := tx.Run(ctx, cypher, map[string]any{
				"id":    c.ID,
				"props": componentToMap(c),
			}); err != nil {
				return nil, err
			}
		}
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
func collectComponents(ctx context.Context, result CypherResult) ([]Component, error) {
	var items []Component
	for result.Next(ctx) {
		nVal, ok := result.Record().Get("n")
		if !ok {
			continue
		}
		node, ok := nVal.(dbtype.Node)
		if !ok {
			continue
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
	for i := range safe {
		if safe[i] >= 'a' && safe[i] <= 'z' {
			safe[i] -= 32
		}
	}
	return string(safe)
}
