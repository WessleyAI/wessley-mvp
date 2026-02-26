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

// Make represents a vehicle manufacturer.
type Make struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// VehicleModel represents a specific model produced by a make.
type VehicleModel struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	MakeID string `json:"make_id"`
}

// Generation represents a model generation/platform.
type Generation struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Platform  string `json:"platform"`
	StartYear int    `json:"start_year"`
	EndYear   int    `json:"end_year"`
	ModelID   string `json:"model_id"`
}

// Trim represents a specific trim level of a generation.
type Trim struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Engine       string `json:"engine"`
	Transmission string `json:"transmission"`
	Drivetrain   string `json:"drivetrain"`
	GenerationID string `json:"generation_id"`
}

// ModelYear represents a specific year of a vehicle make/model/trim.
type ModelYear struct {
	ID    string `json:"id"`
	Year  int    `json:"year"`
	Make  string `json:"make"`
	Model string `json:"model"`
	Trim  string `json:"trim"`
}

// System represents a vehicle system (e.g. Engine, Brakes).
type System struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Subsystem represents a subsystem within a system.
type Subsystem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	SystemID string `json:"system_id"`
}

// VehicleInfo holds vehicle identification extracted from content.
type VehicleInfo struct {
	Make  string `json:"make"`
	Model string `json:"model"`
	Year  int    `json:"year"`
	Trim  string `json:"trim,omitempty"`
}
