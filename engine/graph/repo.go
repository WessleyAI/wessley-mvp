package graph

import (
	"github.com/WessleyAI/wessley-mvp/pkg/repo"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

// newComponentRepo creates a Neo4j-backed repository for Component nodes.
func newComponentRepo(driver neo4j.DriverWithContext) *repo.Neo4jRepo[Component, string] {
	return repo.NewNeo4jRepo[Component, string](
		driver,
		"Component",
		componentToMap,
		componentFromRecord,
	)
}

func componentToMap(c Component) map[string]any {
	m := map[string]any{
		"id":      c.ID,
		"name":    c.Name,
		"type":    c.Type,
		"vehicle": c.Vehicle,
	}
	for k, v := range c.Properties {
		m["prop_"+k] = v
	}
	return m
}

func componentFromRecord(rec *neo4j.Record) (Component, error) {
	node, _, err := neo4j.GetRecordValue[dbtype.Node](rec, "n")
	if err != nil {
		return Component{}, err
	}
	props := node.Props
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
	return c, nil
}

func strProp(props map[string]any, key string) string {
	if v, ok := props[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
