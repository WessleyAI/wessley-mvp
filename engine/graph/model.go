// Package graph provides Neo4j knowledge graph operations for automotive electrical data.
package graph

// Component represents an electrical component in a vehicle's wiring system.
type Component struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"` // ecu, sensor, actuator, connector, wire, fuse, relay
	Vehicle    string            `json:"vehicle"`
	Properties map[string]string `json:"properties"`
}

// Edge represents a relationship between two components.
type Edge struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"` // connects_to, part_of, powers, grounds
	Wire string `json:"wire,omitempty"`
}

// Vehicle represents a vehicle make/model/year.
type Vehicle struct {
	ID    string `json:"id"`
	Year  int    `json:"year"`
	Make  string `json:"make"`
	Model string `json:"model"`
}
