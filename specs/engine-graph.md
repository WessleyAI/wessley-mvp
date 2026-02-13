# Spec: engine-graph — Neo4j Knowledge Graph

**Branch:** `spec/engine-graph`
**Effort:** 3-4 days
**Priority:** P1 — Phase 2

---

## Scope

Neo4j knowledge graph operations for storing and querying automotive electrical data. Uses `pkg/repo` generic repository.

### Files

```
engine/graph/model.go      # Domain types (Component, Wire, Vehicle, Edge)
engine/graph/repo.go       # Neo4j repository using pkg/repo
engine/graph/graph.go      # Graph operations (neighbors, traversal, search)
engine/graph/graph_test.go
```

## Key Types

```go
type Component struct {
    ID           string            `json:"id"`
    Name         string            `json:"name"`
    Type         string            `json:"type"` // ecu, sensor, actuator, connector, wire, fuse, relay
    Vehicle      string            `json:"vehicle"`
    Properties   map[string]string `json:"properties"`
}

type Edge struct {
    ID       string `json:"id"`
    From     string `json:"from"`
    To       string `json:"to"`
    Type     string `json:"type"` // connects_to, part_of, powers, grounds
    Wire     string `json:"wire,omitempty"`
}

type Vehicle struct {
    ID    string `json:"id"`
    Year  int    `json:"year"`
    Make  string `json:"make"`
    Model string `json:"model"`
}
```

## Operations

```go
type GraphStore struct {
    driver neo4j.DriverWithContext
    components *repo.Neo4jRepo[Component, string]
}

func New(driver neo4j.DriverWithContext) *GraphStore

// CRUD via generic repo
func (g *GraphStore) GetComponent(ctx context.Context, id string) (Component, error)
func (g *GraphStore) SaveComponent(ctx context.Context, c Component) error
func (g *GraphStore) SaveEdge(ctx context.Context, e Edge) error

// Graph queries
func (g *GraphStore) Neighbors(ctx context.Context, nodeID string, depth int) ([]Component, error)
func (g *GraphStore) FindByVehicle(ctx context.Context, year int, make, model string) ([]Component, error)
func (g *GraphStore) FindByType(ctx context.Context, componentType string) ([]Component, error)
func (g *GraphStore) TracePath(ctx context.Context, fromID, toID string) ([]Component, error)

// Bulk
func (g *GraphStore) SaveBatch(ctx context.Context, components []Component, edges []Edge) error
```

## Acceptance Criteria

- [ ] Generic repo for Component CRUD
- [ ] Cypher-based neighbor traversal with configurable depth
- [ ] Vehicle-scoped queries
- [ ] Path tracing between components
- [ ] Batch save for ingestion pipeline
- [ ] Transaction support for batch operations
- [ ] Unit tests with testcontainers or mock

## Dependencies

- `pkg/repo` (Neo4j generic implementation)
- Neo4j 5 Community (Docker)

## Reference

- FINAL_ARCHITECTURE.md §8.3
