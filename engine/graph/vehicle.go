package graph

import (
	"context"
	"fmt"
	"strings"
)

// SaveMake creates or updates a Make node.
func (g *GraphStore) SaveMake(ctx context.Context, m Make) error {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MERGE (n:Make {id: $id}) SET n.name = $name`
	_, err := sess.Run(ctx, cypher, map[string]any{
		"id":   m.ID,
		"name": m.Name,
	})
	return err
}

// SaveVehicleModel creates or updates a VehicleModel node and links it to its Make.
func (g *GraphStore) SaveVehicleModel(ctx context.Context, m VehicleModel) error {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MERGE (n:VehicleModel {id: $id}) SET n.name = $name, n.make_id = $makeID
	           WITH n
	           MATCH (mk:Make {id: $makeID})
	           MERGE (mk)-[:HAS_MODEL]->(n)`
	_, err := sess.Run(ctx, cypher, map[string]any{
		"id":     m.ID,
		"name":   m.Name,
		"makeID": m.MakeID,
	})
	return err
}

// SaveGeneration creates or updates a Generation node and links it to its VehicleModel.
func (g *GraphStore) SaveGeneration(ctx context.Context, gen Generation) error {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MERGE (n:Generation {id: $id})
	           SET n.name = $name, n.platform = $platform, n.start_year = $startYear, n.end_year = $endYear, n.model_id = $modelID
	           WITH n
	           MATCH (m:VehicleModel {id: $modelID})
	           MERGE (m)-[:HAS_GENERATION]->(n)`
	_, err := sess.Run(ctx, cypher, map[string]any{
		"id":        gen.ID,
		"name":      gen.Name,
		"platform":  gen.Platform,
		"startYear": gen.StartYear,
		"endYear":   gen.EndYear,
		"modelID":   gen.ModelID,
	})
	return err
}

// SaveTrim creates or updates a Trim node and links it to its Generation.
func (g *GraphStore) SaveTrim(ctx context.Context, tr Trim) error {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MERGE (n:Trim {id: $id})
	           SET n.name = $name, n.engine = $engine, n.transmission = $transmission, n.drivetrain = $drivetrain, n.generation_id = $genID
	           WITH n
	           MATCH (g:Generation {id: $genID})
	           MERGE (g)-[:HAS_TRIM]->(n)`
	_, err := sess.Run(ctx, cypher, map[string]any{
		"id":           tr.ID,
		"name":         tr.Name,
		"engine":       tr.Engine,
		"transmission": tr.Transmission,
		"drivetrain":   tr.Drivetrain,
		"genID":        tr.GenerationID,
	})
	return err
}

// SaveModelYear creates or updates a ModelYear node and links it to its VehicleModel.
func (g *GraphStore) SaveModelYear(ctx context.Context, my ModelYear) error {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MERGE (n:ModelYear {id: $id})
	           SET n.year = $year, n.make = $make, n.model = $model, n.trim = $trim
	           WITH n
	           MATCH (m:VehicleModel {name: $model})
	           MERGE (n)-[:OF_MODEL]->(m)`
	_, err := sess.Run(ctx, cypher, map[string]any{
		"id":    my.ID,
		"year":  my.Year,
		"make":  my.Make,
		"model": my.Model,
		"trim":  my.Trim,
	})
	return err
}

// SaveSystem creates or updates a System node.
func (g *GraphStore) SaveSystem(ctx context.Context, s System) error {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MERGE (n:System {id: $id}) SET n.name = $name`
	_, err := sess.Run(ctx, cypher, map[string]any{
		"id":   s.ID,
		"name": s.Name,
	})
	return err
}

// SaveSubsystem creates or updates a Subsystem node and links it to its System.
func (g *GraphStore) SaveSubsystem(ctx context.Context, ss Subsystem) error {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MERGE (n:Subsystem {id: $id}) SET n.name = $name, n.system_id = $systemID
	           WITH n
	           MATCH (s:System {id: $systemID})
	           MERGE (s)-[:HAS_SUBSYSTEM]->(n)`
	_, err := sess.Run(ctx, cypher, map[string]any{
		"id":       ss.ID,
		"name":     ss.Name,
		"systemID": ss.SystemID,
	})
	return err
}

// EnsureVehicleHierarchy creates Make→VehicleModel→ModelYear in a single transaction.
func (g *GraphStore) EnsureVehicleHierarchy(ctx context.Context, vi VehicleInfo) error {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	makeID := strings.ToLower(vi.Make)
	modelID := fmt.Sprintf("%s-%s", makeID, strings.ToLower(strings.ReplaceAll(vi.Model, " ", "-")))
	myID := fmt.Sprintf("%s-%d", modelID, vi.Year)

	_, err := sess.ExecuteWrite(ctx, func(tx CypherRunner) (any, error) {
		// Create Make
		cypher := `MERGE (mk:Make {id: $id}) SET mk.name = $name`
		if _, err := tx.Run(ctx, cypher, map[string]any{"id": makeID, "name": vi.Make}); err != nil {
			return nil, err
		}

		// Create VehicleModel linked to Make
		cypher = `MERGE (m:VehicleModel {id: $id}) SET m.name = $name, m.make_id = $makeID
		          WITH m
		          MATCH (mk:Make {id: $makeID})
		          MERGE (mk)-[:HAS_MODEL]->(m)`
		if _, err := tx.Run(ctx, cypher, map[string]any{"id": modelID, "name": vi.Model, "makeID": makeID}); err != nil {
			return nil, err
		}

		// Create ModelYear linked to VehicleModel
		cypher = `MERGE (my:ModelYear {id: $id}) SET my.year = $year, my.make = $make, my.model = $model, my.trim = $trim
		          WITH my
		          MATCH (m:VehicleModel {id: $modelID})
		          MERGE (my)-[:OF_MODEL]->(m)`
		if _, err := tx.Run(ctx, cypher, map[string]any{
			"id": myID, "year": vi.Year, "make": vi.Make, "model": vi.Model, "trim": vi.Trim, "modelID": modelID,
		}); err != nil {
			return nil, err
		}

		return nil, nil
	})
	return err
}

// LinkComponentToVehicle creates a relationship between a component and a ModelYear.
func (g *GraphStore) LinkComponentToVehicle(ctx context.Context, componentID, modelYearID string) error {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MATCH (c:Component {id: $compID}), (my:ModelYear {id: $myID})
	           MERGE (my)-[:HAS_COMPONENT]->(c)`
	_, err := sess.Run(ctx, cypher, map[string]any{
		"compID": componentID,
		"myID":   modelYearID,
	})
	return err
}

