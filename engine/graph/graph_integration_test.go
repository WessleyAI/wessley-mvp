//go:build integration

package graph

import (
	"context"
	"os"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func testDriver(t *testing.T) neo4j.DriverWithContext {
	t.Helper()
	url := envOr("NEO4J_URL", "neo4j://localhost:7687")
	driver, err := neo4j.NewDriverWithContext(url, neo4j.NoAuth())
	if err != nil {
		t.Fatalf("neo4j connect: %v", err)
	}
	ctx := context.Background()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		t.Fatalf("neo4j verify: %v", err)
	}
	t.Cleanup(func() {
		// Clean up test data
		sess := driver.NewSession(ctx, neo4j.SessionConfig{})
		sess.Run(ctx, "MATCH (n) DETACH DELETE n", nil)
		sess.Close(ctx)
		driver.Close(ctx)
	})
	return driver
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestNeo4j_SaveAndGetComponent(t *testing.T) {
	driver := testDriver(t)
	store := New(driver)
	ctx := context.Background()

	comp := Component{
		ID:      "test-ecu-1",
		Name:    "Engine ECU",
		Type:    "ecu",
		Vehicle: "2020-Toyota-Camry",
		Properties: map[string]string{
			"location": "engine bay",
		},
	}

	if err := store.SaveComponent(ctx, comp); err != nil {
		t.Fatalf("SaveComponent: %v", err)
	}

	got, err := store.GetComponent(ctx, "test-ecu-1")
	if err != nil {
		t.Fatalf("GetComponent: %v", err)
	}

	if got.ID != comp.ID || got.Name != comp.Name || got.Type != comp.Type {
		t.Fatalf("mismatch: got %+v", got)
	}
	if got.Properties["location"] != "engine bay" {
		t.Fatalf("props mismatch: %v", got.Properties)
	}
}

func TestNeo4j_SaveEdge(t *testing.T) {
	driver := testDriver(t)
	store := New(driver)
	ctx := context.Background()

	store.SaveComponent(ctx, Component{ID: "a", Name: "Sensor A", Type: "sensor", Vehicle: "2020-Toyota-Camry"})
	store.SaveComponent(ctx, Component{ID: "b", Name: "ECU B", Type: "ecu", Vehicle: "2020-Toyota-Camry"})

	err := store.SaveEdge(ctx, Edge{ID: "e1", From: "a", To: "b", Type: "connects_to", Wire: "W101"})
	if err != nil {
		t.Fatalf("SaveEdge: %v", err)
	}
}

func TestNeo4j_Neighbors(t *testing.T) {
	driver := testDriver(t)
	store := New(driver)
	ctx := context.Background()

	// A -> B -> C
	store.SaveComponent(ctx, Component{ID: "n1", Name: "Node1", Type: "sensor", Vehicle: "2020-Honda-Civic"})
	store.SaveComponent(ctx, Component{ID: "n2", Name: "Node2", Type: "ecu", Vehicle: "2020-Honda-Civic"})
	store.SaveComponent(ctx, Component{ID: "n3", Name: "Node3", Type: "actuator", Vehicle: "2020-Honda-Civic"})
	store.SaveEdge(ctx, Edge{ID: "e1", From: "n1", To: "n2", Type: "connects_to"})
	store.SaveEdge(ctx, Edge{ID: "e2", From: "n2", To: "n3", Type: "connects_to"})

	// Depth 1 from n1 should get n2 only
	neighbors, err := store.Neighbors(ctx, "n1", 1)
	if err != nil {
		t.Fatalf("Neighbors depth 1: %v", err)
	}
	if len(neighbors) != 1 || neighbors[0].ID != "n2" {
		t.Fatalf("expected [n2], got %v", neighbors)
	}

	// Depth 2 from n1 should get n2 and n3
	neighbors, err = store.Neighbors(ctx, "n1", 2)
	if err != nil {
		t.Fatalf("Neighbors depth 2: %v", err)
	}
	if len(neighbors) != 2 {
		t.Fatalf("expected 2 neighbors, got %d", len(neighbors))
	}
}

func TestNeo4j_FindByVehicle(t *testing.T) {
	driver := testDriver(t)
	store := New(driver)
	ctx := context.Background()

	store.SaveComponent(ctx, Component{ID: "v1", Name: "Fuse", Type: "fuse", Vehicle: "2021-Ford-F150"})
	store.SaveComponent(ctx, Component{ID: "v2", Name: "Relay", Type: "relay", Vehicle: "2021-Ford-F150"})
	store.SaveComponent(ctx, Component{ID: "v3", Name: "Other", Type: "sensor", Vehicle: "2020-Toyota-Camry"})

	comps, err := store.FindByVehicle(ctx, 2021, "Ford", "F150")
	if err != nil {
		t.Fatalf("FindByVehicle: %v", err)
	}
	if len(comps) != 2 {
		t.Fatalf("expected 2, got %d", len(comps))
	}
}

func TestNeo4j_TracePath(t *testing.T) {
	driver := testDriver(t)
	store := New(driver)
	ctx := context.Background()

	// A -> B -> C
	store.SaveComponent(ctx, Component{ID: "p1", Name: "Start", Type: "sensor"})
	store.SaveComponent(ctx, Component{ID: "p2", Name: "Mid", Type: "ecu"})
	store.SaveComponent(ctx, Component{ID: "p3", Name: "End", Type: "actuator"})
	store.SaveEdge(ctx, Edge{ID: "pe1", From: "p1", To: "p2", Type: "connects_to"})
	store.SaveEdge(ctx, Edge{ID: "pe2", From: "p2", To: "p3", Type: "connects_to"})

	path, err := store.TracePath(ctx, "p1", "p3")
	if err != nil {
		t.Fatalf("TracePath: %v", err)
	}
	if len(path) != 3 {
		t.Fatalf("expected 3 nodes in path, got %d", len(path))
	}
	if path[0].ID != "p1" || path[2].ID != "p3" {
		t.Fatalf("unexpected path: %v", path)
	}
}

func TestNeo4j_SaveBatch(t *testing.T) {
	driver := testDriver(t)
	store := New(driver)
	ctx := context.Background()

	comps := []Component{
		{ID: "b1", Name: "Comp1", Type: "sensor", Vehicle: "2020-Toyota-Camry"},
		{ID: "b2", Name: "Comp2", Type: "ecu", Vehicle: "2020-Toyota-Camry"},
		{ID: "b3", Name: "Comp3", Type: "actuator", Vehicle: "2020-Toyota-Camry"},
	}
	edges := []Edge{
		{ID: "be1", From: "b1", To: "b2", Type: "connects_to", Wire: "W1"},
		{ID: "be2", From: "b2", To: "b3", Type: "powers"},
	}

	if err := store.SaveBatch(ctx, comps, edges); err != nil {
		t.Fatalf("SaveBatch: %v", err)
	}

	// Verify all components saved
	for _, c := range comps {
		got, err := store.GetComponent(ctx, c.ID)
		if err != nil {
			t.Fatalf("GetComponent %s: %v", c.ID, err)
		}
		if got.Name != c.Name {
			t.Fatalf("expected %s, got %s", c.Name, got.Name)
		}
	}

	// Verify edges by checking neighbors
	neighbors, err := store.Neighbors(ctx, "b1", 2)
	if err != nil {
		t.Fatalf("Neighbors: %v", err)
	}
	if len(neighbors) != 2 {
		t.Fatalf("expected 2 neighbors, got %d", len(neighbors))
	}
}
