package graph

import (
	"context"
	"time"
)

// MakeStats holds statistics about a vehicle make.
type MakeStats struct {
	Name      string `json:"name"`
	Models    int64  `json:"models"`
	Documents int64  `json:"documents"`
}

// VehicleStats holds statistics about a vehicle.
type VehicleStats struct {
	Vehicle    string `json:"vehicle"`
	Documents  int64  `json:"documents,omitempty"`
	Components int64  `json:"components,omitempty"`
	AddedAt    string `json:"added_at,omitempty"`
}

// NodeCounts returns node counts grouped by label.
func (g *GraphStore) NodeCounts(ctx context.Context) (map[string]int64, error) {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MATCH (n) RETURN labels(n)[0] AS type, count(*) AS count`
	result, err := sess.Run(ctx, cypher, nil)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int64)
	for result.Next(ctx) {
		rec := result.Record()
		typ, _ := rec.Get("type")
		cnt, _ := rec.Get("count")
		if t, ok := typ.(string); ok {
			if c, ok := cnt.(int64); ok {
				counts[t] = c
			}
		}
	}
	return counts, nil
}

// RelationshipCounts returns relationship counts grouped by type.
func (g *GraphStore) RelationshipCounts(ctx context.Context) (map[string]int64, error) {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MATCH ()-[r]->() RETURN type(r) AS type, count(*) AS count`
	result, err := sess.Run(ctx, cypher, nil)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int64)
	for result.Next(ctx) {
		rec := result.Record()
		typ, _ := rec.Get("type")
		cnt, _ := rec.Get("count")
		if t, ok := typ.(string); ok {
			if c, ok := cnt.(int64); ok {
				counts[t] = c
			}
		}
	}
	return counts, nil
}

// TopMakes returns the top makes by document count.
func (g *GraphStore) TopMakes(ctx context.Context, limit int) ([]MakeStats, error) {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MATCH (mk:Make)
		OPTIONAL MATCH (mk)-[:HAS_MODEL]->(m)
		OPTIONAL MATCH (d:Document)-[*]->(m)
		RETURN mk.name AS name, count(DISTINCT m) AS models, count(DISTINCT d) AS documents
		ORDER BY documents DESC LIMIT $limit`
	result, err := sess.Run(ctx, cypher, map[string]any{"limit": int64(limit)})
	if err != nil {
		return nil, err
	}
	var stats []MakeStats
	for result.Next(ctx) {
		rec := result.Record()
		name, _ := rec.Get("name")
		models, _ := rec.Get("models")
		docs, _ := rec.Get("documents")
		s := MakeStats{}
		if n, ok := name.(string); ok {
			s.Name = n
		}
		if m, ok := models.(int64); ok {
			s.Models = m
		}
		if d, ok := docs.(int64); ok {
			s.Documents = d
		}
		stats = append(stats, s)
	}
	return stats, nil
}

// TopVehicles returns the top vehicles by document count.
func (g *GraphStore) TopVehicles(ctx context.Context, limit int) ([]VehicleStats, error) {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MATCH (my:ModelYear)
		OPTIONAL MATCH (d:Document)-[:FOR_VEHICLE]->(my)
		OPTIONAL MATCH (c:Component)-[*]->(my)
		WITH my.year + ' ' + my.make + ' ' + my.model AS vehicle, 
		     count(DISTINCT d) AS documents, count(DISTINCT c) AS components
		WHERE documents > 0
		RETURN vehicle, documents, components
		ORDER BY documents DESC LIMIT $limit`
	result, err := sess.Run(ctx, cypher, map[string]any{"limit": int64(limit)})
	if err != nil {
		return nil, err
	}
	var stats []VehicleStats
	for result.Next(ctx) {
		rec := result.Record()
		v, _ := rec.Get("vehicle")
		d, _ := rec.Get("documents")
		c, _ := rec.Get("components")
		s := VehicleStats{}
		if vs, ok := v.(string); ok {
			s.Vehicle = vs
		}
		if di, ok := d.(int64); ok {
			s.Documents = di
		}
		if ci, ok := c.(int64); ok {
			s.Components = ci
		}
		stats = append(stats, s)
	}
	return stats, nil
}

// RecentVehicles returns the most recently added vehicles.
func (g *GraphStore) RecentVehicles(ctx context.Context, limit int) ([]VehicleStats, error) {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MATCH (my:ModelYear)
		WHERE my.created_at IS NOT NULL
		WITH my.year + ' ' + my.make + ' ' + my.model AS vehicle, my.created_at AS added_at
		RETURN vehicle, added_at
		ORDER BY added_at DESC LIMIT $limit`
	result, err := sess.Run(ctx, cypher, map[string]any{"limit": int64(limit)})
	if err != nil {
		return nil, err
	}
	var stats []VehicleStats
	for result.Next(ctx) {
		rec := result.Record()
		v, _ := rec.Get("vehicle")
		a, _ := rec.Get("added_at")
		s := VehicleStats{}
		if vs, ok := v.(string); ok {
			s.Vehicle = vs
		}
		switch at := a.(type) {
		case string:
			s.AddedAt = at
		case time.Time:
			s.AddedAt = at.Format(time.RFC3339)
		}
		stats = append(stats, s)
	}
	return stats, nil
}