// FindComponentsByVehicle returns components linked to a specific vehicle.
func (g *GraphStore) FindComponentsByVehicle(ctx context.Context, vi VehicleInfo) ([]Component, error) {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	makeID := strings.ToLower(vi.Make)
	modelID := fmt.Sprintf("%s-%s", makeID, strings.ToLower(strings.ReplaceAll(vi.Model, " ", "-")))
	myID := fmt.Sprintf("%s-%d", modelID, vi.Year)

	cypher := `MATCH (my:ModelYear {id: $myID})-[:HAS_SYSTEM]->(:System)-[:HAS_SUBSYSTEM]->(:Subsystem)-[:HAS_COMPONENT]->(c:Component)
	           RETURN DISTINCT c AS n`
	result, err := sess.Run(ctx, cypher, map[string]any{"myID": myID})
	if err != nil {
		return nil, err
	}
	return collectComponents(ctx, result)
}

// GetVehicleHierarchy returns the Make, VehicleModel, and ModelYear for a vehicle.
func (g *GraphStore) GetVehicleHierarchy(ctx context.Context, vi VehicleInfo) (Make, VehicleModel, ModelYear, error) {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	makeID := strings.ToLower(vi.Make)
	modelID := fmt.Sprintf("%s-%s", makeID, strings.ToLower(strings.ReplaceAll(vi.Model, " ", "-")))
	myID := fmt.Sprintf("%s-%d", modelID, vi.Year)

	cypher := `MATCH (mk:Make {id: $makeID})-[:HAS_MODEL]->(m:VehicleModel {id: $modelID})<-[:OF_MODEL]-(my:ModelYear {id: $myID})
	           RETURN mk, m, my`
	result, err := sess.Run(ctx, cypher, map[string]any{
		"makeID":  makeID,
		"modelID": modelID,
		"myID":    myID,
	})
	if err != nil {
		return Make{}, VehicleModel{}, ModelYear{}, err
	}
	if !result.Next(ctx) {
		return Make{}, VehicleModel{}, ModelYear{}, fmt.Errorf("vehicle hierarchy not found for %s %s %d", vi.Make, vi.Model, vi.Year)
	}

	rec := result.Record()
	mkVal, _ := rec.Get("mk")
	mVal, _ := rec.Get("m")
	myVal, _ := rec.Get("my")

	mk := Make{
		ID:   strFromNode(mkVal, "id"),
		Name: strFromNode(mkVal, "name"),
	}
	vm := VehicleModel{
		ID:     strFromNode(mVal, "id"),
		Name:   strFromNode(mVal, "name"),
		MakeID: strFromNode(mVal, "make_id"),
	}
	my := ModelYear{
		ID:    strFromNode(myVal, "id"),
		Year:  intFromNode(myVal, "year"),
		Make:  strFromNode(myVal, "make"),
		Model: strFromNode(myVal, "model"),
		Trim:  strFromNode(myVal, "trim"),
	}

	return mk, vm, my, nil
}

// strFromNode extracts a string property from a node-like interface.
func strFromNode(val any, key string) string {
	type propsHolder interface {
		GetProperties() map[string]any
	}
	if ph, ok := val.(propsHolder); ok {
		return strProp(ph.GetProperties(), key)
	}
	// Try map directly for test mocks.
	if m, ok := val.(map[string]any); ok {
		return strProp(m, key)
	}
	return ""
}

// intFromNode extracts an int property from a node-like interface.
func intFromNode(val any, key string) int {
	type propsHolder interface {
		GetProperties() map[string]any
	}
	var props map[string]any
	if ph, ok := val.(propsHolder); ok {
		props = ph.GetProperties()
	} else if m, ok := val.(map[string]any); ok {
		props = m
	}
	if props == nil {
		return 0
	}
	switch v := props[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}
